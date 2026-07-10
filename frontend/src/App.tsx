import { useWebSocket } from "./hooks/useWebSocket";
import { useWsStore } from "./store/wsStore";
import { GraphCanvas, ControlPanel, MessageLog, NodeInfoPanel, FilterPanel, ErrorBoundary } from "./components";

function ConnectionBadge() {
  const status = useWsStore((s) => s.status);
  const error = useWsStore((s) => s.error);
  const colors: Record<string, string> = {
    connected: "bg-hive-green",
    connecting: "bg-hive-yellow",
    disconnected: "bg-hive-muted",
    error: "bg-hive-red",
  };

  return (
    <div className="absolute top-4 left-4 z-10 flex items-center gap-2">
      <span className={`w-2 h-2 rounded-full ${colors[status]}`} />
      <span className="text-xs text-hive-muted font-mono">{status}</span>
      {status === "disconnected" && error && (
        <span className="text-xs text-hive-red font-mono ml-1">{error}</span>
      )}
    </div>
  );
}

export default function App() {
  useWebSocket();

  return (
    <ErrorBoundary>
      <div className="relative w-full h-full bg-hive-bg transition-colors">
        <ConnectionBadge />
        <FilterPanel />
        <ControlPanel />
        <NodeInfoPanel />
        <GraphCanvas />
        <MessageLog />
      </div>
    </ErrorBoundary>
  );
}
