import { useGraphStore } from "../store/graphStore";

export function ControlPanel() {
  const paused = useGraphStore((s) => s.paused);
  const setPaused = useGraphStore((s) => s.setPaused);
  const clear = useGraphStore((s) => s.clear);

  return (
    <div className="absolute top-4 right-4 z-10 flex gap-2">
      <button
        onClick={() => setPaused(!paused)}
        className="px-3 py-1.5 text-sm rounded border border-hive-border bg-hive-surface text-hive-text hover:bg-hive-border transition-colors"
      >
        {paused ? "▶ Resume" : "⏸ Pause"}
      </button>
      <button
        onClick={clear}
        className="px-3 py-1.5 text-sm rounded border border-hive-border bg-hive-surface text-hive-text hover:bg-hive-border transition-colors"
      >
        ✕ Clear
      </button>
    </div>
  );
}
