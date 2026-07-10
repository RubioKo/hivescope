from datetime import datetime, timezone
from typing import Literal

from pydantic import BaseModel, Field


class HiveEvent(BaseModel):
    schema_version: int = Field(default=1, ge=1)
    timestamp: str = Field(default_factory=lambda: datetime.now(timezone.utc).isoformat())
    source: str
    target: str | None = None
    phase: str
    action: str
    message: str | None = None
    files: list[str] = Field(default_factory=list)
    code: str | None = None
    status: Literal["running", "success", "error", "idle"]
    node_type: Literal["agent", "tool", "user", "gateway"] = "agent"
    parent_id: str | None = None
    metadata: dict = Field(default_factory=dict)


class WsCommand(BaseModel):
    type: Literal["pause", "resume", "filter", "clear"]
    payload: dict = Field(default_factory=dict)
