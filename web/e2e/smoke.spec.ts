import { test, expect } from "@playwright/test";

test("app shell loads and reports backend health", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "VetScribe" })).toBeVisible();
  await expect(page.getByTestId("health-status")).toContainText("ok");
});
