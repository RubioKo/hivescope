import { useState } from "react";
import { useGraphStore } from "../store/graphStore";
import { useThemeStore } from "../store/themeStore";

const API_BASE = `http://${import.meta.env.VITE_WS_HOST || "localhost"}:${import.meta.env.VITE_WS_PORT || 8765}`;

export function ControlPanel() {
  const paused = useGraphStore((s) => s.paused);
  const setPaused = useGraphStore((s) => s.setPaused);
  const clear = useGraphStore((s) => s.clear);
  const { theme, toggle: toggleTheme } = useThemeStore();
  const [demoLoading, setDemoLoading] = useState(false);

  const startDemo = async () => {
    setDemoLoading(true);
    try {
      await fetch(`${API_BASE}/demo/start`, { method: "POST" });
    } finally {
      setDemoLoading(false);
    }
  };

  return (
    <div className="absolute top-4 right-4 z-10 flex gap-2">
      <button
        onClick={toggleTheme}
        className="px-3 py-1.5 text-sm rounded border border-hive-border bg-hive-surface text-hive-text hover:bg-hive-border transition-colors"
        title={`Switch to ${theme === "dark" ? "light" : "dark"} theme`}
      >
        {theme === "dark" ? "☀" : "☾"}
      </button>
      <button
        onClick={() => setPaused(!paused)}
        className="px-3 py-1.5 text-sm rounded border border-hive-border bg-hive-surface text-hive-text hover:bg-hive-border transition-colors"
      >
        {paused ? "▶ Resume" : "⏸ Pause"}
      </button>
      <button
        onClick={startDemo}
        disabled={demoLoading}
        className="px-3 py-1.5 text-sm rounded border border-hive-border bg-hive-surface text-hive-green hover:bg-hive-border transition-colors disabled:opacity-50"
      >
        {demoLoading ? "..." : "▶ Demo"}
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
