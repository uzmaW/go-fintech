import asyncio
import json
import logging
import time

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class NotificationWorker:
    def __init__(self, max_concurrent: int = 100):
        self.semaphore = asyncio.Semaphore(max_concurrent)
        logger.info("NotificationWorker ready (max_concurrent=%d)", max_concurrent)

    async def send_email(self, user_id: str, subject: str, body: str) -> bool:
        async with self.semaphore:
            await asyncio.sleep(0.05)
            logger.info("Email sent to %s: %s", user_id, subject)
            return True

    async def send_push(self, user_id: str, title: str, message: str) -> bool:
        async with self.semaphore:
            await asyncio.sleep(0.02)
            logger.info("Push sent to %s: %s", user_id, title)
            return True

    async def send_sms(self, user_id: str, message: str) -> bool:
        async with self.semaphore:
            await asyncio.sleep(0.03)
            logger.info("SMS sent to %s", user_id)
            return True

    async def process_notification(self, notification: dict) -> dict:
        user_id = notification["user_id"]
        channel = notification["channel"]
        payload = notification["payload"]

        try:
            if channel == "email":
                ok = await self.send_email(user_id, payload["subject"], payload["body"])
            elif channel == "push":
                ok = await self.send_push(user_id, payload["title"], payload["message"])
            elif channel == "sms":
                ok = await self.send_sms(user_id, payload["message"])
            else:
                logger.warning("Unknown channel: %s", channel)
                ok = False

            return {"user_id": user_id, "channel": channel, "sent": ok}
        except Exception as e:
            logger.error("Failed to send %s to %s: %s", channel, user_id, e)
            return {"user_id": user_id, "channel": channel, "sent": False, "error": str(e)}

    async def process_batch(self, notifications: list[dict]) -> list[dict]:
        tasks = [self.process_notification(n) for n in notifications]
        return list(await asyncio.gather(*tasks))


if __name__ == "__main__":
    notifications = [
        {
            "user_id": f"USR-{i:03d}",
            "channel": channel,
            "payload": payload,
        }
        for i, (channel, payload) in enumerate(
            [
                ("email", {"subject": "Transaction Alert", "body": "You have a new transaction."}),
                ("push", {"title": "Low Balance", "message": "Your balance is below $100."}),
                ("sms", {"message": "Your OTP is 123456"}),
                ("email", {"subject": "Monthly Statement", "body": "See attached."}),
                ("push", {"title": "Transfer Complete", "message": "$500 sent."}),
            ],
            start=1,
        )
    ]

    async def main():
        worker = NotificationWorker(max_concurrent=10)
        start = time.perf_counter()
        results = await worker.process_batch(notifications)
        elapsed = time.perf_counter() - start
        logger.info("Processed %d notifications in %.3fs", len(results), elapsed)
        for r in results:
            logger.info(json.dumps(r))

    asyncio.run(main())
