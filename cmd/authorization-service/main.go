package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"

	"github.com/company/go-fintech/internal/config"
	"github.com/company/go-fintech/internal/health"
	"github.com/company/go-fintech/internal/metrics"
)

type Transaction struct {
	TransactionID   string  `json:"transaction_id"`
	IdempotencyKey  string  `json:"idempotency_key"`
	TransactionType string  `json:"transaction_type"`
	Status          string  `json:"status"`
	SourceAccountID string  `json:"source_account_id"`
	TargetAccountID string  `json:"target_account_id"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	FeeAmount       float64 `json:"fee_amount"`
}

type FraudResult struct {
	FraudScore float64 `json:"fraud_score"`
	RiskLevel  string  `json:"risk_level"`
}

type AuthorizationMessage struct {
	Transaction Transaction `json:"transaction"`
	FraudResult FraudResult `json:"fraud_result"`
}

var redisClient *redis.Client

var placeHoldScript = redis.NewScript(`
local current = tonumber(redis.call('GET', KEYS[1]))
if current == nil then
    return -1
end
local amount = tonumber(ARGV[1])
local holdTTL = tonumber(ARGV[2])
if current < amount then
    return 0
end
redis.call('DECRBYFLOAT', KEYS[1], amount)
redis.call('SET', KEYS[2], ARGV[1], 'EX', holdTTL)
return 1
`)

var ErrBalanceUnavailable = errors.New("balance unavailable")
var ErrInsufficientFunds = errors.New("insufficient funds")

func main() {
	log.Println("Authorization Service starting...")

	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	if len(cfg.Kafka.Brokers) == 0 {
		log.Fatalf("kafka brokers must be configured (KAFKA_BROKERS env or config file)")
	}
	if cfg.Redis.Addr == "" {
		log.Fatalf("redis address must be configured (REDIS_ADDR env or config file)")
	}
	if cfg.Kafka.ConsumerGroup == "" {
		cfg.Kafka.ConsumerGroup = "authorization-service"
	}

	m := metrics.New("authorization-service")
	_ = m

	redisClient = redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-sigCh
		log.Printf("Received signal %s, shutting down...", s)
		cancel()
	}()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis not available: %v", err)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Kafka.Brokers,
		Topic:    cfg.Kafka.Topics.TransactionsAuthorized,
		GroupID:  cfg.Kafka.ConsumerGroup,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	settledWriter := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Kafka.Brokers...),
		Topic:    cfg.Kafka.Topics.TransactionsSettled,
		Balancer: &kafka.Hash{},
	}
	defer settledWriter.Close()

	failedWriter := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Kafka.Brokers...),
		Topic:    cfg.Kafka.Topics.TransactionsFailed,
		Balancer: &kafka.Hash{},
	}
	defer failedWriter.Close()

	addr := fmt.Sprintf(":%d", cfg.App.Port)
	if cfg.App.Port == 0 {
		addr = ":8083"
	}
	srv := health.New("authorization-service", addr)
	defer srv.Shutdown(ctx)
	go func() {
		log.Printf("Health server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Health server error: %v", err)
		}
	}()
	srv.SetReady(true)

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

		var authMsg AuthorizationMessage
		if err := json.Unmarshal(msg.Value, &authMsg); err != nil {
			log.Printf("Error unmarshaling: %v", err)
			reader.CommitMessages(ctx, msg)
			continue
		}

		result := authorizeTransaction(ctx, &authMsg.Transaction)

		if result.Approved {
			produceSettled(ctx, settledWriter, &authMsg.Transaction)
		} else {
			produceFailed(ctx, failedWriter, &authMsg.Transaction, result.Reason)
		}

		reader.CommitMessages(ctx, msg)
	}
}

type AuthorizationResult struct {
	Approved bool
	Reason   string
}

func authorizeTransaction(ctx context.Context, tx *Transaction) AuthorizationResult {
	if isProcessed(ctx, tx.TransactionID) {
		return AuthorizationResult{Approved: false, Reason: "DUPLICATE_TRANSACTION"}
	}

	balance, err := getAccountBalance(ctx, tx.SourceAccountID)
	if err != nil {
		if errors.Is(err, ErrBalanceUnavailable) {
			return AuthorizationResult{Approved: false, Reason: "BALANCE_UNAVAILABLE"}
		}
		return AuthorizationResult{Approved: false, Reason: "BALANCE_LOOKUP_ERROR"}
	}

	totalAmount := tx.Amount + tx.FeeAmount
	if balance < totalAmount {
		return AuthorizationResult{Approved: false, Reason: "INSUFFICIENT_FUNDS"}
	}

	holdOK, err := placeHold(ctx, tx.SourceAccountID, totalAmount, tx.TransactionID)
	if err != nil {
		return AuthorizationResult{Approved: false, Reason: "HOLD_ERROR"}
	}
	if !holdOK {
		return AuthorizationResult{Approved: false, Reason: "INSUFFICIENT_FUNDS"}
	}

	if err := markProcessed(ctx, tx.TransactionID); err != nil {
		log.Printf("Warning: failed to mark transaction %s as processed: %v", tx.TransactionID, err)
	}

	return AuthorizationResult{Approved: true}
}

func getAccountBalance(ctx context.Context, accountID string) (float64, error) {
	key := fmt.Sprintf("balance:available:%s", accountID)
	val, err := redisClient.Get(ctx, key).Float64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, ErrBalanceUnavailable
		}
		return 0, fmt.Errorf("redis get balance: %w", err)
	}
	return val, nil
}

func placeHold(ctx context.Context, accountID string, amount float64, txID string) (bool, error) {
	balanceKey := fmt.Sprintf("balance:available:%s", accountID)
	holdKey := fmt.Sprintf("hold:%s:%s", accountID, txID)
	holdTTL := int((24 * time.Hour).Seconds())

	res, err := placeHoldScript.Run(ctx, redisClient,
		[]string{balanceKey, holdKey},
		amount, holdTTL,
	).Int()
	if err != nil {
		return false, fmt.Errorf("placeHold script: %w", err)
	}
	switch res {
	case 1:
		return true, nil
	case 0:
		return false, nil
	default:
		return false, ErrBalanceUnavailable
	}
}

func isProcessed(ctx context.Context, txID string) bool {
	key := fmt.Sprintf("tx:processed:%s", txID)
	val, err := redisClient.Get(ctx, key).Result()
	return err == nil && val == "1"
}

func markProcessed(ctx context.Context, txID string) error {
	key := fmt.Sprintf("tx:processed:%s", txID)
	return redisClient.Set(ctx, key, "1", 24*time.Hour).Err()
}

func produceSettled(ctx context.Context, w *kafka.Writer, tx *Transaction) {
	tx.Status = "SETTLED"
	data, err := json.Marshal(tx)
	if err != nil {
		log.Printf("Failed to marshal settled transaction %s: %v", tx.TransactionID, err)
		return
	}
	if err := w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(tx.SourceAccountID),
		Value: data,
	}); err != nil {
		log.Printf("Failed to produce settled message for %s: %v", tx.TransactionID, err)
		return
	}
	log.Printf("Transaction %s authorized and settled", tx.TransactionID)
}

func produceFailed(ctx context.Context, w *kafka.Writer, tx *Transaction, reason string) {
	tx.Status = "FAILED"
	data, err := json.Marshal(map[string]interface{}{
		"transaction":    tx,
		"failure_reason": reason,
	})
	if err != nil {
		log.Printf("Failed to marshal failed transaction %s: %v", tx.TransactionID, err)
		return
	}
	if err := w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(tx.SourceAccountID),
		Value: data,
	}); err != nil {
		log.Printf("Failed to produce failed message for %s: %v", tx.TransactionID, err)
		return
	}
	log.Printf("Transaction %s failed: %s", tx.TransactionID, reason)
}
