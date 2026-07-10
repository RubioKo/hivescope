import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { ErrorBoundary } from "../ErrorBoundary";

describe("ErrorBoundary", () => {
  beforeEach(() => {
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  it("renders children when no error", () => {
    render(
      <ErrorBoundary>
        <p>Hello</p>
      </ErrorBoundary>,
    );
    expect(screen.getByText("Hello")).toBeDefined();
  });

  it("renders fallback when child throws", () => {
    const Broken = () => {
      throw new Error("Boom!");
    };
    render(
      <ErrorBoundary>
        <Broken />
      </ErrorBoundary>,
    );
    expect(screen.getByText("Something went wrong")).toBeDefined();
    expect(screen.getByText("Boom!")).toBeDefined();
  });

  it("renders custom fallback when provided", () => {
    const Broken = () => {
      throw new Error("Boom!");
    };
    render(
      <ErrorBoundary fallback={<p>Custom error</p>}>
        <Broken />
      </ErrorBoundary>,
    );
    expect(screen.getByText("Custom error")).toBeDefined();
  });
});
