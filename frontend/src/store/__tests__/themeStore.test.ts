import { describe, it, expect, beforeEach } from "vitest";
import { useThemeStore } from "../themeStore";

describe("themeStore", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute("data-theme");
    useThemeStore.setState({ theme: "dark" });
  });

  it("defaults to dark theme", () => {
    expect(useThemeStore.getState().theme).toBe("dark");
  });

  it("reads stored theme from localStorage", () => {
    localStorage.setItem("hivescope-theme", "light");
    useThemeStore.setState({ theme: "light" });
    expect(useThemeStore.getState().theme).toBe("light");
  });

  it("toggle switches from dark to light", () => {
    useThemeStore.getState().toggle();
    expect(useThemeStore.getState().theme).toBe("light");
  });

  it("toggle switches from light to dark", () => {
    useThemeStore.setState({ theme: "light" });
    useThemeStore.getState().toggle();
    expect(useThemeStore.getState().theme).toBe("dark");
  });

  it("toggle persists to localStorage", () => {
    useThemeStore.getState().toggle();
    expect(localStorage.getItem("hivescope-theme")).toBe("light");
  });

  it("toggle updates data-theme attribute", () => {
    useThemeStore.getState().toggle();
    expect(document.documentElement.getAttribute("data-theme")).toBe("light");
    useThemeStore.getState().toggle();
    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
  });
});
