# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY frontend/ .
RUN pnpm build

# Stage 2: Bridge
FROM python:3.11-slim AS bridge
WORKDIR /app/bridge
COPY bridge/ .
RUN pip install --no-cache-dir -e ".[dev]"

# Stage 3: Runtime
FROM python:3.11-slim
RUN pip install --no-cache-dir uvicorn

WORKDIR /app
COPY --from=bridge /app/bridge /app/bridge
COPY --from=frontend /app/frontend/dist /app/frontend/dist
RUN pip install --no-cache-dir -e /app/bridge

EXPOSE 8765 5173

CMD ["python", "-m", "uvicorn", "hivescope_bridge.main:app", "--host", "0.0.0.0", "--port", "8765"]
