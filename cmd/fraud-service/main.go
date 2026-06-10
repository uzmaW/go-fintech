package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/attribute"

	"github.com/company/go-fintech/internal/config"
	"github.com/company/go-fintech/internal/health"
	fintechkafka "github.com/company/go-fintech/internal/kafka"
	"github.com/company/go-fintech/internal/metrics"
	"github.com/company/go-fintech/internal/tracing"
)

type Transaction struct {
	TransactionID    string  `json:"transaction_id"`
	IdempotencyKey   string  `json:"idempotency_key"`
	TransactionType  string  `json:"transaction_type"`
	Status           string  `json:"status"`
	SourceAccountID  string  `json:"source_account_id"`
	TargetAccountID  string  `json:"target_account_id"`
	Amount           float64 `json:"amount"`
	Currency         string  `json:"currency"`
	MerchantCategory string  `json:"merchant_category"`
	InitiatedAt      string  `json:"initiated_at"`
}

type FraudResult struct {
	TransactionID  string  `json:"transaction_id"`
	FraudScore     float64 `json:"fraud_score"`
	RiskLevel      string  `json:"risk_level"`
	Recommendation string  `json:"recommendation"`
}

var avroSer *fintechkafka.AvroSerializer

func main() {
	log.Println("Fraud Detection Service starting...")

	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	if len(cfg.Kafka.Brokers) == 0 {
		log.Fatalf("kafka brokers must be configured (KAFKA_BROKERS env or config file)")
	}
	if cfg.Kafka.ConsumerGroup == "" {
		cfg.Kafka.ConsumerGroup = "fraud-service"
	}

	if cfg.Tracing.Enabled {
		shutdown, err := tracing.Init(cfg.App.Name, cfg.Tracing.CollectorURL)
		if err != nil {
			log.Printf("Warning: failed to init tracing: %v", err)
		} else {
			defer shutdown()
		}
	}

	if cfg.Schema.SchemasDir != "" {
		var err error
		avroSer, err = fintechkafka.NewAvroSerializer(cfg.Schema.SchemasDir)
		if err != nil {
			log.Printf("Warning: failed to load Avro schemas: %v", err)
		}
	}

	m := metrics.New("fraud-service")
	_ = m

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-sigCh
		log.Printf("Received signal %s, shutting down...", s)
		cancel()
	}()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Kafka.Brokers,
		Topic:    cfg.Kafka.Topics.TransactionsPending,
		GroupID:  cfg.Kafka.ConsumerGroup,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer func() { _ = reader.Close() }()

	authorizedWriter := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Kafka.Brokers...),
		Topic:    cfg.Kafka.Topics.TransactionsAuthorized,
		Balancer: &kafka.Hash{},
	}
	defer func() { _ = authorizedWriter.Close() }()

	failedWriter := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Kafka.Brokers...),
		Topic:    cfg.Kafka.Topics.TransactionsFailed,
		Balancer: &kafka.Hash{},
	}
	defer func() { _ = failedWriter.Close() }()

	fraudWriter := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Kafka.Brokers...),
		Topic:    cfg.Kafka.Topics.FraudEvents,
		Balancer: &kafka.Hash{},
	}
	defer func() { _ = fraudWriter.Close() }()

	if cfg.App.Port == 0 {
		cfg.App.Port = 8081
	}
	healthSrv := health.New("fraud-service", fmt.Sprintf(":%d", cfg.App.Port))
	defer func() { _ = healthSrv.Shutdown(ctx) }()
	go func() {
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Health server error: %v", err)
		}
	}()
	healthSrv.SetReady(true)

	metricsAddr := fmt.Sprintf(":%d", cfg.App.Port+100)
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", metrics.Handler())
		log.Printf("Metrics server on %s", metricsAddr)
		if err := http.ListenAndServe(metricsAddr, mux); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("Error fetching message: %v", err)
			continue
		}

		var tx Transaction
		if err := json.Unmarshal(msg.Value, &tx); err != nil {
			log.Printf("Error unmarshaling transaction: %v", err)
			if err := reader.CommitMessages(ctx, msg); err != nil {
				log.Printf("Error committing messages: %v", err)
			}
			continue
		}

		processCtx, span := tracing.Tracer("fraud-service").Start(ctx, "EvaluateFraud")
		span.SetAttributes(
			attribute.Float64("transaction.amount", tx.Amount),
		)

		result := evaluateFraud(&tx)
		span.SetAttributes(
			attribute.Float64("fraud.score", result.FraudScore),
			attribute.String("fraud.risk_level", result.RiskLevel),
			attribute.String("fraud.recommendation", result.Recommendation),
		)
		log.Printf("Transaction %s: fraud_score=%.2f, risk=%s",
			tx.TransactionID, result.FraudScore, result.RiskLevel)

		var writeErr error
		switch result.Recommendation {
		case "APPROVE":
			writeErr = produceAuthorized(processCtx, authorizedWriter, &tx, result)
		case "DECLINE":
			writeErr = produceFailed(processCtx, failedWriter, &tx, result)
		case "REVIEW":
			writeErr = produceFraudEvent(processCtx, fraudWriter, &tx, result)
		}

		if writeErr != nil {
			span.RecordError(writeErr)
			log.Printf("Error producing message for tx %s, committing to avoid reprocess: %v", tx.TransactionID, writeErr)
			span.End()
			if err := reader.CommitMessages(ctx, msg); err != nil {
				log.Printf("Error committing message: %v", err)
			}
			continue
		}

		span.End()

		if err := reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("Error committing message: %v", err)
		}
	}
}

func evaluateFraud(tx *Transaction) FraudResult {
	fraudScore := calculateFraudScore(tx)

	var riskLevel string
	var recommendation string

	switch {
	case fraudScore < 50:
		riskLevel = "LOW"
		recommendation = "APPROVE"
	case fraudScore < 80:
		riskLevel = "MEDIUM"
		recommendation = "REVIEW"
	default:
		riskLevel = "HIGH"
		recommendation = "DECLINE"
	}

	return FraudResult{
		TransactionID:  tx.TransactionID,
		FraudScore:     fraudScore,
		RiskLevel:      riskLevel,
		Recommendation: recommendation,
	}
}

func calculateFraudScore(tx *Transaction) float64 {
	score := rand.Float64() * 30

	if tx.Amount > 10000 {
		score += 20
	}
	if tx.Amount > 50000 {
		score += 25
	}

	switch tx.MerchantCategory {
	case "5411", "5812", "5814":
		score += 15
	case "7995", "7832", "7922":
		score += 10
	}

	return score
}

func produceAuthorized(ctx context.Context, w *kafka.Writer, tx *Transaction, result FraudResult) error {
	tx.Status = "AUTHORIZED"
	payload := map[string]interface{}{
		"transaction":  tx,
		"fraud_result": result,
	}
	if avroSer != nil {
		if avroData, err := avroSer.Serialize("Transaction", map[string]interface{}{
			"transaction_id":    tx.TransactionID,
			"idempotency_key":   tx.IdempotencyKey,
			"transaction_type":  tx.TransactionType,
			"status":            tx.Status,
			"source_account_id": tx.SourceAccountID,
			"target_account_id": tx.TargetAccountID,
			"amount":            tx.Amount,
			"currency":          tx.Currency,
			"fee_amount":        0.0,
			"initiated_at":      tx.InitiatedAt,
			"created_at":        tx.InitiatedAt,
		}); err == nil {
			payload["transaction_avro"] = avroData
		}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(tx.SourceAccountID),
		Value: data,
	})
}

func produceFailed(ctx context.Context, w *kafka.Writer, tx *Transaction, result FraudResult) error {
	tx.Status = "FAILED"
	payload := map[string]interface{}{
		"transaction":    tx,
		"fraud_result":   result,
		"failure_reason": "FRAUD_DETECTED",
	}
	if avroSer != nil {
		if avroData, err := avroSer.Serialize("Transaction", map[string]interface{}{
			"transaction_id":    tx.TransactionID,
			"idempotency_key":   tx.IdempotencyKey,
			"transaction_type":  tx.TransactionType,
			"status":            tx.Status,
			"source_account_id": tx.SourceAccountID,
			"target_account_id": tx.TargetAccountID,
			"amount":            tx.Amount,
			"currency":          tx.Currency,
			"fee_amount":        0.0,
			"initiated_at":      tx.InitiatedAt,
			"created_at":        tx.InitiatedAt,
		}); err == nil {
			payload["transaction_avro"] = avroData
		}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(tx.SourceAccountID),
		Value: data,
	})
}

func produceFraudEvent(ctx context.Context, w *kafka.Writer, tx *Transaction, result FraudResult) error {
	payload := map[string]interface{}{
		"transaction":     tx,
		"fraud_result":    result,
		"requires_review": true,
	}
	if avroSer != nil {
		if avroData, err := avroSer.Serialize("Transaction", map[string]interface{}{
			"transaction_id":    tx.TransactionID,
			"idempotency_key":   tx.IdempotencyKey,
			"transaction_type":  tx.TransactionType,
			"status":            tx.Status,
			"source_account_id": tx.SourceAccountID,
			"target_account_id": tx.TargetAccountID,
			"amount":            tx.Amount,
			"currency":          tx.Currency,
			"fee_amount":        0.0,
			"initiated_at":      tx.InitiatedAt,
			"created_at":        tx.InitiatedAt,
		}); err == nil {
			payload["transaction_avro"] = avroData
		}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(tx.SourceAccountID),
		Value: data,
	})
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}
