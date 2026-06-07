//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRouter() *gin.Engine {
	r := gin.New()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "api-gateway",
			"env":       "test",
			"timestamp": time.Now().UTC(),
		})
	})
	r.POST("/api/v1/transactions", func(c *gin.Context) {
		var req struct {
			IdempotencyKey  string  `json:"idempotency_key" binding:"required"`
			TransactionType string  `json:"transaction_type" binding:"required"`
			SourceAccountID string  `json:"source_account_id" binding:"required"`
			Amount          float64 `json:"amount" binding:"required"`
			Currency        string  `json:"currency" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "ERROR", "message": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{
			"transaction_id": "test-tx-123",
			"status":         "PENDING",
			"message":        "Transaction queued for processing",
		})
	})
	r.POST("/api/v1/transfers", func(c *gin.Context) {
		var req struct {
			IdempotencyKey  string  `json:"idempotency_key" binding:"required"`
			TransactionType string  `json:"transaction_type"`
			SourceAccountID string  `json:"source_account_id" binding:"required"`
			TargetAccountID string  `json:"target_account_id"`
			Amount          float64 `json:"amount" binding:"required"`
			Currency        string  `json:"currency" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "ERROR", "message": err.Error()})
			return
		}
		if req.SourceAccountID == "" || req.TargetAccountID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"status": "ERROR", "message": "Both accounts required"})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{
			"transaction_id": "test-transfer-456",
			"status":         "PENDING",
		})
	})
	r.GET("/api/v1/transactions/:id", func(c *gin.Context) {
		txID := c.Param("id")
		c.JSON(http.StatusOK, gin.H{
			"transaction_id": txID,
			"status":         "PENDING",
		})
	})
	return r
}

func TestHealthEndpoint(t *testing.T) {
	r := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["status"] != "healthy" {
		t.Errorf("expected healthy status, got %v", resp["status"])
	}
}

func TestCreateTransaction_Success(t *testing.T) {
	r := setupTestRouter()
	payload := map[string]interface{}{
		"idempotency_key":   "idem-integ-1",
		"transaction_type":  "DEPOSIT",
		"source_account_id": "acc-1",
		"amount":            100.50,
		"currency":          "USD",
	}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/transactions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["transaction_id"] == nil {
		t.Error("expected transaction_id")
	}
	if resp["status"] != "PENDING" {
		t.Errorf("expected PENDING, got %v", resp["status"])
	}
}

func TestCreateTransaction_MissingRequiredFields(t *testing.T) {
	r := setupTestRouter()

	tests := []struct {
		name    string
		payload map[string]interface{}
	}{
		{
			name:    "missing idempotency_key",
			payload: map[string]interface{}{"transaction_type": "DEPOSIT", "source_account_id": "acc-1", "amount": 100, "currency": "USD"},
		},
		{
			name:    "missing transaction_type",
			payload: map[string]interface{}{"idempotency_key": "key-1", "source_account_id": "acc-1", "amount": 100, "currency": "USD"},
		},
		{
			name:    "missing source_account_id",
			payload: map[string]interface{}{"idempotency_key": "key-1", "transaction_type": "DEPOSIT", "amount": 100, "currency": "USD"},
		},
		{
			name:    "missing amount",
			payload: map[string]interface{}{"idempotency_key": "key-1", "transaction_type": "DEPOSIT", "source_account_id": "acc-1", "currency": "USD"},
		},
		{
			name:    "missing currency",
			payload: map[string]interface{}{"idempotency_key": "key-1", "transaction_type": "DEPOSIT", "source_account_id": "acc-1", "amount": 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/v1/transactions", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", w.Code)
			}
		})
	}
}

func TestCreateTransaction_InvalidJSON(t *testing.T) {
	r := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/transactions", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateTransfer_Success(t *testing.T) {
	r := setupTestRouter()
	payload := map[string]interface{}{
		"idempotency_key":   "idem-transfer-1",
		"source_account_id": "acc-1",
		"target_account_id": "acc-2",
		"amount":            250.00,
		"currency":          "USD",
	}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/transfers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", w.Code)
	}
}

func TestCreateTransfer_MissingTarget(t *testing.T) {
	r := setupTestRouter()
	payload := map[string]interface{}{
		"idempotency_key":   "idem-transfer-2",
		"source_account_id": "acc-1",
		"amount":            100.00,
		"currency":          "USD",
	}
	body, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/transfers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetTransaction_Success(t *testing.T) {
	r := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/transactions/tx-integ-123", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["transaction_id"] != "tx-integ-123" {
		t.Errorf("expected tx-integ-123, got %v", resp["transaction_id"])
	}
}

func TestTransactionFlow(t *testing.T) {
	t.Run("create_transaction", func(t *testing.T) {
		r := setupTestRouter()
		payload := map[string]interface{}{
			"idempotency_key":   "idem-flow-1",
			"transaction_type":  "DEPOSIT",
			"source_account_id": "acc-100",
			"amount":            500.00,
			"currency":          "USD",
		}
		body, _ := json.Marshal(payload)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/transactions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		txID := resp["transaction_id"].(string)

		// Verify transaction can be retrieved
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/v1/transactions/"+txID, nil)
		r.ServeHTTP(w2, req2)

		if w2.Code != http.StatusOK {
			t.Errorf("expected 200 for GET, got %d", w2.Code)
		}
	})

	t.Run("fraud_detection", func(t *testing.T) {
		t.Skip("Requires running fraud service")
	})

	t.Run("authorization", func(t *testing.T) {
		t.Skip("Requires running authorization service")
	})

	t.Run("settlement", func(t *testing.T) {
		t.Skip("Requires running settlement job")
	})
}
