from hivescope_bridge.models import HiveEvent, WsCommand


class TestHiveEvent:
    def test_minimal_event(self):
        event = HiveEvent(source="Programmer", phase="coding", action="message", status="running")
        assert event.schema_version == 1
        assert event.source == "Programmer"
        assert event.node_type == "agent"

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
        assert event.code == "print('hello')"
        assert event.parent_id == "session-1"

    def test_timestamp_defaults_to_iso(self):
        event = HiveEvent(source="Tester", phase="testing", action="message", status="running")
        assert "T" in event.timestamp
        assert event.timestamp.endswith("Z") or "+" in event.timestamp

    def test_node_type_default_agent(self):
        event = HiveEvent(source="X", phase="p", action="m", status="idle")
        assert event.node_type == "agent"

    def test_rejects_empty_source(self):
        try:
            HiveEvent(source="", phase="p", action="m", status="running")
            assert False, "Should have raised"
        except Exception as e:
            assert "source" in str(e).lower()

    def test_rejects_empty_phase(self):
        try:
            HiveEvent(source="X", phase="", action="m", status="running")
            assert False, "Should have raised"
        except Exception as e:
            assert "phase" in str(e).lower()

    def test_rejects_empty_action(self):
        try:
            HiveEvent(source="X", phase="p", action="", status="running")
            assert False, "Should have raised"
        except Exception as e:
            assert "action" in str(e).lower()

    def test_all_status_literals(self):
        for s in ("running", "success", "error", "idle"):
            event = HiveEvent(source="X", phase="p", action="m", status=s)  # type: ignore
            assert event.status == s

    def test_all_node_type_literals(self):
        for t in ("agent", "tool", "user", "gateway"):
            event = HiveEvent(source="X", phase="p", action="m", status="running", node_type=t)  # type: ignore
            assert event.node_type == t

    def test_defaults_for_optionals(self):
        event = HiveEvent(source="X", phase="p", action="m", status="running")
        assert event.files == []
        assert event.metadata == {}
        assert event.target is None
        assert event.message is None
        assert event.code is None
        assert event.parent_id is None


class TestWsCommand:
    def test_pause_command(self):
        cmd = WsCommand(type="pause")
        assert cmd.type == "pause"

    def test_clear_command(self):
        cmd = WsCommand(type="clear")
        assert cmd.type == "clear"

    def test_filter_command_with_payload(self):
        cmd = WsCommand(type="filter", payload={"phases": ["coding"]})
        assert cmd.payload == {"phases": ["coding"]}
