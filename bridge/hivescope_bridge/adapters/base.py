from abc import ABC, abstractmethod

from ..models import HiveEvent


class BaseAdapter(ABC):
    @abstractmethod
    def start(self) -> None: ...

    @abstractmethod
    def stop(self) -> None: ...

    @abstractmethod
    def translate(self, raw_event: dict) -> HiveEvent: ...
