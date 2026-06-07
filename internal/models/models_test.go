package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTransactionTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant TransactionType
		want     string
	}{
		{"TypeDeposit", TypeDeposit, "DEPOSIT"},
		{"TypeWithdrawal", TypeWithdrawal, "WITHDRAWAL"},
		{"TypeTransfer", TypeTransfer, "TRANSFER"},
		{"TypeCardAuth", TypeCardAuth, "CARD_AUTH"},
		{"TypeACHDebit", TypeACHDebit, "ACH_DEBIT"},
		{"TypeACHCredit", TypeACHCredit, "ACH_CREDIT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.want {
				t.Errorf("%s = %s, want %s", tt.name, string(tt.constant), tt.want)
			}
		})
	}
}

func TestTransactionStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant TransactionStatus
		want     string
	}{
		{"StatusPending", StatusPending, "PENDING"},
		{"StatusAuthorized", StatusAuthorized, "AUTHORIZED"},
		{"StatusSettled", StatusSettled, "SETTLED"},
		{"StatusFailed", StatusFailed, "FAILED"},
		{"StatusReversed", StatusReversed, "REVERSED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.want {
				t.Errorf("%s = %s, want %s", tt.name, string(tt.constant), tt.want)
			}
		})
	}
}

func TestEntryTypeConstants(t *testing.T) {
	if string(EntryTypeDebit) != "DEBIT" {
		t.Errorf("EntryTypeDebit = %s, want DEBIT", string(EntryTypeDebit))
	}
	if string(EntryTypeCredit) != "CREDIT" {
		t.Errorf("EntryTypeCredit = %s, want CREDIT", string(EntryTypeCredit))
	}
}

func TestTransactionJSON(t *testing.T) {
	txID := uuid.New()
	srcID := uuid.New()
	tgtID := uuid.New()
	merchantID := uuid.New()
	cardID := uuid.New()
	now := time.Now().Truncate(time.Microsecond)

	tx := Transaction{
		TransactionID:     txID,
		IdempotencyKey:    "idem-123",
		TransactionType:   TypeTransfer,
		Status:            StatusPending,
		SourceAccountID:   srcID,
		TargetAccountID:   tgtID,
		MerchantID:        merchantID,
		Amount:            150.75,
		Currency:          "USD",
		FeeAmount:         2.50,
		FXRate:            1.0,
		FraudScore:        0.05,
		RiskLevel:         "LOW",
		MFAVerified:       true,
		CardID:            cardID,
		CardLast4:         "4242",
		MerchantCategory:  "groceries",
		AuthorizationCode: "AUTH001",
		InitiatedBy:       "user-001",
		InitiatedAt:       now,
		IPAddress:         "192.168.1.1",
		UserAgent:         "Mozilla/5.0",
		DeviceFingerprint: "fp-abc",
		Metadata:          json.RawMessage(`{"source":"mobile"}`),
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	data, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Transaction
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.TransactionID != txID {
		t.Errorf("TransactionID = %v, want %v", decoded.TransactionID, txID)
	}
	if decoded.IdempotencyKey != "idem-123" {
		t.Errorf("IdempotencyKey = %s, want idem-123", decoded.IdempotencyKey)
	}
	if decoded.TransactionType != TypeTransfer {
		t.Errorf("TransactionType = %s, want TRANSFER", decoded.TransactionType)
	}
	if decoded.Status != StatusPending {
		t.Errorf("Status = %s, want PENDING", decoded.Status)
	}
	if decoded.SourceAccountID != srcID {
		t.Errorf("SourceAccountID = %v, want %v", decoded.SourceAccountID, srcID)
	}
	if decoded.TargetAccountID != tgtID {
		t.Errorf("TargetAccountID = %v, want %v", decoded.TargetAccountID, tgtID)
	}
	if decoded.MerchantID != merchantID {
		t.Errorf("MerchantID = %v, want %v", decoded.MerchantID, merchantID)
	}
	if decoded.Amount != 150.75 {
		t.Errorf("Amount = %f, want 150.75", decoded.Amount)
	}
	if decoded.Currency != "USD" {
		t.Errorf("Currency = %s, want USD", decoded.Currency)
	}
	if decoded.FeeAmount != 2.50 {
		t.Errorf("FeeAmount = %f, want 2.50", decoded.FeeAmount)
	}
	if decoded.CardID != cardID {
		t.Errorf("CardID = %v, want %v", decoded.CardID, cardID)
	}
	if decoded.CardLast4 != "4242" {
		t.Errorf("CardLast4 = %s, want 4242", decoded.CardLast4)
	}
}

func TestAccountJSON(t *testing.T) {
	accountID := uuid.New()
	customerID := uuid.New()
	now := time.Now().Truncate(time.Microsecond)

	account := Account{
		AccountID:            accountID,
		CustomerID:           customerID,
		AccountNumber:        "1234567890",
		AccountType:          "CHECKING",
		Currency:             "USD",
		Status:               "ACTIVE",
		BalanceAvailable:     1000.00,
		BalanceCurrent:       1000.00,
		OverdraftLimit:       200.00,
		DailyWithdrawalLimit: 500.00,
		CreatedAt:            now,
		UpdatedAt:            now,
		Metadata:             json.RawMessage(`{"verified":true}`),
	}

	data, err := json.Marshal(account)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Account
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.AccountID != accountID {
		t.Errorf("AccountID = %v, want %v", decoded.AccountID, accountID)
	}
	if decoded.CustomerID != customerID {
		t.Errorf("CustomerID = %v, want %v", decoded.CustomerID, customerID)
	}
	if decoded.AccountNumber != "1234567890" {
		t.Errorf("AccountNumber = %s, want 1234567890", decoded.AccountNumber)
	}
	if decoded.AccountType != "CHECKING" {
		t.Errorf("AccountType = %s, want CHECKING", decoded.AccountType)
	}
	if decoded.Currency != "USD" {
		t.Errorf("Currency = %s, want USD", decoded.Currency)
	}
	if decoded.Status != "ACTIVE" {
		t.Errorf("Status = %s, want ACTIVE", decoded.Status)
	}
	if decoded.BalanceAvailable != 1000.00 {
		t.Errorf("BalanceAvailable = %f, want 1000.00", decoded.BalanceAvailable)
	}
	if decoded.BalanceCurrent != 1000.00 {
		t.Errorf("BalanceCurrent = %f, want 1000.00", decoded.BalanceCurrent)
	}
	if decoded.OverdraftLimit != 200.00 {
		t.Errorf("OverdraftLimit = %f, want 200.00", decoded.OverdraftLimit)
	}
	if decoded.DailyWithdrawalLimit != 500.00 {
		t.Errorf("DailyWithdrawalLimit = %f, want 500.00", decoded.DailyWithdrawalLimit)
	}
}

func TestLedgerEntryJSON(t *testing.T) {
	txID := uuid.New()
	accountID := uuid.New()
	now := time.Now().Truncate(time.Microsecond)

	entry := LedgerEntry{
		EntryID:        1,
		TransactionID:  txID,
		AccountID:      accountID,
		EntryType:      EntryTypeDebit,
		Amount:         50.00,
		Currency:       "USD",
		BalanceAfter:   950.00,
		Description:    "Purchase at store",
		ReferenceID:    "ref-001",
		CreatedAt:      now,
		PostedAt:       now,
		IdempotencyKey: "idem-456",
		Metadata:       json.RawMessage(`{"category":"food"}`),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded LedgerEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.EntryID != 1 {
		t.Errorf("EntryID = %d, want 1", decoded.EntryID)
	}
	if decoded.TransactionID != txID {
		t.Errorf("TransactionID = %v, want %v", decoded.TransactionID, txID)
	}
	if decoded.AccountID != accountID {
		t.Errorf("AccountID = %v, want %v", decoded.AccountID, accountID)
	}
	if decoded.EntryType != EntryTypeDebit {
		t.Errorf("EntryType = %s, want DEBIT", decoded.EntryType)
	}
	if decoded.Amount != 50.00 {
		t.Errorf("Amount = %f, want 50.00", decoded.Amount)
	}
	if decoded.Currency != "USD" {
		t.Errorf("Currency = %s, want USD", decoded.Currency)
	}
	if decoded.BalanceAfter != 950.00 {
		t.Errorf("BalanceAfter = %f, want 950.00", decoded.BalanceAfter)
	}
	if decoded.Description != "Purchase at store" {
		t.Errorf("Description = %s, want 'Purchase at store'", decoded.Description)
	}
	if decoded.ReferenceID != "ref-001" {
		t.Errorf("ReferenceID = %s, want ref-001", decoded.ReferenceID)
	}
	if decoded.IdempotencyKey != "idem-456" {
		t.Errorf("IdempotencyKey = %s, want idem-456", decoded.IdempotencyKey)
	}
}

func TestCardJSON(t *testing.T) {
	cardID := uuid.New()
	accountID := uuid.New()
	now := time.Now().Truncate(time.Microsecond)

	card := Card{
		CardID:         cardID,
		AccountID:      accountID,
		CardNumberHash: "hash-abc",
		CardLast4:      "1234",
		CardType:       "DEBIT",
		Network:        "VISA",
		Status:         "ACTIVE",
		DailyLimit:     1000.00,
		ExpiryMonth:    12,
		ExpiryYear:     2028,
		CVVHash:        "cvv-hash",
		PinHash:        "pin-hash",
		CreatedAt:      now,
	}

	data, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Card
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.CardID != cardID {
		t.Errorf("CardID = %v, want %v", decoded.CardID, cardID)
	}
	if decoded.AccountID != accountID {
		t.Errorf("AccountID = %v, want %v", decoded.AccountID, accountID)
	}
	if decoded.CardNumberHash != "hash-abc" {
		t.Errorf("CardNumberHash = %s, want hash-abc", decoded.CardNumberHash)
	}
	if decoded.CardLast4 != "1234" {
		t.Errorf("CardLast4 = %s, want 1234", decoded.CardLast4)
	}
	if decoded.CardType != "DEBIT" {
		t.Errorf("CardType = %s, want DEBIT", decoded.CardType)
	}
	if decoded.Network != "VISA" {
		t.Errorf("Network = %s, want VISA", decoded.Network)
	}
	if decoded.Status != "ACTIVE" {
		t.Errorf("Status = %s, want ACTIVE", decoded.Status)
	}
	if decoded.DailyLimit != 1000.00 {
		t.Errorf("DailyLimit = %f, want 1000.00", decoded.DailyLimit)
	}
	if decoded.ExpiryMonth != 12 {
		t.Errorf("ExpiryMonth = %d, want 12", decoded.ExpiryMonth)
	}
	if decoded.ExpiryYear != 2028 {
		t.Errorf("ExpiryYear = %d, want 2028", decoded.ExpiryYear)
	}
	if decoded.CVVHash != "cvv-hash" {
		t.Errorf("CVVHash = %s, want cvv-hash", decoded.CVVHash)
	}
	if decoded.PinHash != "pin-hash" {
		t.Errorf("PinHash = %s, want pin-hash", decoded.PinHash)
	}
}

func TestHoldJSON(t *testing.T) {
	holdID := uuid.New()
	accountID := uuid.New()
	txID := uuid.New()
	now := time.Now().Truncate(time.Microsecond)

	hold := Hold{
		HoldID:        holdID,
		AccountID:     accountID,
		TransactionID: txID,
		Amount:        200.00,
		Currency:      "EUR",
		Status:        "ACTIVE",
		ExpiresAt:     now.Add(24 * time.Hour),
		CreatedAt:     now,
	}

	data, err := json.Marshal(hold)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Hold
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.HoldID != holdID {
		t.Errorf("HoldID = %v, want %v", decoded.HoldID, holdID)
	}
	if decoded.AccountID != accountID {
		t.Errorf("AccountID = %v, want %v", decoded.AccountID, accountID)
	}
	if decoded.TransactionID != txID {
		t.Errorf("TransactionID = %v, want %v", decoded.TransactionID, txID)
	}
	if decoded.Amount != 200.00 {
		t.Errorf("Amount = %f, want 200.00", decoded.Amount)
	}
	if decoded.Currency != "EUR" {
		t.Errorf("Currency = %s, want EUR", decoded.Currency)
	}
	if decoded.Status != "ACTIVE" {
		t.Errorf("Status = %s, want ACTIVE", decoded.Status)
	}
}

func TestAuditLogJSON(t *testing.T) {
	actorID := uuid.New()
	resourceID := uuid.New()
	now := time.Now().Truncate(time.Microsecond)

	log := AuditLog{
		AuditID:         42,
		EventType:       "TRANSACTION_CREATED",
		ActorID:         actorID,
		ActorType:       "CUSTOMER",
		ResourceType:    "TRANSACTION",
		ResourceID:      resourceID,
		Action:          "CREATE",
		Status:          "SUCCESS",
		IPAddress:       "10.0.0.1",
		UserAgent:       "GoClient/1.0",
		RequestPayload:  json.RawMessage(`{"amount":100}`),
		ResponsePayload: json.RawMessage(`{"status":"ok"}`),
		ErrorMessage:    "",
		Timestamp:       now,
	}

	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded AuditLog
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.AuditID != 42 {
		t.Errorf("AuditID = %d, want 42", decoded.AuditID)
	}
	if decoded.EventType != "TRANSACTION_CREATED" {
		t.Errorf("EventType = %s, want TRANSACTION_CREATED", decoded.EventType)
	}
	if decoded.ActorID != actorID {
		t.Errorf("ActorID = %v, want %v", decoded.ActorID, actorID)
	}
	if decoded.ActorType != "CUSTOMER" {
		t.Errorf("ActorType = %s, want CUSTOMER", decoded.ActorType)
	}
	if decoded.ResourceType != "TRANSACTION" {
		t.Errorf("ResourceType = %s, want TRANSACTION", decoded.ResourceType)
	}
	if decoded.ResourceID != resourceID {
		t.Errorf("ResourceID = %v, want %v", decoded.ResourceID, resourceID)
	}
	if decoded.Action != "CREATE" {
		t.Errorf("Action = %s, want CREATE", decoded.Action)
	}
	if decoded.Status != "SUCCESS" {
		t.Errorf("Status = %s, want SUCCESS", decoded.Status)
	}
	if decoded.IPAddress != "10.0.0.1" {
		t.Errorf("IPAddress = %s, want 10.0.0.1", decoded.IPAddress)
	}
	if decoded.UserAgent != "GoClient/1.0" {
		t.Errorf("UserAgent = %s, want GoClient/1.0", decoded.UserAgent)
	}
	if string(decoded.RequestPayload) != `{"amount":100}` {
		t.Errorf("RequestPayload = %s, want {\"amount\":100}", decoded.RequestPayload)
	}
	if string(decoded.ResponsePayload) != `{"status":"ok"}` {
		t.Errorf("ResponsePayload = %s, want {\"status\":\"ok\"}", decoded.ResponsePayload)
	}
}
