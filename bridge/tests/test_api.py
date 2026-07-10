from fastapi.testclient import TestClient

import hivescope_bridge.main as bridge_main

client = TestClient(bridge_main.app)


class TestHealth:
    def test_health_returns_ok(self):
        resp = client.get("/health")
        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "ok"
        assert "events_buffered" in data

    def test_health_after_events(self):
        client.post(
            "/event", json={"source": "T", "phase": "p", "action": "m", "status": "running"}
        )
        client.post(
            "/event", json={"source": "T2", "phase": "p", "action": "m", "status": "success"}
        )
        resp = client.get("/health")
        assert resp.json()["events_buffered"] == 2


class TestEvents:
    def test_ingest_event(self):
        resp = client.post(
            "/event",
            json={
                "source": "Programmer",
                "phase": "coding",
                "action": "message",
                "message": "hello",
                "status": "running",
            },
        )
        assert resp.status_code == 200
        assert resp.json()["ok"] is True

    def test_ingest_batch(self):
        resp = client.post(
            "/events",
            json=[
                {"source": "A", "phase": "p", "action": "m", "status": "running"},
                {"source": "B", "phase": "p", "action": "m", "status": "success"},
                {"source": "C", "phase": "p", "action": "m", "status": "error"},
            ],
        )
        assert resp.status_code == 200
        data = resp.json()
        assert data["ok"] is True
        assert data["count"] == 3

    def test_ingest_empty_batch(self):
        resp = client.post("/events", json=[])
        assert resp.status_code == 200
        assert resp.json()["count"] == 0

    def test_get_events(self):
        resp = client.get("/events")
        assert resp.status_code == 200
        assert isinstance(resp.json(), list)

    def test_ingest_invalid_event(self):
        resp = client.post("/event", json={"source": "x"})
        assert resp.status_code == 422

    def test_ingest_batch_invalid_item(self):
        resp = client.post("/events", json=[{"source": "A"}, {"bad": True}])
        assert resp.status_code == 422


class TestDemo:
    def test_demo_start(self):
        resp = client.post("/demo/start")
        assert resp.status_code == 200
        data = resp.json()
        assert data["ok"] is True

    def test_demo_already_running(self):
        bridge_main._demo_running = True
        resp = client.post("/demo/start")
        assert resp.status_code == 200
        data = resp.json()
        assert data["ok"] is False
        assert "already running" in data["message"].lower()
        bridge_main._demo_running = False


class TestCors:
    def test_cors_headers(self):
        resp = client.options(
            "/health",
            headers={
                "Origin": "http://localhost:5173",
                "Access-Control-Request-Method": "GET",
            },
        )
        assert resp.status_code == 200
        assert resp.headers.get("access-control-allow-origin") == "http://localhost:5173"
