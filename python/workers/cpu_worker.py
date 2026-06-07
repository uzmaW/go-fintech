import asyncio
import concurrent.futures
import json
import logging
import hashlib
import time

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

HIGH_RISK_CATEGORIES = {"gambling", "crypto", "wire_transfer", "foreign"}


def compute_fraud_score(transaction: dict) -> dict:
    tx_id = transaction["transaction_id"]
    amount = transaction["amount"]
    category = transaction.get("merchant_category", "").lower()

    digest = hashlib.sha256(f"{tx_id}:{amount}".encode()).hexdigest()
    base_score = int(digest[:4], 16) % 30

    score = base_score

    if amount > 10000:
        score += 20
    if amount > 50000:
        score += 25
    if category in HIGH_RISK_CATEGORIES:
        score += 15
    elif category in {"retail", "groceries", "utilities"}:
        score += 10

    score = min(score, 100)

    if score >= 60:
        risk_level = "HIGH"
    elif score >= 30:
        risk_level = "MEDIUM"
    else:
        risk_level = "LOW"

    return {
        "transaction_id": tx_id,
        "fraud_score": score,
        "risk_level": risk_level,
    }


class CPUWorkerPool:
    def __init__(self, max_workers: int = 4):
        self.max_workers = max_workers
        self.executor = concurrent.futures.ProcessPoolExecutor(
            max_workers=max_workers
        )
        logger.info("CPUWorkerPool started with %d workers", max_workers)

    async def process_batch(self, transactions: list[dict]) -> list[dict]:
        loop = asyncio.get_running_loop()
        tasks = [
            loop.run_in_executor(self.executor, compute_fraud_score, tx)
            for tx in transactions
        ]
        results = await asyncio.gather(*tasks)
        return list(results)

    async def shutdown(self):
        self.executor.shutdown(wait=True)
        logger.info("CPUWorkerPool shut down")


if __name__ == "__main__":
    sample = [
        {
            "transaction_id": f"TX-{i:04d}",
            "amount": amount,
            "merchant_category": cat,
            "source_account_id": f"ACC-{i:03d}",
        }
        for i, (amount, cat) in enumerate(
            [
                (1500, "retail"),
                (55000, "crypto"),
                (3000, "groceries"),
                (120000, "wire_transfer"),
                (200, "utilities"),
            ],
            start=1,
        )
    ]

    async def main():
        pool = CPUWorkerPool(max_workers=2)
        start = time.perf_counter()
        results = await pool.process_batch(sample)
        elapsed = time.perf_counter() - start
        logger.info("Processed %d transactions in %.3fs", len(results), elapsed)
        for r in results:
            logger.info(json.dumps(r))
        await pool.shutdown()

    asyncio.run(main())
