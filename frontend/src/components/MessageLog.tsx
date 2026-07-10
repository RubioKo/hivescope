import { useRef, useEffect } from "react";
import { useWsStore } from "../store/wsStore";

export function MessageLog() {
  const events = useWsStore((s) => s.events);
  const logRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight;
    }
  }, [events.length]);

  return (
    <div className="absolute bottom-4 left-4 right-4 z-10">
      <div
        ref={logRef}
        className="max-h-40 overflow-y-auto rounded border border-hive-border bg-hive-surface/90 backdrop-blur-sm p-3"
      >
        {events.length === 0 && (
          <p className="text-hive-muted text-sm">Waiting for events...</p>
        )}
        {events.map((event, i) => (
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
    </div>
  );
}
