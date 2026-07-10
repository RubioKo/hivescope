import { create } from "zustand";

type Theme = "dark" | "light";

interface ThemeState {
  theme: Theme;
  toggle: () => void;
}

export const useThemeStore = create<ThemeState>((set) => {
  const stored = localStorage.getItem("hivescope-theme") as Theme | null;
  const initial = stored || "dark";
  document.documentElement.setAttribute("data-theme", initial);

  return {
    theme: initial,
    toggle: () => {
      set((state) => {
        const next = state.theme === "dark" ? "light" : "dark";
        localStorage.setItem("hivescope-theme", next);
        document.documentElement.setAttribute("data-theme", next);
        return { theme: next };
      });
    },
  };
});
