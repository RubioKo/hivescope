import { describe, it, expect, beforeEach } from "vitest";
import { useGraphStore } from "../graphStore";
import type { HiveEvent } from "../../types";

function makeEvent(overrides: Partial<HiveEvent> = {}): HiveEvent {
  return {
    schema_version: 1,
    timestamp: "2026-01-01T00:00:00Z",
    source: "AgentA",
    target: null,
    phase: "coding",
    action: "message",
    message: "hello",
    files: [],
    code: null,
    status: "running",
    node_type: "agent",
    parent_id: null,
    metadata: {},
    ...overrides,
  };
}

describe("graphStore", () => {
  beforeEach(() => {
    useGraphStore.getState().clear();
  });

  it("starts empty", () => {
    const { nodes, links, eventLog, paused } = useGraphStore.getState();
    expect(nodes.size).toBe(0);
    expect(links.length).toBe(0);
    expect(eventLog.length).toBe(0);
    expect(paused).toBe(false);
  });

  it("addEvent creates a node", () => {
    useGraphStore.getState().addEvent(makeEvent({ source: "Programmer" }));
    const nodes = useGraphStore.getState().nodes;
    expect(nodes.size).toBe(1);
    expect(nodes.has("Programmer")).toBe(true);
    expect(nodes.get("Programmer")!.type).toBe("agent");
  });

  it("addEvent updates existing node status", () => {
    useGraphStore.getState().addEvent(makeEvent({ source: "A", status: "running" }));
    useGraphStore.getState().addEvent(makeEvent({ source: "A", status: "success" }));
    const node = useGraphStore.getState().nodes.get("A")!;
    expect(node.status).toBe("success");
  });

  it("addEvent creates link when target is set", () => {
    useGraphStore.getState().addEvent(makeEvent({ source: "A", target: "B" }));
    const links = useGraphStore.getState().links;
    expect(links.length).toBe(1);
    expect(links[0].source).toBe("A");
    expect(links[0].target).toBe("B");
  });

  it("addEvent does not create link to self", () => {
    useGraphStore.getState().addEvent(makeEvent({ source: "A", target: "A" }));
    expect(useGraphStore.getState().links.length).toBe(0);
  });

  it("addEvent deduplicates recent links", () => {
    useGraphStore.getState().addEvent(makeEvent({ source: "A", target: "B" }));
    useGraphStore.getState().addEvent(makeEvent({ source: "A", target: "B" }));
    expect(useGraphStore.getState().links.length).toBe(1);
  });

  it("addEvent appends to eventLog", () => {
    useGraphStore.getState().addEvent(makeEvent({ source: "A" }));
    useGraphStore.getState().addEvent(makeEvent({ source: "B" }));
    expect(useGraphStore.getState().eventLog.length).toBe(2);
  });

  it("addEvent respects paused state", () => {
    useGraphStore.getState().setPaused(true);
    useGraphStore.getState().addEvent(makeEvent({ source: "A" }));
    expect(useGraphStore.getState().nodes.size).toBe(0);
  });

  it("selectNode updates selectedNodeId", () => {
    useGraphStore.getState().selectNode("AgentX");
    expect(useGraphStore.getState().selectedNodeId).toBe("AgentX");
    useGraphStore.getState().selectNode(null);
    expect(useGraphStore.getState().selectedNodeId).toBeNull();
  });

  it("setPaused toggles paused", () => {
    useGraphStore.getState().setPaused(true);
    expect(useGraphStore.getState().paused).toBe(true);
    useGraphStore.getState().setPaused(false);
    expect(useGraphStore.getState().paused).toBe(false);
  });

  it("setFilter adds phase filter", () => {
    useGraphStore.getState().setFilter("phases", "coding", true);
    expect(useGraphStore.getState().filters.phases.has("coding")).toBe(true);
  });

  it("setFilter removes phase filter", () => {
    useGraphStore.getState().setFilter("phases", "coding", true);
    useGraphStore.getState().setFilter("phases", "coding", false);
    expect(useGraphStore.getState().filters.phases.has("coding")).toBe(false);
  });

  it("setFilter adds status filter", () => {
    useGraphStore.getState().setFilter("statuses", "error", true);
    expect(useGraphStore.getState().filters.statuses.has("error")).toBe(true);
  });

  it("setSearch updates search text", () => {
    useGraphStore.getState().setSearch("CEO");
    expect(useGraphStore.getState().filters.search).toBe("CEO");
  });

  it("clear resets all state", () => {
    useGraphStore.getState().addEvent(makeEvent({ source: "A" }));
    useGraphStore.getState().setPaused(true);
    useGraphStore.getState().setSearch("test");
    useGraphStore.getState().clear();
    const s = useGraphStore.getState();
    expect(s.nodes.size).toBe(0);
    expect(s.links.length).toBe(0);
    expect(s.selectedNodeId).toBeNull();
    expect(s.eventLog.length).toBe(0);
  });

  it("eventLog caps at 500", () => {
    useGraphStore.setState({ paused: false, eventLog: [] });
    for (let i = 0; i < 600; i++) {
      useGraphStore.getState().addEvent(makeEvent({ source: `A${i}` }));
    }
    expect(useGraphStore.getState().eventLog.length).toBe(500);
  });
});
