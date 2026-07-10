"""Stress test: sends rapid events to HiveScope bridge."""

import asyncio
import random
import sys
import time

import httpx

BRIDGE_URL = "http://localhost:8765"
EVENT_COUNT = 1000
BATCH_SIZE = 50

PHASES = ["planning", "coding", "reviewing", "testing"]
AGENTS = ["CEO", "CTO", "Programmer", "Reviewer", "Tester", "Designer"]
TOOLS = ["file_system", "web_search", "terminal", "database"]
STATUSES = ["running", "success", "error", "idle"]


async def send_events(client: httpx.AsyncClient, count: int) -> tuple[int, float]:
    start = time.perf_counter()
    sent = 0

    for i in range(0, count, BATCH_SIZE):
        batch: list[dict] = []
        for j in range(min(BATCH_SIZE, count - i)):
            phase = random.choice(PHASES)
            source = random.choice(AGENTS)
            node_type = "agent"
            if random.random() < 0.2:
                source = random.choice(TOOLS)
                node_type = "tool"

            event = {
                "schema_version": 1,
                "timestamp": time.strftime("%Y-%m-%dT%H:%M:%S", time.gmtime()),
                "source": source,
                "target": random.choice(AGENTS) if random.random() < 0.5 else None,
                "phase": phase,
                "action": random.choice(["message", "file_create", "file_edit", "state_change"]),
                "message": f"Stress test event {i + j + 1}/{count}",
                "files": [f"src/module_{random.randint(1, 5)}.py"] if random.random() < 0.3 else [],
                "code": None,
                "status": random.choice(STATUSES),
                "node_type": node_type,
                "parent_id": None,
                "metadata": {"stress": True, "index": i + j + 1},
            }
            batch.append(event)

        try:
            r = await client.post(f"{BRIDGE_URL}/events", json=batch, timeout=10)
            r.raise_for_status()
            sent += len(batch)
        except Exception as e:
            print(f"  Error at event {sent}: {e}", file=sys.stderr)

    elapsed = time.perf_counter() - start
    return sent, elapsed


async def main():
    print(f"Connecting to {BRIDGE_URL}")
    async with httpx.AsyncClient() as client:
        print(f"Sending {EVENT_COUNT} events in batches of {BATCH_SIZE}...")
        sent, elapsed = await send_events(client, EVENT_COUNT)
        rate = sent / elapsed if elapsed > 0 else 0
        print(f"\nSent {sent} events in {elapsed:.2f}s ({rate:.0f} events/s)")
        print("Check the WebSocket client (UI) for performance.")


if __name__ == "__main__":
    asyncio.run(main())
