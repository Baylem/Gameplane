# Contract: Theme Tokens v2

**Feature**: `016-user-theme-customization`  
**Binding Modules**: `web/src/styles/globals.css`, `web/src/components/AppLayout.tsx`, `web/index.html`  
**Status**: Binding (revised 2026-09-21 after clarification session 2026-09-21)  

---

## 1. Overview

This contract extends `specs/done_014-heroui-web-rebuild/contracts/theme-tokens.md` by defining:
1. The token declarations for the **Legacy Theme** preset (`data-theme-preset="legacy"`).
2. The dynamic override mechanism for **Simple Custom Color Scheme** (`data-theme-type="custom_colors"`).
3. The injection, ordering, and scoping contract for the **Custom CSS overlay** (`#gameplane-custom-css`, gated by `customCssEnabled`).

---

## 2. Token Comparison Table

| Semantic Token | Pink Preset (Dark) | Pink Preset (Light) | Legacy Preset (Dark) | Legacy Preset (Light) |
|---|---|---|---|---|
| `--accent` | `#FF4FA3`<br>`oklch(69.50% 0.2229 355.31)` | `#DB2777`<br>`oklch(59.16% 0.2180 0.58)` | `#F97316`<br>`oklch(69.11% 0.1944 44.01)` | `#EA580C`<br>`oklch(62.00% 0.2000 40.00)` |
| `--accent-foreground` | `#FFFFFF`<br>`oklch(100% 0 0)` | `#FFFFFF`<br>`oklch(100% 0 0)` | `#FFFFFF`<br>`oklch(100% 0 0)` | `#FFFFFF`<br>`oklch(100% 0 0)` |
| `--accent-soft` | `#331525`<br>`oklch(24.65% 0.0537 348.54)` | `#FFFFFF`<br>`oklch(100% 0 0)` | `#381E11`<br>`oklch(25.00% 0.0600 45.00)` | `#FFF7ED`<br>`oklch(97.00% 0.0200 45.00)` |
| `--accent-soft-foreground` | `#FF8AC4`<br>`oklch(77.75% 0.1553 350.26)` | `#BE185D`<br>`oklch(52.46% 0.1990 3.96)` | `#FDBA74`<br>`oklch(80.00% 0.1500 45.00)` | `#C2410C`<br>`oklch(55.00% 0.2000 40.00)` |
| `--background` | `#121114`<br>`oklch(18.02% 0.0063 300.93)` | `#FFFFFF`<br>`oklch(100% 0 0)` | `#0F0F0F`<br>`oklch(14.50% 0 0)` | `#FFFFFF`<br>`oklch(100% 0 0)` |
| `--surface` | `#1C1A20`<br>`oklch(22.27% 0.0119 300.63)` | `#FFF7FB`<br>`oklch(98.34% 0.0100 345.41)` | `#171717`<br>`oklch(18.50% 0 0)` | `#F8FAFC`<br>`oklch(98.50% 0.0050 260.00)` |
| `--surface-secondary` | `#17151A`<br>`oklch(20.03% 0.0103 303.61)` | `#F8DDE9`<br>`oklch(92.30% 0.0335 349.09)` | `#141414`<br>`oklch(16.50% 0 0)` | `#F1F5F9`<br>`oklch(96.00% 0.0100 260.00)` |
| `--surface-tertiary` | `#141317`<br>`oklch(18.97% 0.0081 297.06)` | `#F8DDE9`<br>`oklch(92.30% 0.0335 349.09)` | `#111111`<br>`oklch(15.00% 0 0)` | `#E2E8F0`<br>`oklch(91.00% 0.0200 260.00)` |
| `--overlay` | `#201E24`<br>`oklch(23.98% 0.0117 300.71)` | `#FFF7FB`<br>`oklch(98.34% 0.0100 345.41)` | `#1C1C1C`<br>`oklch(21.00% 0 0)` | `#F8FAFC`<br>`oklch(98.50% 0.0050 260.00)` |
| `--foreground` | `#F5F3F7`<br>`oklch(96.68% 0.0057 308.40)` | `#2A0F1E`<br>`oklch(21.67% 0.0502 347.62)` | `#F5F5F5`<br>`oklch(96.50% 0 0)` | `#0F172A`<br>`oklch(18.00% 0.0300 260.00)` |
| `--muted` | `#9E98A6`<br>`oklch(68.92% 0.0213 305.13)` | `#6E5262`<br>`oklch(47.33% 0.0441 343.67)` | `#949494`<br>`oklch(65.00% 0 0)` | `#64748B`<br>`oklch(50.00% 0.0200 260.00)` |
| `--border` | `#2C2932`<br>`oklch(28.79% 0.0167 300.56)` | `#EFC3D6`<br>`oklch(86.09% 0.0556 350.54)` | `#292929`<br>`oklch(26.00% 0 0)` | `#E2E8F0`<br>`oklch(90.00% 0.0100 260.00)` |
| `--focus` | `#A78BFA`<br>`oklch(70.90% 0.1592 293.54)` | `#7C3AED`<br>`oklch(54.13% 0.2466 293.01)` | `#F97316`<br>`oklch(69.11% 0.1944 44.01)` | `#EA580C`<br>`oklch(62.00% 0.2000 40.00)` |
| `--link` | `#7DB4FF`<br>`oklch(76.21% 0.1231 256.39)` | `#2563EB`<br>`oklch(54.61% 0.2152 262.88)` | `#FB923C`<br>`oklch(72.00% 0.1600 45.00)` | `#C2410C`<br>`oklch(55.00% 0.2000 40.00)` |

---

## 3. DOM Attributes & Binding

Theme selection is represented on the root `<html>` element using standard attributes:

1. **`class` and `data-theme`**:
   - Holds `"dark"` or `"light"`, resolving system preference when appearance is `"system"`.
   - (D4, revised 2026-09-23) While `data-theme-type="custom_colors"` and custom colors are set, this resolves from the custom surface color's brightness instead of `appearanceMode` — see §4a.
2. **`data-theme-preset`**:
   - Holds `"pink"` (default) or `"legacy"`.
3. **`data-theme-type`**:
   - Holds the base mode: `"preset"` or `"custom_colors"`. (Custom CSS is an overlay, not a base mode.)
4. **`data-custom-css`**:
   - Holds `"on"` when the custom CSS overlay is enabled and injected; `"off"` otherwise (including safe mode). Exists to make overlay state inspectable in tests and devtools.

### Example: Legacy Theme in Dark Mode with Custom CSS Overlay Active

```html
<html lang="en" class="dark" data-theme="dark" data-theme-preset="legacy" data-theme-type="preset" data-custom-css="on">
```

---

## 4. Custom Colors Dynamic Injection

When `data-theme-type="custom_colors"`, dynamic tokens are derived from user-selected hex values and mounted via a dedicated style element in `<head>`:

```html
<style id="gameplane-custom-theme-vars">
  html[data-theme-type="custom_colors"][data-theme] {
    --accent: oklch(62.5% 0.22 250);
    --accent-foreground: #FFFFFF;
    --accent-soft: oklch(25% 0.05 250);
    --accent-soft-foreground: oklch(80% 0.12 250);
    --focus: oklch(62.5% 0.22 250);
    --link: oklch(68% 0.18 250);
  }
</style>
```

The selector is scoped to the custom_colors root and carries specificity (0,2,1), so it outranks the preset token blocks (e.g. `.dark[data-theme-preset="legacy"]`, (0,2,0)) regardless of stylesheet order (revised 2026-09-23).

---

## 4a. Mode Resolution for Custom Colors (D4, 2026-09-23)

While `data-theme-type="custom_colors"` and custom colors are set, `class`/`data-theme` is derived from the surface color's own relative luminance instead of `appearanceMode`:
- `surfaceAppearance(surface)` (`web/src/lib/theme-derivation.ts`) returns `"light"` when `relativeLuminance(surface) > 0.179`, else `"dark"`.
- `useThemePreferences.ts` (`applyThemePreferences` / `resolveAppearanceMode`) applies this after every draft change, including live preview on `/settings/theme`.
- `index.html`'s boot script reads the pre-computed value from `localStorage["gameplane-theme-custom-mode"]` (written by `writeThemePreferences` alongside `gameplane-theme-vars-css`), validated to be exactly `"light"` or `"dark"` before use, falling back to `"dark"` otherwise.
- `appearanceMode` itself is never mutated by this — it stays stored and reapplies unchanged the moment `themeType` returns to `"preset"`.
- Never applies on `/login` or `/share/:token` (FR-011) — those routes force the Pink preset and never read the custom-colors path at all.

---

## 5. Custom CSS Overlay Injection Rules

1. **Gating**: Mounted only when the user is authenticated, `customCssEnabled` is true, and safe mode is not active. Unauthenticated surfaces (`/login`, `/share/:token`) never mount it (FR-008, FR-011).
2. **Cascade ordering (user priority)**: Mounted as `<style id="gameplane-custom-css">` as the **last** child of `document.head`, after the preset token declarations and after `<style id="gameplane-custom-theme-vars">`. At equal specificity, user rules win over every base layer; elements not targeted by custom CSS follow the active base theme.
3. **Base-switch behavior**: Switching `data-theme-preset` or `data-theme-type` while the overlay is enabled leaves `#gameplane-custom-css` mounted and last, so customized elements keep their custom appearance across base theme changes.
4. **Safe mode**: Not mounted when the URL contains `?safe-mode=1`, when the safe-mode keyboard shortcut was used, or when the session arrived via the safe-mode login link. `data-custom-css` reads `"off"` and the Safe Mode banner (contracts/theme-ui.md §4) is shown. The stored stylesheet is untouched.
5. **Sanitization**: Content must have passed FR-013 sanitization (no `@import`, no external `url()` references, `data:` URIs allowed, valid syntax, max 32,768 chars, no `<style`/`<script` delimiters) — enforced server-side at save time and re-checked client-side before mounting cached CSS at boot.
6. **Disable vs reset**: Toggling the overlay off unmounts the element but retains the stored stylesheet; only the FR-012 reset deletes it.
7. Strictly unmounted on logout or navigation to `/login` and `/share/:token`.
