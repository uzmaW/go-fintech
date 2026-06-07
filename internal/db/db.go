package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool struct {
	pool *pgxpool.Pool
}

type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

func NewPool(cfg Config) (*Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.MaxConns > 0 {
		poolConfig.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns >= 0 {
		poolConfig.MinConns = cfg.MinConns
	}
	if cfg.MaxConnLifetime > 0 {
		poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	}
	if cfg.MaxConnIdleTime > 0 {
		poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Pool{pool: pool}, nil
}

func (p *Pool) Query(ctx context.Context, sql string, args ...interface{}) ([]byte, error) {
	row := p.pool.QueryRow(ctx, sql, args...)
	var result []byte
	err := row.Scan(&result)
	return result, err
}

func (p *Pool) Exec(ctx context.Context, sql string, args ...interface{}) error {
	_, err := p.pool.Exec(ctx, sql, args...)
	return err
}

func (p *Pool) Close() {
	p.pool.Close()
}

func (p *Pool) BatchInsertLedgerEntries(ctx context.Context, entries []LedgerEntry) error {
	batch := &pgx.Batch{}

	for _, entry := range entries {
		batch.Queue(`
			INSERT INTO ledger_entries
			(transaction_id, account_id, entry_type, amount, currency, balance_after, posted_at, idempotency_key, metadata)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, entry.TransactionID, entry.AccountID, entry.EntryType, entry.Amount,
			entry.Currency, entry.BalanceAfter, entry.PostedAt, entry.IdempotencyKey, entry.Metadata)
	}

	br := p.pool.SendBatch(ctx, batch)
	defer br.Close()

	_, err := br.Exec()
	return err
}

func (p *Pool) BatchInsertTransactions(ctx context.Context, txs []Transaction) error {
	batch := &pgx.Batch{}

	for _, tx := range txs {
		batch.Queue(`
			INSERT INTO transactions
			(transaction_id, idempotency_key, transaction_type, status, source_account_id,
			 target_account_id, merchant_id, amount, currency, fee_amount, fraud_score,
			 initiated_by, initiated_at, metadata, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
			ON CONFLICT (idempotency_key) DO NOTHING
		`, tx.TransactionID, tx.IdempotencyKey, tx.TransactionType, tx.Status,
			tx.SourceAccountID, tx.TargetAccountID, tx.MerchantID, tx.Amount, tx.Currency,
			tx.FeeAmount, tx.FraudScore, tx.InitiatedBy, tx.InitiatedAt, tx.Metadata, tx.CreatedAt, tx.UpdatedAt)
	}

	br := p.pool.SendBatch(ctx, batch)
	defer br.Close()

	_, err := br.Exec()
	return err
}

func (p *Pool) GetAccountBalance(ctx context.Context, accountID string) (float64, error) {
	var balance float64
	err := p.pool.QueryRow(ctx, `
		SELECT balance_available FROM accounts WHERE account_id = $1
	`, accountID).Scan(&balance)
	return balance, err
}

func (p *Pool) UpdateAccountBalance(ctx context.Context, accountID string, amount float64) error {
	_, err := p.pool.Exec(ctx, `
		UPDATE accounts SET balance_available = balance_available - $1,
		balance_current = balance_current - $1, updated_at = NOW()
		WHERE account_id = $2
	`, amount, accountID)
	return err
}

type LedgerEntry struct {
	TransactionID  string
	AccountID      string
	EntryType      string
	Amount         float64
	Currency       string
	BalanceAfter   float64
	PostedAt       time.Time
	IdempotencyKey string
	Metadata       []byte
}

type Transaction struct {
	TransactionID   string
	IdempotencyKey  string
	TransactionType string
	Status          string
	SourceAccountID string
	TargetAccountID string
	MerchantID      string
	Amount          float64
	Currency        string
	FeeAmount       float64
	FraudScore      float64
	InitiatedBy     string
	InitiatedAt     time.Time
	Metadata        []byte
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
