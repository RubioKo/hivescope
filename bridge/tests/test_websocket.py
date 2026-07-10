from fastapi.testclient import TestClient

from hivescope_bridge.main import app

client = TestClient(app)


class TestWebSocket:
    def test_connect_receives_init(self):
        with client.websocket_connect("/ws") as ws:
            data = ws.receive_text()
            assert "init" in data

    def test_connect_receives_events(self):
        client.post(
            "/event", json={"source": "P", "phase": "p", "action": "m", "status": "running"}
        )
        with client.websocket_connect("/ws") as ws:
            first = ws.receive_text()
            assert "init" not in first
            second = ws.receive_text()
            assert "init" in second

    def test_send_command(self):
        with client.websocket_connect("/ws") as ws:
            ws.receive_text()
            ws.send_json({"type": "clear", "payload": {}})
            ws.send_json({"type": "pause", "payload": {}})
            ws.send_json({"type": "resume", "payload": {}})

    def test_event_received_after_post(self):
        with client.websocket_connect("/ws") as ws:
            ws.receive_text()
            client.post(
                "/event", json={"source": "WS", "phase": "t", "action": "m", "status": "running"}
            )
            received = ws.receive_text()
            assert "WS" in received

    def test_batch_broadcast(self):
        with client.websocket_connect("/ws") as ws:
            ws.receive_text()
            client.post(
                "/events",
                json=[
                    {"source": "B1", "phase": "p", "action": "m", "status": "running"},
                    {"source": "B2", "phase": "p", "action": "m", "status": "success"},
                ],
            )
            received = ws.receive_text()
            assert "B1" in received
            assert "B2" in received

    def test_reconnect_receives_history(self):
        client.post(
            "/event", json={"source": "HIST", "phase": "p", "action": "m", "status": "running"}
        )
        with client.websocket_connect("/ws") as ws:
            first = ws.receive_text()
            assert "HIST" in first
