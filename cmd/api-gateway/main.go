package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"

	"github.com/company/go-fintech/internal/config"
	fintechkafka "github.com/company/go-fintech/internal/kafka"
	"github.com/company/go-fintech/internal/metrics"
	"github.com/company/go-fintech/internal/tracing"
)

type TransactionRequest struct {
	IdempotencyKey   string  `json:"idempotency_key" binding:"required"`
	TransactionType  string  `json:"transaction_type" binding:"required"`
	SourceAccountID  string  `json:"source_account_id" binding:"required"`
	TargetAccountID  string  `json:"target_account_id"`
	Amount           float64 `json:"amount" binding:"required"`
	Currency         string  `json:"currency" binding:"required"`
	FeeAmount        float64 `json:"fee_amount"`
	MerchantID       string  `json:"merchant_id"`
	MerchantCategory string  `json:"merchant_category"`
	CardNumberHash   string  `json:"card_number_hash"`
	CardLast4        string  `json:"card_last4"`
	InitiatedBy      string  `json:"initiated_by"`
}

type TransactionResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
	Message       string `json:"message,omitempty"`
}

var producer *kafkago.Writer
var avroSerializer *fintechkafka.AvroSerializer
var serviceConfig *config.Config

func main() {
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	serviceConfig = cfg
	if serviceConfig.App.Name == "" {
		serviceConfig.App.Name = "api-gateway"
	}
	if serviceConfig.App.Port == 0 {
		serviceConfig.App.Port = 8080
	}
	if len(serviceConfig.Kafka.Brokers) == 0 {
		log.Fatalf("kafka brokers must be configured (KAFKA_BROKERS env or config file)")
	}

	if serviceConfig.Tracing.Enabled {
		shutdown, err := tracing.Init(serviceConfig.App.Name, serviceConfig.Tracing.CollectorURL)
		if err != nil {
			log.Printf("Warning: failed to init tracing: %v", err)
		} else {
			defer shutdown()
		}
	}

	if serviceConfig.Schema.SchemasDir != "" {
		var err error
		avroSerializer, err = fintechkafka.NewAvroSerializer(serviceConfig.Schema.SchemasDir)
		if err != nil {
			log.Printf("Warning: failed to load Avro schemas: %v", err)
		}
	}

	producer = &kafkago.Writer{
		Addr:         kafkago.TCP(serviceConfig.Kafka.Brokers...),
		Topic:        serviceConfig.Kafka.Topics.TransactionsPending,
		Balancer:     &kafkago.Hash{},
		BatchSize:    serviceConfig.Kafka.Producer.BatchSize,
		BatchTimeout: time.Duration(serviceConfig.Kafka.Producer.LingerMs) * time.Millisecond,
	}
	defer producer.Close()

	m := metrics.New("api-gateway")

	r := gin.Default()

	r.Use(metrics.PrometheusMiddleware(m))
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	r.POST("/api/v1/transactions", CreateTransaction)
	r.POST("/api/v1/transfers", CreateTransfer)
	r.GET("/api/v1/transactions/:id", GetTransaction)
	r.GET("/api/v1/health", HealthCheck)

	addr := fmt.Sprintf(":%d", serviceConfig.App.Port)
	log.Printf("%s starting on %s (env=%s)", serviceConfig.App.Name, addr, serviceConfig.App.Env)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
		os.Exit(1)
	}
}

func CreateTransaction(c *gin.Context) {
	var req TransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, TransactionResponse{Status: "ERROR", Message: err.Error()})
		return
	}
	createTransaction(c, &req)
}

func createTransaction(c *gin.Context, req *TransactionRequest) {
	ctx, span := tracing.Tracer("api-gateway").Start(c.Request.Context(), "CreateTransaction")
	defer span.End()

	txID := uuid.New().String()
	tx := map[string]interface{}{
		"transaction_id":    txID,
		"idempotency_key":   req.IdempotencyKey,
		"transaction_type":  req.TransactionType,
		"source_account_id": req.SourceAccountID,
		"target_account_id": req.TargetAccountID,
		"amount":            req.Amount,
		"currency":          req.Currency,
		"fee_amount":        req.FeeAmount,
		"merchant_id":       req.MerchantID,
		"merchant_category": req.MerchantCategory,
		"card_number_hash":  req.CardNumberHash,
		"card_last4":        req.CardLast4,
		"initiated_by":      req.InitiatedBy,
		"initiated_at":      time.Now().UTC(),
		"status":            "PENDING",
	}

	data, err := json.Marshal(tx)
	if err != nil {
		span.RecordError(err)
		c.JSON(http.StatusInternalServerError, TransactionResponse{
			Status:  "ERROR",
			Message: "Failed to serialize transaction",
		})
		return
	}

	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	avroData := data
	if avroSerializer != nil {
		if serialized, err := avroSerializer.Serialize("Transaction", tx); err == nil {
			avroData = serialized
		}
	}

	if err := producer.WriteMessages(writeCtx, kafkago.Message{
		Key:   []byte(req.SourceAccountID),
		Value: avroData,
	}); err != nil {
		span.RecordError(err)
		c.JSON(http.StatusInternalServerError, TransactionResponse{
			Status:  "ERROR",
			Message: "Failed to queue transaction",
		})
		return
	}

	c.JSON(http.StatusAccepted, TransactionResponse{
		TransactionID: txID,
		Status:        "PENDING",
		Message:       "Transaction queued for processing",
	})
}

func CreateTransfer(c *gin.Context) {
	var req TransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, TransactionResponse{Status: "ERROR", Message: err.Error()})
		return
	}

	if req.SourceAccountID == "" || req.TargetAccountID == "" {
		c.JSON(http.StatusBadRequest, TransactionResponse{
			Status:  "ERROR",
			Message: "Both source and target account IDs are required for transfers",
		})
		return
	}

	req.TransactionType = "TRANSFER"
	createTransaction(c, &req)
}

func GetTransaction(c *gin.Context) {
	txID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"transaction_id": txID,
		"status":         "PENDING",
	})
}

func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   serviceConfig.App.Name,
		"env":       serviceConfig.App.Env,
		"timestamp": time.Now().UTC(),
	})
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}
