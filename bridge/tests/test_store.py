from hivescope_bridge.models import HiveEvent
from hivescope_bridge.store import EventStore


class TestEventStore:
    def test_push_and_get_all(self):
        store = EventStore()
        event = HiveEvent(source="A", phase="p", action="m", status="running")
        store.push(event)
        assert len(store) == 1
        assert store.get_all()[0].source == "A"

    def test_clear(self):
        store = EventStore()
        store.push(HiveEvent(source="A", phase="p", action="m", status="running"))
        store.clear()
        assert len(store) == 0

    def test_maxlen_eviction(self):
        store = EventStore(maxlen=3)
        for i in range(5):
            store.push(HiveEvent(source=str(i), phase="p", action="m", status="running"))
        assert len(store) == 3
        assert store.get_all()[0].source == "2"
        assert store.get_all()[-1].source == "4"

    def test_get_all_returns_copy(self):
        store = EventStore()
        store.push(HiveEvent(source="A", phase="p", action="m", status="running"))
        events = store.get_all()
        events.append(HiveEvent(source="B", phase="p", action="m", status="running"))
        assert len(store) == 1
