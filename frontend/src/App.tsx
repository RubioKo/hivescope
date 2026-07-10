import { useWebSocket } from "./hooks/useWebSocket";
import { useWsStore } from "./store/wsStore";
import { GraphCanvas, ControlPanel, MessageLog } from "./components";

function ConnectionBadge() {
  const status = useWsStore((s) => s.status);
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
    </div>
  );
}

export default function App() {
  useWebSocket();

  return (
    <div className="relative w-full h-full">
      <ConnectionBadge />
      <ControlPanel />
      <GraphCanvas />
      <MessageLog />
    </div>
  );
}
