import asyncio
import json
import pytest
from types import SimpleNamespace
from kafka_consumer import AsyncTransactionConsumer


def _make_consumer_record(value, partition=0, offset=0):
    return SimpleNamespace(
        value=value,
        partition=partition,
        offset=offset,
    )


class TestAsyncTransactionConsumer:
    def _make_consumer(self):
        return AsyncTransactionConsumer(
            brokers="localhost:9092",
            topic="transactions",
            group_id="test-group",
        )

    @pytest.mark.asyncio
    async def test_process_message_low_amount(self):
        consumer = self._make_consumer()
        tx = {"transaction_id": "TX-0001", "amount": 500}
        msg = _make_consumer_record(tx, partition=0, offset=10)
        result = await consumer.process_message(msg)
        assert result["transaction_id"] == "TX-0001"
        assert result["amount"] == 500
        assert result["risk_level"] == "LOW"
        assert result["partition"] == 0
        assert result["offset"] == 10

    @pytest.mark.asyncio
    async def test_process_message_medium_amount(self):
        consumer = self._make_consumer()
        tx = {"transaction_id": "TX-0002", "amount": 15000}
        msg = _make_consumer_record(tx, partition=1, offset=20)
        result = await consumer.process_message(msg)
        assert result["risk_level"] == "MEDIUM"

    @pytest.mark.asyncio
    async def test_process_message_high_amount(self):
        consumer = self._make_consumer()
        tx = {"transaction_id": "TX-0003", "amount": 60000}
        msg = _make_consumer_record(tx, partition=2, offset=30)
        result = await consumer.process_message(msg)
        assert result["risk_level"] == "HIGH"

    @pytest.mark.asyncio
    async def test_process_message_missing_transaction_id(self):
        consumer = self._make_consumer()
        tx = {"amount": 100}
        msg = _make_consumer_record(tx)
        result = await consumer.process_message(msg)
        assert result["transaction_id"] == "unknown"

    @pytest.mark.asyncio
    async def test_process_message_missing_amount_defaults_zero(self):
        consumer = self._make_consumer()
        tx = {"transaction_id": "TX-0004"}
        msg = _make_consumer_record(tx)
        result = await consumer.process_message(msg)
        assert result["amount"] == 0
        assert result["risk_level"] == "LOW"

    @pytest.mark.asyncio
    async def test_stop_when_not_started(self):
        consumer = self._make_consumer()
        await consumer.stop()
        assert consumer._running is False

    def test_init_stores_params(self):
        consumer = self._make_consumer()
        assert consumer.brokers == "localhost:9092"
        assert consumer.topic == "transactions"
        assert consumer.group_id == "test-group"
        assert consumer.consumer is None
        assert consumer._running is False
