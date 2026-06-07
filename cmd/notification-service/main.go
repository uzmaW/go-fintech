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
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"

	"github.com/company/go-fintech/internal/config"
	"github.com/company/go-fintech/internal/metrics"
)

type NotificationMessage struct {
	Type        string                 `json:"type"`
	UserID      string                 `json:"user_id"`
	Channel     string                 `json:"channel"`
	TemplateID  string                 `json:"template_id"`
	Data        map[string]interface{} `json:"data"`
	Priority    string                 `json:"priority"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
}

type OutboxMessage struct {
	MessageID    string              `json:"message_id"`
	Notification NotificationMessage `json:"notification"`
	Status       string              `json:"status"`
	CreatedAt    time.Time           `json:"created_at"`
	SentAt       *time.Time          `json:"sent_at,omitempty"`
	FailedAt     *time.Time          `json:"failed_at,omitempty"`
	Error        string              `json:"error,omitempty"`
}

func main() {
	log.Println("Notification Service starting...")

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
	if cfg.App.Port == 0 {
		cfg.App.Port = 8082
	}
	if cfg.Kafka.ConsumerGroup == "" {
		cfg.Kafka.ConsumerGroup = "notification-service"
	}

	m := metrics.New("notification-service")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis not available: %v", err)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Kafka.Brokers,
		Topic:    cfg.Kafka.Topics.NotificationsOutbox,
		GroupID:  cfg.Kafka.ConsumerGroup,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	sentWriter := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Kafka.Brokers...),
		Topic:    cfg.Kafka.Topics.NotificationsSent,
		Balancer: &kafka.Hash{},
	}
	defer sentWriter.Close()

	failedWriter := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Kafka.Brokers...),
		Topic:    cfg.Kafka.Topics.NotificationsFailed,
		Balancer: &kafka.Hash{},
	}
	defer failedWriter.Close()

	writer := sentWriter

	r := gin.Default()
	r.Use(metrics.PrometheusMiddleware(m))

	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": cfg.App.Name,
		})
	})

	r.POST("/notifications/send", func(c *gin.Context) {
		var notif NotificationMessage
		if err := c.ShouldBindJSON(&notif); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := enqueueNotification(ctx, writer, &notif); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue"})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{"status": "queued"})
	})

	go processNotifications(ctx, reader, sentWriter, failedWriter, redisClient)

	addr := fmt.Sprintf(":%d", cfg.App.Port)
	log.Printf("Notification Service running on %s", addr)
	srv := &http.Server{Addr: addr, Handler: r}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
		os.Exit(0)
	}()

	<-ctx.Done()
}

func enqueueNotification(ctx context.Context, writer *kafka.Writer, notif *NotificationMessage) error {
	data, err := json.Marshal(notif)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(notif.UserID),
		Value: data,
		Time:  time.Now(),
	}

	return writer.WriteMessages(ctx, msg)
}

func processNotifications(ctx context.Context, reader *kafka.Reader, sentWriter, failedWriter *kafka.Writer, redis *redis.Client) {
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("Error fetching message: %v", err)
			continue
		}

		var notif NotificationMessage
		if err := json.Unmarshal(msg.Value, &notif); err != nil {
			log.Printf("Error unmarshaling: %v", err)
			reader.CommitMessages(ctx, msg)
			continue
		}

		if err := sendNotification(ctx, redis, &notif); err != nil {
			log.Printf("Notification send failed for user=%s type=%s: %v", notif.UserID, notif.Type, err)
			publishResult(ctx, failedWriter, &notif, err.Error())
		} else {
			publishResult(ctx, sentWriter, &notif, "")
		}

		reader.CommitMessages(ctx, msg)
	}
}

func sendNotification(ctx context.Context, redis *redis.Client, notif *NotificationMessage) error {
	log.Printf("Sending %s notification to %s via %s",
		notif.Type, notif.UserID, notif.Channel)

	key := "notification:processed:" + notif.UserID + ":" + notif.Type
	if err := redis.Set(ctx, key, "1", 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

func publishResult(ctx context.Context, w *kafka.Writer, notif *NotificationMessage, errMsg string) {
	out := OutboxMessage{
		MessageID:    uuid.NewString(),
		Notification: *notif,
		CreatedAt:    time.Now().UTC(),
	}
	now := time.Now().UTC()
	if errMsg != "" {
		out.Status = "FAILED"
		out.FailedAt = &now
		out.Error = errMsg
	} else {
		out.Status = "SENT"
		out.SentAt = &now
	}
	data, err := json.Marshal(out)
	if err != nil {
		log.Printf("Failed to marshal outbox message: %v", err)
		return
	}
	if err := w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(notif.UserID),
		Value: data,
	}); err != nil {
		log.Printf("Failed to publish result to %s: %v", w.Topic, err)
	}
}
