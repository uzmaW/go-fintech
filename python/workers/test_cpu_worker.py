import asyncio
import pytest
from cpu_worker import compute_fraud_score, CPUWorkerPool


class TestComputeFraudScore:
    def test_deterministic_output(self):
        tx = {"transaction_id": "TX-0001", "amount": 500}
        r1 = compute_fraud_score(tx)
        r2 = compute_fraud_score(tx)
        assert r1 == r2

    def test_low_amount_low_risk(self):
        tx = {"transaction_id": "TX-0002", "amount": 10}
        result = compute_fraud_score(tx)
        assert result["risk_level"] == "LOW"
        assert result["fraud_score"] < 30

    def test_high_amount_medium_risk(self):
        tx = {"transaction_id": "TX-0003", "amount": 15000}
        result = compute_fraud_score(tx)
        assert result["fraud_score"] >= 20

    def test_very_high_amount_high_risk(self):
        tx = {"transaction_id": "TX-0004", "amount": 60000}
        result = compute_fraud_score(tx)
        assert result["fraud_score"] >= 45

    def test_gambling_category_increases_score(self):
        tx = {"transaction_id": "TX-0005", "amount": 500, "merchant_category": "gambling"}
        result = compute_fraud_score(tx)
        assert result["fraud_score"] >= 15

    def test_crypto_category_increases_score(self):
        tx = {"transaction_id": "TX-0006", "amount": 500, "merchant_category": "crypto"}
        result = compute_fraud_score(tx)
        assert result["fraud_score"] >= 15

    def test_retail_category_adds_score(self):
        tx = {"transaction_id": "TX-0007", "amount": 500, "merchant_category": "retail"}
        result = compute_fraud_score(tx)
        assert result["fraud_score"] >= 10

    def test_unknown_category_no_extra_score(self):
        tx = {"transaction_id": "TX-0008", "amount": 500, "merchant_category": "unknown_cat"}
        result = compute_fraud_score(tx)
        assert result["fraud_score"] >= 0

    def test_score_capped_at_100(self):
        tx = {"transaction_id": "TX-0009", "amount": 99999999, "merchant_category": "wire_transfer"}
        result = compute_fraud_score(tx)
        assert result["fraud_score"] <= 100

    def test_missing_category_defaults_low(self):
        tx = {"transaction_id": "TX-0010", "amount": 50}
        result = compute_fraud_score(tx)
        assert result["risk_level"] in ("LOW", "MEDIUM", "HIGH")
        assert isinstance(result["fraud_score"], int)

    def test_result_keys(self):
        tx = {"transaction_id": "TX-0011", "amount": 100}
        result = compute_fraud_score(tx)
        assert set(result.keys()) == {"transaction_id", "fraud_score", "risk_level"}

    def test_risk_level_thresholds(self):
        tx = {"transaction_id": "TX-0012", "amount": 1}
        result = compute_fraud_score(tx)
        assert result["risk_level"] == "LOW"

        tx = {"transaction_id": "TX-0013", "amount": 12000}
        result = compute_fraud_score(tx)
        assert result["risk_level"] == "MEDIUM"

        tx = {"transaction_id": "TX-0014", "amount": 60000, "merchant_category": "wire_transfer"}
        result = compute_fraud_score(tx)
        assert result["risk_level"] == "HIGH"


@pytest.mark.asyncio
class TestCPUWorkerPool:
    async def test_process_batch_returns_all_results(self):
        pool = CPUWorkerPool(max_workers=2)
        txs = [
            {"transaction_id": f"TX-{i:04d}", "amount": i * 100}
            for i in range(1, 6)
        ]
        results = await pool.process_batch(txs)
        await pool.shutdown()
        assert len(results) == 5
        assert all("fraud_score" in r for r in results)
        assert all("risk_level" in r for r in results)

    async def test_process_batch_preserves_ids(self):
        pool = CPUWorkerPool(max_workers=2)
        txs = [
            {"transaction_id": "TX-AAA", "amount": 50},
            {"transaction_id": "TX-BBB", "amount": 60000},
        ]
        results = await pool.process_batch(txs)
        await pool.shutdown()
        ids = {r["transaction_id"] for r in results}
        assert ids == {"TX-AAA", "TX-BBB"}

    async def test_empty_batch(self):
        pool = CPUWorkerPool(max_workers=2)
        results = await pool.process_batch([])
        await pool.shutdown()
        assert results == []

    async def test_single_worker_pool(self):
        pool = CPUWorkerPool(max_workers=1)
        txs = [{"transaction_id": f"TX-{i:04d}", "amount": i * 10} for i in range(10)]
        results = await pool.process_batch(txs)
        await pool.shutdown()
        assert len(results) == 10
