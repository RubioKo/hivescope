import asyncio
import random
from contextlib import asynccontextmanager
from datetime import datetime, timezone

from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware

from .models import HiveEvent, WsCommand
from .store import store


@asynccontextmanager
async def lifespan(app: FastAPI):
    yield


app = FastAPI(title="HiveScope Bridge", version="0.0.1", lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:5173", "http://127.0.0.1:5173"],
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/health")
async def health():
    return {"status": "ok", "events_buffered": len(store)}


@app.get("/events")
async def get_events():
    return [e.model_dump(mode="json") for e in store.get_all()]


@app.post("/event")
async def ingest_event(event: HiveEvent):
    store.push(event)
    await _broadcast(event.model_dump_json())
    return {"ok": True}


@app.post("/demo/start")
async def demo_start():
    asyncio.create_task(_demo_loop())
    return {"ok": True, "message": "Demo events started"}


@app.websocket("/ws")
async def websocket_endpoint(ws: WebSocket):
    await ws.accept()
    _clients.add(ws)
    for event in store.get_all():
        await ws.send_text(event.model_dump_json())
    try:
        while True:
            raw = await ws.receive_text()
            cmd = WsCommand.model_validate_json(raw)
            if cmd.type == "clear":
                store.clear()
            elif cmd.type == "pause":
                pass
            elif cmd.type == "resume":
                pass
    except WebSocketDisconnect:
        pass
    finally:
        _clients.discard(ws)


_demo_running = False

_AGENTS = [
    {"id": "CEO", "type": "agent", "phase": "planning"},
    {"id": "Programmer", "type": "agent", "phase": "coding"},
    {"id": "Code Reviewer", "type": "agent", "phase": "reviewing"},
    {"id": "Tester", "type": "agent", "phase": "testing"},
    {"id": "Ollama", "type": "tool", "phase": "coding"},
    {"id": "File System", "type": "tool", "phase": "coding"},
]

_ACTIONS = [
    "message", "file_create", "file_edit", "state_change"
]

_PHASES = ["planning", "coding", "reviewing", "testing"]

_MESSAGES = {
    "CEO": [
        "We need to build a snake game with a scoring system and multiple difficulty levels.",
        "User wants a clean UI with dark mode support and keyboard controls.",
        "Prioritize the core game loop first, then add power-ups.",
        "Let's use Python with Pygame for the implementation.",
        "The deadline is end of week. Focus on MVP.",
    ],
    "Programmer": [
        "Implementing the main game loop with Pygame.",
        "Adding collision detection for walls and self-intersection.",
        "Creating the scoring system with level progression.",
        "Refactoring the input handler for arrow key support.",
        "Implementing the snake growth mechanic on food consumption.",
        "Writing the game state manager for pause/resume functionality.",
        "Adding sound effects for eating and collision events.",
    ],
    "Code Reviewer": [
        "The game loop logic looks solid. Consider adding delta time for consistent speed.",
        "Good use of enum for game states. The input queue pattern prevents missed keys.",
        "The collision detection could be optimized with a spatial hash map.",
        "Nice separation of concerns between rendering and game logic.",
        "Consider adding type hints to improve code maintainability.",
        "The food spawning logic needs to avoid spawning inside the snake body.",
    ],
    "Tester": [
        "Testing wall collision at coordinates (0,0) passes correctly.",
        "Snake growth test: food consumption increases length by 1.",
        "Edge case: snake at max length does not crash on food eat.",
        "Performance test: 1000 iterations at 60 FPS maintains stable frame time.",
        "Regression test: pause/resume does not reset score.",
    ],
    "Ollama": [
        "Generated code for snake game main loop.",
        "Processing review comments and generating fixes.",
        "Optimizing collision detection algorithm.",
        "Generating test suite for edge cases.",
    ],
    "File System": [
        "Created: main.py",
        "Created: game/engine.py",
        "Updated: game/engine.py",
        "Created: game/models.py",
        "Updated: config/settings.py",
        "Created: tests/test_collision.py",
        "Updated: tests/test_collision.py",
        "Created: assets/sprites.py",
    ],
}

_TARGETS = {
    "CEO": ["Programmer"],
    "Programmer": ["Code Reviewer", "File System", "Ollama"],
    "Code Reviewer": ["Programmer", "CEO"],
    "Tester": ["Programmer"],
    "Ollama": ["Programmer"],
    "File System": ["Programmer"],
}


async def _demo_loop():
    global _demo_running
    if _demo_running:
        return
    _demo_running = True

    store.clear()
    cycle = 0
    try:
        while _demo_running and cycle < 50:
            agent = random.choice(_AGENTS)
            targets = _TARGETS.get(agent["id"], [])
            target = random.choice(targets) if targets else None
            phase = random.choice(_PHASES)
            action = random.choice(_ACTIONS)
            messages = _MESSAGES.get(agent["id"], ["Working..."])
            message = random.choice(messages)

            file_list = []
            if action in ("file_create", "file_edit"):
                file_list = [random.choice(_MESSAGES["File System"])]

            event = HiveEvent(
                source=agent["id"],
                target=target,
                phase=phase,
                action=action,
                message=message,
                files=file_list,
                status=random.choice(["running", "success"]),
                node_type=agent["type"],
                timestamp=datetime.now(timezone.utc).isoformat(),
            )

            store.push(event)
            await _broadcast(event.model_dump_json())
            await asyncio.sleep(random.uniform(0.3, 1.5))
            cycle += 1
    finally:
        _demo_running = False


_clients: set[WebSocket] = set()


async def _broadcast(message: str):
    dead: list[WebSocket] = []
    for ws in _clients:
        try:
            await ws.send_text(message)
        except Exception:
            dead.append(ws)
    for ws in dead:
        _clients.discard(ws)
