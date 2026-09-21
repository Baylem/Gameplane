# Quickstart: User Theme Customization Validation Guide

**Feature**: `016-user-theme-customization`  
**Date**: 2026-09-19 (revised 2026-09-21 after clarification session 2026-09-21)  
**Status**: Ready for Verification  

**Execution venue**: The checks in this guide run on the CI pipeline per Constitution Principle VI (tests and lint do not run on developer/agent local machines; the sole exception is an operator-provided remote host). This document is the executable checklist the CI run is validated against.

---

## 1. Overview

This guide provides runnable end-to-end verification procedures to validate that:
1. The database migration correctly defaults existing users to `legacy` and new users to `pink`.
2. The user preferences API (`GET/PUT /api/v1/users/me/preferences`, `POST .../preferences/reset`) persists, validates, and sanitizes settings, and enforces retain-until-reset semantics.
3. The dashboard UI switches between Pink and Legacy presets without reloading.
4. Custom colors apply as a base theme; custom CSS applies as an overlay that wins over the base and survives base theme switches.
5. Safe Mode recovers broken stylesheets via all three entry points: `?safe-mode=1`, the keyboard shortcut, and the login-page safe-mode link.
6. Theme export/import round-trips a full configuration, with imported CSS sanitized per FR-013.

---

## 2. Prerequisites

- Go 1.25+ installed.
- Node.js 20+ and npm installed.
- Repository cloned and current on `016-user-theme-customization` branch.

---

## 3. Database Migration & API Validation

### 3.1 Run Database Migration Test

Verify that migration `011_user_theme_preferences.sql` executes and migrates pre-existing users:

```bash
cd /home/valgul/project/Gameplane-Sec/api
go test -v ./internal/db -run TestUserThemePreferencesMigration
```

**Expected Outcome**:
- `user_preferences` table is created, including the `custom_css_enabled` column (default 0).
- Pre-existing users in the test database hold `preset_id = 'legacy'`.
- Newly inserted users hold default `preset_id = 'pink'`, `theme_type = 'preset'`, `custom_css_enabled = 0`.

### 3.2 Run User Handler Preferences Test

Test preferences API endpoint validation, sanitization, retention, and permissions:

```bash
cd /home/valgul/project/Gameplane-Sec/api
go test -v ./internal/handlers -run TestUserPreferences
```

**Expected Outcome**:
- `GET /users/me/preferences` returns the caller's preferences (default pink when no row exists).
- `PUT /users/me/preferences` accepts valid payloads including `customCssEnabled`.
- `PUT` returns `400 Bad Request` with a rule-identifying message for: invalid hex `#XYZ`; CSS containing `@import`; CSS containing an external `url()` (`https://...` or `//host/...`); CSS over 32,768 chars; CSS containing `<style>`/`<script>` delimiters. CSS with inline `data:` URIs is accepted.
- `PUT` that switches `presetId` or toggles `customCssEnabled` leaves stored `customColors`/`customCss` intact (retention, FR-012).
- `POST /users/me/preferences/reset` nulls all custom fields, disables the overlay, and restores the requested preset.
- Returns `401 Unauthorized` for unauthenticated requests.

---

## 4. Frontend Component & Unit Validation

### 4.1 Theme Tokens & Preset Tests

Verify token values for both Pink and Legacy presets in light and dark modes:

```bash
cd /home/valgul/project/Gameplane-Sec/web
npm test -- src/__tests__/theme.test.tsx src/__tests__/themeBoot.test.ts
```

**Expected Outcome**:
- Light mode tokens assert `--accent` is `#DB2777` for Pink and `#EA580C` for Legacy.
- Dark mode tokens assert `--accent` is `#FF4FA3` for Pink and `#F97316` for Legacy.
- `themeBoot.test.ts` validates that the `index.html` boot script reads `gameplane-theme-prefs` from `localStorage`, initializes `data-theme-preset` / `data-theme-type` / `data-custom-css`, injects cached custom CSS last in `<head>`, and skips injection when the URL contains `safe-mode=1`.

### 4.2 Sanitization & Export/Import Unit Tests

```bash
cd /home/valgul/project/Gameplane-Sec/web
npm test -- src/lib/theme-sanitize.test.ts src/lib/theme-export.test.ts src/lib/theme-derivation.test.ts
```

**Expected Outcome**:
- `theme-sanitize.test.ts`: `@import` and external `url()` rejected with the offending rule named; `data:` URIs pass; 32 KB cap enforced; unbalanced braces flagged.
- `theme-export.test.ts`: export produces a valid `gameplane-theme` v1 document including retained-but-inactive customs; import validation rejects malformed/oversized/unknown-version documents; valid documents map to the PUT payload.
- `theme-derivation.test.ts`: contrast math picks a legible accent-foreground for extreme accent/surface pairs.

### 4.3 Theme Settings Modal Tests

Verify UI interactions in `ThemeSettingsModal.test.tsx`:

```bash
cd /home/valgul/project/Gameplane-Sec/web
npm test -- src/components/ui/ThemeSettingsModal.test.tsx
```

**Expected Outcome**:
- Modal renders tabs: Presets, Custom Colors, Custom CSS, Export.
- Clicking "Legacy" updates DOM attribute `data-theme-preset="legacy"` without clearing stored custom values.
- The overlay toggle flips `data-custom-css` between `"on"`/`"off"` and mounts/unmounts `#gameplane-custom-css` as the last `<head>` child.
- Clicking "Reset to Defaults" asks for one confirmation, then calls the reset endpoint and clears custom CSS and colors.

---

## 5. End-to-End Browser Flow Validation

### 5.1 Run Playwright Live E2E

Run the automated UI test suite for theme customization against the live stack:

```bash
cd /home/valgul/project/Gameplane-Sec/web
npx playwright test e2e/specs/live/theme-customization.spec.ts
```

**Coverage**: preset switching without reload, migration defaults (existing user → Legacy, new user → Pink), persistence across reload/devices, custom colors base, overlay cascade priority across a preset switch, retention after overlay toggle, safe mode via URL parameter and via the login-page link, sanitization rejection in the editor, export → import round-trip on a second account, and unauthenticated exclusion (login + share link render Pink, zero custom CSS).

### 5.2 Manual Browser Walkthrough

1. **Sign in as an existing user**:
   - Open browser to `http://localhost:5173/login`.
   - Verify the login page renders with modern Pink branding and shows the *"Sign in with safe mode"* link.
   - Sign in with an account created prior to migration.
   - Verify dashboard shell immediately renders in **Legacy Theme** (orange buttons, classic dark background).

2. **Switch to Modern Pink**:
   - Click user avatar in TopBar -> Select **Theme & Appearance**.
   - Select **Modern Pink** -> Click **Save**.
   - Verify buttons shift to vibrant pink and backgrounds shift to deep violet-dark.
   - Refresh page; verify Modern Pink remains active with no flash of the wrong theme.

3. **Configure Custom Colors**:
   - Reopen **Theme & Appearance** -> Switch to **Custom Colors** tab.
   - Choose an Emerald green accent (`#10B981`) -> Click **Save**.
   - Verify primary buttons and active navigation markers turn emerald green.

4. **Verify Overlay Priority Across a Base Switch**:
   - In the **Custom CSS** tab, enable the overlay and save:
     ```css
     .topbar { border-bottom: 3px solid lime; }
     ```
   - Verify the lime border appears.
   - Switch the base preset from Pink to Legacy and back.
   - Verify the lime border persists through both switches while all untargeted elements follow each base theme.

5. **Verify Custom CSS Sanitization**:
   - In the **Custom CSS** tab, paste:
     ```css
     @import url("https://evil.example.com/track.css");
     ```
   - Click **Save** and verify an inline error names the offending `@import` rule; the stylesheet is not saved.
   - Paste 33 KB of valid CSS and verify the size error.

6. **Verify Safe Mode (all entry points)**:
   - Save `body { opacity: 0.1 !important; }` with the overlay enabled. Observe the screen dim drastically.
   - **URL parameter**: Append `?safe-mode=1` to the URL and press Enter. Verify opacity restores, `data-custom-css="off"`, and the Safe Mode banner appears.
   - **Keyboard shortcut**: Re-enable the overlay, reload, then press the safe-mode shortcut (`Ctrl + Shift + Alt + T`). Verify the same suspension + banner.
   - **Login-page link**: Sign out, click *"Sign in with safe mode"* on the login page, and sign in. Verify the session starts with the overlay suspended and the banner visible.
   - From the banner, open Appearance Settings and click **Reset to Defaults** (one confirmation). Verify custom styling is deleted and the preset restored.

7. **Verify Export / Import Round-Trip**:
   - With a full configuration active (Legacy preset + emerald custom colors retained + CSS overlay), open the **Export** tab and copy/download the JSON.
   - Sign in as a second user, open **Export** tab, paste the JSON, preview, and apply the import.
   - Verify the second user's dashboard reproduces the full setup; verify an exported document hand-edited to include `@import` is rejected on import with the server message.

8. **Verify Unauthenticated Exclusion**:
   - Sign out and open `/login` and a public `/share/:token` link.
   - Verify both render in the modern Pink preset and `#gameplane-custom-css` is absent from the DOM.
