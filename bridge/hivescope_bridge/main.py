from contextlib import asynccontextmanager

from fastapi import FastAPI, WebSocket, WebSocketDisconnect

from .models import HiveEvent, WsCommand
from .store import store


@asynccontextmanager
async def lifespan(app: FastAPI):
    yield


app = FastAPI(title="HiveScope Bridge", version="0.0.1", lifespan=lifespan)


@app.get("/health")
async def health():
    return {"status": "ok", "events_buffered": len(store)}


@app.post("/event")
async def ingest_event(event: HiveEvent):
    store.push(event)
    await _broadcast(event.model_dump_json())
    return {"ok": True}


@app.websocket("/ws")
async def websocket_endpoint(ws: WebSocket):
    await ws.accept()
    _clients.add(ws)
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
