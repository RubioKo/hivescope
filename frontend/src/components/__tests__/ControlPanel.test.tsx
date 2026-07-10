import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ControlPanel } from "../ControlPanel";
import { useGraphStore } from "../../store/graphStore";
import { useThemeStore } from "../../store/themeStore";

describe("ControlPanel", () => {
  beforeEach(() => {
    useGraphStore.getState().clear();
    useThemeStore.setState({ theme: "dark" });
    vi.restoreAllMocks();
  });

  it("renders all buttons", () => {
    render(<ControlPanel />);
    expect(screen.getByText("☀")).toBeDefined(); // dark mode toggle shows sun
    expect(screen.getByText("⏸ Pause")).toBeDefined();
    expect(screen.getByText("▶ Demo")).toBeDefined();
    expect(screen.getByText("✕ Clear")).toBeDefined();
  });

  it("toggles theme on click", () => {
    render(<ControlPanel />);
    fireEvent.click(screen.getByTitle("Switch to light theme"));
    expect(useThemeStore.getState().theme).toBe("light");
  });

  it("shows sun icon in dark mode and moon in light mode", () => {
    const { rerender } = render(<ControlPanel />);
    expect(screen.getByText("☀")).toBeDefined();
    useThemeStore.setState({ theme: "light" });
    rerender(<ControlPanel />);
    expect(screen.getByText("☾")).toBeDefined();
  });

  it("toggles pause on click", () => {
    render(<ControlPanel />);
    fireEvent.click(screen.getByText("⏸ Pause"));
    expect(useGraphStore.getState().paused).toBe(true);
  });

  it("shows Resume when paused", () => {
    useGraphStore.getState().setPaused(true);
    render(<ControlPanel />);
    expect(screen.getByText("▶ Resume")).toBeDefined();
  });
});
