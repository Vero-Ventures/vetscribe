import { render, screen, waitFor } from "@testing-library/preact";
import { beforeEach, afterEach, expect, test, vi } from "vitest";
import { App } from "../src/app";

beforeEach(() => {
  vi.stubGlobal(
    "fetch",
    vi.fn(() =>
      Promise.resolve({
        ok: true,
        json: () =>
          Promise.resolve({ status: "ok", build: { version: "test", commit: "c", date: "d" } }),
      }),
    ),
  );
});

afterEach(() => {
  vi.unstubAllGlobals();
});

test("renders the app title and tagline", () => {
  render(<App />);
  expect(screen.getByText("VetScribe")).toBeTruthy();
  expect(screen.getByTestId("tagline")).toBeTruthy();
});

test("shows backend health once loaded", async () => {
  render(<App />);
  await waitFor(() => expect(screen.getByTestId("health-status").textContent).toContain("ok"));
});
