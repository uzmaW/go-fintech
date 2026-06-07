//go:build integration

package integration

import (
	"testing"
)

// TestTransactionFlow tests the full transaction lifecycle:
// API Gateway → Kafka → Fraud Service → Authorization Service → Settlement
func TestTransactionFlow(t *testing.T) {
	// This test documents the expected flow
	// In a real integration environment with running services,
	// it would POST a transaction and verify the full pipeline

	t.Run("create_transaction", func(t *testing.T) {
		// Verify transaction creation endpoint
		t.Skip("Requires running API gateway")
	})

	t.Run("fraud_detection", func(t *testing.T) {
		// Verify fraud service processes pending transactions
		t.Skip("Requires running fraud service")
	})

	t.Run("authorization", func(t *testing.T) {
		// Verify authorization service processes authorized transactions
		t.Skip("Requires running authorization service")
	})

	t.Run("settlement", func(t *testing.T) {
		// Verify settlement job processes settled transactions
		t.Skip("Requires running settlement job")
	})
}
