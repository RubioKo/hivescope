import { useGraphStore } from "../store/graphStore";

const STATUS_LABELS: Record<string, string> = {
  running: "Running",
  success: "Success",
  error: "Error",
  idle: "Idle",
};

const STATUS_COLORS: Record<string, string> = {
  running: "text-hive-blue",
  success: "text-hive-green",
  error: "text-hive-red",
  idle: "text-hive-muted",
};

const STATUS_DOTS: Record<string, string> = {
  running: "bg-hive-blue",
  success: "bg-hive-green",
  error: "bg-hive-red",
  idle: "bg-hive-muted",
};

export function NodeInfoPanel() {
  const selectedNodeId = useGraphStore((s) => s.selectedNodeId);
  const nodes = useGraphStore((s) => s.nodes);
  const selectNode = useGraphStore((s) => s.selectNode);

  if (!selectedNodeId) return null;

  const node = nodes.get(selectedNodeId);
  if (!node) return null;

  const event = node.lastEvent;

  return (
    <div className="absolute top-4 left-1/2 -translate-x-1/2 z-20 w-80 max-w-[90vw]">
      <div className="rounded-lg border border-hive-border bg-hive-surface/95 backdrop-blur-sm shadow-xl">
        <div className="flex items-center justify-between px-4 py-3 border-b border-hive-border">
          <div className="flex items-center gap-2 min-w-0">
            <span className={`w-2 h-2 rounded-full shrink-0 ${STATUS_DOTS[node.status]}`} />
            <h3 className="text-sm font-semibold text-hive-text truncate">{node.label}</h3>
          </div>
          <button
            onClick={() => selectNode(null)}
            className="text-hive-muted hover:text-hive-text transition-colors shrink-0 ml-2"
          >
            ✕
          </button>
        </div>

        <div className="px-4 py-3 space-y-2 text-xs">
          <div className="flex gap-4">
            <span className="text-hive-muted">Type:</span>
            <span className="text-hive-text capitalize">{node.type}</span>
          </div>
          <div className="flex gap-4">
            <span className="text-hive-muted">Status:</span>
            <span className={STATUS_COLORS[node.status]}>{STATUS_LABELS[node.status]}</span>
          </div>
          <div className="flex gap-4">
            <span className="text-hive-muted">Phase:</span>
            <span className="text-hive-text capitalize">{node.phase || "—"}</span>
          </div>

          {event && event.message && (
            <div className="pt-2 border-t border-hive-border">
              <span className="text-hive-muted block mb-1">Last message:</span>
              <p className="text-hive-text leading-relaxed">{event.message}</p>
            </div>
          )}

          {event && event.files.length > 0 && (
            <div className="pt-2 border-t border-hive-border">
              <span className="text-hive-muted block mb-1">Files:</span>
              <div className="flex flex-wrap gap-1">
                {event.files.map((f, i) => (
                  <span
                    key={i}
                    className="px-1.5 py-0.5 rounded bg-hive-bg text-hive-blue text-[10px] font-mono"
                  >
                    {f}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
