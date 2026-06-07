package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"

	"github.com/company/go-fintech/internal/config"
	"github.com/company/go-fintech/internal/metrics"
)

const (
	BatchSize = 500
)

type Transaction struct {
	ID              string
	SourceAccount   string
	TargetAccount   string
	Amount          float64
	FeeAmount       float64
	AuthorizationAt time.Time
}

func main() {
	log.Println("Settlement Job starting...")

	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	if cfg.Database.Host == "" || cfg.Database.User == "" || cfg.Database.Name == "" {
		log.Fatalf("database host/user/name must be configured (DATABASE_HOST/USER/NAME env or config file)")
	}
	if cfg.Database.Port == 0 {
		cfg.Database.Port = 5432
	}

	m := metrics.New("settlement-job")
	_ = m

	metricsAddr := fmt.Sprintf(":%d", cfg.App.Port+100)
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", metrics.Handler())
		log.Printf("Metrics server on %s", metricsAddr)
		if err := http.ListenAndServe(metricsAddr, mux); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	settleDate := time.Now().AddDate(0, 0, -1)
	if v := os.Getenv("SETTLEMENT_DATE"); v != "" {
		if parsed, err := time.Parse("2006-01-02", v); err == nil {
			settleDate = parsed
		} else {
			log.Printf("Invalid SETTLEMENT_DATE=%s, falling back to yesterday", v)
		}
	}

	ctx := context.Background()
	db, err := sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.Name,
	))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Printf("Running settlement for date: %s", settleDate.Format("2006-01-02"))

	txns, err := fetchAuthorizedTransactions(ctx, db, settleDate)
	if err != nil {
		log.Fatalf("Failed to fetch transactions: %v", err)
	}

	log.Printf("Found %d authorized transactions to settle", len(txns))

	settled := 0
	batch := make([]Transaction, 0, BatchSize)

	for _, txn := range txns {
		batch = append(batch, txn)

		if len(batch) >= BatchSize {
			if err := settleBatch(ctx, db, batch); err != nil {
				log.Printf("Batch settlement failed: %v", err)
				continue
			}
			settled += len(batch)
			batch = batch[:0]
			log.Printf("Settled %d transactions", settled)
		}
	}

	if len(batch) > 0 {
		if err := settleBatch(ctx, db, batch); err != nil {
			log.Printf("Final batch settlement failed: %v", err)
		} else {
			settled += len(batch)
		}
	}

	log.Printf("Settlement completed: %d transactions settled", settled)

	if err := generateSettlementReport(ctx, db, settleDate); err != nil {
		log.Printf("Failed to generate report: %v", err)
	}
}

func fetchAuthorizedTransactions(ctx context.Context, db *sql.DB, date time.Time) ([]Transaction, error) {
	query := `
		SELECT transaction_id, source_account_id, target_account_id, amount, fee_amount
		FROM transactions
		WHERE status = 'AUTHORIZED'
		  AND DATE(authorized_at) = $1
		  AND transaction_type IN ('CARD_AUTH', 'ACH_DEBIT')
		ORDER BY authorized_at ASC
	`

	rows, err := db.QueryContext(ctx, query, date.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txns []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.SourceAccount, &t.TargetAccount, &t.Amount, &t.FeeAmount); err != nil {
			return nil, err
		}
		txns = append(txns, t)
	}

	return txns, rows.Err()
}

func settleBatch(ctx context.Context, db *sql.DB, batch []Transaction) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO ledger_entries
		(transaction_id, account_id, entry_type, amount, currency, balance_after, posted_at, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (idempotency_key) DO NOTHING
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()

	for _, txn := range batch {
		debitBalance, err := getBalanceForUpdate(ctx, tx, txn.SourceAccount)
		if err != nil {
			return fmt.Errorf("failed to lock source account %s: %w", txn.SourceAccount, err)
		}
		newDebitBalance := debitBalance - txn.Amount - txn.FeeAmount

		_, err = stmt.ExecContext(ctx,
			txn.ID, txn.SourceAccount, "DEBIT",
			txn.Amount+txn.FeeAmount, "USD", newDebitBalance, now,
			fmt.Sprintf("settle-%s-debit", txn.ID),
		)
		if err != nil {
			return fmt.Errorf("failed to insert debit entry: %w", err)
		}

		creditBalance, err := getBalanceForUpdate(ctx, tx, txn.TargetAccount)
		if err != nil {
			return fmt.Errorf("failed to lock target account %s: %w", txn.TargetAccount, err)
		}
		newCreditBalance := creditBalance + txn.Amount

		_, err = stmt.ExecContext(ctx,
			txn.ID, txn.TargetAccount, "CREDIT",
			txn.Amount, "USD", newCreditBalance, now,
			fmt.Sprintf("settle-%s-credit", txn.ID),
		)
		if err != nil {
			return fmt.Errorf("failed to insert credit entry: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE transactions
			SET status = 'SETTLED', settled_at = $1, updated_at = $1
			WHERE transaction_id = $2 AND status = 'AUTHORIZED'
		`, now, txn.ID); err != nil {
			return fmt.Errorf("failed to update transaction: %w", err)
		}

		res, err := tx.ExecContext(ctx, `
			UPDATE accounts
			SET balance_current = $1, balance_available = $1, updated_at = $2
			WHERE account_id = $3
		`, newDebitBalance, now, txn.SourceAccount)
		if err != nil {
			return fmt.Errorf("failed to update source account: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("source account %s not found", txn.SourceAccount)
		}

		res, err = tx.ExecContext(ctx, `
			UPDATE accounts
			SET balance_current = $1, balance_available = $1, updated_at = $2
			WHERE account_id = $3
		`, newCreditBalance, now, txn.TargetAccount)
		if err != nil {
			return fmt.Errorf("failed to update target account: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("target account %s not found", txn.TargetAccount)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}

	return nil
}

func getBalanceForUpdate(ctx context.Context, tx *sql.Tx, accountID string) (float64, error) {
	var balance float64
	err := tx.QueryRowContext(ctx, `
		SELECT balance_current FROM accounts WHERE account_id = $1 FOR UPDATE
	`, accountID).Scan(&balance)
	return balance, err
}

func generateSettlementReport(ctx context.Context, db *sql.DB, date time.Time) error {
	log.Printf("Generating settlement report for %s", date.Format("2006-01-02"))

	var totalCount int
	var totalAmount float64

	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE status = 'SETTLED' AND DATE(settled_at) = $1
	`, date.Format("2006-01-02")).Scan(&totalCount, &totalAmount)

	if err != nil {
		return err
	}

	log.Printf("Settlement Report: %d transactions, total amount: %.2f", totalCount, totalAmount)
	return nil
}
