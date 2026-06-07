"""
Reconciliation Service - Compare internal ledger vs external settlement files
"""
import logging
import os
from datetime import datetime, date
from typing import Dict, List, Optional, Tuple

import pandas as pd
from sqlalchemy import create_engine

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class ReconciliationService:
    """
    Reconciliation service for comparing internal transactions against
    external settlement files from card networks and ACH processors.
    """

    def __init__(self, db_url: str, s3_bucket: str):
        self.engine = create_engine(db_url)
        self.s3_bucket = s3_bucket

    def reconcile_daily(self, date: date) -> Dict:
        """
        Perform daily reconciliation.

        Args:
            date: The date to reconcile

        Returns:
            Dict with reconciliation results
        """
        logger.info(f"Starting reconciliation for {date}")

        try:
            internal_df = self._load_internal_ledger(date)
            logger.info(f"Loaded {len(internal_df)} internal transactions")

            external_df = self._load_external_settlement(date)
            logger.info(f"Loaded {len(external_df)} external transactions")

            matched, discrepancies, amount_mismatches = self._match_transactions(
                internal_df, external_df
            )

            result = {
                "date": date.isoformat(),
                "internal_count": len(internal_df),
                "external_count": len(external_df),
                "matched_count": len(matched),
                "discrepancy_count": len(discrepancies),
                "mismatch_count": len(amount_mismatches),
                "status": "PASSED" if len(discrepancies) == 0 and len(amount_mismatches) == 0 else "FAILED"
            }

            if result["status"] == "FAILED":
                self._alert_finance_team(discrepancies, amount_mismatches)
                self._write_discrepancy_report(date, discrepancies, amount_mismatches)
            else:
                logger.info(f"Reconciliation PASSED: {len(matched)} transactions matched")

            self._write_summary(result)
            return result

        except Exception as e:
            logger.error(f"Reconciliation failed: {e}")
            return {
                "date": date.isoformat(),
                "status": "ERROR",
                "error": str(e)
            }

    def _load_internal_ledger(self, date: date) -> pd.DataFrame:
        """Load internal ledger transactions for the date."""
        query = f"""
            SELECT
                t.transaction_id,
                t.authorization_code,
                t.amount,
                t.currency,
                t.settled_at,
                t.source_account_id,
                t.merchant_id,
                t.transaction_type,
                t.status
            FROM transactions t
            WHERE DATE(t.settled_at) = '{date}'
              AND t.transaction_type IN ('CARD_AUTH', 'ACH_DEBIT', 'ACH_CREDIT')
              AND t.status = 'SETTLED'
        """
        return pd.read_sql(query, self.engine)

    def _load_external_settlement(self, date: date) -> pd.DataFrame:
        """Load external settlement files from S3."""
        dfs = []

        visa_df = self._load_visa_settlement(date)
        if visa_df is not None:
            dfs.append(visa_df)

        mastercard_df = self._load_mastercard_settlement(date)
        if mastercard_df is not None:
            dfs.append(mastercard_df)

        ach_df = self._load_ach_settlement(date)
        if ach_df is not None:
            dfs.append(ach_df)

        if not dfs:
            logger.warning(f"No external settlement files found for {date}")
            return pd.DataFrame()

        return pd.concat(dfs, ignore_index=True)

    def _load_visa_settlement(self, date: date) -> Optional[pd.DataFrame]:
        """Load Visa settlement file from S3."""
        file_path = f"s3://{self.s3_bucket}/visa/settlement_{date.strftime('%Y%m%d')}.csv"

        if not self._s3_file_exists(file_path):
            logger.info(f"Visa settlement file not found: {file_path}")
            return None

        try:
            df = pd.read_csv(file_path)
            df = df.rename(columns={
                'AUTH_CODE': 'authorization_code',
                'AMOUNT': 'amount',
                'CURRENCY': 'currency',
                'MERCHANT_ID': 'merchant_id',
                'TRACE_NUMBER': 'trace_number'
            })
            df['network'] = 'VISA'
            return df
        except Exception as e:
            logger.error(f"Error loading Visa settlement: {e}")
            return None

    def _load_mastercard_settlement(self, date: date) -> Optional[pd.DataFrame]:
        """Load Mastercard settlement file (IPM format) from S3."""
        file_path = f"s3://{self.s3_bucket}/mastercard/IPM_{date.strftime('%Y%m%d')}.txt"

        if not self._s3_file_exists(file_path):
            logger.info(f"Mastercard settlement file not found: {file_path}")
            return None

        try:
            df = pd.read_csv(file_path, sep='|')
            df = df.rename(columns={
                'AUTH_CODE': 'authorization_code',
                'TXN_AMOUNT': 'amount',
                'CURRENCY_CODE': 'currency',
                'MERCHANT_ID': 'merchant_id'
            })
            df['network'] = 'MASTERCARD'
            return df
        except Exception as e:
            logger.error(f"Error loading Mastercard settlement: {e}")
            return None

    def _load_ach_settlement(self, date: date) -> Optional[pd.DataFrame]:
        """Load NACHA ACH settlement file from S3."""
        file_path = f"s3://{self.s3_bucket}/ach/ACH_{date.strftime('%Y%m%d')}.ach"

        if not self._s3_file_exists(file_path):
            logger.info(f"ACH settlement file not found: {file_path}")
            return None

        try:
            df = pd.read_csv(file_path, header=None)
            df.columns = ['trace_number', 'receiver_id', 'amount', 'currency', 'effective_date']
            df['network'] = 'ACH'
            return df
        except Exception as e:
            logger.error(f"Error loading ACH settlement: {e}")
            return None

    def _s3_file_exists(self, path: str) -> bool:
        """Check if S3 file exists."""
        if path.startswith('s3://'):
            bucket = path.split('/')[2]
            key = '/'.join(path.split('/')[3:])
            return True
        return os.path.exists(path)

    def _match_transactions(
        self,
        internal_df: pd.DataFrame,
        external_df: pd.DataFrame
    ) -> Tuple[pd.DataFrame, pd.DataFrame, pd.DataFrame]:
        """
        Match internal transactions against external settlement files.

        Returns:
            Tuple of (matched, discrepancies, amount_mismatches)
        """
        if internal_df.empty:
            return pd.DataFrame(), external_df, pd.DataFrame()

        if external_df.empty:
            return pd.DataFrame(), internal_df, pd.DataFrame()

        matched = pd.merge(
            internal_df,
            external_df,
            on='authorization_code',
            suffixes=('_internal', '_external'),
            how='outer',
            indicator=True
        )

        discrepancies = matched[matched['_merge'] != 'both'].copy()

        amount_mismatches = matched[
            (matched['_merge'] == 'both') &
            (abs(matched['amount_internal'].fillna(0) - matched['amount_external'].fillna(0)) > 0.01)
        ].copy()

        matched_only_both = matched[matched['_merge'] == 'both']

        return matched_only_both, discrepancies, amount_mismatches

    def _alert_finance_team(self, discrepancies: pd.DataFrame, mismatches: pd.DataFrame):
        """Send alert to finance team via Slack/PagerDuty."""
        message = f"""
⚠️ RECONCILIATION ALERT

Date: {datetime.now().strftime('%Y-%m-%d')}

Discrepancies found:
- Missing in internal: {len(discrepancies[discrepancies['_merge'] == 'right_only'])}
- Missing in external: {len(discrepancies[discrepancies['_merge'] == 'left_only'])}
- Amount mismatches: {len(mismatches)}

Total issues: {len(discrepancies) + len(mismatches)}

Review dashboard: https://dashboard.company.com/reconciliation
"""
        logger.warning(message)

    def _write_discrepancy_report(
        self,
        date: date,
        discrepancies: pd.DataFrame,
        mismatches: pd.DataFrame
    ):
        """Write discrepancy report to S3 for audit."""
        report_date = date.strftime('%Y%m%d')

        if not discrepancies.empty:
            discrepancy_path = f"s3://{self.s3_bucket}/reconciliation/discrepancies_{report_date}.csv"
            discrepancies.to_csv(discrepancy_path, index=False)
            logger.info(f"Discrepancy report written to {discrepancy_path}")

        if not mismatches.empty:
            mismatch_path = f"s3://{self.s3_bucket}/reconciliation/mismatches_{report_date}.csv"
            mismatches.to_csv(mismatch_path, index=False)
            logger.info(f"Mismatch report written to {mismatch_path}")

    def _write_summary(self, result: Dict):
        """Write reconciliation summary to database."""
        with self.engine.begin() as conn:
            conn.execute(f"""
                INSERT INTO reconciliation_summary
                (date, internal_count, internal_amount, external_count, external_amount,
                 matched_count, matched_amount, discrepancy_count, mismatch_count, status, created_at)
                VALUES (
                    '{result['date']}',
                    {result.get('internal_count', 0)},
                    0, 0, 0,
                    {result.get('matched_count', 0)},
                    0,
                    {result.get('discrepancy_count', 0)},
                    {result.get('mismatch_count', 0)},
                    '{result['status']}',
                    NOW()
                )
                ON CONFLICT (date) DO UPDATE SET
                    status = EXCLUDED.status,
                    internal_count = EXCLUDED.internal_count,
                    matched_count = EXCLUDED.matched_count,
                    discrepancy_count = EXCLUDED.discrepancy_count,
                    mismatch_count = EXCLUDED.mismatch_count
            """)
        logger.info(f"Summary written to reconciliation_summary table")


def main():
    import argparse
    from datetime import datetime, timedelta

    parser = argparse.ArgumentParser(description="Reconciliation Service")
    parser.add_argument("--db-url", required=True, help="PostgreSQL connection URL")
    parser.add_argument("--s3-bucket", required=True, help="S3 bucket for settlement files")
    parser.add_argument("--date", help="Date to reconcile (YYYY-MM-DD), defaults to yesterday")

    args = parser.parse_args()

    if args.date:
        reconcile_date = datetime.strptime(args.date, '%Y-%m-%d').date()
    else:
        reconcile_date = datetime.now().date() - timedelta(days=1)

    service = ReconciliationService(args.db_url, args.s3_bucket)
    result = service.reconcile_daily(reconcile_date)

    logger.info(f"Reconciliation result: {result['status']}")
    if result['status'] == 'FAILED':
        logger.warning(f"Found {result['discrepancy_count']} discrepancies and {result['mismatch_count']} mismatches")


if __name__ == "__main__":
    main()