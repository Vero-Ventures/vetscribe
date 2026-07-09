import { defineConfig } from "vite";
import preact from "@preact/preset-vite";

// The frontend builds into the Go embed directory so the single binary serves it.
export default defineConfig({
  plugins: [preact()],
  build: {
    outDir: "../internal/api/dist",
    emptyOutDir: true,
  },
  server: {
    proxy: {
      "/api": "http://localhost:8080",
      "/healthz": "http://localhost:8080",
    },
  },
  test: {
    environment: "jsdom",
    globals: true,
    include: ["tests/**/*.test.ts", "tests/**/*.test.tsx"],
    coverage: {
      provider: "v8",
      include: ["src/**"],
      // main.tsx is the DOM bootstrap; it is exercised by the Playwright e2e, not units.
      exclude: ["src/main.tsx"],
      thresholds: { lines: 80, functions: 80, branches: 70, statements: 80 },
    },
  },
});
