export interface HiveEvent {
  schema_version: number;
  timestamp: string;
  source: string;
  target: string | null;
  phase: string;
  action: string;
  message: string | null;
  files: string[];
  code: string | null;
  status: "running" | "success" | "error" | "idle";
  node_type: "agent" | "tool" | "user" | "gateway";
  parent_id: string | null;
  metadata: Record<string, unknown>;
}

export interface GraphNode {
  id: string;
  label: string;
  type: HiveEvent["node_type"];
  status: HiveEvent["status"];
  phase: string;
  parentId: string | null;
  depth: number;
  lastEvent: HiveEvent | null;
}

export interface GraphLink {
  source: string;
  target: string;
  phase: string;
  timestamp: string;
  createdAt: number;
}

export type WsStatus = "connecting" | "connected" | "disconnected" | "error";
