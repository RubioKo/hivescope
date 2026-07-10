import { useRef, useEffect, useState } from "react";
import { useWsStore } from "../store/wsStore";

export function MessageLog() {
  const events = useWsStore((s) => s.events);
  const logRef = useRef<HTMLDivElement>(null);
  const [collapsed, setCollapsed] = useState(false);
  const [search, setSearch] = useState("");

  useEffect(() => {
    if (logRef.current && !collapsed) {
      logRef.current.scrollTop = logRef.current.scrollHeight;
    }
  }, [events.length, collapsed]);

  const filtered = search
    ? events.filter(
        (e) =>
          e.source.toLowerCase().includes(search.toLowerCase()) ||
          (e.message && e.message.toLowerCase().includes(search.toLowerCase())),
      )
    : events;

  return (
    <div className="absolute bottom-4 left-4 right-4 z-10">
      <div className="rounded-lg border border-hive-border bg-hive-surface/95 backdrop-blur-sm shadow-xl">
        <div className="flex items-center justify-between px-3 py-2 border-b border-hive-border">
          <div className="flex items-center gap-2">
            <span className="text-xs text-hive-muted font-semibold uppercase tracking-wide">
              Events
            </span>
            <span className="text-[10px] text-hive-muted">({events.length})</span>
          </div>
          <div className="flex items-center gap-2">
            {!collapsed && (
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search..."
                className="w-28 px-2 py-0.5 text-[11px] rounded border border-hive-border bg-hive-bg text-hive-text placeholder-hive-muted outline-none focus:border-hive-blue"
              />
            )}
            <button
              onClick={() => setCollapsed(!collapsed)}
              className="text-xs text-hive-muted hover:text-hive-text transition-colors"
            >
              {collapsed ? "▲" : "▼"}
            </button>
          </div>
        </div>

        {!collapsed && (
          <div
            ref={logRef}
            className="max-h-40 overflow-y-auto p-3"
          >
            {filtered.length === 0 && (
              <p className="text-hive-muted text-xs">
                {search ? "No matches." : "Waiting for events..."}
              </p>
            )}
            {filtered.map((event, i) => (
              <div key={i} className="text-xs font-mono text-hive-text mb-1 last:mb-0">
                <span className="text-hive-muted">{event.timestamp.slice(11, 19)}</span>{" "}
                <span
                  className={
                    event.status === "error"
                      ? "text-hive-red"
                      : event.status === "success"
                        ? "text-hive-green"
                        : "text-hive-blue"
                  }
                >
                  {event.source}
                </span>
                <span className="text-hive-muted"> → </span>
                <span className="text-hive-blue">{event.target || "*"}</span>{" "}
                <span className="text-hive-muted">|</span>{" "}
                {event.message && <span>{event.message.slice(0, 120)}</span>}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
