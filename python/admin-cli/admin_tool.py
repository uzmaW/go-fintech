"""
Kafka Admin CLI - Topic creation, management, and backfill operations
"""
import asyncio
import argparse
import json
import logging
import subprocess
from datetime import datetime, timedelta
from typing import List, Optional

from aiokafka import KafkaAdminClient, NewTopic
from aiokafka.admin import ConfigResource, ConfigResourceType

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class KafkaAdmin:
    def __init__(self, bootstrap_servers: str):
        self.bootstrap_servers = bootstrap_servers
        self.admin: Optional[KafkaAdminClient] = None

    async def connect(self):
        self.admin = KafkaAdminClient(
            bootstrap_servers=self.bootstrap_servers,
            client_id="admin-cli"
        )
        logger.info(f"Connected to Kafka at {self.bootstrap_servers}")

    async def close(self):
        if self.admin:
            await self.admin.close()

    async def create_topics(self, topics: List[dict]):
        """Create Kafka topics with specified configuration."""
        topic_configs = []
        for t in topics:
            topic = NewTopic(
                name=t["name"],
                num_partitions=t.get("partitions", 24),
                replication_factor=t.get("replication_factor", 3),
                topic_configs=t.get("configs", {})
            )
            topic_configs.append(topic)

        try:
            await self.admin.create_topics(topic_configs, timeout_ms=10000)
            logger.info(f"Created {len(topics)} topics")
        except Exception as e:
            logger.error(f"Failed to create topics: {e}")
            raise

    async def list_topics(self) -> List[str]:
        """List all Kafka topics."""
        try:
            topics = await self.admin.list_topics()
            return topics
        except Exception as e:
            logger.error(f"Failed to list topics: {e}")
            return []

    async def describe_topic(self, topic_name: str) -> dict:
        """Describe a topic's configuration and partition info."""
        try:
            configs = await self.admin.describe_configs(
                ConfigResource(ConfigResourceType.TOPIC, topic_name)
            )
            return dict(configs)
        except Exception as e:
            logger.error(f"Failed to describe topic {topic_name}: {e}")
            return {}


async def create_transaction_topics(bootstrap_servers: str):
    """Create all transaction processing topics."""
    admin = KafkaAdmin(bootstrap_servers)
    await admin.connect()

    topics = [
        {
            "name": "transactions.pending",
            "partitions": 24,
            "replication_factor": 3,
            "configs": {
                "retention.ms": "604800000",
                "cleanup.policy": "delete",
                "compression.type": "snappy"
            }
        },
        {
            "name": "transactions.authorized",
            "partitions": 24,
            "replication_factor": 3,
            "configs": {
                "retention.ms": "604800000",
                "cleanup.policy": "delete"
            }
        },
        {
            "name": "transactions.settled",
            "partitions": 24,
            "replication_factor": 3,
            "configs": {
                "retention.ms": "2592000000",
                "cleanup.policy": "delete"
            }
        },
        {
            "name": "transactions.failed",
            "partitions": 12,
            "replication_factor": 3,
            "configs": {
                "retention.ms": "2592000000"
            }
        },
        {
            "name": "transactions.reversed",
            "partitions": 12,
            "replication_factor": 3
        },
        {
            "name": "fraud.events",
            "partitions": 12,
            "replication_factor": 3,
            "configs": {
                "retention.ms": "2592000000"
            }
        },
        {
            "name": "audit.log",
            "partitions": 24,
            "replication_factor": 3,
            "configs": {
                "retention.ms": "31536000000"
            }
        },
        {
            "name": "notifications.outbox",
            "partitions": 12,
            "replication_factor": 3
        },
    ]

    try:
        await admin.create_topics(topics)
        logger.info("All transaction topics created successfully")
    finally:
        await admin.close()


async def list_all_topics(bootstrap_servers: str):
    """List all topics in the cluster."""
    admin = KafkaAdmin(bootstrap_servers)
    await admin.connect()

    topics = await admin.list_topics()
    logger.info(f"Found {len(topics)} topics:")
    for topic in sorted(topics):
        logger.info(f"  - {topic}")

    await admin.close()
    return topics


async def backfill_partition(partition: int, s3_path: str, kafka_topic: str):
    """
    Backfill a partition from S3 Parquet files.

    This orchestrates a Go binary that reads Parquet and produces to Kafka.
    """
    cmd = [
        "./bin/backfill-go",
        "--partition", str(partition),
        "--s3-path", s3_path,
        "--kafka-topic", kafka_topic
    ]

    proc = await asyncio.create_subprocess_exec(
        *cmd,
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE
    )

    stdout, stderr = await proc.communicate()

    if proc.returncode != 0:
        raise Exception(f"Backfill partition {partition} failed: {stderr.decode()}")

    logger.info(f"Partition {partition} backfilled: {stdout.decode().strip()}")


async def backfill_from_s3(bootstrap_servers: str, s3_bucket: str, date: str):
    """
    Backfill all partitions from S3 for a given date.

    Args:
        bootstrap_servers: Kafka bootstrap servers
        s3_bucket: S3 bucket containing Parquet files
        date: Date to backfill (YYYY-MM-DD format)
    """
    logger.info(f"Starting backfill for {date} from {s3_bucket}")

    tasks = []
    for partition in range(24):
        s3_path = f"s3://{s3_bucket}/date={date}/part-{partition}.parquet"
        task = backfill_partition(partition, s3_path, "transactions.raw")
        tasks.append(task)

    await asyncio.gather(*tasks, return_exceptions=True)
    logger.info(f"Backfill completed for {date}")


def main():
    parser = argparse.ArgumentParser(description="Kafka Admin CLI")
    subparsers = parser.add_subparsers(dest="command", help="Commands")

    create_parser = subparsers.add_parser("create-topics", help="Create transaction topics")
    create_parser.add_argument("--bootstrap-servers", default="localhost:9092",
                              help="Kafka bootstrap servers")

    list_parser = subparsers.add_parser("list-topics", help="List all topics")
    list_parser.add_argument("--bootstrap-servers", default="localhost:9092",
                           help="Kafka bootstrap servers")

    backfill_parser = subparsers.add_parser("backfill", help="Backfill from S3")
    backfill_parser.add_argument("--bootstrap-servers", default="localhost:9092")
    backfill_parser.add_argument("--s3-bucket", required=True)
    backfill_parser.add_argument("--date", required=True, help="Date (YYYY-MM-DD)")

    args = parser.parse_args()

    if args.command == "create-topics":
        asyncio.run(create_transaction_topics(args.bootstrap_servers))
    elif args.command == "list-topics":
        asyncio.run(list_all_topics(args.bootstrap_servers))
    elif args.command == "backfill":
        asyncio.run(backfill_from_s3(args.bootstrap_servers, args.s3_bucket, args.date))
    else:
        parser.print_help()


if __name__ == "__main__":
    main()