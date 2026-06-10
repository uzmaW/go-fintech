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

	"github.com/segmentio/kafka-go"

	"github.com/company/go-fintech/internal/config"
	"github.com/company/go-fintech/internal/db"
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
	InitiatedAt     string  `json:"initiated_at"`
}

type LedgerEntry struct {
	TransactionID  string  `json:"transaction_id"`
	AccountID      string  `json:"account_id"`
	EntryType      string  `json:"entry_type"`
	Amount         float64 `json:"amount"`
	Currency       string  `json:"currency"`
	PostedAt       string  `json:"posted_at"`
	IdempotencyKey string  `json:"idempotency_key"`
}

type DoubleEntryBatch struct {
	TransactionID string      `json:"transaction_id"`
	DebitEntry    LedgerEntry `json:"debit_entry"`
	CreditEntry   LedgerEntry `json:"credit_entry"`
}

type pendingTx struct {
	msg kafka.Message
	tx  Transaction
}

var pool *db.Pool

func main() {
	log.Println("Ledger Processor Service starting...")

	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	if len(cfg.Kafka.Brokers) == 0 {
		log.Fatalf("kafka brokers must be configured (KAFKA_BROKERS env or config file)")
	}
	if cfg.Database.Host == "" {
		log.Fatalf("database host must be configured (DATABASE_HOST env or config file)")
	}
	if cfg.Kafka.ConsumerGroup == "" {
		cfg.Kafka.ConsumerGroup = "ledger-processor"
	}

	m := metrics.New("ledger-processor")
	_ = m

	pool, err = db.NewPoolFromConfig(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to ledger database")

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Kafka.Brokers,
		Topic:    cfg.Kafka.Topics.TransactionsSettled,
		GroupID:  cfg.Kafka.ConsumerGroup,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer func() { _ = reader.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-sigCh
		log.Printf("Received signal %s, shutting down...", s)
		cancel()
	}()

	healthAddr := fmt.Sprintf(":%d", cfg.App.Port)
	if cfg.App.Port == 0 {
		healthAddr = ":8084"
	}
	hs := health.New("ledger-processor", healthAddr)
	defer func() { _ = hs.Shutdown(ctx) }()
	go func() {
		if err := hs.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Health server error: %v", err)
		}
	}()
	hs.SetReady(true)

	metricsAddr := fmt.Sprintf(":%d", cfg.App.Port+100)
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", metrics.Handler())
		log.Printf("Metrics server on %s", metricsAddr)
		if err := http.ListenAndServe(metricsAddr, mux); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	msgCh := make(chan kafka.Message, 500)
	errCh := make(chan error, 10)

	go func() {
		defer close(msgCh)
		for {
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				select {
				case errCh <- err:
				default:
					log.Printf("Error fetching message (dropped from errCh): %v", err)
				}
				continue
			}
			msgCh <- msg
		}
	}()

	const flushSize = 500
	const flushInterval = 100 * time.Millisecond

	batch := make([]pendingTx, 0, flushSize)
	batchTimer := time.NewTimer(flushInterval)
	defer batchTimer.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		processBatch(ctx, reader, batch)
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return

		case <-batchTimer.C:
			flush()
			batchTimer.Reset(flushInterval)

		case err := <-errCh:
			log.Printf("Error fetching message: %v", err)

		case msg, ok := <-msgCh:
			if !ok {
				flush()
				return
			}
			var tx Transaction
			if err := json.Unmarshal(msg.Value, &tx); err != nil {
				log.Printf("Error unmarshaling: %v", err)
				if err := reader.CommitMessages(ctx, msg); err != nil {
					log.Printf("Error committing messages: %v", err)
				}
				continue
			}
			batch = append(batch, pendingTx{msg: msg, tx: tx})

			if len(batch) >= flushSize {
				flush()
				batchTimer.Reset(flushInterval)
			}
		}
	}
}

func processBatch(ctx context.Context, reader *kafka.Reader, batch []pendingTx) {
	log.Printf("Processing batch of %d transactions", len(batch))

	successMsgs := make([]kafka.Message, 0, len(batch))
	for _, p := range batch {
		entries := createDoubleEntry(&p.tx)
		log.Printf("Created double-entry for transaction %s: DEBIT %s %.2f %s, CREDIT %s %.2f %s",
			p.tx.TransactionID,
			entries.DebitEntry.AccountID, entries.DebitEntry.Amount, entries.DebitEntry.Currency,
			entries.CreditEntry.AccountID, entries.CreditEntry.Amount, entries.CreditEntry.Currency)

		if err := writeLedgerEntries(ctx, &entries); err != nil {
			log.Printf("Error writing ledger entries for %s: %v", p.tx.TransactionID, err)
			continue
		}
		successMsgs = append(successMsgs, p.msg)
	}

	if len(successMsgs) == 0 {
		return
	}
	if err := reader.CommitMessages(ctx, successMsgs...); err != nil {
		log.Printf("Error committing offsets: %v", err)
		return
	}
	log.Printf("Batch committed to ledger: %d transactions", len(successMsgs))
}

func createDoubleEntry(tx *Transaction) DoubleEntryBatch {
	now := time.Now().UTC().Format(time.RFC3339)

	debitEntry := LedgerEntry{
		TransactionID:  tx.TransactionID,
		AccountID:      tx.SourceAccountID,
		EntryType:      "DEBIT",
		Amount:         tx.Amount + tx.FeeAmount,
		Currency:       tx.Currency,
		PostedAt:       now,
		IdempotencyKey: fmt.Sprintf("debit-%s", tx.TransactionID),
	}

	creditEntry := LedgerEntry{
		TransactionID:  tx.TransactionID,
		AccountID:      tx.TargetAccountID,
		EntryType:      "CREDIT",
		Amount:         tx.Amount,
		Currency:       tx.Currency,
		PostedAt:       now,
		IdempotencyKey: fmt.Sprintf("credit-%s", tx.TransactionID),
	}

	return DoubleEntryBatch{
		TransactionID: tx.TransactionID,
		DebitEntry:    debitEntry,
		CreditEntry:   creditEntry,
	}
}

func writeLedgerEntries(ctx context.Context, batch *DoubleEntryBatch) error {
	postedAt, err := time.Parse(time.RFC3339, batch.DebitEntry.PostedAt)
	if err != nil {
		return fmt.Errorf("parse posted_at: %w", err)
	}

	sourceBalance, err := pool.GetAccountBalance(ctx, batch.DebitEntry.AccountID)
	if err != nil {
		return fmt.Errorf("get source balance: %w", err)
	}
	newSourceBalance := sourceBalance - batch.DebitEntry.Amount

	targetBalance, err := pool.GetAccountBalance(ctx, batch.CreditEntry.AccountID)
	if err != nil {
		return fmt.Errorf("get target balance: %w", err)
	}
	newTargetBalance := targetBalance + batch.CreditEntry.Amount

	entries := []db.LedgerEntry{
		{
			TransactionID:  batch.DebitEntry.TransactionID,
			AccountID:      batch.DebitEntry.AccountID,
			EntryType:      batch.DebitEntry.EntryType,
			Amount:         batch.DebitEntry.Amount,
			Currency:       batch.DebitEntry.Currency,
			BalanceAfter:   newSourceBalance,
			PostedAt:       postedAt,
			IdempotencyKey: batch.DebitEntry.IdempotencyKey,
		},
		{
			TransactionID:  batch.CreditEntry.TransactionID,
			AccountID:      batch.CreditEntry.AccountID,
			EntryType:      batch.CreditEntry.EntryType,
			Amount:         batch.CreditEntry.Amount,
			Currency:       batch.CreditEntry.Currency,
			BalanceAfter:   newTargetBalance,
			PostedAt:       postedAt,
			IdempotencyKey: batch.CreditEntry.IdempotencyKey,
		},
	}

	return pool.BatchInsertLedgerEntries(ctx, entries)
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}
