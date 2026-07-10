import json
import tempfile
import time
from pathlib import Path

from hivescope_bridge.adapters.chatdev import (
    ChatDevAdapter,
    discover_chatdev_tasks,
    find_latest_task_dir,
    find_phase_dirs,
    parse_chatdev_event,
    read_dialogue_files,
)
from hivescope_bridge.models import HiveEvent


class TestParseChatDevEvent:
    def test_minimal_entry(self):
        raw = {"role": "Programmer", "content": "Hello", "phase": "Coding"}
        event = parse_chatdev_event(raw)
        assert event is not None
        assert event.source == "Programmer"
        assert event.message == "Hello"
        assert event.phase == "coding"
        assert event.status == "running"
        assert event.node_type == "agent"

    def test_with_files(self):
        raw = {
            "role": "Programmer",
            "content": "Created main.py",
            "phase": "Coding",
            "files": ["main.py", "utils.py"],
        }
        event = parse_chatdev_event(raw)
        assert event is not None
        assert event.action == "file_create"
        assert event.files == ["main.py", "utils.py"]

    def test_terminated_status(self):
        raw = {"role": "CEO", "content": "Done", "phase": "Planning", "terminated": True}
        event = parse_chatdev_event(raw)
        assert event is not None
        assert event.status == "success"

    def test_with_target(self):
        raw = {
            "role": "Programmer",
            "target": "Code Reviewer",
            "content": "Review please",
            "phase": "Review",
        }
        event = parse_chatdev_event(raw)
        assert event is not None
        assert event.target == "Code Reviewer"
        assert event.phase == "reviewing"

    def test_list_of_entries(self):
        raws = [
            {"role": "A", "content": "hi", "phase": "Coding"},
            {"role": "B", "content": "ho", "phase": "Coding"},
        ]
        events = [e for e in (parse_chatdev_event(r) for r in raws) if e is not None]
        assert len(events) == 2
        assert events[0].source == "A"
        assert events[1].source == "B"

    def test_missing_role_uses_unknown(self):
        raw = {"content": "no role", "phase": "Coding"}
        event = parse_chatdev_event(raw)
        assert event is not None
        assert event.source == "unknown"

    def test_missing_message(self):
        raw = {"role": "Tester", "phase": "Testing"}
        event = parse_chatdev_event(raw)
        assert event is not None
        assert event.message == ""
        assert event.action == "message"

    def test_source_from_source_key(self):
        raw = {"source": "CustomAgent", "message": "hi", "phase": "Coding"}
        event = parse_chatdev_event(raw)
        assert event.source == "CustomAgent"

    def test_source_from_agent_key(self):
        raw = {"agent": "AgentX", "message": "hi", "phase": "Coding"}
        event = parse_chatdev_event(raw)
        assert event.source == "AgentX"

    def test_files_string_converted_to_list(self):
        raw = {"role": "P", "content": "file", "phase": "Coding", "files": "single_file.py"}
        event = parse_chatdev_event(raw)
        assert event is not None
        assert event.files == ["single_file.py"]

    def test_invalid_phase_preserved(self):
        raw = {"role": "P", "content": "hi", "phase": "UnknownPhase"}
        event = parse_chatdev_event(raw)
        assert event is not None
        assert event.phase == "UnknownPhase"


class TestDiscoverTasks:
    def test_discover_chatdev_tasks(self):
        with tempfile.TemporaryDirectory() as tmp:
            warehouse = Path(tmp)
            (warehouse / "Task_001").mkdir()
            (warehouse / "Task_002").mkdir()
            (warehouse / "not_a_dir").write_text("file", encoding="utf-8")
            tasks = discover_chatdev_tasks(warehouse)
            assert len(tasks) == 2
            assert {t.name for t in tasks} == {"Task_001", "Task_002"}

    def test_discover_no_warehouse(self):
        tasks = discover_chatdev_tasks(Path("/nonexistent/path"))
        assert tasks == []


class TestChatDevFileDiscovery:
    def test_find_latest_task(self):
        with tempfile.TemporaryDirectory() as tmp:
            warehouse = Path(tmp)
            (warehouse / "Task_001").mkdir()
            time.sleep(0.05)
            (warehouse / "Task_002").mkdir()
            latest = find_latest_task_dir(warehouse)
            assert latest is not None
            assert latest.name == "Task_002"

    def test_no_tasks(self):
        with tempfile.TemporaryDirectory() as tmp:
            warehouse = Path(tmp)
            assert find_latest_task_dir(warehouse) is None

    def test_find_phase_dirs(self):
        with tempfile.TemporaryDirectory() as tmp:
            task = Path(tmp) / "Task"
            task.mkdir()
            (task / "Planning").mkdir()
            (task / "Coding").mkdir()
            (task / "Extra").mkdir()
            phases = find_phase_dirs(task)
            assert len(phases) == 2
            assert {p.name for p in phases} == {"Planning", "Coding"}

    def test_find_phase_dirs_no_task_dir(self):
        assert find_phase_dirs(Path("/nonexistent")) == []


class TestReadDialogueFiles:
    def test_read_single_entry(self):
        with tempfile.TemporaryDirectory() as tmp:
            phase_dir = Path(tmp) / "Coding"
            phase_dir.mkdir(parents=True)
            dialogue = {"role": "Programmer", "content": "hello", "phase": "Coding"}
            (phase_dir / "agent.json").write_text(json.dumps(dialogue), encoding="utf-8")
            entries = read_dialogue_files(phase_dir)
            assert len(entries) == 1
            assert entries[0]["role"] == "Programmer"

    def test_read_list_entries(self):
        with tempfile.TemporaryDirectory() as tmp:
            phase_dir = Path(tmp) / "Planning"
            phase_dir.mkdir(parents=True)
            entries_data = [
                {"role": "CEO", "content": "Plan"},
                {"role": "Programmer", "content": "OK"},
            ]
            (phase_dir / "chat.json").write_text(json.dumps(entries_data), encoding="utf-8")
            entries = read_dialogue_files(phase_dir)
            assert len(entries) == 2
            assert entries[0]["role"] == "CEO"
            assert entries[1]["role"] == "Programmer"

    def test_invalid_json_skipped(self):
        with tempfile.TemporaryDirectory() as tmp:
            phase_dir = Path(tmp) / "Coding"
            phase_dir.mkdir(parents=True)
            (phase_dir / "bad.json").write_text("not json", encoding="utf-8")
            entries = read_dialogue_files(phase_dir)
            assert len(entries) == 0

    def test_non_json_files_skipped(self):
        with tempfile.TemporaryDirectory() as tmp:
            phase_dir = Path(tmp) / "Coding"
            phase_dir.mkdir(parents=True)
            (phase_dir / "readme.txt").write_text("hello", encoding="utf-8")
            (phase_dir / "data.csv").write_text("a,b,c", encoding="utf-8")
            entries = read_dialogue_files(phase_dir)
            assert len(entries) == 0

    def test_empty_phase_dir(self):
        with tempfile.TemporaryDirectory() as tmp:
            phase_dir = Path(tmp) / "Coding"
            phase_dir.mkdir(parents=True)
            entries = read_dialogue_files(phase_dir)
            assert entries == []

    def test_no_phase_dir(self):
        entries = read_dialogue_files(Path("/nonexistent"))
        assert entries == []

    def test_metadata_added(self):
        with tempfile.TemporaryDirectory() as tmp:
            phase_dir = Path(tmp) / "Coding"
            phase_dir.mkdir(parents=True)
            dialogue = {"role": "P", "content": "hi"}
            (phase_dir / "agent.json").write_text(json.dumps(dialogue), encoding="utf-8")
            entries = read_dialogue_files(phase_dir)
            assert "_phase" in entries[0]
            assert entries[0]["_phase"] == "Coding"
            assert "_file" in entries[0]


class TestChatDevAdapter:
    def test_start_stop(self):
        with tempfile.TemporaryDirectory() as tmp:
            warehouse = Path(tmp)
            (warehouse / "Task_001" / "Coding").mkdir(parents=True)
            dialogue = {"role": "Programmer", "content": "test", "phase": "Coding"}
            (warehouse / "Task_001" / "Coding" / "agent.json").write_text(
                json.dumps(dialogue), encoding="utf-8"
            )
            adapter = ChatDevAdapter(warehouse=warehouse, bridge_url="http://localhost:18765")
            adapter.start()
            time.sleep(0.5)
            assert adapter._thread is not None
            assert adapter._thread.is_alive()
            adapter.stop()
            assert not adapter._thread.is_alive()

    def test_get_task_summary_no_tasks(self):
        with tempfile.TemporaryDirectory() as tmp:
            adapter = ChatDevAdapter(warehouse=Path(tmp))
            summary = adapter.get_task_summary()
            assert "No ChatDev tasks found" in summary

    def test_get_task_summary_with_tasks(self):
        with tempfile.TemporaryDirectory() as tmp:
            warehouse = Path(tmp)
            task = warehouse / "Task_Snake" / "Coding"
            task.mkdir(parents=True)
            dialogue = {"role": "P", "content": "code"}
            (task / "agent.json").write_text(json.dumps(dialogue), encoding="utf-8")
            adapter = ChatDevAdapter(warehouse=warehouse)
            summary = adapter.get_task_summary()
            assert "Task_Snake" in summary
            assert "Coding" in summary
            assert "Events:" in summary

    def test_translate(self):
        adapter = ChatDevAdapter(warehouse=Path("/tmp"))
        raw = {"role": "CEO", "content": "Build it", "phase": "Planning"}
        event = adapter.translate(raw)
        assert isinstance(event, HiveEvent)
        assert event.source == "CEO"

    def test_task_summary_with_known_files(self):
        with tempfile.TemporaryDirectory() as tmp:
            warehouse = Path(tmp)
            coding = warehouse / "Task_001" / "Coding"
            coding.mkdir(parents=True)
            (coding / "agent.json").write_text(
                json.dumps({"role": "P", "content": "hi"}), encoding="utf-8"
            )
            adapter = ChatDevAdapter(warehouse=warehouse)
            adapter._known_files.add("Task_001/Coding/agent.json")
            summary = adapter.get_task_summary()
            assert "Task_001" in summary
