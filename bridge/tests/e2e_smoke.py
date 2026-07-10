"""Quick end-to-end smoke test for the bridge."""

import asyncio

import httpx


async def main():
    async with httpx.AsyncClient() as c:
        h = await c.get("http://localhost:8765/health")
        print(f"HEALTH: {h.json()}")

        r = await c.post(
            "http://localhost:8765/event",
            json={"source": "Test", "phase": "x", "action": "y", "status": "running"},
        )
        print(f"INGEST: {r.json()}")

        ev = await c.get("http://localhost:8765/events")
        print(f"EVENTS: {len(ev.json())} stored")

        demo = await c.post("http://localhost:8765/demo/start")
        print(f"DEMO: {demo.json()}")

        await asyncio.sleep(2)
        ev2 = await c.get("http://localhost:8765/events")
        print(f"EVENTS AFTER DEMO: {len(ev2.json())} stored")

        print("ALL OK")


asyncio.run(main())
