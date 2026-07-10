import json
import time
from datetime import datetime, timezone
from pathlib import Path
from threading import Event, Thread

from ..adapters.base import BaseAdapter
from ..adapters.registry import hookimpl
from ..models import HiveEvent

CHATDEV_PHASES = {
    "Planning": "planning",
    "Coding": "coding",
    "Review": "reviewing",
    "Testing": "testing",
}


def parse_chatdev_event(raw: dict) -> HiveEvent | None:
    """Convert a raw ChatDev dialogue entry into a HiveEvent."""
    role = raw.get("role") or raw.get("source") or raw.get("agent", "unknown")
    content = raw.get("content") or raw.get("message", "")
    phase = CHATDEV_PHASES.get(raw.get("phase", ""), raw.get("phase", "coding"))
    files = raw.get("files", [])
    action = "file_create" if files else "message"
    status = "success" if raw.get("terminated") else "running"

    target = None
    if raw.get("target"):
        target = raw["target"]

    return HiveEvent(
        source=role,
        target=target,
        phase=phase,
        action=action,
        message=content,
        files=files if isinstance(files, list) else [files],
        status=status,
        node_type="agent",
        timestamp=datetime.now(timezone.utc).isoformat(),
        metadata={"adapter": "chatdev"},
    )


def discover_chatdev_tasks(warehouse: Path) -> list[Path]:
    """Discover ChatDev task directories in WareHouse/."""
    if not warehouse.exists():
        return []
    return [p for p in warehouse.iterdir() if p.is_dir()]


def find_latest_task_dir(warehouse: Path) -> Path | None:
    """Find the most recently modified task directory."""
    dirs = discover_chatdev_tasks(warehouse)
    if not dirs:
        return None
    return max(dirs, key=lambda d: d.stat().st_mtime)


def find_phase_dirs(task_dir: Path) -> list[Path]:
    """Find phase subdirectories (Planning, Coding, Review, Testing)."""
    if not task_dir.exists():
        return []
    return [p for p in task_dir.iterdir() if p.is_dir() and p.name in CHATDEV_PHASES]


def read_dialogue_files(phase_dir: Path) -> list[dict]:
    """Read all JSON dialogue files in a phase directory."""
    events: list[dict] = []
    if not phase_dir.exists():
        return events
    for f in sorted(phase_dir.iterdir()):
        if f.suffix == ".json":
            try:
                data = json.loads(f.read_text(encoding="utf-8"))
                if isinstance(data, dict):
                    data["_phase"] = phase_dir.name
                    data["_file"] = str(f.relative_to(phase_dir.parent.parent))
                    events.append(data)
                elif isinstance(data, list):
                    for item in data:
                        if isinstance(item, dict):
                            item["_phase"] = phase_dir.name
                            item["_file"] = str(f.relative_to(phase_dir.parent.parent))
                            events.append(item)
            except (json.JSONDecodeError, UnicodeDecodeError):
                pass
    return events


class ChatDevAdapter(BaseAdapter):
    """Adapter that watches a ChatDev WareHouse/ directory and streams events to the bridge."""

    def __init__(self, warehouse: str | Path, bridge_url: str = "http://localhost:8765"):
        self.warehouse = Path(warehouse)
        self.bridge_url = bridge_url.rstrip("/")
        self._stop_event = Event()
        self._thread: Thread | None = None
        self._known_files: set[str] = set()

    def start(self) -> None:
        self._stop_event.clear()
        self._thread = Thread(target=self._poll_loop, daemon=True)
        self._thread.start()

    def stop(self) -> None:
        self._stop_event.set()
        if self._thread:
            self._thread.join(timeout=5)

    def translate(self, raw_event: dict) -> HiveEvent:
        return parse_chatdev_event(raw_event)

    def _poll_loop(self) -> None:
        import httpx

        while not self._stop_event.is_set():
            task_dir = find_latest_task_dir(self.warehouse)
            if task_dir:
                phase_dirs = find_phase_dirs(task_dir)
                for phase_dir in phase_dirs:
                    for file_path in sorted(phase_dir.iterdir()):
                        if file_path.suffix != ".json":
                            continue
                        key = str(file_path.relative_to(self.warehouse))
                        if key in self._known_files:
                            continue
                        self._known_files.add(key)

                        entries = read_dialogue_files(phase_dir)
                        for entry in entries:
                            event = parse_chatdev_event(entry)
                            if event:
                                try:
                                    with httpx.Client() as c:
                                        c.post(
                                            f"{self.bridge_url}/event",
                                            json=event.model_dump(mode="json"),
                                            timeout=2,
                                        )
                                except Exception:
                                    pass
            time.sleep(1)

    def get_task_summary(self) -> str:
        """Return a summary of detected ChatDev tasks."""
        task_dir = find_latest_task_dir(self.warehouse)
        if not task_dir:
            return "No ChatDev tasks found."
        phase_dirs = find_phase_dirs(task_dir)
        event_count = sum(len(read_dialogue_files(p)) for p in phase_dirs)
        phases = ", ".join(p.name for p in phase_dirs)
        return f"Task: {task_dir.name} | Phases: {phases} | Events: {event_count}"


@hookimpl
def hivescope_adapter() -> type:
    return ChatDevAdapter


def cli_main():
    """Entry point for the `chatdev-with-hivescope` CLI command."""
    import argparse
    import subprocess
    import sys
    import time
    from pathlib import Path

    parser = argparse.ArgumentParser(description="Run ChatDev with HiveScope visualization")
    parser.add_argument("task", nargs="?", help="Task description for ChatDev")
    parser.add_argument("--attach", help="Attach to existing ChatDev task directory")
    parser.add_argument(
        "--bridge-url",
        default="http://localhost:8765",
    )
    parser.add_argument(
        "--warehouse",
        default="WareHouse",
    )
    parser.add_argument(
        "--chatdev-cmd",
        default="chatdev",
    )
    args = parser.parse_args()

    warehouse = Path(args.warehouse).resolve()

    if args.attach:
        task_dir = Path(args.attach).resolve()
        if not task_dir.exists():
            print(f"Error: task directory not found: {task_dir}", file=sys.stderr)
            sys.exit(1)
        warehouse = task_dir.parent
        print(f"Attaching to task: {task_dir.name}")
    elif args.task:
        print(f"Launching ChatDev: {args.task}")
        cwd = str(warehouse.parent) if warehouse.parent.exists() else None
        subprocess.Popen([args.chatdev_cmd, args.task], cwd=cwd)
        print("Waiting for task directory...")
        for _ in range(30):
            td = find_latest_task_dir(warehouse)
            if td:
                print(f"Found: {td.name}")
                break
            time.sleep(1)
    else:
        print("Watching for new ChatDev tasks...")

    adapter = ChatDevAdapter(warehouse=warehouse, bridge_url=args.bridge_url)
    adapter.start()
    print(f"Watching: {warehouse} → {args.bridge_url}")
    print("Ctrl+C to stop.")

    try:
        while True:
            time.sleep(5)
            print(adapter.get_task_summary())
    except KeyboardInterrupt:
        print("\nStopping...")
    finally:
        adapter.stop()
        print("Done.")
