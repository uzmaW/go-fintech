package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type TransactionType string
type TransactionStatus string
type EntryType string

const (
	TypeDeposit    TransactionType = "DEPOSIT"
	TypeWithdrawal TransactionType = "WITHDRAWAL"
	TypeTransfer   TransactionType = "TRANSFER"
	TypeCardAuth   TransactionType = "CARD_AUTH"
	TypeACHDebit   TransactionType = "ACH_DEBIT"
	TypeACHCredit  TransactionType = "ACH_CREDIT"
)

const (
	StatusPending    TransactionStatus = "PENDING"
	StatusAuthorized TransactionStatus = "AUTHORIZED"
	StatusSettled    TransactionStatus = "SETTLED"
	StatusFailed     TransactionStatus = "FAILED"
	StatusReversed   TransactionStatus = "REVERSED"
)

const (
	EntryTypeDebit  EntryType = "DEBIT"
	EntryTypeCredit EntryType = "CREDIT"
)

type Transaction struct {
	TransactionID     uuid.UUID         `json:"transaction_id" db:"transaction_id"`
	IdempotencyKey    string            `json:"idempotency_key" db:"idempotency_key"`
	TransactionType   TransactionType   `json:"transaction_type" db:"transaction_type"`
	Status            TransactionStatus `json:"status" db:"status"`
	SourceAccountID   uuid.UUID         `json:"source_account_id" db:"source_account_id"`
	TargetAccountID   uuid.UUID         `json:"target_account_id" db:"target_account_id"`
	MerchantID        uuid.UUID         `json:"merchant_id" db:"merchant_id"`
	Amount            float64           `json:"amount" db:"amount"`
	Currency          string            `json:"currency" db:"currency"`
	FeeAmount         float64           `json:"fee_amount" db:"fee_amount"`
	FXRate            float64           `json:"fx_rate" db:"fx_rate"`
	FraudScore        float64           `json:"fraud_score" db:"fraud_score"`
	RiskLevel         string            `json:"risk_level" db:"risk_level"`
	MFAVerified       bool              `json:"mfa_verified" db:"mfa_verified"`
	CardID            uuid.UUID         `json:"card_id" db:"card_id"`
	CardLast4         string            `json:"card_last4" db:"card_last4"`
	MerchantCategory  string            `json:"merchant_category" db:"merchant_category"`
	AuthorizationCode string            `json:"authorization_code" db:"authorization_code"`
	InitiatedBy       string            `json:"initiated_by" db:"initiated_by"`
	InitiatedAt       time.Time         `json:"initiated_at" db:"initiated_at"`
	AuthorizedAt      *time.Time        `json:"authorized_at" db:"authorized_at"`
	SettledAt         *time.Time        `json:"settled_at" db:"settled_at"`
	ReversedAt        *time.Time        `json:"reversed_at" db:"reversed_at"`
	IPAddress         string            `json:"ip_address" db:"ip_address"`
	UserAgent         string            `json:"user_agent" db:"user_agent"`
	DeviceFingerprint string            `json:"device_fingerprint" db:"device_fingerprint"`
	Metadata          json.RawMessage   `json:"metadata" db:"metadata"`
	CreatedAt         time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at" db:"updated_at"`
}

type Account struct {
	AccountID            uuid.UUID       `json:"account_id" db:"account_id"`
	CustomerID           uuid.UUID       `json:"customer_id" db:"customer_id"`
	AccountNumber        string          `json:"account_number" db:"account_number"`
	AccountType          string          `json:"account_type" db:"account_type"`
	Currency             string          `json:"currency" db:"currency"`
	Status               string          `json:"status" db:"status"`
	BalanceAvailable     float64         `json:"balance_available" db:"balance_available"`
	BalanceCurrent       float64         `json:"balance_current" db:"balance_current"`
	OverdraftLimit       float64         `json:"overdraft_limit" db:"overdraft_limit"`
	DailyWithdrawalLimit float64         `json:"daily_withdrawal_limit" db:"daily_withdrawal_limit"`
	CreatedAt            time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at" db:"updated_at"`
	Metadata             json.RawMessage `json:"metadata" db:"metadata"`
}

type LedgerEntry struct {
	EntryID        int64           `json:"entry_id" db:"entry_id"`
	TransactionID  uuid.UUID       `json:"transaction_id" db:"transaction_id"`
	AccountID      uuid.UUID       `json:"account_id" db:"account_id"`
	EntryType      EntryType       `json:"entry_type" db:"entry_type"`
	Amount         float64         `json:"amount" db:"amount"`
	Currency       string          `json:"currency" db:"currency"`
	BalanceAfter   float64         `json:"balance_after" db:"balance_after"`
	Description    string          `json:"description" db:"description"`
	ReferenceID    string          `json:"reference_id" db:"reference_id"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	PostedAt       time.Time       `json:"posted_at" db:"posted_at"`
	IdempotencyKey string          `json:"idempotency_key" db:"idempotency_key"`
	Metadata       json.RawMessage `json:"metadata" db:"metadata"`
}

type Card struct {
	CardID         uuid.UUID  `json:"card_id" db:"card_id"`
	AccountID      uuid.UUID  `json:"account_id" db:"account_id"`
	CardNumberHash string     `json:"card_number_hash" db:"card_number_hash"`
	CardLast4      string     `json:"card_last4" db:"card_last4"`
	CardType       string     `json:"card_type" db:"card_type"`
	Network        string     `json:"network" db:"network"`
	Status         string     `json:"status" db:"status"`
	DailyLimit     float64    `json:"daily_limit" db:"daily_limit"`
	ExpiryMonth    int        `json:"expiry_month" db:"expiry_month"`
	ExpiryYear     int        `json:"expiry_year" db:"expiry_year"`
	CVVHash        string     `json:"cvv_hash" db:"cvv_hash"`
	PinHash        string     `json:"pin_hash" db:"pin_hash"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	ActivatedAt    *time.Time `json:"activated_at" db:"activated_at"`
	BlockedAt      *time.Time `json:"blocked_at" db:"blocked_at"`
}

type Hold struct {
	HoldID        uuid.UUID `json:"hold_id" db:"hold_id"`
	AccountID     uuid.UUID `json:"account_id" db:"account_id"`
	TransactionID uuid.UUID `json:"transaction_id" db:"transaction_id"`
	Amount        float64   `json:"amount" db:"amount"`
	Currency      string    `json:"currency" db:"currency"`
	Status        string    `json:"status" db:"status"`
	ExpiresAt     time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

type AuditLog struct {
	AuditID         int64           `json:"audit_id" db:"audit_id"`
	EventType       string          `json:"event_type" db:"event_type"`
	ActorID         uuid.UUID       `json:"actor_id" db:"actor_id"`
	ActorType       string          `json:"actor_type" db:"actor_type"`
	ResourceType    string          `json:"resource_type" db:"resource_type"`
	ResourceID      uuid.UUID       `json:"resource_id" db:"resource_id"`
	Action          string          `json:"action" db:"action"`
	Status          string          `json:"status" db:"status"`
	IPAddress       string          `json:"ip_address" db:"ip_address"`
	UserAgent       string          `json:"user_agent" db:"user_agent"`
	RequestPayload  json.RawMessage `json:"request_payload" db:"request_payload"`
	ResponsePayload json.RawMessage `json:"response_payload" db:"response_payload"`
	ErrorMessage    string          `json:"error_message" db:"error_message"`
	Timestamp       time.Time       `json:"timestamp" db:"timestamp"`
}
