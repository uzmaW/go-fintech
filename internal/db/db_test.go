package db

import (
	"testing"
	"time"
)

func TestNewPoolInvalidConfig(t *testing.T) {
	cfg := Config{
		Host:   "",
		Port:   5432,
		User:   "test",
		DBName: "testdb",
	}

	pool, err := NewPool(cfg)
	if err == nil {
		t.Fatal("expected error for empty host, got nil")
	}
	if pool != nil {
		t.Fatal("expected nil pool for invalid config")
	}
}

func TestLedgerEntryFields(t *testing.T) {
	now := time.Now()
	entry := LedgerEntry{
		TransactionID:  "txn-001",
		AccountID:      "acct-001",
		EntryType:      "DEBIT",
		Amount:         100.50,
		Currency:       "USD",
		BalanceAfter:   899.50,
		PostedAt:       now,
		IdempotencyKey: "idem-001",
		Metadata:       []byte(`{"key":"value"}`),
	}

	if entry.TransactionID != "txn-001" {
		t.Errorf("TransactionID = %s, want txn-001", entry.TransactionID)
	}
	if entry.AccountID != "acct-001" {
		t.Errorf("AccountID = %s, want acct-001", entry.AccountID)
	}
	if entry.EntryType != "DEBIT" {
		t.Errorf("EntryType = %s, want DEBIT", entry.EntryType)
	}
	if entry.Amount != 100.50 {
		t.Errorf("Amount = %f, want 100.50", entry.Amount)
	}
	if entry.Currency != "USD" {
		t.Errorf("Currency = %s, want USD", entry.Currency)
	}
	if entry.BalanceAfter != 899.50 {
		t.Errorf("BalanceAfter = %f, want 899.50", entry.BalanceAfter)
	}
	if !entry.PostedAt.Equal(now) {
		t.Errorf("PostedAt = %v, want %v", entry.PostedAt, now)
	}
	if entry.IdempotencyKey != "idem-001" {
		t.Errorf("IdempotencyKey = %s, want idem-001", entry.IdempotencyKey)
	}
	if string(entry.Metadata) != `{"key":"value"}` {
		t.Errorf("Metadata = %s, want {\"key\":\"value\"}", entry.Metadata)
	}
}

func TestTransactionFields(t *testing.T) {
	now := time.Now()
	tx := Transaction{
		TransactionID:   "txn-002",
		IdempotencyKey:  "idem-002",
		TransactionType: "TRANSFER",
		Status:          "PENDING",
		SourceAccountID: "src-001",
		TargetAccountID: "tgt-001",
		MerchantID:      "",
		Amount:          250.00,
		Currency:        "PKR",
		FeeAmount:       5.00,
		FraudScore:      0.1,
		InitiatedBy:     "user-001",
		InitiatedAt:     now,
		Metadata:        []byte(`{"note":"test"}`),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if tx.TransactionID != "txn-002" {
		t.Errorf("TransactionID = %s, want txn-002", tx.TransactionID)
	}
	if tx.IdempotencyKey != "idem-002" {
		t.Errorf("IdempotencyKey = %s, want idem-002", tx.IdempotencyKey)
	}
	if tx.TransactionType != "TRANSFER" {
		t.Errorf("TransactionType = %s, want TRANSFER", tx.TransactionType)
	}
	if tx.Status != "PENDING" {
		t.Errorf("Status = %s, want PENDING", tx.Status)
	}
	if tx.SourceAccountID != "src-001" {
		t.Errorf("SourceAccountID = %s, want src-001", tx.SourceAccountID)
	}
	if tx.TargetAccountID != "tgt-001" {
		t.Errorf("TargetAccountID = %s, want tgt-001", tx.TargetAccountID)
	}
	if tx.Amount != 250.00 {
		t.Errorf("Amount = %f, want 250.00", tx.Amount)
	}
	if tx.Currency != "PKR" {
		t.Errorf("Currency = %s, want PKR", tx.Currency)
	}
	if tx.FeeAmount != 5.00 {
		t.Errorf("FeeAmount = %f, want 5.00", tx.FeeAmount)
	}
	if tx.FraudScore != 0.1 {
		t.Errorf("FraudScore = %f, want 0.1", tx.FraudScore)
	}
	if tx.InitiatedBy != "user-001" {
		t.Errorf("InitiatedBy = %s, want user-001", tx.InitiatedBy)
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := Config{}

	if cfg.Host != "" {
		t.Errorf("Host = %s, want empty string", cfg.Host)
	}
	if cfg.Port != 0 {
		t.Errorf("Port = %d, want 0", cfg.Port)
	}
	if cfg.MaxConns != 0 {
		t.Errorf("MaxConns = %d, want 0", cfg.MaxConns)
	}
	if cfg.MinConns != 0 {
		t.Errorf("MinConns = %d, want 0", cfg.MinConns)
	}
	if cfg.MaxConnLifetime != 0 {
		t.Errorf("MaxConnLifetime = %v, want 0", cfg.MaxConnLifetime)
	}
	if cfg.MaxConnIdleTime != 0 {
		t.Errorf("MaxConnIdleTime = %v, want 0", cfg.MaxConnIdleTime)
	}
}
