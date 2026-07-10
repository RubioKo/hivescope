from fastapi.testclient import TestClient

from hivescope_bridge.main import app

client = TestClient(app)


class TestHealth:
    def test_health_returns_ok(self):
        resp = client.get("/health")
        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "ok"
        assert "events_buffered" in data


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

    def test_get_events(self):
        resp = client.get("/events")
        assert resp.status_code == 200
        assert isinstance(resp.json(), list)

    def test_ingest_invalid_event(self):
        resp = client.post("/event", json={"source": "x"})
        assert resp.status_code == 422


class TestDemo:
    def test_demo_start(self):
        global _demo_running
        _demo_running = False
        resp = client.post("/demo/start")
        assert resp.status_code == 200
        data = resp.json()
        assert data["ok"] is True
