# Research: User Theme Customization

**Feature**: `016-user-theme-customization`  
**Date**: 2026-09-19 (revised 2026-09-21 after clarification session 2026-09-21)  
**Status**: Completed  

---

## Executive Summary

This document establishes the technical decisions, architecture, and verification models for multi-theme support, user-specific theme persistence, legacy migration, simple custom color calculation, custom CSS as a sanitized overlay layer, safe-mode recovery, and theme export/import in Gameplane.

The 2026-09-21 revision absorbs five spec clarifications: CSS sanitization with a 32 KB cap (FR-013), custom CSS as an independent overlay with user cascade priority, three safe-mode entry points including a login-page link (FR-009), retain-until-explicit-reset lifecycle (FR-012), and theme export/import (FR-014).

---

## Research Items

### R-01: User Preferences Database Storage & Migration Design

**Decision**:  
Create a dedicated `user_preferences` table with a 1-to-1 relationship to `users(id)` with cascading deletion. Use migration `011_user_theme_preferences.sql` (migration `010` is already taken by `010_share_links_expiry_nullable.sql`). During migration execution:
1. Create the `user_preferences` table.
2. Seed rows for all pre-existing users in the `users` table setting `preset_id = 'legacy'`, `theme_type = 'preset'`, and `appearance_mode = 'system'`.
3. In application code, whenever a user profile is queried without a corresponding `user_preferences` row (e.g. newly created users), the system returns the default: `preset_id = 'pink'`, `theme_type = 'preset'`, `appearance_mode = 'system'`, `custom_css_enabled = false`.

**Rationale**:
- **Clean separation of concerns**: Keeps authentication, RBAC, and credential tables focused on security; preferences can evolve without altering `users` table layout.
- **Portability**: Plain SQL `CREATE TABLE` and `INSERT INTO ... SELECT` works identically across SQLite and PostgreSQL drivers without driver-specific JSON parsing functions.
- **Deterministic Migration**: Ensures existing user accounts are immediately assigned `legacy` upon migration execution, fulfilling requirement FR-003 and SC-001.

**DDL Structure**:
```sql
CREATE TABLE user_preferences (
    user_id            INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    theme_type         TEXT NOT NULL DEFAULT 'preset',   -- base mode: 'preset' | 'custom_colors'
    preset_id          TEXT NOT NULL DEFAULT 'pink',     -- 'pink' | 'legacy'
    appearance_mode    TEXT NOT NULL DEFAULT 'system',   -- 'light' | 'dark' | 'system'
    custom_accent      TEXT,                             -- hex string, e.g. '#3B82F6'
    custom_surface     TEXT,                             -- hex string, e.g. '#1E1E2E'
    custom_css_enabled INTEGER NOT NULL DEFAULT 0,       -- overlay on/off flag
    custom_css         TEXT,                             -- user CSS rules (max 32 KB, sanitized)
    updated_at         TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_user_preferences_user ON user_preferences(user_id);

-- Migrate all pre-existing accounts to legacy theme
INSERT INTO user_preferences (user_id, theme_type, preset_id, appearance_mode)
SELECT id, 'preset', 'legacy', 'system'
FROM users;
```

**Retention semantics (FR-012 clarification)**: `custom_accent`, `custom_surface`, and `custom_css` are never overwritten or nulled by ordinary updates (preset switches, overlay toggles). Only the dedicated reset action (`POST /api/v1/users/me/preferences/reset`, see contracts/user-preferences-api.md) sets them to `NULL`.

**Alternatives Considered**:
- *JSON Column on `users` table*: Rejected because SQLite and Postgres handle JSON extraction differently in SQL queries; a dedicated relational table is simpler to type and query across both database engines.
- *Client-side `localStorage` only*: Rejected per user decision in Session 2026-09-19 Q1; client-only storage cannot reliably identify existing accounts across devices or after browser cache clear.
- *Storing overlay state inside `theme_type` as a third value (`'custom_css'`)*: Rejected per clarification session 2026-09-21; custom CSS is an overlay, not a base mode (see R-06).

---

### R-02: CSS Variable System & Token Mapping for Pink vs Legacy Themes

**Decision**:  
Structure theme tokens in `web/src/styles/globals.css` using `data-theme-preset` attributes on `<html>`:
- `data-theme-preset="pink"` (default): Uses the existing HeroUI semantic tokens approved in `specs/done_014-heroui-web-rebuild/contracts/theme-tokens.md` (Pink accent `#FF4FA3` dark / `#DB2777` light).
- `data-theme-preset="legacy"`: Rebinds HeroUI semantic variables (`--accent`, `--surface`, `--background`, `--foreground`, `--border`, etc.) to the original Gameplane palette (Orange accent `#F97316`, dark neutral ground `#0F0F0F`, card `#1C1C1C`, border `#292929`).

**Rationale**:
- Rebinding HeroUI semantic variables ensures that **all rebuilt screens automatically adapt** without code changes or conditional class rendering in React components.
- The existing legacy `--gp-*` tokens in `globals.css` already hold the historical orange palette values. Re-pointing `--accent` and `--surface` when `data-theme-preset="legacy"` delivers pixel-accurate restoration of the classic theme.

**Token Mapping for Legacy Theme**:
```css
/* Legacy Dark Theme */
.dark[data-theme-preset="legacy"],
[data-theme="dark"][data-theme-preset="legacy"] {
  --background: oklch(14.5% 0 0);           /* #0F0F0F */
  --foreground: oklch(96.5% 0 0);           /* #F5F5F5 */
  --surface: oklch(18.5% 0 0);              /* #171717 */
  --surface-secondary: oklch(16.5% 0 0);    /* #141414 */
  --surface-tertiary: oklch(15.0% 0 0);     /* #111111 */
  --overlay: oklch(21.0% 0 0);              /* #1C1C1C */
  --accent: oklch(69.11% 0.1944 44.01);     /* #F97316 orange */
  --accent-foreground: oklch(100% 0 0);     /* #FFFFFF */
  --accent-soft: oklch(25.0% 0.06 45.0);
  --accent-soft-foreground: oklch(80.0% 0.15 45.0);
  --default: oklch(21.0% 0 0);              /* #1C1C1C */
  --default-foreground: oklch(96.5% 0 0);   /* #F5F5F5 */
  --border: oklch(26.0% 0 0);               /* #292929 */
  --separator: oklch(26.0% 0 0);            /* #292929 */
  --muted: oklch(65.0% 0 0);                /* #949494 */
  --field-background: oklch(18.5% 0 0);     /* #171717 */
  --field-border: oklch(26.0% 0 0);         /* #292929 */
  --field-placeholder: oklch(50.0% 0 0);    /* #737373 */
  --field-foreground: oklch(96.5% 0 0);     /* #F5F5F5 */
  --focus: oklch(69.11% 0.1944 44.01);      /* #F97316 */
  --link: oklch(72.0% 0.16 45.0);           /* #FB923C */
  --segment: oklch(21.0% 0 0);              /* #1C1C1C */
  --segment-foreground: oklch(96.5% 0 0);   /* #F5F5F5 */
}

/* Legacy Light Theme */
.light[data-theme-preset="legacy"],
[data-theme="light"][data-theme-preset="legacy"] {
  --background: oklch(100% 0 0);            /* #FFFFFF */
  --foreground: oklch(18.0% 0.03 260);      /* #0F172A */
  --surface: oklch(98.5% 0.005 260);        /* #F8FAFC */
  --surface-secondary: oklch(96.0% 0.01 260);/* #F1F5F9 */
  --surface-tertiary: oklch(91.0% 0.02 260); /* #E2E8F0 */
  --overlay: oklch(98.5% 0.005 260);        /* #F8FAFC */
  --accent: oklch(62.0% 0.20 40.0);         /* #EA580C */
  --accent-foreground: oklch(100% 0 0);     /* #FFFFFF */
  --accent-soft: oklch(95.0% 0.04 45.0);
  --accent-soft-foreground: oklch(55.0% 0.20 40.0);
  --border: oklch(90.0% 0.01 260);          /* #E2E8F0 */
  --separator: oklch(90.0% 0.01 260);       /* #E2E8F0 */
  --muted: oklch(50.0% 0.02 260);           /* #64748B */
  --field-background: oklch(100% 0 0);
  --field-border: oklch(90.0% 0.01 260);
  --field-foreground: oklch(18.0% 0.03 260);
  --focus: oklch(62.0% 0.20 40.0);
  --link: oklch(55.0% 0.20 40.0);
}
```

**Alternatives Considered**:
- *Duplicating component styling*: Rejected because hardcoding classnames per theme produces massive bloat; CSS variable swapping is O(1) in CSS bundle size.

---

### R-03: Simple Custom Color Scheme Derivation Algorithm

**Decision**:  
When `theme_type === "custom_colors"`, the user inputs two values:
1. `custom_accent`: A hex color string (e.g. `#10B981` emerald, `#3B82F6` blue).
2. `custom_surface`: A surface tone selector or hex string (defaulting to dark neutral `#141318` or light `#F8FAFC`).

From these two inputs, a pure TypeScript utility (`deriveCustomThemeTokens` in `web/src/lib/theme-derivation.ts`) computes all derivative CSS tokens:
- **Accent foreground**: White `#FFFFFF` or Black `#000000` depending on relative luminance (WCAG AA ratio >= 4.5:1).
- **Accent soft**: 15% opacity tint of accent over the surface.
- **Surface layers**: Base surface, secondary (±3% lightness), tertiary (±5% lightness).
- **Borders**: Surface with elevated lightness (+10% in dark mode, -10% in light mode).
- **Muted text**: Intermediate luminance between surface and foreground ensuring >= 4.5:1 contrast against surface.

**Application**:
The derived tokens are injected into a dedicated `<style id="gameplane-custom-theme-vars">` element in `<head>`. Custom colors form the *base* theme; an enabled custom CSS overlay still layers on top of it (see R-06).

**Alternatives Considered**:
- *Full manual token editor*: Rejected per user decision in Session 2026-09-19 Q2; primary accent + surface tone provides 95% of desired customization with zero risk of broken intermediate states.

---

### R-04: Custom CSS Sanitization & Safe-Mode Recovery Architecture

**Decision (revised 2026-09-21 per FR-013 and FR-009)**:  

**Sanitization** — applied on the server at save time (enforcement gate) and mirrored on the client for immediate feedback, in a shared-rules module (`web/src/lib/theme-sanitize.ts` client, equivalent checks in `api/internal/handlers/users.go`):
1. **External reference rejection**: Stylesheets containing `@import` rules or external `url()` references (absolute `http://`/`https://` or protocol-relative `//host/...` URLs, including remote font loads) are rejected with `400 Bad Request` and a validation message identifying the offending rule. Inline `data:` URIs are explicitly permitted.
2. **Syntax validation**: Unbalanced braces or unparseable declarations are rejected (client warns before save; server is authoritative).
3. **Size cap**: Maximum 32,768 characters (32 KB). The cap was raised from an initial 16 KB proposal so users can manually paste third-party CSS content inline that they would otherwise have imported.
4. **DOM-escape defense**: Text containing `<style`, `</style`, `<script`, or `</script` is rejected, preventing breakout from the `<style>` element regardless of injection path.

**Isolation (FR-008 / FR-011)**:
- Evaluated **only** when a user is authenticated (`me` is loaded) and `customCssEnabled` is true.
- Unauthenticated pages (Login `/login` and Share `/share/:token`) **never** inject user custom CSS.

**Safe-Mode Recovery — three entry points (FR-009)**:
1. **URL query parameter** `?safe-mode=1` — the guaranteed path; works regardless of how badly the CSS broke the page; scriptable in E2E tests and shareable as a support link.
2. **Keyboard shortcut** (`Ctrl + Shift + Alt + T`) — a convenience for desktop power users.
3. **Safe-mode sign-in link on the login page** — the login page is a guaranteed-clean surface (custom CSS never executes there, FR-011), so it offers a *"Sign in with safe mode (custom styling disabled)"* link that carries the safe-mode flag into the authenticated session after successful sign-in.

When Safe Mode is active:
- The custom CSS overlay is suspended **for the session only** — the stored stylesheet is not deleted (the user edits or clears it after recovery).
- A discreet floating banner appears: *"Safe Mode active (Custom CSS suspended). [Open Appearance Settings] [Dismiss]"*.

**Rationale**: CSS is strictly isolated to its author, so the real risks are privacy leakage to third parties (external fetches can track IP/activity or exfiltrate data) and unbounded payloads — not self-inflicted styling. Blocking remote loads while allowing arbitrary local rules preserves full styling freedom. A URL parameter is the only recovery mechanism that cannot be defeated by the broken CSS itself (a dialog can be hidden; a keybinding is unavailable on touch devices), and the login page extends that guarantee to users whose UI is too broken to even reach the address bar with a session open.

**Alternatives Considered**:
- *No restrictions (verbatim injection)*: Rejected — permits silent third-party tracking/exfiltration via `url()` and unbounded profile payloads.
- *Strict property allowlist*: Rejected — largely defeats the purpose of "custom CSS" for power users.
- *Iframe sandboxing*: Rejected because custom CSS is intended to style the application shell and dashboard screens directly; iframing the entire app would destroy layout and state management.
- *Recovery dialog as the sole mechanism*: Rejected — the broken CSS can hide the dialog itself.

---

### R-05: Client-Side Hydration, Boot Script & Multi-Device Sync

**Decision**:  
To prevent flash of unstyled content (FOUC):
1. **Local Cache**: The application caches `{ themeType, presetId, appearanceMode, customColors, customCssEnabled, customCss }` in `localStorage` under key `gameplane-theme-prefs`. (The existing key `gameplane-theme` holding only light/dark/system remains the source for the boot mode class until the prefs cache exists; the boot script migrates to reading `gameplane-theme-prefs` first.)
2. **Boot Script (`index.html`)**:
   - Reads `gameplane-theme-prefs`.
   - Sets `data-theme-preset` (`"pink"` | `"legacy"`), `data-theme-type` (`"preset"` | `"custom_colors"`), and `.dark` / `.light` class synchronously before the first paint.
   - If `customCssEnabled` and `customCss` exist **and** the URL does not contain `safe-mode=1`, injects the initial `<style id="gameplane-custom-css">` as the **last** element of `<head>` (see R-06 ordering guarantee).
3. **Profile Reconciliation (`AppLayout.tsx`)**:
   - When `useMe()` (in `web/src/lib/auth.ts`) loads, compare backend preferences with `localStorage`.
   - If backend preferences differ (e.g. user updated theme on another machine), update `localStorage` and smoothly re-apply the DOM attributes.
4. **Mutations**:
   - `useThemePreferences` hook (in `web/src/lib/useThemePreferences.ts`, following the codebase convention of keeping hooks in `lib/`) provides `updatePreferences(...)`.
   - Immediately updates DOM and `localStorage` (optimistic UI), then calls `PUT /api/v1/users/me/preferences`.
   - On error, reverts local state and shows an error toast.
5. **Offline behavior (edge case)**: Optimistic local updates apply immediately; the mutation retries/synchronizes when connectivity resumes, matching the spec's Offline / Transient Connectivity edge case.

---

### R-06: Custom CSS Overlay Cascade Model

**Decision (new, 2026-09-21 clarification)**:  
Custom CSS is an **independent overlay layer**, not a third theme mode:
1. `themeType` describes only the **base** theme: `"preset"` (Pink or Legacy) or `"custom_colors"`.
2. `customCssEnabled` independently controls whether the user's stylesheet is injected on top of whatever base is active. Both can be used together.
3. **Cascade priority**: `<style id="gameplane-custom-css">` is always mounted as the **last** child of `document.head`, after both the preset token declarations and `<style id="gameplane-custom-theme-vars">`. At equal specificity, user rules therefore win over every base layer.
4. **Base-switch behavior**: When the user switches the base theme (preset or custom colors) while the overlay is enabled, elements not targeted by custom CSS immediately adopt the new base styles; elements targeted by custom CSS keep their customized appearance (color, rounding, positioning) because the overlay remains last in the cascade.
5. **Disable vs reset**: Toggling the overlay off unmounts the `<style>` element but retains the stored stylesheet (it can be re-enabled later). Only the explicit "Reset to Defaults" action (FR-012) deletes it server-side.

**Rationale**: The spec names the feature "Custom CSS *Overrides*"; power users (US4) want to tweak fonts, spacing, and component treatments on top of their chosen theme — including existing Legacy users — not rebuild a theme from scratch. An independent flag also keeps safe mode trivially scoped: suspending the overlay never touches base theme state.

**Alternatives Considered**:
- *Mutually exclusive third mode over the selected preset*: Rejected — deactivates custom colors when CSS is used and makes "which base applies" ambiguous.
- *Exclusive mode on a fixed Pink base*: Rejected — surprising for Legacy users, and contradicts the "overrides" framing.

---

### R-07: Theme Export / Import Format

**Decision (new, 2026-09-21 clarification, FR-014)**:  
Users can export their complete theme configuration as portable JSON text and import it on another Gameplane instance.
1. **Format**: A versioned JSON document (`gameplane-theme` v1) containing `themeType`, `presetId`, `appearanceMode`, `customColors`, `customCssEnabled`, and `customCss`. Full schema in `contracts/theme-export.md`.
2. **Export**: Generated **client-side** from the loaded preferences (`web/src/lib/theme-export.ts`) and offered as copy-to-clipboard and `.json` file download. No backend endpoint required — the data is already in the user's own profile payload.
3. **Import**: Paste or file-upload in the Theme Settings modal → client validates shape, enums, hex colors, and CSS size → calls the existing `PUT /api/v1/users/me/preferences`.
4. **Enforcement gate**: The server-side PUT validation and FR-013 sanitization are authoritative — an imported stylesheet containing `@import` or external `url()` references is rejected exactly as if typed by hand. Import therefore cannot bypass sanitization.
5. **Scope boundaries**: Theme sharing galleries, admin-enforced default themes, and per-server/per-page themes are explicitly out of scope.

**Revision (2026-09-22)**: The operator replaced the Theme Settings modal with a full settings page at route `/settings/theme` (standard app shell: sidebar + top bar + page header; Presets / Custom Colors / Custom CSS / Export as in-page tabs per contracts/theme-ui.md). The export/import format and flows above are unchanged — copy/download and paste/file-upload now live in the page's Export / Import tab.

**Alternatives Considered**:
- *Dedicated server export/import endpoints*: Rejected as unnecessary — export is a re-serialization of data the client already holds, and import must pass through PUT validation anyway.
- *Import applying raw CSS without sanitization*: Rejected outright (FR-014 requires imported CSS to pass FR-013).

---

## Constitution Compliance Analysis

| Constitution Principle | Compliance Assessment |
|---|---|
| **I. E2E-Tested Delivery** | Playwright live specs in `web/e2e/specs/live/theme-customization.spec.ts` will test preset switching, migration defaults, persistence across reloads, custom colors, overlay cascade priority across base switches, retention after toggles, safe-mode recovery via URL parameter and login-page link, sanitization rejection, and export/import round-trip. Tests call `t.Parallel()` equivalents per project e2e conventions with unique resource names. |
| **II. Design-First** | The theme settings page at `/settings/theme`, appearance controls, Safe Mode banner, export/import controls, and the login-page safe-mode link will be mapped and designed in `design.pen` via the Pencil MCP server before code implementation, with exports to `design-export/`. |
| **III. Language & Ecosystem** | Strict TypeScript; Go handlers wrapped with `%w`; zero suppression directives (`//nolint`, `// @ts-ignore`). |
| **IV. Spec-Driven Development** | Follows spec -> clarify -> plan -> research -> data-model -> contracts -> quickstart. `web/specs.md` and `api/specs.md` will be updated in the same change as the implementation. |
| **V. Delegate to Workflows** | Tasks will be decomposed into independent subagent units (API/DB slice, token slice, overlay/sanitization slice, UI settings-page slice, export/import slice, E2E slice). |
| **VI. CI Bears the Heavy Lifting** | Verified on GitHub Actions CI; local checks limited to `go build ./...` and `tsc --noEmit`. |
