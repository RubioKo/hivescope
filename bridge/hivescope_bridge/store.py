from collections import deque

from .models import HiveEvent


class EventStore:
    def __init__(self, maxlen: int = 5000):
        self._events: deque[HiveEvent] = deque(maxlen=maxlen)

    def push(self, event: HiveEvent) -> None:
        self._events.append(event)

    def get_all(self) -> list[HiveEvent]:
        return list(self._events)

    def clear(self) -> None:
        self._events.clear()

    def __len__(self) -> int:
        return len(self._events)


store = EventStore()
