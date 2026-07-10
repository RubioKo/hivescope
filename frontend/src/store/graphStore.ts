import { create } from "zustand";
import type { GraphNode, GraphLink, HiveEvent } from "../types";

interface GraphState {
  nodes: Map<string, GraphNode>;
  links: GraphLink[];
  selectedNodeId: string | null;
  paused: boolean;
  filters: {
    phases: Set<string>;
    statuses: Set<string>;
    search: string;
  };

  addEvent: (event: HiveEvent) => void;
  selectNode: (id: string | null) => void;
  setPaused: (paused: boolean) => void;
  setFilter: (key: "phases" | "statuses", value: string, active: boolean) => void;
  setSearch: (search: string) => void;
  clear: () => void;
}

export const useGraphStore = create<GraphState>((set) => ({
  nodes: new Map(),
  links: [],
  selectedNodeId: null,
  paused: false,
  filters: {
    phases: new Set(),
    statuses: new Set(),
    search: "",
  },

  addEvent: (event: HiveEvent) => {
    set((state) => {
      const nodes = new Map(state.nodes);
      const nodeId = event.source;

      if (!nodes.has(nodeId)) {
        nodes.set(nodeId, {
          id: nodeId,
          label: nodeId,
          type: event.node_type,
          status: event.status,
          phase: event.phase,
          parentId: event.parent_id,
          depth: 0,
        });
      } else {
        const existing = nodes.get(nodeId)!;
        nodes.set(nodeId, { ...existing, status: event.status, phase: event.phase });
      }

      const links = [...state.links];
      if (event.target && event.target !== nodeId) {
        links.push({
          source: nodeId,
          target: event.target,
          phase: event.phase,
          timestamp: event.timestamp,
        });
      }

      return { nodes, links };
    });
  },

  selectNode: (id) => set({ selectedNodeId: id }),

  setPaused: (paused) => set({ paused }),

  setFilter: (key, value, active) => {
    set((state) => {
      const filters = { ...state.filters };
      const set = new Set(filters[key]);
      if (active) set.add(value);
      else set.delete(value);
      filters[key] = set;
      return { filters };
    });
  },

  setSearch: (search) => {
    set((state) => ({ filters: { ...state.filters, search } }));
  },

  clear: () => set({ nodes: new Map(), links: [], selectedNodeId: null }),
}));
