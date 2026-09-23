# Contract: Theme Export Format

**Feature**: `016-user-theme-customization` (FR-014)  
**Binding Modules**: `web/src/lib/theme-export.ts`, `web/src/routes/ThemeSettings.tsx`, `web/src/lib/theme-sanitize.ts`  
**Status**: Binding (revised 2026-09-22: `ThemeSettingsModal` superseded by the `/settings/theme` settings page; export format unchanged)  

---

## 1. Overview

The Theme Export is a versioned JSON document capturing a user's complete theme configuration for manual transfer between Gameplane instances. Export is generated client-side from the user's own preferences payload; import is applied through `PUT /api/v1/users/me/preferences` so server-side validation and FR-013 sanitization remain the authoritative gate.

Out of scope (per clarification session 2026-09-21): theme sharing galleries, admin-enforced default themes, per-server or per-page themes.

---

## 2. Document Schema (`gameplane-theme` v1)

```json
{
  "format": "gameplane-theme",
  "version": 1,
  "preferences": {
    "themeType": "custom_colors",
    "presetId": "legacy",
    "appearanceMode": "dark",
    "customColors": {
      "accent": "#10B981",
      "surface": "#121114"
    },
    "customCssEnabled": true,
    "customCss": ".dashboard-card { border-radius: 12px; }"
  }
}
```

| Field | Type | Required | Rules |
|---|---|---|---|
| `format` | `string` | Yes | Must equal `"gameplane-theme"`. |
| `version` | `number` | Yes | Must equal `1`. Unknown higher versions are rejected with an upgrade prompt. |
| `preferences.themeType` | `string` | Yes | `"preset"` or `"custom_colors"`. |
| `preferences.presetId` | `string` | Yes | `"pink"` or `"legacy"`. |
| `preferences.appearanceMode` | `string` | Yes | `"light"`, `"dark"`, or `"system"`. |
| `preferences.customColors` | `object \| null` | No | `accent` and `surface` matching `^#([0-9a-fA-F]{6})$`. |
| `preferences.customCssEnabled` | `boolean` | Yes | Overlay flag; independent of `themeType`. |
| `preferences.customCss` | `string \| null` | No | Max 32,768 chars; subject to FR-013 sanitization on import. |

`updatedAt` is intentionally excluded — it is instance-local metadata.

---

## 3. Export Behavior

1. Built in `web/src/lib/theme-export.ts` from the currently loaded `UserThemePreferences`.
2. Offered on the Theme Settings page (Export / Import section) as:
   - **Copy to clipboard** (JSON text, pretty-printed, 2-space indent).
   - **Download** as `gameplane-theme.json` (`application/json`).
3. The export always reflects the saved profile state, including retained-but-inactive custom settings (e.g. `customCss` with `customCssEnabled: false`), so a full setup survives the transfer.

---

## 4. Import Behavior

1. User pastes JSON text or selects a `.json` file on the Theme Settings page (Export / Import section).
2. Client-side validation in `theme-export.ts` rejects with a specific message when:
   - JSON is unparseable, `format`/`version` mismatch, or required fields are missing;
   - any enum or hex color fails the rules in §2;
   - `customCss` exceeds 32,768 characters.
3. A valid document is previewed (preset name, accent swatch, CSS byte size) and applied on confirmation via `PUT /api/v1/users/me/preferences` with `customCssEnabled` taken from the document.
4. The server re-validates everything and applies FR-013 sanitization to `customCss`; a `400` response is surfaced inline on the page with the server message identifying the offending rule.
5. Import is a wholesale replace of the theme configuration (it is an explicit, confirmed action) — it does not merge with existing settings.
