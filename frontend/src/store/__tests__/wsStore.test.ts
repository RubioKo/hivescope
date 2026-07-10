import { describe, it, expect, beforeEach } from "vitest";
import { useWsStore } from "../wsStore";
import type { HiveEvent } from "../../types";

function makeEvent(): HiveEvent {
  return {
    schema_version: 1,
    timestamp: "2026-01-01T00:00:00Z",
    source: "Agent",
    target: null,
    phase: "p",
    action: "message",
    message: "test",
    files: [],
    code: null,
    status: "running",
    node_type: "agent",
    parent_id: null,
    metadata: {},
  };
}

describe("wsStore", () => {
  beforeEach(() => {
    useWsStore.setState({ status: "disconnected", events: [], error: null });
  });

  it("starts disconnected with empty events", () => {
    const { status, events, error } = useWsStore.getState();
    expect(status).toBe("disconnected");
    expect(events).toEqual([]);
    expect(error).toBeNull();
  });

  it("setStatus updates status", () => {
    useWsStore.getState().setStatus("connected");
    expect(useWsStore.getState().status).toBe("connected");
  });

  it("addEvent appends to events", () => {
    useWsStore.getState().addEvent(makeEvent());
    useWsStore.getState().addEvent(makeEvent());
    expect(useWsStore.getState().events.length).toBe(2);
  });

  it("setError sets error message", () => {
    useWsStore.getState().setError("connection failed");
    expect(useWsStore.getState().error).toBe("connection failed");
  });

  it("setError(null) clears error", () => {
    useWsStore.getState().setError("err");
    useWsStore.getState().setError(null);
    expect(useWsStore.getState().error).toBeNull();
  });
});
