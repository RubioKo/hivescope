from fastapi.testclient import TestClient

from hivescope_bridge.main import app

client = TestClient(app)


class TestWebSocket:
    def test_connect_and_receive(self):
        with client.websocket_connect("/ws") as ws:
            data = ws.receive_text()
            assert data is not None

    def test_send_command(self):
        with client.websocket_connect("/ws") as ws:
            ws.receive_text()
            ws.send_json({"type": "clear", "payload": {}})
            ws.send_json({"type": "pause", "payload": {}})
            ws.send_json({"type": "resume", "payload": {}})
