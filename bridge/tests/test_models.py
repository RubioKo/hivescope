from hivescope_bridge.models import HiveEvent, WsCommand


class TestHiveEvent:
    def test_minimal_event(self):
        event = HiveEvent(
            source="Programmer",
            phase="coding",
            action="message",
            status="running",
        )
        assert event.schema_version == 1
        assert event.source == "Programmer"
        assert event.phase == "coding"
        assert event.action == "message"
        assert event.status == "running"
        assert event.node_type == "agent"
        assert event.target is None
        assert event.message is None
        assert event.files == []
        assert event.code is None
        assert event.parent_id is None
        assert event.metadata == {}

    def test_full_event(self):
        event = HiveEvent(
            source="CEO",
            target="Programmer",
            phase="planning",
            action="message",
            message="Build a snake game",
            files=["main.py"],
            code="print('hello')",
            status="success",
            node_type="agent",
            parent_id="session-1",
            metadata={"task": "snake"},
        )
        assert event.source == "CEO"
        assert event.target == "Programmer"
        assert event.phase == "planning"
        assert event.message == "Build a snake game"
        assert event.files == ["main.py"]
        assert event.code == "print('hello')"
        assert event.status == "success"
        assert event.node_type == "agent"
        assert event.parent_id == "session-1"
        assert event.metadata == {"task": "snake"}

    def test_timestamp_defaults_to_iso(self):
        event = HiveEvent(source="Tester", phase="testing", action="message", status="running")
        assert "T" in event.timestamp
        assert event.timestamp.endswith("Z") or "+" in event.timestamp


class TestWsCommand:
    def test_pause_command(self):
        cmd = WsCommand(type="pause")
        assert cmd.type == "pause"
        assert cmd.payload == {}

    def test_clear_command(self):
        cmd = WsCommand(type="clear")
        assert cmd.type == "clear"
