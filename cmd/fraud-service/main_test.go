package main

import (
	"encoding/json"
	"math"
	"testing"
)

func TestCalculateFraudScore_LowAmount(t *testing.T) {
	tx := &Transaction{
		TransactionID:    "tx-1",
		Amount:           50.0,
		MerchantCategory: "",
	}
	score := calculateFraudScore(tx)
	if score < 0 || score > 30 {
		t.Errorf("low amount score should be 0-30, got %f", score)
	}
}

func TestCalculateFraudScore_AmountOver10k(t *testing.T) {
	tx := &Transaction{
		TransactionID:    "tx-2",
		Amount:           15000.0,
		MerchantCategory: "",
	}
	// Run multiple times due to random component
	for i := 0; i < 100; i++ {
		score := calculateFraudScore(tx)
		if score < 20 || score > 50 {
			t.Errorf("amount >10k score should be 20-50, got %f", score)
			break
		}
	}
}

func TestCalculateFraudScore_AmountOver50k(t *testing.T) {
	tx := &Transaction{
		TransactionID:    "tx-3",
		Amount:           60000.0,
		MerchantCategory: "",
	}
	for i := 0; i < 100; i++ {
		score := calculateFraudScore(tx)
		if score < 45 || score > 75 {
			t.Errorf("amount >50k score should be 45-75, got %f", score)
			break
		}
	}
}

func TestCalculateFraudScore_GroceryMerchant(t *testing.T) {
	tx := &Transaction{
		TransactionID:    "tx-4",
		Amount:           100.0,
		MerchantCategory: "5411",
	}
	for i := 0; i < 100; i++ {
		score := calculateFraudScore(tx)
		if score < 15 || score > 45 {
			t.Errorf("grocery merchant score should be 15-45, got %f", score)
			break
		}
	}
}

func TestCalculateFraudScore_GamblingMerchant(t *testing.T) {
	tx := &Transaction{
		TransactionID:    "tx-5",
		Amount:           100.0,
		MerchantCategory: "7995",
	}
	for i := 0; i < 100; i++ {
		score := calculateFraudScore(tx)
		if score < 10 || score > 40 {
			t.Errorf("gambling merchant score should be 10-40, got %f", score)
			break
		}
	}
}

func TestCalculateFraudScore_Combined(t *testing.T) {
	tx := &Transaction{
		TransactionID:    "tx-6",
		Amount:           60000.0,
		MerchantCategory: "5812",
	}
	for i := 0; i < 100; i++ {
		score := calculateFraudScore(tx)
		// random(0-30) + 20 (10k) + 25 (50k) + 15 (restaurant) = 60-90
		if score < 60 || score > 90 {
			t.Errorf("combined score should be 60-90, got %f", score)
			break
		}
	}
}

func TestEvaluateFraud_Approve(t *testing.T) {
	tx := &Transaction{
		TransactionID: "tx-approve",
		Amount:        10.0,
	}
	result := evaluateFraud(tx)
	if result.TransactionID != "tx-approve" {
		t.Errorf("expected tx-approve, got %s", result.TransactionID)
	}
	if result.Recommendation != "APPROVE" {
		t.Errorf("expected APPROVE, got %s", result.Recommendation)
	}
	if result.RiskLevel != "LOW" {
		t.Errorf("expected LOW, got %s", result.RiskLevel)
	}
}

func TestEvaluateFraud_TransactionID(t *testing.T) {
	tx := &Transaction{
		TransactionID: "tx-id-check",
		Amount:        10.0,
	}
	result := evaluateFraud(tx)
	if result.TransactionID != "tx-id-check" {
		t.Errorf("expected tx-id-check, got %s", result.TransactionID)
	}
}

func TestEvaluateFraud_ScoreRange(t *testing.T) {
	tx := &Transaction{
		TransactionID: "tx-range",
		Amount:        100.0,
	}
	result := evaluateFraud(tx)
	if result.FraudScore < 0 || result.FraudScore > 100 {
		t.Errorf("score should be 0-100, got %f", result.FraudScore)
	}
}

func TestEvaluateFraud_JSON(t *testing.T) {
	tx := &Transaction{
		TransactionID: "tx-json",
		Amount:        100.0,
	}
	result := evaluateFraud(tx)
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded FraudResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.TransactionID != "tx-json" {
		t.Error("TransactionID mismatch")
	}
	if decoded.FraudScore != result.FraudScore {
		t.Error("FraudScore mismatch")
	}
	if decoded.RiskLevel != result.RiskLevel {
		t.Error("RiskLevel mismatch")
	}
	if decoded.Recommendation != result.Recommendation {
		t.Error("Recommendation mismatch")
	}
}

func TestEvaluateFraud_RiskLevels(t *testing.T) {
	lowCount := 0
	mediumCount := 0
	highCount := 0

	for i := 0; i < 1000; i++ {
		tx := &Transaction{
			TransactionID: "tx-stat",
			Amount:        100.0,
		}
		result := evaluateFraud(tx)
		switch result.RiskLevel {
		case "LOW":
			lowCount++
		case "MEDIUM":
			mediumCount++
		case "HIGH":
			highCount++
		default:
			t.Errorf("unknown risk level: %s", result.RiskLevel)
		}
	}

	if lowCount == 0 && mediumCount == 0 && highCount == 0 {
		t.Error("no risk levels assigned")
	}
}

func TestEvaluateFraud_Recommendations(t *testing.T) {
	valid := map[string]bool{"APPROVE": true, "REVIEW": true, "DECLINE": true}
	for i := 0; i < 100; i++ {
		tx := &Transaction{
			TransactionID: "tx-rec",
			Amount:        100.0,
		}
		result := evaluateFraud(tx)
		if !valid[result.Recommendation] {
			t.Errorf("invalid recommendation: %s", result.Recommendation)
		}
	}
}

func TestEvaluateFraud_ScoreConsistency(t *testing.T) {
	tx := &Transaction{
		TransactionID: "tx-consist",
		Amount:        100.0,
	}
	// Same transaction should produce same result structure
	// (score is random, but risk level mapping should be consistent)
	for i := 0; i < 50; i++ {
		result := evaluateFraud(tx)
		score := result.FraudScore
		switch {
		case score < 50:
			if result.RiskLevel != "LOW" || result.Recommendation != "APPROVE" {
				t.Errorf("score %f should map to LOW/APPROVE, got %s/%s", score, result.RiskLevel, result.Recommendation)
			}
		case score < 80:
			if result.RiskLevel != "MEDIUM" || result.Recommendation != "REVIEW" {
				t.Errorf("score %f should map to MEDIUM/REVIEW, got %s/%s", score, result.RiskLevel, result.Recommendation)
			}
		default:
			if result.RiskLevel != "HIGH" || result.Recommendation != "DECLINE" {
				t.Errorf("score %f should map to HIGH/DECLINE, got %s/%s", score, result.RiskLevel, result.Recommendation)
			}
		}
	}
}

func TestTransaction_Fields(t *testing.T) {
	tx := Transaction{
		TransactionID:    "tx-fields",
		IdempotencyKey:   "idem-1",
		TransactionType:  "TRANSFER",
		Status:           "PENDING",
		SourceAccountID:  "acc-1",
		TargetAccountID:  "acc-2",
		Amount:           500.0,
		Currency:         "USD",
		MerchantCategory: "5812",
		InitiatedAt:      "2026-01-01T00:00:00Z",
	}

	data, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Transaction
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.TransactionID != "tx-fields" {
		t.Error("TransactionID mismatch")
	}
	if decoded.Amount != 500.0 {
		t.Error("Amount mismatch")
	}
	if decoded.MerchantCategory != "5812" {
		t.Error("MerchantCategory mismatch")
	}
}

func TestFraudResult_RoundTrip(t *testing.T) {
	fr := FraudResult{
		TransactionID:  "tx-rt",
		FraudScore:     75.5,
		RiskLevel:      "MEDIUM",
		Recommendation: "REVIEW",
	}

	data, err := json.Marshal(fr)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded FraudResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if math.Abs(decoded.FraudScore-75.5) > 0.001 {
		t.Errorf("FraudScore mismatch: %f != 75.5", decoded.FraudScore)
	}
}

func TestCalculateFraudScore_Restaurant(t *testing.T) {
	tx := &Transaction{
		TransactionID:    "tx-rest",
		Amount:           100.0,
		MerchantCategory: "5814",
	}
	for i := 0; i < 100; i++ {
		score := calculateFraudScore(tx)
		if score < 15 || score > 45 {
			t.Errorf("restaurant merchant score should be 15-45, got %f", score)
			break
		}
	}
}

func TestCalculateFraudScore_FuelStation(t *testing.T) {
	tx := &Transaction{
		TransactionID:    "tx-fuel",
		Amount:           100.0,
		MerchantCategory: "5812",
	}
	for i := 0; i < 100; i++ {
		score := calculateFraudScore(tx)
		if score < 15 || score > 45 {
			t.Errorf("fuel station merchant score should be 15-45, got %f", score)
			break
		}
	}
}

func TestCalculateFraudScore_UnknownCategory(t *testing.T) {
	tx := &Transaction{
		TransactionID:    "tx-unk",
		Amount:           100.0,
		MerchantCategory: "9999",
	}
	for i := 0; i < 100; i++ {
		score := calculateFraudScore(tx)
		if score < 0 || score > 30 {
			t.Errorf("unknown category score should be 0-30, got %f", score)
			break
		}
	}
}
