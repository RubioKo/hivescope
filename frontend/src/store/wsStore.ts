import { create } from "zustand";
import type { WsStatus, HiveEvent } from "../types";

interface WsState {
  status: WsStatus;
  events: HiveEvent[];
  error: string | null;
  setStatus: (status: WsStatus) => void;
  addEvent: (event: HiveEvent) => void;
  setError: (error: string | null) => void;
}

export const useWsStore = create<WsState>((set) => ({
  status: "disconnected",
  events: [],
  error: null,

  setStatus: (status) => set({ status }),

  addEvent: (event) => {
    set((state) => {
      const events = [...state.events, event];
      if (events.length > 5000) events.splice(0, events.length - 5000);
      return { events };
    });
  },

  setError: (error) => set({ error }),
}));
