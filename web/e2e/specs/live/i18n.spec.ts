import { test, expect } from "@playwright/test";
import type { APIRequestContext, Page } from "@playwright/test";
import { apiSend, loginIfNeeded, seedTemplate } from "./_seed";

// Live: spec 019 T037, language switching end to end on the real backend.
// Covers:
//   A. Switching language keeps the create-server wizard's typed input and
//      does not reload the page.
//   B. A reload and a new browser context keep French, with no English flash.
//   C. The logged-out login and share pages offer a picker that is remembered
//      per browser, and a fresh logged-out context starts in English.
//   D. A 0%-complete fixture language shows English text everywhere (SC-008).
//      It runs only when the live web server was started with
//      VITE_I18N_FIXTURES=1; otherwise it is reported as skipped.
//
// The specs share one admin account, and the language choice is saved to that
// account. The suite runs serially and resets the account language to null
// when it finishes, so later live specs see English again.

const ADMIN_STATE = "./e2e/.auth/storage.json";
// Empty storage state for logged-out contexts (no cookies, no origins).
const EMPTY_STATE: { cookies: []; origins: [] } = { cookies: [], origins: [] };

// Window extensions used as markers across a reload or a language switch.
type I18nWindow = Window & { __i18nMark?: number; __englishSeen?: boolean };

// resetAccountLanguage clears the admin's saved language, so the shared account
// falls back to the browser language and then the instance default.
async function resetAccountLanguage(request: APIRequestContext): Promise<void> {
  const res = await apiSend(request, "PUT", "/users/me/language", { language: null });
  if (!res.ok()) {
    throw new Error(`reset account language: ${res.status()} ${await res.text().catch(() => "")}`);
  }
}

// pickFrench opens the compact picker on a logged-out page and chooses
// Français. The trigger is named "Language" in English and "Langue" in French.
async function pickFrenchFromCompactPicker(page: Page): Promise<void> {
  await page.getByRole("button", { name: /^(Langue|Language)$/ }).click();
  await page.getByRole("menu", { name: "Options de langue" }).getByText("Français").click();
}

test.describe("live: i18n", () => {
  test.skip(
    process.env.GAMEPLANE_E2E_TARGET !== "live",
    "live-only — language switching is covered by the mock specs",
  );

  // Every test changes the admin's language, so the order matters.
  test.describe.configure({ mode: "serial" });

  test.afterAll(async ({ playwright }) => {
    const ctx = await playwright.request.newContext({
      baseURL: "http://localhost:5173",
      storageState: ADMIN_STATE,
    });
    try {
      await resetAccountLanguage(ctx);
    } finally {
      await ctx.dispose();
    }
  });

  test("switching language keeps the wizard input and does not reload", async ({ page, request }) => {
    const stamp = Date.now();
    const templateName = `e2e-pw-i18n-tpl-${stamp}`;
    const serverName = `e2e-pw-i18n-${stamp}`;
    const template = await seedTemplate(request, templateName);
    try {
      // The template is preselected by the query string. The Configure step,
      // which holds the server name, is the next step.
      await page.goto(`/servers/new?template=${templateName}`);
      await page.waitForLoadState("domcontentloaded");
      await loginIfNeeded(page);

      await page.getByRole("button", { name: /^Continue to/ }).click();
      const nameInput = page.getByPlaceholder("mc-hardcore");
      await expect(nameInput).toBeVisible({ timeout: 15_000 });
      await nameInput.fill(serverName);
      await page.evaluate(() => {
        (window as I18nWindow).__i18nMark = 1;
      });

      // Switch the account language from a second page of the same browser.
      const second = await page.context().newPage();
      try {
        await second.goto("/settings/theme");
        await second.waitForLoadState("domcontentloaded");
        await second.getByRole("radio", { name: /Français/ }).click();

        const secondHtml = second.locator("html");
        await expect(secondHtml).toHaveAttribute("lang", "fr", { timeout: 15_000 });
        await expect(secondHtml).toHaveAttribute("dir", "ltr");
        await expect(secondHtml).toHaveAttribute("data-i18n-loaded", "fr");
      } finally {
        await second.close();
      }

      // Back on the wizard page: the typed input and the marker survive, and the
      // French label replaces the English one without a reload.
      await page.bringToFront();
      await expect(page.locator("html")).toHaveAttribute("lang", "fr", { timeout: 15_000 });
      await expect(page.getByText("Nom du serveur")).toBeVisible();
      await expect(nameInput).toHaveValue(serverName);
      const mark = await page.evaluate(() => (window as I18nWindow).__i18nMark);
      expect(mark).toBe(1);
    } finally {
      await template.cleanup(request);
    }
  });

  test("reload and a new browser context keep French without an English flash", async ({ page, browser }) => {
    // Watch the DOM from before React renders. A later "Create server" text is
    // the English label leaking into the French page.
    await page.addInitScript(() => {
      const w = window as I18nWindow;
      w.__englishSeen = false;
      const watch = () => {
        const observer = new MutationObserver(() => {
          if (/Create server/.test(document.body?.textContent ?? "")) {
            w.__englishSeen = true;
          }
        });
        observer.observe(document.documentElement, { childList: true, subtree: true, characterData: true });
      };
      if (document.readyState === "loading") {
        document.addEventListener("DOMContentLoaded", watch);
      } else {
        watch();
      }
    });

    await page.goto("/servers");
    await page.waitForLoadState("domcontentloaded");
    await loginIfNeeded(page);

    await page.reload({ waitUntil: "domcontentloaded" });
    await expect(page.locator("html")).toHaveAttribute("lang", "fr");
    await expect(page.getByText("Créer un serveur").first()).toBeVisible({ timeout: 15_000 });
    expect(await page.evaluate(() => (window as I18nWindow).__englishSeen)).toBe(false);

    // A new browser context has no local storage, so the account choice decides.
    const ctx2 = await browser.newContext({ storageState: ADMIN_STATE });
    try {
      const p2 = await ctx2.newPage();
      await p2.goto("/servers");
      await p2.waitForLoadState("domcontentloaded");
      await expect(p2.locator("html")).toHaveAttribute("lang", "fr", { timeout: 15_000 });
      await expect(p2.getByText("Créer un serveur").first()).toBeVisible({ timeout: 15_000 });
    } finally {
      await ctx2.close();
    }
  });

  test.describe("logged out", () => {
    // Logged-out contexts, as login-and-shell.spec.ts does, so the shared admin
    // session is not touched.
    test.use({ storageState: EMPTY_STATE });

    test("login and share pages offer a picker, remembered per browser", async ({ page, browser }) => {
      await page.goto("/login");
      await page.waitForLoadState("domcontentloaded");
      await pickFrenchFromCompactPicker(page);

      await expect(page.locator("html")).toHaveAttribute("lang", "fr");
      await expect(page.getByRole("textbox", { name: /E-mail ou nom d'utilisateur/ })).toBeVisible();
      await expect(page.getByRole("button", { name: /^Connexion$/ })).toBeVisible();

      try {
        // The browser choice persists across a reload.
        await page.reload({ waitUntil: "domcontentloaded" });
        await expect(page.locator("html")).toHaveAttribute("lang", "fr");
        const stored = await page.evaluate(() => window.localStorage.getItem("gameplane-language"));
        expect(JSON.parse(stored ?? "null")).toMatchObject({ choice: "fr", applied: "fr" });

        // A fresh logged-out context has no stored choice, so it starts in English.
        const fresh = await browser.newContext({ storageState: EMPTY_STATE });
        try {
          const freshPage = await fresh.newPage();
          await freshPage.goto("/login");
          await expect(freshPage.getByRole("button", { name: /^Sign in$/ })).toBeVisible();
          await expect(freshPage.locator("html")).toHaveAttribute("lang", "en");
        } finally {
          await fresh.close();
        }

        // The share page shows the same picker in its invalid-token state.
        await page.goto("/share/e2e-pw-i18n-missing");
        await page.waitForLoadState("domcontentloaded");
        await pickFrenchFromCompactPicker(page);
        await expect(page.locator("html")).toHaveAttribute("lang", "fr");
      } finally {
        await page.evaluate(() => {
          try {
            window.localStorage.removeItem("gameplane-language");
          } catch {
            // Storage can be blocked; the context is discarded at the end anyway.
          }
        });
      }
    });
  });

  test("a 0% fixture language shows English everywhere (SC-008)", async ({ page, request }) => {
    try {
      await page.goto("/settings/theme");
      await page.waitForLoadState("domcontentloaded");
      await expect(page.getByRole("radio", { name: /Français/ })).toBeVisible({ timeout: 15_000 });

      // The fixture language exists only when the web server was started with
      // VITE_I18N_FIXTURES=1 (vite.config.ts). The CI live job does not set it,
      // so this check reports the subtest as skipped rather than passed.
      const fixture = page.getByRole("radio", { name: /Fixture/ });
      if ((await fixture.count()) === 0) {
        test.skip(true, "VITE_I18N_FIXTURES is not set for the live web server, so the zz fixture language is absent");
      }

      await fixture.click();
      await expect(page.locator("html")).toHaveAttribute("lang", "zz", { timeout: 15_000 });

      // Every string falls back to English, and no raw message id is shown.
      await page.goto("/servers");
      await page.waitForLoadState("domcontentloaded");
      await expect(page.getByText("Create server").first()).toBeVisible({ timeout: 15_000 });
      const visible = await page.locator("body").innerText();
      expect(visible.split("\n").some((line) => /^web\.[a-z_.]+$/.test(line.trim()))).toBe(false);
    } finally {
      await resetAccountLanguage(request);
    }
  });
});
