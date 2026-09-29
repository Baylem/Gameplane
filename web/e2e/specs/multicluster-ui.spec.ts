import { test, expect, type Page } from "@playwright/test";

const remoteNode = "gp-demo-remote-control-plane";
const localNode = "gp-demo-central-control-plane";

async function expectInventoryFits(page: Page, nodeName: string) {
  const main = page.getByRole("main");
  const heading = main.getByRole("heading", { name: "Cluster", exact: true });
  const viewport = page.viewportSize()!;
  for (const control of [
    heading,
    main.getByRole("button", { name: /Add node/i }),
    main.getByRole("button", { name: /Download kubeconfig/i }),
  ]) {
    const bounds = await control.boundingBox();
    expect(bounds).not.toBeNull();
    expect(bounds!.width).toBeGreaterThan(0);
    expect(bounds!.x).toBeGreaterThanOrEqual(0);
    expect(bounds!.y).toBeGreaterThanOrEqual(0);
    expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(viewport.width);
    expect(bounds!.y + bounds!.height).toBeLessThanOrEqual(viewport.height);
  }
  expect(await heading.evaluate((element) => element.scrollWidth <= element.clientWidth)).toBe(true);
  expect(await main.evaluate((element) => element.scrollWidth <= element.clientWidth)).toBe(true);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);

  const nodeBounds = await main.getByText(nodeName, { exact: true }).boundingBox();
  const statusBounds = await main.getByText(/^●\s*Ready$/).boundingBox();
  expect(nodeBounds).not.toBeNull();
  expect(statusBounds).not.toBeNull();
  expect(nodeBounds!.x + nodeBounds!.width).toBeLessThanOrEqual(statusBounds!.x);
}

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
      await expect(page.getByText(remoteNode, { exact: true })).toBeVisible();
      await expect(page.getByText(localNode, { exact: true })).toHaveCount(0);
      await expect(page.getByRole("button", { name: /Add node/i })).toBeDisabled();
      await expect(page.getByRole("button", { name: /Download kubeconfig/i })).toBeDisabled();
      await expectInventoryFits(page, remoteNode);
      await page.getByRole("button", { name: "Select cluster", exact: true }).click();
      await page.getByRole("menuitem", { name: "local", exact: true }).click();
      await expect(page.getByText(localNode, { exact: true })).toBeVisible();
      await expect(page.getByText(remoteNode, { exact: true })).toHaveCount(0);
      await expectInventoryFits(page, localNode);
    });
  }
});
