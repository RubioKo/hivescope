import { useEffect, useRef, useCallback, useState } from "react";
import { useWsStore } from "../store/wsStore";
import { useGraphStore } from "../store/graphStore";
import type { HiveEvent } from "../types";

const WS_HOST = import.meta.env.VITE_WS_HOST || "localhost";
const WS_PORT = import.meta.env.VITE_WS_PORT || 8765;
const WS_URL = `ws://${WS_HOST}:${WS_PORT}/ws`;
const MAX_RETRY_DELAY = 30000;

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout>>();
  const retryCountRef = useRef(0);
  const mountedRef = useRef(true);
  const [retryDelay, setRetryDelay] = useState(0);
  const { setStatus, addEvent, setError, error } = useWsStore();
  const addGraphEvent = useGraphStore((s) => s.addEvent);

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) return;

    setStatus("connecting");
    const ws = new WebSocket(WS_URL);

    ws.onopen = () => {
      retryCountRef.current = 0;
      setRetryDelay(0);
      setStatus("connected");
      setError(null);
    };

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
      if (!mountedRef.current) return;
      setStatus("disconnected");
      const delay = Math.min(1000 * Math.pow(2, retryCountRef.current), MAX_RETRY_DELAY);
      retryCountRef.current += 1;
      setRetryDelay(delay);
      reconnectTimer.current = setTimeout(connect, delay);
    };

    wsRef.current = ws;
  }, [setStatus, addEvent, addGraphEvent, setError]);

  useEffect(() => {
    mountedRef.current = true;
    connect();
    return () => {
      mountedRef.current = false;
      clearTimeout(reconnectTimer.current);
      wsRef.current?.close();
    };
  }, [connect]);

  const send = useCallback((data: unknown) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data));
    }
  }, []);

  return { send, retryDelay, error };
}
