import { useState } from "react";
import { useGraphStore } from "../store/graphStore";

const PHASES = ["planning", "coding", "reviewing", "testing"];
const STATUSES = ["running", "success", "error", "idle"];

export function FilterPanel() {
  const [open, setOpen] = useState(false);
  const filters = useGraphStore((s) => s.filters);
  const setFilter = useGraphStore((s) => s.setFilter);
  const setSearch = useGraphStore((s) => s.setSearch);
  const nodes = useGraphStore((s) => s.nodes);

  const activeCount = filters.phases.size + filters.statuses.size + (filters.search ? 1 : 0);

  return (
    <div className="absolute top-4 left-16 z-10">
      <button
        onClick={() => setOpen(!open)}
        className="relative px-3 py-1.5 text-sm rounded border border-hive-border bg-hive-surface text-hive-text hover:bg-hive-border transition-colors"
      >
        Filters
        {activeCount > 0 && (
          <span className="absolute -top-1.5 -right-1.5 w-4 h-4 rounded-full bg-hive-blue text-[10px] flex items-center justify-center text-white">
            {activeCount}
          </span>
        )}
      </button>

      {open && (
        <div className="mt-2 w-56 rounded-lg border border-hive-border bg-hive-surface/95 backdrop-blur-sm shadow-xl p-3 space-y-3">
          <div>
            <span className="text-xs text-hive-muted font-semibold uppercase tracking-wide">Phase</span>
            <div className="mt-1 space-y-1">
              {PHASES.map((p) => (
                <label key={p} className="flex items-center gap-2 text-xs text-hive-text cursor-pointer">
                  <input
                    type="checkbox"
                    checked={filters.phases.has(p)}
                    onChange={(e) => setFilter("phases", p, e.target.checked)}
                    className="accent-hive-blue"
                  />
                  {p}
                </label>
              ))}
            </div>
          </div>

          <div>
            <span className="text-xs text-hive-muted font-semibold uppercase tracking-wide">Status</span>
            <div className="mt-1 space-y-1">
              {STATUSES.map((s) => (
                <label key={s} className="flex items-center gap-2 text-xs text-hive-text cursor-pointer">
                  <input
                    type="checkbox"
                    checked={filters.statuses.has(s)}
                    onChange={(e) => setFilter("statuses", s, e.target.checked)}
                    className="accent-hive-blue"
                  />
                  {s}
                </label>
              ))}
            </div>
          </div>

          <div>
            <span className="text-xs text-hive-muted font-semibold uppercase tracking-wide">Search node</span>
            <input
              type="text"
              value={filters.search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Type agent name..."
              className="mt-1 w-full px-2 py-1 text-xs rounded border border-hive-border bg-hive-bg text-hive-text placeholder-hive-muted outline-none focus:border-hive-blue"
            />
          </div>

          <div className="text-[10px] text-hive-muted">
            {nodes.size} node{nodes.size !== 1 ? "s" : ""} visible
          </div>
        </div>
      )}
    </div>
  );
}
