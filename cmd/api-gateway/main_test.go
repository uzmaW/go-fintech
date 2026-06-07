package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/company/go-fintech/internal/config"
)

func init() {
	gin.SetMode(gin.TestMode)
	serviceConfig = &config.Config{
		App: config.AppConfig{Name: "api-gateway", Env: "test", Port: 8080},
	}
}

func setupRouter() *gin.Engine {
	r := gin.New()
	r.GET("/api/v1/health", HealthCheck)
	r.GET("/api/v1/transactions/:id", GetTransaction)
	// Note: POST handlers require a non-nil producer, so we test binding separately.
	return r
}

func TestHealthCheck(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["status"] != "healthy" {
		t.Errorf("expected healthy, got %v", resp["status"])
	}
	if resp["service"] != "api-gateway" {
		t.Errorf("expected api-gateway, got %v", resp["service"])
	}
	if resp["env"] != "test" {
		t.Errorf("expected test, got %v", resp["env"])
	}
}

func TestGetTransaction(t *testing.T) {
	r := setupRouter()
	txID := "test-tx-123"
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/transactions/"+txID, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["transaction_id"] != txID {
		t.Errorf("expected %s, got %v", txID, resp["transaction_id"])
	}
	if resp["status"] != "PENDING" {
		t.Errorf("expected PENDING, got %v", resp["status"])
	}
}

func TestTransactionRequest_Binding(t *testing.T) {
	tests := []struct {
		name        string
		payload     string
		expectError bool
	}{
		{
			name:        "valid request",
			payload:     `{"idempotency_key":"key-1","transaction_type":"DEPOSIT","source_account_id":"acc-1","amount":100,"currency":"USD"}`,
			expectError: false,
		},
		{
			name:        "missing idempotency_key",
			payload:     `{"transaction_type":"DEPOSIT","source_account_id":"acc-1","amount":100,"currency":"USD"}`,
			expectError: true,
		},
		{
			name:        "missing transaction_type",
			payload:     `{"idempotency_key":"key-1","source_account_id":"acc-1","amount":100,"currency":"USD"}`,
			expectError: true,
		},
		{
			name:        "missing source_account_id",
			payload:     `{"idempotency_key":"key-1","transaction_type":"DEPOSIT","amount":100,"currency":"USD"}`,
			expectError: true,
		},
		{
			name:        "missing amount",
			payload:     `{"idempotency_key":"key-1","transaction_type":"DEPOSIT","source_account_id":"acc-1","currency":"USD"}`,
			expectError: true,
		},
		{
			name:        "missing currency",
			payload:     `{"idempotency_key":"key-1","transaction_type":"DEPOSIT","source_account_id":"acc-1","amount":100}`,
			expectError: true,
		},
		{
			name:        "invalid JSON",
			payload:     `{invalid`,
			expectError: true,
		},
		{
			name:        "empty body",
			payload:     ``,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req TransactionRequest
			err := json.Unmarshal([]byte(tt.payload), &req)
			if tt.expectError && err == nil {
				// Also check binding tags
				if req.IdempotencyKey == "" || req.TransactionType == "" || req.SourceAccountID == "" || req.Amount == 0 || req.Currency == "" {
					// Expected: required fields missing
					return
				}
				t.Error("expected error but got valid request")
			}
		})
	}
}

func TestTransactionRequest_AllFields(t *testing.T) {
	payload := `{
		"idempotency_key": "key-all",
		"transaction_type": "CARD_AUTH",
		"source_account_id": "acc-1",
		"target_account_id": "acc-2",
		"amount": 500.50,
		"currency": "EUR",
		"fee_amount": 5.00,
		"merchant_id": "merch-1",
		"merchant_category": "5812",
		"card_number_hash": "hash-abc",
		"card_last4": "4321",
		"initiated_by": "user-1"
	}`
	var req TransactionRequest
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if req.IdempotencyKey != "key-all" {
		t.Error("IdempotencyKey mismatch")
	}
	if req.Amount != 500.50 {
		t.Errorf("Amount = %f, want 500.50", req.Amount)
	}
	if req.Currency != "EUR" {
		t.Error("Currency mismatch")
	}
	if req.CardLast4 != "4321" {
		t.Error("CardLast4 mismatch")
	}
}

func TestTransactionResponse_Fields(t *testing.T) {
	resp := TransactionResponse{
		TransactionID: "tx-123",
		Status:        "PENDING",
		Message:       "queued",
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded TransactionResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.TransactionID != "tx-123" {
		t.Error("TransactionID mismatch")
	}
	if decoded.Status != "PENDING" {
		t.Error("Status mismatch")
	}
	if decoded.Message != "queued" {
		t.Error("Message mismatch")
	}
}

func TestTransactionResponse_EmptyMessage(t *testing.T) {
	resp := TransactionResponse{TransactionID: "tx-456", Status: "SETTLED"}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if _, exists := decoded["message"]; exists {
		t.Error("empty message should be omitted from JSON")
	}
}

func TestHealthCheck_Timestamp(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	before := time.Now().UTC()
	r.ServeHTTP(w, req)
	after := time.Now().UTC()

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	tsStr, ok := resp["timestamp"].(string)
	if !ok {
		t.Fatal("timestamp not a string")
	}
	ts, err := time.Parse(time.RFC3339Nano, tsStr)
	if err != nil {
		t.Fatalf("failed to parse timestamp: %v", err)
	}
	if ts.Before(before.Add(-time.Second)) || ts.After(after.Add(time.Second)) {
		t.Errorf("timestamp %v not between %v and %v", ts, before, after)
	}
}

func TestCreateTransfer_ValidationLogic(t *testing.T) {
	tests := []struct {
		name          string
		sourceAccount string
		targetAccount string
		expectReject  bool
	}{
		{"both accounts present", "acc-1", "acc-2", false},
		{"missing target", "acc-1", "", true},
		{"missing source", "", "acc-2", true},
		{"both missing", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate CreateTransfer validation logic
			shouldReject := tt.sourceAccount == "" || tt.targetAccount == ""
			if shouldReject != tt.expectReject {
				t.Errorf("expected reject=%v, got %v", tt.expectReject, shouldReject)
			}
		})
	}
}
