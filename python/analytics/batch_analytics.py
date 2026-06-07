"""
Batch Analytics - Process historical data for analytics and reporting
Uses ProcessPoolExecutor for CPU-intensive aggregation tasks
"""
import asyncio
import logging
import multiprocessing as mp
from concurrent.futures import ProcessPoolExecutor, as_completed
from datetime import datetime, timedelta
from typing import List, Dict, Any

import pandas as pd
from sqlalchemy import create_engine

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


def aggregate_partition(parquet_path: str) -> pd.DataFrame:
    """
    CPU-heavy aggregation for a single partition's Parquet file.

    Reads 1M+ rows, groups by account, sums amounts, counts transactions.

    Args:
        parquet_path: Path to Parquet file (local or S3)

    Returns:
        DataFrame with aggregated results per account
    """
    logger.info(f"Processing {parquet_path}")

    df = pd.read_parquet(parquet_path)

    if df.empty:
        return pd.DataFrame()

    result = df.groupby("account_id").agg(
        total_amount=("amount", "sum"),
        transaction_count=("amount", "count"),
        avg_amount=("amount", "mean"),
        max_amount=("amount", "max"),
        min_amount=("amount", "min")
    ).reset_index()

    logger.info(f"Aggregated {len(result)} accounts from {parquet_path}")
    return result


def aggregate_daily_volume(parquet_paths: List[str], date: str) -> Dict[str, Any]:
    """
    Aggregate daily transaction volume across all partitions.

    Uses ProcessPoolExecutor to parallelize CPU-intensive work across cores.
    """
    logger.info(f"Aggregating {len(parquet_paths)} partitions for {date}")

    results = []
    with ProcessPoolExecutor(max_workers=mp.cpu_count()) as executor:
        futures = {executor.submit(aggregate_partition, path): path
                   for path in parquet_paths}

        for future in as_completed(futures):
            path = futures[future]
            try:
                result = future.result()
                results.append(result)
            except Exception as e:
                logger.error(f"Failed to aggregate {path}: {e}")

    if not results:
        return {"date": date, "total_accounts": 0, "total_transactions": 0}

    combined = pd.concat(results, ignore_index=True)

    final = combined.groupby("account_id").agg({
        "total_amount": "sum",
        "transaction_count": "sum",
        "avg_amount": "mean",
        "max_amount": "max",
        "min_amount": "min"
    }).reset_index()

    summary = {
        "date": date,
        "total_accounts": len(final),
        "total_transactions": int(final["transaction_count"].sum()),
        "total_volume": float(final["total_amount"].sum()),
        "avg_transaction_value": float(final["total_amount"].sum() / final["transaction_count"].sum())
    }

    logger.info(f"Daily aggregation complete: {summary}")
    return summary


def generate_account_statement(account_id: str, start_date: str, end_date: str) -> pd.DataFrame:
    """
    Generate account statement for a date range.

    Used for customer statements and regulatory reporting.
    """
    logger.info(f"Generating statement for account {account_id} from {start_date} to {end_date}")

    query = f"""
        SELECT
            le.entry_id,
            le.transaction_id,
            le.entry_type,
            le.amount,
            le.currency,
            le.balance_after,
            le.description,
            le.posted_at,
            le.reference_id,
            t.merchant_id,
            t.merchant_category
        FROM ledger_entries le
        JOIN transactions t ON t.transaction_id = le.transaction_id
        WHERE le.account_id = '{account_id}'
          AND le.posted_at >= '{start_date}'
          AND le.posted_at < '{end_date}'
        ORDER BY le.posted_at ASC
    """

    return pd.read_sql(query, "postgresql://user:pass@localhost/db")


class AsyncAnalyticsService:
    """
    Async service for orchestrating analytics jobs.

    Uses asyncio for I/O-bound operations (S3, database) while
    delegating CPU-intensive work to ProcessPoolExecutor.
    """

    def __init__(self, db_url: str, s3_bucket: str):
        self.db_url = db_url
        self.s3_bucket = s3_bucket
        self.engine = create_engine(db_url)

    async def run_daily_aggregation(self, date: str) -> Dict[str, Any]:
        """
        Run daily aggregation job.

        1. List all Parquet files for the date from S3
        2. Submit partition aggregation tasks to ProcessPoolExecutor
        3. Combine results and write to analytics table
        """
        logger.info(f"Starting daily aggregation for {date}")

        parquet_files = await self._list_s3_files(date)

        if not parquet_files:
            logger.warning(f"No parquet files found for {date}")
            return {"date": date, "status": "NO_DATA"}

        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(
            None,
            aggregate_daily_volume,
            parquet_files,
            date
        )

        await self._write_analytics_result(result)

        return result

    async def _list_s3_files(self, date: str) -> List[str]:
        """List Parquet files for a given date from S3."""
        prefix = f"s3://{self.s3_bucket}/date={date}/"

        logger.info(f"Listing files with prefix: {prefix}")

        files = [f"{prefix}part-{i}.parquet" for i in range(24)]
        return files

    async def _write_analytics_result(self, result: Dict):
        """Write aggregation result to analytics table."""
        with self.engine.begin() as conn:
            conn.execute(f"""
                INSERT INTO analytics_daily_summary
                (date, total_accounts, total_transactions, total_volume,
                 avg_transaction_value, created_at)
                VALUES (
                    '{result['date']}',
                    {result['total_accounts']},
                    {result['total_transactions']},
                    {result['total_volume']},
                    {result['avg_transaction_value']},
                    NOW()
                )
                ON CONFLICT (date) DO UPDATE SET
                    total_accounts = EXCLUDED.total_accounts,
                    total_transactions = EXCLUDED.total_transactions,
                    total_volume = EXCLUDED.total_volume,
                    avg_transaction_value = EXCLUDED.avg_transaction_value
            """)
        logger.info(f"Analytics result written for {result['date']}")


async def main():
    import argparse

    parser = argparse.ArgumentParser(description="Batch Analytics Service")
    parser.add_argument("--db-url", required=True)
    parser.add_argument("--s3-bucket", required=True)
    parser.add_argument("--date", help="Date (YYYY-MM-DD)")

    args = parser.parse_args()

    service = AsyncAnalyticsService(args.db_url, args.s3_bucket)

    date = args.date or datetime.now().strftime('%Y-%m-%d')
    result = await service.run_daily_aggregation(date)

    logger.info(f"Daily aggregation result: {result}")


if __name__ == "__main__":
    asyncio.run(main())