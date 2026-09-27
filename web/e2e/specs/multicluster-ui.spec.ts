import { test, expect } from "@playwright/test";

test.describe("multicluster navigation", () => {
  test.skip(process.env.GAMEPLANE_E2E_TARGET === "live", "Deterministic independent node fixtures use the mock API.");

  test.beforeEach(async ({ context }) => {
    await context.addCookies([{ name: "e2e_multicluster", value: "1", url: "http://localhost:5173" }]);
  });

  for (const width of [1440, 375]) {
    test(`shows both clusters and selected nodes at ${width}px`, async ({ page }) => {
      await page.setViewportSize({ width, height: 900 });
      await page.goto("/clusters");
      await expect(page.getByRole("heading", { name: "Clusters", exact: true })).toBeVisible();
      await expect(page.getByRole("button", { name: "Select cluster", exact: true })).toBeVisible();
      await expect(page.getByRole("heading", { name: "Remote demo", exact: true })).toBeVisible();
      await page.getByRole("button", { name: "View nodes in Remote demo", exact: true }).click();
      await expect(page).toHaveURL(/\/cluster$/);
      await expect(page.getByText("remote-node", { exact: true })).toBeVisible();
      await expect(page.getByText("local-node", { exact: true })).toHaveCount(0);
      await expect(page.getByRole("button", { name: /Add node/i })).toBeDisabled();
      await expect(page.getByRole("button", { name: /Download kubeconfig/i })).toBeDisabled();
      await page.getByRole("button", { name: "Select cluster", exact: true }).click();
      await page.getByRole("menuitem", { name: "local", exact: true }).click();
      await expect(page.getByText("local-node", { exact: true })).toBeVisible();
      await expect(page.getByText("remote-node", { exact: true })).toHaveCount(0);
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth);
      expect(overflow).toBe(false);
    });
  }
});
