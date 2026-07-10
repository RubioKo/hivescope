<div align="center">

# HiveScope

**Real-time visualizer for multi-agent AI systems**

Every agent is a node. Every message is a pulse. Every file is a trail.

[![License: MIT](https://img.shields.io/badge/LICENSE-MIT-58A6FF?style=for-the-badge)](LICENSE)
[![Python](https://img.shields.io/badge/python-3.10%2B-3ECF8E?style=for-the-badge&logo=python)](bridge/)
[![TypeScript](https://img.shields.io/badge/typescript-5.6%2B-3178C6?style=for-the-badge&logo=typescript)](frontend/)
[![PRs Welcome](https://img.shields.io/badge/PRs-WELCOME-1F6FEB?style=for-the-badge)]()

`「 Local-first · Plug & Play · Zero cloud 」`

</div>

---

## What is HiveScope?

HiveScope turns plain-text agent logs into a **living graph**: agents are nodes, messages are pulses of light traveling between them, and generated files are inspectable in an info panel.

- **No cloud** — everything runs on `localhost`
- **No accounts** — zero telemetry leaves your machine
- **No framework modification** — non-invasive adapter architecture

---

## Quick start

```bash
# 1. Prerequisites
node >= 18
python >= 3.10
pnpm >= 9  (or npm)

# 2. Clone
git clone https://github.com/RubioKo/hivescope.git
cd hivescope

# 3. Frontend
cd frontend
pnpm install
pnpm dev        # → http://localhost:5173

# 4. Bridge (separate terminal)
cd bridge
pip install -e ".[dev]"
uvicorn hivescope_bridge.main:app --reload --port 8765

# 5. Open http://localhost:5173 and click "▶ Demo"
```

### Docker (one command)

```bash
docker compose up --build
# → http://localhost:5173
```

---

## Architecture

```
Agent Engine ──→ Adapter ──→ WebSocket ──→ Zustand Store ──→ D3.js Canvas
(ChatDev, etc.)  (pluggy)    (port 8765)                   (React + Vite)
                                                          ──→ ControlPanel
                                                          ──→ MessageLog
```

| Layer | Tech | Role |
|-------|------|------|
| **Frontend** | React 18 + TypeScript + Vite + Tailwind CSS + Zustand + react-force-graph-2d | Real-time graph visualization, filter/search, theme toggle |
| **Bridge** | FastAPI + Pydantic v2 + Uvicorn + WebSocket | Event ingestion, ring buffer, broadcast to all clients |
| **Adapters** | pluggy hookspec + ABC (`BaseAdapter`) | Translate engine-specific events to `HiveEvent` schema |
| **Transport** | WebSocket (bidirectional) | Live JSON stream + command channel (clear, pause) |

### Data schema

```typescript
interface HiveEvent {
  schema_version: 1;
  timestamp: string;              // ISO 8601 UTC
  source: string;                 // agent / tool name
  target: string | null;          // recipient
  phase: "planning" | "coding" | "reviewing" | "testing";
  action: "message" | "file_create" | "file_edit" | "state_change";
  message: string | null;
  files: string[];
  code: string | null;
  status: "running" | "success" | "error" | "idle";
  node_type: "agent" | "tool" | "user" | "gateway";
  parent_id: string | null;
  metadata: Record<string, unknown>;
}
```

---

## Features

| | Feature | Status |
|---|---|---|
| 🕸️ | Force-directed agent graph with custom node shapes | ✅ Phase 2 |
| 💫 | Animated link particles on recent messages | ✅ Phase 2 |
| 🫀 | Pulsing ring animation for active agents | ✅ Phase 2 |
| 🔍 | Node click → info panel (status, phase, last message, files) | ✅ Phase 2 |
| 🎚️ | Phase/status filter checkboxes + node search | ✅ Phase 4 |
| 📜 | Searchable, collapsible message log with event counter | ✅ Phase 4 |
| ☀/☾ | Dark/light theme toggle with localStorage persistence | ✅ Phase 4 |
| 🔄 | Exponential backoff reconnection (1s → 30s max) | ✅ Phase 5 |
| 🧩 | ChatDev 2.0 adapter (filesystem poller, CLI entry point) | ✅ Phase 3 |
| ▶ | Demo mode: 6 synthetic agents, 100 events, phase-colored | ✅ Phase 1 |
| 🛡️ | React ErrorBoundary, WebSocket init message, model validation | ✅ Phase 6 |
| 🧪 | 100 tests: 65 Python (bridge) + 35 TypeScript (frontend) | ✅ Phase 6 |

---

## API Reference

### REST Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Bridge status + buffered event count |
| `GET` | `/events` | Retrieve all buffered events (JSON array) |
| `POST` | `/event` | Ingest a single `HiveEvent` |
| `POST` | `/events` | Ingest a batch of `HiveEvent` (array) |
| `POST` | `/demo/start` | Start synthetic demo event stream |

### WebSocket (`/ws`)

- **On connect**: receives all buffered events + `{"type":"init","count":N}`
- **Incoming commands**: `{"type":"clear"}` / `{"type":"pause"}` / `{"type":"resume"}` / `{"type":"filter","payload":{...}}`
- **Outgoing**: each ingested event is broadcast to all connected clients

---

## Development

```bash
# Bridge
cd bridge
pip install -e ".[dev]"
ruff check . && ruff format --check .
pytest

# Frontend
cd frontend
pnpm install
pnpm test
pnpm build
```

### Stress test

```bash
cd bridge
python scripts/stress_test.py
# → Sends 1000 events in batches of 50, measures throughput
```

---

## Project structure

```
hivescope/
├── frontend/                 # React + Vite + TypeScript
│   ├── src/
│   │   ├── components/       # GraphCanvas, ControlPanel, FilterPanel, MessageLog, NodeInfoPanel, ErrorBoundary
│   │   ├── store/            # Zustand: graphStore, wsStore, themeStore
│   │   ├── hooks/            # useWebSocket (exponential backoff)
│   │   ├── types.ts          # HiveEvent, GraphNode, GraphLink
│   │   └── test/             # Vitest setup
│   ├── index.html
│   └── vite.config.ts
├── bridge/
│   ├── hivescope_bridge/     # FastAPI + Pydantic models + ring buffer
│   │   ├── adapters/         # BaseAdapter, registry (pluggy), ChatDev adapter
│   │   ├── main.py           # FastAPI app + WebSocket + demo loop
│   │   ├── models.py         # HiveEvent, WsCommand
│   │   └── store.py          # Ring buffer (deque, max 5000)
│   ├── scripts/
│   │   └── stress_test.py    # 1000-event burst test
│   └── tests/                # 65 tests (pytest)
├── Dockerfile                # Multi-stage build
├── docker-compose.yml        # Frontend + bridge orchestration
└── README.md
```

---

## Adapters

HiveScope uses a **pluggy-based adapter system** to support multiple agent frameworks:

| Adapter | Status | Method |
|---------|--------|--------|
| **ChatDev 2.0** | ✅ | Filesystem poller (WareHouse/ directory) |
| CrewAI | 📋 Planned | — |
| LangGraph | 📋 Planned | — |
| AutoGen | 📋 Planned | — |

### Using the ChatDev adapter

```bash
# While HiveScope bridge is running, in another terminal:
chatdev-with-hivescope "build a snake game"
# or attach to an existing task:
chatdev-with-hivescope --attach /path/to/ChatDev/WareHouse/Task_001
```

---

## Release

```bash
# Create a new release
git tag v1.0.0
git push origin v1.0.0
# → GitHub Action builds, tests, creates release, publishes to PyPI + npm
```

---

## Contributing

1. Fork the repo
2. Create a feature branch from `dev`
3. Run tests: `pytest` (bridge) + `pnpm test` (frontend)
4. Run `gitleaks detect --source . -v` before committing
5. Open a PR to `dev` (never `main`)

---

<div align="center">

Built with open source, for the open-source community.

</div>
