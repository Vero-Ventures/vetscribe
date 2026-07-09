import { defineConfig, devices } from "@playwright/test";

// E2E runs against the built single binary (see CI). BASE_URL points at a running
// vetscribe instance. Fake-audio flags are added per-project in later milestones so
// the record→transcribe→SOAP path can be driven deterministically.
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? "github" : "list",
  use: {
    baseURL: process.env.VETSCRIBE_BASE_URL ?? "http://localhost:8080",
    trace: "on-first-retry",
  },
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "firefox", use: { ...devices["Desktop Firefox"] } },
  ],
});
