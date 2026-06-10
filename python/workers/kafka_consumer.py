from __future__ import annotations

import asyncio
import json
import logging
import os
import time

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

try:
    from aiokafka import AIOKafkaConsumer, ConsumerRecord

    HAS_AIOKAFKA = True
except ImportError:
    HAS_AIOKAFKA = False
    logger.warning("aiokafka not installed. Install with: pip install aiokafka")


class AsyncTransactionConsumer:
    def __init__(self, brokers: str, topic: str, group_id: str):
        self.brokers = brokers
        self.topic = topic
        self.group_id = group_id
        self.consumer: AIOKafkaConsumer | None = None
        self._running = False

    async def start(self):
        if not HAS_AIOKAFKA:
            raise RuntimeError("aiokafka is required but not installed")

        self.consumer = AIOKafkaConsumer(
            self.topic,
            bootstrap_servers=self.brokers,
            group_id=self.group_id,
            auto_offset_reset="earliest",
            enable_auto_commit=True,
            value_deserializer=lambda v: json.loads(v.decode("utf-8")),
        )
        await self.consumer.start()
        self._running = True
        logger.info(
            "Consumer started — topic=%s group=%s brokers=%s",
            self.topic,
            self.group_id,
            self.brokers,
        )

    async def process_message(self, message: ConsumerRecord) -> dict:
        tx = message.value
        amount = tx.get("amount", 0)

        risk = "LOW"
        if amount > 50000:
            risk = "HIGH"
        elif amount > 10000:
            risk = "MEDIUM"

        result = {
            "transaction_id": tx.get("transaction_id", "unknown"),
            "amount": amount,
            "risk_level": risk,
            "partition": message.partition,
            "offset": message.offset,
        }
        logger.info("Processed: %s", json.dumps(result))
        return result

    async def run(self):
        await self.start()
        try:
            async for msg in self.consumer:
                if not self._running:
                    break
                await self.process_message(msg)
        except asyncio.CancelledError:
            pass
        finally:
            await self.stop()

    async def stop(self):
        self._running = False
        if self.consumer:
            await self.consumer.stop()
            logger.info("Consumer stopped")


if __name__ == "__main__":
    BROKERS = os.getenv("KAFKA_BROKERS", "localhost:9092")
    TOPIC = os.getenv("KAFKA_TOPIC", "transactions")
    GROUP_ID = os.getenv("KAFKA_GROUP_ID", "fintech-processor")

    async def main():
        consumer = AsyncTransactionConsumer(BROKERS, TOPIC, GROUP_ID)

        async def shutdown(signal):
            logger.info("Received %s, shutting down...", signal.name)
            await consumer.stop()

        loop = asyncio.get_running_loop()
        for sig in (None, None):
            pass

        try:
            await consumer.run()
        except KeyboardInterrupt:
            await consumer.stop()

    asyncio.run(main())
