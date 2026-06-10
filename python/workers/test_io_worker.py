import asyncio
import pytest
from io_worker import NotificationWorker


@pytest.fixture
def worker():
    return NotificationWorker(max_concurrent=10)


class TestNotificationWorker:
    @pytest.mark.asyncio
    async def test_send_email_returns_true(self, worker):
        result = await worker.send_email("USR-001", "Alert", "Hello")
        assert result is True

    @pytest.mark.asyncio
    async def test_send_push_returns_true(self, worker):
        result = await worker.send_push("USR-001", "Title", "Message")
        assert result is True

    @pytest.mark.asyncio
    async def test_send_sms_returns_true(self, worker):
        result = await worker.send_sms("USR-001", "OTP is 123456")
        assert result is True

    @pytest.mark.asyncio
    async def test_process_email_notification(self, worker):
        notification = {
            "user_id": "USR-001",
            "channel": "email",
            "payload": {"subject": "Test", "body": "Hello"},
        }
        result = await worker.process_notification(notification)
        assert result["sent"] is True
        assert result["channel"] == "email"
        assert result["user_id"] == "USR-001"

    @pytest.mark.asyncio
    async def test_process_push_notification(self, worker):
        notification = {
            "user_id": "USR-002",
            "channel": "push",
            "payload": {"title": "Title", "message": "Body"},
        }
        result = await worker.process_notification(notification)
        assert result["sent"] is True
        assert result["channel"] == "push"

    @pytest.mark.asyncio
    async def test_process_sms_notification(self, worker):
        notification = {
            "user_id": "USR-003",
            "channel": "sms",
            "payload": {"message": "Your OTP is 123456"},
        }
        result = await worker.process_notification(notification)
        assert result["sent"] is True
        assert result["channel"] == "sms"

    @pytest.mark.asyncio
    async def test_unknown_channel_returns_false(self, worker):
        notification = {
            "user_id": "USR-004",
            "channel": "carrier_pigeon",
            "payload": {},
        }
        result = await worker.process_notification(notification)
        assert result["sent"] is False
        assert result["channel"] == "carrier_pigeon"

    @pytest.mark.asyncio
    async def test_process_batch_multiple(self, worker):
        notifications = [
            {"user_id": "USR-1", "channel": "email", "payload": {"subject": "A", "body": "B"}},
            {"user_id": "USR-2", "channel": "push", "payload": {"title": "C", "message": "D"}},
            {"user_id": "USR-3", "channel": "sms", "payload": {"message": "E"}},
        ]
        results = await worker.process_batch(notifications)
        assert len(results) == 3
        assert all(r["sent"] is True for r in results)

    @pytest.mark.asyncio
    async def test_process_batch_empty(self, worker):
        results = await worker.process_batch([])
        assert results == []

    @pytest.mark.asyncio
    async def test_concurrent_execution(self, worker):
        notifications = [
            {"user_id": f"USR-{i}", "channel": "email", "payload": {"subject": f"S{i}", "body": f"B{i}"}}
            for i in range(20)
        ]
        start = asyncio.get_event_loop().time()
        results = await worker.process_batch(notifications)
        elapsed = asyncio.get_event_loop().time() - start
        assert len(results) == 20
        assert elapsed < 5.0
