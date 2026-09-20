import { test, expect, type Locator, type Page } from "@playwright/test";
import { LoginPage, loginIfNeeded } from "../pages/LoginPage";
import { ServersPage } from "../pages/ServersPage";
import { ServerDetailPage } from "../pages/ServerDetailPage";

// SC-007 keyboard-only pass (specs/done_014-heroui-web-rebuild, T207), automated.
//
// After the initial page.goto, every step uses page.keyboard only (Tab,
// Shift+Tab, Enter, arrow keys, Home/End, Escape). Every Tab stop on the way
// must carry a non-empty accessible name ("no unlabeled control") and paint a
// :focus-visible indicator. OD-16 settles the ring colour as the theme accent;
// the colour itself is covered by the screenshot rulings, so this only asserts
// that an indicator is painted.
//
// Mock-only, like settingsSubTabs.spec.ts: it relies on the deterministic MSW
// fixtures (the `alpha` server, the Keycloak SSO provider) that live mode does
// not have.

const MAX_TAB_STOPS = 80;

async function expectFocusedStopIsUsable(page: Page): Promise<void> {
  const focused = page.locator(":focus");
  await expect(focused, "a Tab stop must hold focus").toHaveCount(1);
  await expect(focused, "focused control has no accessible name").toHaveAccessibleName(/\S/);
  const problem = await page.evaluate(() => {
    const el = document.activeElement;
    if (!(el instanceof HTMLElement) || el === document.body) return "no focused element";
    const label = el.getAttribute("aria-label") ?? el.textContent?.trim().slice(0, 40) ?? "";
    if (!el.matches(":focus-visible")) return `${el.tagName} "${label}" is focused but not :focus-visible`;
    const painted = (cs: CSSStyleDeclaration): boolean =>
      (cs.outlineStyle !== "none" && parseFloat(cs.outlineWidth) > 0) ||
      (cs.boxShadow !== "" && cs.boxShadow !== "none");
    const styles = [
      getComputedStyle(el),
      getComputedStyle(el, "::before"),
      getComputedStyle(el, "::after"),
    ];
    return styles.some(painted) ? "" : `${el.tagName} "${label}" paints no focus indicator`;
  });
  expect(problem, "visible focus indicator").toBe("");
}

function hasFocus(target: Locator): () => Promise<boolean> {
  return () => target.evaluate((el) => el === document.activeElement);
}

function focusWithin(container: Locator): () => Promise<boolean> {
  return () => container.evaluate((el) => el.contains(document.activeElement));
}

// Presses Tab (or Shift+Tab) until `reached` holds, checking every stop on
// the way. Failing to arrive within `max` presses means the control is not
// keyboard-reachable (or focus is trapped somewhere before it).
async function tabUntil(
  page: Page,
  reached: () => Promise<boolean>,
  what: string,
  opts: { key?: "Tab" | "Shift+Tab"; max?: number } = {},
): Promise<void> {
  const key = opts.key ?? "Tab";
  const max = opts.max ?? MAX_TAB_STOPS;
  for (let i = 0; i < max; i++) {
    await page.keyboard.press(key);
    await expectFocusedStopIsUsable(page);
    if (await reached()) return;
  }
  throw new Error(`${key} did not reach ${what} within ${max} stops`);
}

async function tabTo(page: Page, target: Locator, what: string, max?: number): Promise<void> {
  await expect(target).toBeVisible();
  await tabUntil(page, hasFocus(target), what, { max });
}

test.describe("keyboard-only pass (SC-007)", () => {
  test.skip(
    process.env.GAMEPLANE_E2E_TARGET === "live",
    "depends on deterministic mock fixtures (alpha server, SSO provider); mock mode is the reproducible path",
  );

  test("login: every control is reachable in order, labelled, visibly focused, and the form submits by keyboard", async ({
    page,
  }) => {
    const login = new LoginPage(page);
    await login.goto();

    // Wait for /auth/providers so the SSO button (the last stop) is rendered.
    const sso = page.getByRole("button", { name: /continue with keycloak/i });
    await expect(sso).toBeVisible();

    // autoFocus puts the caret in the username field on load.
    await expect(login.username).toBeFocused();
    await expectFocusedStopIsUsable(page);

    const forgot = page.getByRole("button", { name: /^forgot\?$/i });
    const reveal = page.getByRole("button", { name: /^show password$/i });

    for (const next of [forgot, login.password, reveal, login.submit, sso]) {
      await page.keyboard.press("Tab");
      await expect(next).toBeFocused();
      await expectFocusedStopIsUsable(page);
    }
    // No one-way trap: Shift+Tab walks back through the same controls.
    for (const prev of [login.submit, reveal, login.password, forgot, login.username]) {
      await page.keyboard.press("Shift+Tab");
      await expect(prev).toBeFocused();
      await expectFocusedStopIsUsable(page);
    }

    // Operate the form without a mouse: type, move with Tab, submit with Enter.
    await page.keyboard.type("e2e-admin");
    await page.keyboard.press("Tab"); // Forgot?
    await page.keyboard.press("Tab"); // password
    await expect(login.password).toBeFocused();
    await page.keyboard.type("any-non-empty");
    await page.keyboard.press("Enter");
    await page.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 10_000 });
  });

  test("servers list: toolbar, filter popover and a server row are reachable and operable by keyboard", async ({
    page,
  }) => {
    await page.goto("/servers");
    await page.waitForLoadState("domcontentloaded");
    await loginIfNeeded(page);

    const servers = new ServersPage(page);
    const alphaLink = servers.getServerLink("alpha");
    await expect(alphaLink).toBeVisible();

    // Shell (sidebar, top bar) comes first; every stop on the way is checked.
    await tabTo(page, servers.createServerButton, "the Create server link");

    // Status filter: WAI-ARIA tablist, one Tab stop, arrows move and select.
    const allTab = servers.statusTabs.getByRole("tab", { name: /^all/i });
    const runningTab = servers.statusTabs.getByRole("tab", { name: /^running/i });
    await tabTo(page, allTab, "the All status tab", 5);
    await page.keyboard.press("ArrowRight");
    await expect(runningTab).toBeFocused();
    await expect(runningTab).toHaveAttribute("aria-selected", "true");
    await expectFocusedStopIsUsable(page);
    await page.keyboard.press("ArrowLeft");
    await expect(allTab).toBeFocused();
    await expect(allTab).toHaveAttribute("aria-selected", "true");
    await expect(alphaLink).toBeVisible();

    await tabTo(page, servers.searchInput, "the Search servers field", 5);
    await tabTo(page, servers.filterButton, "the Filter button", 5);

    // Popover opens by keyboard; Escape closes it and focus returns to the trigger.
    await page.keyboard.press("Enter");
    const apply = page.getByRole("button", { name: /^apply$/i });
    await expect(apply).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(apply).toBeHidden();
    await expect(servers.filterButton).toBeFocused();

    // Grid: one Tab stop, arrows move inside it, Enter opens the server.
    await tabUntil(page, focusWithin(servers.serversTable), "the Server list grid", { max: 5 });
    for (let i = 0; i < 4 && !(await hasFocus(alphaLink)()); i++) {
      await page.keyboard.press("ArrowRight");
    }
    await expect(alphaLink).toBeFocused();
    await expectFocusedStopIsUsable(page);
    await page.keyboard.press("Enter");
    await page.waitForURL(/\/servers\/alpha(\?|$)/, { timeout: 10_000 });
  });

  test("server settings: Danger zone reached by keyboard; delete dialog traps focus and Escape returns it to the trigger", async ({
    page,
  }) => {
    const detail = new ServerDetailPage(page);
    await detail.goto("alpha");
    await page.waitForLoadState("domcontentloaded");
    await loginIfNeeded(page);

    // Server detail tablist: one Tab stop on the selected (Overview) tab;
    // Settings is the last tab, so End selects it (automatic activation).
    const overviewTab = detail.tablist.getByRole("tab", { name: /^overview$/i });
    await tabTo(page, overviewTab, "the Overview tab");
    await page.keyboard.press("End");
    const settingsTab = detail.tablist.getByRole("tab", { name: /^settings$/i });
    await expect(settingsTab).toBeFocused();
    await expect(settingsTab).toHaveAttribute("aria-selected", "true");

    // Settings sections are a vertical tablist (Settings.tsx); Danger zone is last.
    const generalTab = page.getByRole("tab", { name: /^general$/i });
    await tabTo(page, generalTab, "the General settings tab", 10);
    await page.keyboard.press("End");
    const dangerTab = page.getByRole("tab", { name: /^danger zone$/i });
    await expect(dangerTab).toBeFocused();
    await expect(dangerTab).toHaveAttribute("aria-selected", "true");

    // Danger zone rows: Wipe world…, Transfer…, Delete.
    const deleteButton = page.getByRole("button", { name: /^delete$/i });
    await tabTo(page, deleteButton, "the Danger zone Delete button", 10);
    await page.keyboard.press("Enter");

    const dialog = page
      .locator('[role="alertdialog"], [role="dialog"]')
      .filter({ hasText: /delete alpha\?/i });
    await expect(dialog).toBeVisible();

    // ConfirmDialog autofocuses the confirm-phrase input.
    await expect(dialog.getByRole("textbox")).toBeFocused();
    await expectFocusedStopIsUsable(page);

    // Focus trap: the cycle is input -> Cancel (Delete server stays disabled
    // until the phrase matches); neither direction may leave the dialog.
    for (const key of ["Tab", "Tab", "Tab", "Shift+Tab", "Shift+Tab", "Shift+Tab"]) {
      await page.keyboard.press(key);
      await expect(dialog.locator(":focus"), `focus escaped the dialog on ${key}`).toHaveCount(1);
      await expectFocusedStopIsUsable(page);
    }

    // Escape closes the dialog and restores focus to the control that opened it.
    await page.keyboard.press("Escape");
    await expect(dialog).toBeHidden();
    await expect(deleteButton).toBeFocused();
  });
});
