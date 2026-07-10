import { useEffect, useRef, useCallback } from "react";
import { useWsStore } from "../store/wsStore";
import { useGraphStore } from "../store/graphStore";
import type { HiveEvent } from "../types";

const WS_URL = `ws://${import.meta.env.VITE_WS_HOST || "localhost"}:${import.meta.env.VITE_WS_PORT || 8765}/ws`;

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout>>();
  const { setStatus, addEvent, setError } = useWsStore();
  const addGraphEvent = useGraphStore((s) => s.addEvent);

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) return;

    setStatus("connecting");
    const ws = new WebSocket(WS_URL);

    ws.onopen = () => setStatus("connected");

    ws.onmessage = (msg) => {
      try {
        const event: HiveEvent = JSON.parse(msg.data);
        addEvent(event);
        addGraphEvent(event);
      } catch {
        setError("Failed to parse event");
      }
    };

    ws.onerror = () => {
      setError("WebSocket error");
      setStatus("error");
    };

    ws.onclose = () => {
      setStatus("disconnected");
      reconnectTimer.current = setTimeout(connect, 3000);
    };

    wsRef.current = ws;
  }, [setStatus, addEvent, addGraphEvent, setError]);

  useEffect(() => {
    connect();
    return () => {
      clearTimeout(reconnectTimer.current);
      wsRef.current?.close();
    };
  }, [connect]);

  const send = useCallback((data: unknown) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data));
    }
  }, []);

  return { send };
}
