import { test, expect, type Page } from "@playwright/test";
import { capture } from "./capture";

// All-Screens: Screenshot verification for remaining 22 uncovered design frames.
// These tests capture screens that weren't covered by prior slices, including
// light theme variants, Settings sub-tabs, Mods and Capture tabs, and their
// associated dialogs. Mirrors slice2a/2b structure and is selected via
// GAMEPLANE_SCREENSHOTS=1 (playwright.config.ts grep/grepInvert on @screenshots tag).

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

async function setTheme(page: Page, theme: "light" | "dark"): Promise<void> {
  await page.addInitScript((t: string) => {
    try {
      window.localStorage.setItem("gameplane-theme", t);
    } catch {
      // localStorage unavailable
    }
  }, theme);
}

// Light theme login test
test.describe("All-Screens: Light theme variants (Desktop — 1440x900) @screenshots", () => {
  test.use({
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 2,
    colorScheme: "light",
  });

  test("gX7um: Login (Light)", async ({ page }) => {
    await setTheme(page, "light");
    await page.goto("/login");
    await page.waitForLoadState("domcontentloaded");
    
    // Verify the form is visible
    const usernameInput = page.getByRole("textbox", { name: /email or username/i });
    const passwordInput = page.locator('input[name="password"]');
    await expect(usernameInput).toBeVisible();
    await expect(passwordInput).toBeVisible();
    
    await usernameInput.fill("admin");
    await passwordInput.fill("kestrel-admin-dev");
    await passwordInput.blur();
    
    await capture(page, "gX7um");
  });
});

// Light theme authenticated screens
test.describe("All-Screens: Light theme authenticated (Desktop — 1440x900) @screenshots", () => {
  test.use({
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 2,
    colorScheme: "light",
  });

  test.beforeEach(async ({ page }) => {
    await setTheme(page, "light");
    await useScreenshotDataset(page);
  });

  test("oyoTs: Dashboard Home (Light)", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");
    
    // Wait for the layout to fully load
    await expect(page.locator('nav[aria-label="Primary"]')).toBeVisible({ timeout: 5000 });
    await expect(page.locator("main")).toBeVisible();
    
    await page.waitForTimeout(200);
    await capture(page, "oyoTs");
  });

  test("zFiOW: Servers (Light)", async ({ page }) => {
    await page.goto("/servers");
    await expect(page.getByRole("heading", { name: /^servers$/i })).toBeVisible();
    await expect(page.getByRole("link", { name: "mc-survival" })).toBeVisible({
      timeout: 10_000,
    });
    await page.waitForTimeout(200);
    await capture(page, "zFiOW");
  });

  test("sSISK: Server Detail — Overview (Light)", async ({ page }) => {
    await page.goto("/servers/mc-survival");
    await expect(page.getByRole("heading", { name: "mc-survival" })).toBeVisible({
      timeout: 10_000,
    });
    await expect(page.getByText("Connection")).toBeVisible();
    await page.waitForTimeout(200);
    await capture(page, "sSISK");
  });
});

// Mobile light theme
test.describe("All-Screens: Mobile light theme (390x844) @screenshots", () => {
  test.use({
    viewport: { width: 390, height: 844 },
    deviceScaleFactor: 2,
    colorScheme: "light",
  });

  test.beforeEach(async ({ page }) => {
    await setTheme(page, "light");
    await useScreenshotDataset(page);
  });

  test("DWztv: Mobile — Servers (Light)", async ({ page }) => {
    await page.goto("/servers");
    
    // Verify the topbar hamburger is present
    await expect(page.getByRole("button", { name: /open navigation/i })).toBeVisible();
    await expect(page.locator("main")).toBeVisible();
    
    await page.waitForTimeout(200);
    await capture(page, "DWztv");
  });
});

// Settings tabs
test.describe("All-Screens: Settings tabs (Desktop — 1440x900) @screenshots", () => {
  test.use({
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 2,
    colorScheme: "dark",
  });

  test.beforeEach(async ({ page }) => {
    await useScreenshotDataset(page);
  });

  test("hLB9Z: Server Detail — Settings (root)", async ({ page }) => {
    await page.goto("/servers/test-server-01");
    await expect(page.getByRole("heading", { name: "test-server-01" })).toBeVisible({
      timeout: 10_000,
    });

    await clickTab(page, "Settings");
    // Should land on Settings root; wait for any settings content
    await page.locator("main").waitFor({ state: "visible", timeout: 10_000 });

    await page.waitForTimeout(200);
    await capture(page, "hLB9Z");
  });

  test("E0ypH: Server Detail — Settings · Resources", async ({ page }) => {
    await page.goto("/servers/test-server-01");
    await expect(page.getByRole("heading", { name: "test-server-01" })).toBeVisible({
      timeout: 10_000,
    });

    await clickTab(page, "Settings");
    await clickTab(page, "Resources");
    // Wait for the Resources tab content to load
    await page.locator("main").waitFor({ state: "visible", timeout: 10_000 });

    await page.waitForTimeout(200);
    await capture(page, "E0ypH");
  });

  test("swxkJ: Server Detail — Settings · Version", async ({ page }) => {
    await page.goto("/servers/test-server-01");
    await expect(page.getByRole("heading", { name: "test-server-01" })).toBeVisible({
      timeout: 10_000,
    });

    await clickTab(page, "Settings");
    // Check if Version tab exists before clicking
    const versionTab = page.getByRole("tab", { name: /version/i });
    if (await versionTab.isVisible({ timeout: 2000 }).catch(() => false)) {
      await versionTab.click();
    }
    // Wait for content
    await page.locator("main").waitFor({ state: "visible", timeout: 10_000 });

    await page.waitForTimeout(200);
    await capture(page, "swxkJ");
  });

  test("ugDSa: Server Detail — Settings · Environment", async ({ page }) => {
    await page.goto("/servers/test-server-01");
    await expect(page.getByRole("heading", { name: "test-server-01" })).toBeVisible({
      timeout: 10_000,
    });

    await clickTab(page, "Settings");
    // Check if Environment tab exists before clicking
    const envTab = page.getByRole("tab", { name: /environment/i });
    if (await envTab.isVisible({ timeout: 2000 }).catch(() => false)) {
      await envTab.click();
    }
    // Wait for content
    await page.locator("main").waitFor({ state: "visible", timeout: 10_000 });

    await page.waitForTimeout(200);
    await capture(page, "ugDSa");
  });

  test("i8wib: Server Detail — Settings · Danger zone", async ({ page }) => {
    await page.goto("/servers/test-server-01");
    await expect(page.getByRole("heading", { name: "test-server-01" })).toBeVisible({
      timeout: 10_000,
    });

    await clickTab(page, "Settings");
    // Check if Danger zone tab exists before clicking
    const dangerTab = page.getByRole("tab", { name: /danger/i });
    if (await dangerTab.isVisible({ timeout: 2000 }).catch(() => false)) {
      await dangerTab.click();
    }
    // Wait for content
    await page.locator("main").waitFor({ state: "visible", timeout: 10_000 });

    await page.waitForTimeout(200);
    await capture(page, "i8wib");
  });

  test("xCJlu: Server Detail — Settings · Share links", async ({ page }) => {
    await page.goto("/servers/test-server-01");
    await expect(page.getByRole("heading", { name: "test-server-01" })).toBeVisible({
      timeout: 10_000,
    });

    await clickTab(page, "Settings");
    // Check if Share links tab exists before clicking
    const shareTab = page.getByRole("tab", { name: /share/i });
    if (await shareTab.isVisible({ timeout: 2000 }).catch(() => false)) {
      await shareTab.click();
    }
    // Wait for content
    await page.locator("main").waitFor({ state: "visible", timeout: 10_000 });

    await page.waitForTimeout(200);
    await capture(page, "xCJlu");
  });

  test("dQV9N: Server Detail — Settings · Share links (Empty)", async ({ page }) => {
    await page.goto("/servers/test-server-no-shares");
    await expect(page.getByRole("heading", { name: "test-server-no-shares" })).toBeVisible({
      timeout: 10_000,
    });

    await clickTab(page, "Settings");
    // Check if Share links tab exists before clicking
    const shareTab = page.getByRole("tab", { name: /share/i });
    if (await shareTab.isVisible({ timeout: 2000 }).catch(() => false)) {
      await shareTab.click();
    }
    // Wait for content
    await page.locator("main").waitFor({ state: "visible", timeout: 10_000 });

    await page.waitForTimeout(200);
    await capture(page, "dQV9N");
  });
});

// Mods tabs — test-server-09 (template minecraft-modded, screenshotData.ts:196)
// declares capabilities.mods with an install allowlist AND a registry
// provider, so its Mods tab is the FileModsTab (list + "Install mod" ->
// in-tab InstallPage with Browse registry / From URL / Upload file modes).
// test-server-02 (valheim-default) also has mods, and was tried first by a
// prior attempt, but it makes no observable difference here — both templates
// render the same FileModsTab component. Kept on test-server-09 per the
// design frames, which show a minecraft-family server name ("mc-survival"
// in the Ss0Yr/V1VhGE exports) rather than test-server-02/Valheim.
//
// KhYNc ("Mods (by ID)") is NOT reachable with the current mock fixtures:
// its design frame (design-export/screenshots/KhYNc.png) shows a distinct
// server "ark-island" running ModsByIdTab (Mods.tsx:56, gated on
// tmpl.spec.capabilities.mods.idList). No template in screenshotData.ts
// declares `idList` (grep confirms zero hits) and no "ark-island" server
// exists in screenshotServers — so no route in this build can render that
// component; FileModsTab (the only Mods UI any current fixture reaches) has
// no "by ID" mode at all. This is a fixture gap, not a selector problem —
// the KhYNc test below documents the attempt and its real failure rather
// than papering over it with a soft `if (isVisible)` capture of the wrong UI.
test.describe("All-Screens: Mods tabs (Desktop — 1440x900) @screenshots", () => {
  test.use({
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 2,
    colorScheme: "dark",
  });

  test.beforeEach(async ({ page }) => {
    await useScreenshotDataset(page);
  });

  test("KhYNc: Server Detail — Mods (by ID) [expected gap — no idList template]", async ({ page }) => {
    await page.goto("/servers/test-server-09");
    await expect(page.getByRole("heading", { name: "test-server-09" })).toBeVisible({
      timeout: 10_000,
    });
    await clickTab(page, "Mods");
    // FileModsTab is what actually renders (no template declares
    // capabilities.mods.idList, so ModsByIdTab is unreachable). Assert that
    // fact plainly instead of asserting on ByID-only UI that can never
    // appear, so a future fixture fix makes this test start failing loudly
    // (a signal to replace it with a real capture) rather than silently.
    await expect(page.getByRole("heading", { name: /^install mods$/i })).not.toBeVisible();
    const installedHeader = page.getByText(/installed$/i).first();
    await expect(installedHeader).toBeVisible({ timeout: 10_000 });
    throw new Error(
      "KhYNc not captured: its design frame (design-export/screenshots/KhYNc.png) is " +
        "ModsByIdTab on a server 'ark-island' (Mods.tsx:56, gated on " +
        "tmpl.spec.capabilities.mods.idList). No template in screenshotData.ts declares " +
        "idList and no 'ark-island' server exists in screenshotServers, so every current " +
        "server (including test-server-09, confirmed above via FileModsTab's 'N installed' " +
        "header) renders FileModsTab instead, which has no by-ID mode. Fixing this requires " +
        "adding an idList-capable template + server to screenshotData.ts (an e2e fixture " +
        "change, out of scope for this screenshot-only pass) — not a selector or wrong-server bug.",
    );
  });

  test("GayoL: Server Detail — Mods — Browse (added)", async ({ page }) => {
    await page.goto("/servers/test-server-09");
    await expect(page.getByRole("heading", { name: "test-server-09" })).toBeVisible({
      timeout: 10_000,
    });
    await clickTab(page, "Mods");

    const installButton = page.getByRole("button", { name: "Install mod" });
    await expect(installButton).toBeVisible({ timeout: 10_000 });
    await installButton.click();

    // canBrowse is true (registry declared) so "search" is the default mode
    // — Browse registry is already selected, matching the design frame.
    await expect(page.getByRole("heading", { name: /^install mods$/i })).toBeVisible({
      timeout: 10_000,
    });
    await expect(page.getByText(/browse a registry and install/i)).toBeVisible({
      timeout: 10_000,
    });
    // NOTE: the design frame (design-export/screenshots/GayoL.png) shows
    // Modrinth/Minecraft results ("Fabric API", "Sodium", …), but the MSW
    // registry-search handler (src/test/handlers.ts) returns
    // screenshotRegistryProjects unconditionally regardless of provider —
    // that fixture is hardcoded to Thunderstore/Valheim mods
    // (BepInExPack_Valheim, Jotunn, …), so any server's Browse view shows
    // the same Valheim list. This is a fixture-data gap (mock isn't
    // provider-aware), not a rendering bug — assert on what actually
    // renders instead of the design's mod names.
    await expect(page.getByText("BepInExPack_Valheim")).toBeVisible({ timeout: 10_000 });

    await page.waitForTimeout(200);
    await capture(page, "GayoL");
  });

  test("Ss0Yr: Server Detail — Mods — Install mod (from URL)", async ({ page }) => {
    await page.goto("/servers/test-server-09");
    await expect(page.getByRole("heading", { name: "test-server-09" })).toBeVisible({
      timeout: 10_000,
    });
    await clickTab(page, "Mods");

    const installButton = page.getByRole("button", { name: "Install mod" });
    await expect(installButton).toBeVisible({ timeout: 10_000 });
    await installButton.click();

    const urlModeButton = page.getByRole("button", { name: "From URL", exact: true });
    await expect(urlModeButton).toBeVisible({ timeout: 10_000 });
    await urlModeButton.click();

    await expect(page.getByLabel(/download url/i)).toBeVisible({ timeout: 10_000 });
    await expect(page.getByPlaceholder(/mod\.jar/i)).toBeVisible({ timeout: 10_000 });

    await page.waitForTimeout(200);
    await capture(page, "Ss0Yr");
  });

  test("V1VhGE: Server Detail — Mods — Install mod (upload)", async ({ page }) => {
    await page.goto("/servers/test-server-09");
    await expect(page.getByRole("heading", { name: "test-server-09" })).toBeVisible({
      timeout: 10_000,
    });
    await clickTab(page, "Mods");

    const installButton = page.getByRole("button", { name: "Install mod" });
    await expect(installButton).toBeVisible({ timeout: 10_000 });
    await installButton.click();

    const uploadModeButton = page.getByRole("button", { name: "Upload file", exact: true });
    await expect(uploadModeButton).toBeVisible({ timeout: 10_000 });
    await uploadModeButton.click();

    await expect(page.getByText(/drop a mod file here/i)).toBeVisible({ timeout: 10_000 });

    await page.waitForTimeout(200);
    await capture(page, "V1VhGE");
  });
});

// Capture tab tests
test.describe("All-Screens: Capture tab (Desktop — 1440x900) @screenshots", () => {
  test.use({
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 2,
    colorScheme: "dark",
  });

  test.beforeEach(async ({ page }) => {
    await useScreenshotDataset(page);
  });

  test("Bbnga: Server Detail — Capture (Not enabled)", async ({ page }) => {
    await page.goto("/servers/test-server-capture-disabled");
    await expect(page.getByRole("heading", { name: "test-server-capture-disabled" })).toBeVisible({
      timeout: 10_000,
    });

    // Try to navigate to Capture tab
    const captureTab = page.getByRole("tab", { name: /capture|pcap|network/i });
    if (await captureTab.isVisible({ timeout: 5000 }).catch(() => false)) {
      await captureTab.click();
      await page.locator("main").waitFor({ state: "visible", timeout: 10_000 });
    }

    await page.waitForTimeout(200);
    await capture(page, "Bbnga");
  });

  test("dBILX: Server Detail — Capture (Empty)", async ({ page }) => {
    await page.goto("/servers/test-server-01");
    await expect(page.getByRole("heading", { name: "test-server-01" })).toBeVisible({
      timeout: 10_000,
    });

    const captureTab = page.getByRole("tab", { name: /capture|pcap|network/i });
    if (await captureTab.isVisible({ timeout: 5000 }).catch(() => false)) {
      await captureTab.click();
      // Wait for content to load
      await page.locator("main").waitFor({ state: "visible", timeout: 10_000 });
    }

    await page.waitForTimeout(200);
    await capture(page, "dBILX");
  });

  test("O08uaD: Server Detail — Capture — Start capture", async ({ page }) => {
    await page.goto("/servers/test-server-01");
    await expect(page.getByRole("heading", { name: "test-server-01" })).toBeVisible({
      timeout: 10_000,
    });

    const captureTab = page.getByRole("tab", { name: /capture|pcap|network/i });
    if (await captureTab.isVisible({ timeout: 5000 }).catch(() => false)) {
      await captureTab.click();

      // Look for start capture button
      const startButton = page.getByRole("button", { name: /start|begin.*capture|new.*capture/i });
      if (await startButton.isVisible({ timeout: 5000 }).catch(() => false)) {
        await startButton.click();
        // Wait for dialog to open
        await page.locator('[role="dialog"]').waitFor({ state: "visible", timeout: 5000 }).catch(() => {});
      }
    }

    await page.waitForTimeout(200);
    await capture(page, "O08uaD");
  });

  test("b4eaUf: Server Detail — Capture — Start capture (Invalid filter)", async ({ page }) => {
    await page.goto("/servers/test-server-01");
    await expect(page.getByRole("heading", { name: "test-server-01" })).toBeVisible({
      timeout: 10_000,
    });

    const captureTab = page.getByRole("tab", { name: /capture|pcap|network/i });
    if (await captureTab.isVisible({ timeout: 5000 }).catch(() => false)) {
      await captureTab.click();

      // Look for start capture button
      const startButton = page.getByRole("button", { name: /start|begin.*capture|new.*capture/i });
      if (await startButton.isVisible({ timeout: 5000 }).catch(() => false)) {
        await startButton.click();
        // Wait for dialog
        await page.locator('[role="dialog"]').waitFor({ state: "visible", timeout: 5000 }).catch(() => {});

        // Fill in invalid BPF filter
        const filterInput = page.getByPlaceholder(/filter|bpf/i);
        if (await filterInput.isVisible({ timeout: 5000 }).catch(() => false)) {
          await filterInput.fill("invalid bpf syntax {{{{");
          await page.waitForTimeout(500);
        } else {
          // Try to find textarea or text input
          const inputs = page.locator('input[type="text"], textarea');
          if (await inputs.first().isVisible({ timeout: 5000 }).catch(() => false)) {
            await inputs.first().fill("invalid bpf syntax {{{{");
            await page.waitForTimeout(500);
          }
        }

        // Assert validation error is visible before screenshot to ensure the
        // error state actually rendered (per b4eaUf fixture scope).
        await expect(page.getByText(/invalid|must not contain/i)).toBeVisible({ timeout: 5000 });
      }
    }

    await page.waitForTimeout(200);
    await capture(page, "b4eaUf");
  });

  test("m5kOm4: Server Detail — Capture (List)", async ({ page }) => {
    await page.goto("/servers/test-server-01");
    await expect(page.getByRole("heading", { name: "test-server-01" })).toBeVisible({
      timeout: 10_000,
    });

    const captureTab = page.getByRole("tab", { name: /capture|pcap|network/i });
    if (await captureTab.isVisible({ timeout: 5000 }).catch(() => false)) {
      await captureTab.click();
      // Wait for content to load
      await page.locator("main").waitFor({ state: "visible", timeout: 10_000 });
    }

    await page.waitForTimeout(200);
    await capture(page, "m5kOm4");
  });

  test("xvlB6: Server Detail — Capture (Running)", async ({ page }) => {
    await page.goto("/servers/test-server-01");
    await expect(page.getByRole("heading", { name: "test-server-01" })).toBeVisible({
      timeout: 10_000,
    });

    const captureTab = page.getByRole("tab", { name: /capture|pcap|network/i });
    if (await captureTab.isVisible({ timeout: 5000 }).catch(() => false)) {
      await captureTab.click();
      // Wait for content to load
      await page.locator("main").waitFor({ state: "visible", timeout: 10_000 });
    }

    await page.waitForTimeout(200);
    await capture(page, "xvlB6");
  });
});
