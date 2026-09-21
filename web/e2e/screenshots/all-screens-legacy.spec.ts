import { test, expect, type Page } from "@playwright/test";
import { capture } from "./capture";

// Legacy preset parity (feature 016, T043 / SC-006): the core workflow
// screens — server management, log viewing, module browsing, settings — also
// render under data-theme-preset="legacy" with full functional parity and
// legibility versus the Pink baseline (all-screens.spec.ts). Parity is
// asserted functionally, not pixel-compared: the same interactive elements
// must be present, and the computed legacy token values (orange accent
// #F97316 / dark ground #0F0F0F) must actually reach the CSS cascade on
// every screen. Screenshots are captured alongside the pink baseline.
//
// The preset is seeded through the boot contract (no settings UI exists
// yet): an init script writes the gameplane-theme-prefs localStorage cache
// before the first paint, and the inline theme-boot script in index.html
// applies it. The screenshot mock dataset's /users/me carries no preferences
// field, so the backend-wins reconciliation in useThemePreferences leaves
// the seeded cache untouched. Tagged @screenshots like the other specs in
// this directory: selected only when GAMEPLANE_SCREENSHOTS=1
// (playwright.config.ts grep/grepInvert).

// Legacy dark preset tokens (globals.css [data-theme-preset="legacy"] dark
// block, contracts/theme-tokens-v2.md §2). Custom properties are returned by
// getComputedStyle as their declared token stream, so compare normalized.
const LEGACY_DARK_ACCENT = "oklch(69.11% 0.1944 44.01)"; // #F97316
const LEGACY_DARK_BACKGROUND = "oklch(14.5% 0 0)"; // #0F0F0F

// Seeds the Legacy preset through the boot contract before any page load.
async function useLegacyPreset(page: Page): Promise<void> {
  await page.addInitScript(() => {
    try {
      window.localStorage.setItem(
        "gameplane-theme-prefs",
        JSON.stringify({
          themeType: "preset",
          presetId: "legacy",
          appearanceMode: "dark",
          customColors: null,
          customCssEnabled: false,
          customCss: null,
        }),
      );
    } catch {
      // localStorage unavailable (sandboxed) — the page renders the pink
      // default and the preset assertions below fail loudly.
    }
  });
}

// Selects the enriched screenshot dataset before the app's first fetch.
async function useScreenshotDataset(page: Page): Promise<void> {
  await page.addInitScript(() => {
    try {
      window.localStorage.setItem("gameplane-e2e-dataset", "screenshots");
    } catch {
      // localStorage unavailable (sandboxed) — falls back to default handlers.
    }
  });
}

async function clickTab(page: Page, name: string): Promise<void> {
  const tab = page.getByRole("tab", { name: new RegExp(`^${name}$`, "i") });
  await expect(tab).toBeVisible({ timeout: 10_000 });
  await tab.click();
}

function normalizeToken(value: string): string {
  return value.replace(/\s+/g, " ").trim().toLowerCase();
}

async function rootVar(page: Page, name: string): Promise<string> {
  const raw = await page.evaluate((n) => {
    return getComputedStyle(document.documentElement).getPropertyValue(n);
  }, name);
  return normalizeToken(raw);
}

// Functional parity + legibility gate: the legacy preset must be the active
// base theme with no overlay, and its dark accent must reach the cascade.
async function expectLegacyThemeApplied(page: Page): Promise<void> {
  const root = page.locator("html");
  await expect(root).toHaveAttribute("data-theme-preset", "legacy");
  await expect(root).toHaveAttribute("data-theme-type", "preset");
  await expect(root).toHaveAttribute("data-custom-css", "off");
  await expect(root).toHaveAttribute("data-theme", "dark");
  expect(await rootVar(page, "--accent")).toBe(LEGACY_DARK_ACCENT);
}

test.describe("Legacy preset parity: core workflow (Desktop — 1440x900) @screenshots", () => {
  test.use({
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 2,
    colorScheme: "dark",
  });

  test.beforeEach(async ({ page }) => {
    await useLegacyPreset(page);
    await useScreenshotDataset(page);
  });

  test("legacy-dashboard: Dashboard renders with legacy tokens", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");

    await expect(page.locator('nav[aria-label="Primary"]')).toBeVisible({ timeout: 10_000 });
    await expect(page.locator("main")).toBeVisible();

    await expectLegacyThemeApplied(page);
    // Legibility: dark neutral ground and near-white foreground are the
    // historical legacy values, not the pink preset's.
    expect(await rootVar(page, "--background")).toBe(LEGACY_DARK_BACKGROUND);
    expect(await rootVar(page, "--foreground")).toBe("oklch(96.5% 0 0)");

    await page.waitForTimeout(200);
    await capture(page, "legacy-dashboard");
  });

  test("legacy-servers: Server management list renders with legacy tokens", async ({ page }) => {
    await page.goto("/servers");
    await expect(page.getByRole("heading", { name: /^servers$/i })).toBeVisible();
    await expect(page.getByRole("link", { name: "mc-survival" })).toBeVisible({
      timeout: 10_000,
    });

    await expectLegacyThemeApplied(page);

    await page.waitForTimeout(200);
    await capture(page, "legacy-servers");
  });

  test("legacy-server-detail: Server detail (Overview) renders with legacy tokens", async ({
    page,
  }) => {
    await page.goto("/servers/mc-survival");
    await expect(page.getByRole("heading", { name: "mc-survival" })).toBeVisible({
      timeout: 10_000,
    });
    await expect(page.getByText("Connection")).toBeVisible();

    await expectLegacyThemeApplied(page);

    await page.waitForTimeout(200);
    await capture(page, "legacy-server-detail");
  });

  test("legacy-logs: Log viewing renders with legacy tokens", async ({ page }) => {
    await page.goto("/servers/mc-survival");
    await expect(page.getByRole("heading", { name: "mc-survival" })).toBeVisible({
      timeout: 10_000,
    });

    await clickTab(page, "Logs");
    // Same functional assertion as the pink baseline (kPmoo): the streamed
    // log content is present, so the viewer is alive under the legacy preset.
    await expect(page.getByText(/Starting minecraft server version/i).first()).toBeVisible({
      timeout: 15_000,
    });

    await expectLegacyThemeApplied(page);

    await page.waitForTimeout(200);
    await capture(page, "legacy-logs");
  });

  test("legacy-mods: Module browsing (Mods tab) renders with legacy tokens", async ({ page }) => {
    await page.goto("/servers/test-server-09");
    await expect(page.getByRole("heading", { name: "test-server-09" })).toBeVisible({
      timeout: 10_000,
    });

    await clickTab(page, "Mods");
    // Same functional assertion as the pink baseline (GayoL): the install
    // entry point is present.
    await expect(page.getByRole("button", { name: "Install mod" })).toBeVisible({
      timeout: 10_000,
    });

    await expectLegacyThemeApplied(page);

    await page.waitForTimeout(200);
    await capture(page, "legacy-mods");
  });

  test("legacy-modules: Modules catalog renders with legacy tokens", async ({ page }) => {
    await page.goto("/modules");
    await expect(page.getByRole("heading", { name: /^modules$/i })).toBeVisible({
      timeout: 10_000,
    });
    // Same functional assertion as the pink baseline (kK8Ji).
    const grid = page.locator('[data-testid="modules-grid"]');
    await expect(grid.getByText("Minecraft (Vanilla)", { exact: true })).toBeVisible({
      timeout: 10_000,
    });
    await expect(grid.getByText("Valheim", { exact: true })).toBeVisible();

    await expectLegacyThemeApplied(page);

    await page.waitForTimeout(200);
    await capture(page, "legacy-modules");
  });

  test("legacy-settings: Server settings render with legacy tokens", async ({ page }) => {
    await page.goto("/servers/mc-survival");
    await expect(page.getByRole("heading", { name: "mc-survival" })).toBeVisible({
      timeout: 10_000,
    });

    await clickTab(page, "Settings");
    await page.locator("main").waitFor({ state: "visible", timeout: 10_000 });

    await expectLegacyThemeApplied(page);

    await page.waitForTimeout(200);
    await capture(page, "legacy-settings");
  });
});
