# Contract: Theme Customization UI

**Feature**: `016-user-theme-customization`  
**Binding Modules**: `web/src/routes/ThemeSettings.tsx`, `web/src/components/ui/TopBar.tsx`, `web/src/components/ui/Sidebar.tsx`, `web/src/components/ui/SafeModeBanner.tsx`, `web/src/routes/Login.tsx`  
**Status**: Binding (revised 2026-09-22: ThemeSettingsModal replaced by a full settings page at route `/settings/theme`, matching the other `Screen/* Settings` designs; revised 2026-09-21 after clarification session 2026-09-21); revised 2026-09-23 (D1 settings-shell parity, D2 free color choice, D3 login preferences) after clarification session 2026-09-23  

---

## 1. Triggers & Navigation

### 1.1 TopBar User Menu

The user avatar dropdown in `TopBar.tsx` gains a dedicated item:
- **Label**: `"Theme & Appearance"`
- **Icon**: `Palette` (from `lucide-react`)
- **Action**: Navigates to the theme settings page (`/settings/theme`)

```text
+------------------------------+
| [Avatar] Alex Rivera         |
| operator                     |
|------------------------------|
| [Palette] Theme & Appearance |
| [LogOut]  Sign out           |
+------------------------------+
```

### 1.2 Sidebar Footer Appearance Section

The `Sidebar.tsx` footer appearance row is enhanced:
- Retains the quick light/dark/system mode toggle (`AppearanceToggle`).
- Adds a small settings icon button (`Palette` or `Sliders`) with `aria-label="Customize theme"` that also navigates to `/settings/theme`.

### 1.3 Login Page Safe-Mode Link

`web/src/routes/Login.tsx` gains a secondary link below the sign-in form:
- **Label**: `"Sign in with safe mode (custom styling disabled)"`
- **Behavior**: Submits the same credentials flow but carries the safe-mode flag into the authenticated session (equivalent to landing with `?safe-mode=1`).
- **Rendering**: The login page always renders in the modern Pink preset with no custom CSS (FR-011); the link itself is standard pink-themed chrome and carries no custom styling — it is only an entry point.

---

## 2. Theme Settings Page Structure

The theme settings UI is a full settings page at route `/settings/theme`, rendered inside the standard app shell (App Sidebar + Top Bar + Page Header) exactly like the other `Screen/* Settings` designs (e.g. `Screen/Admin Settings`). Page title: `"Theme & Appearance"`. Layout details (revised 2026-09-23, D1):
- No in-page breadcrumb row; the TopBar breadcrumb alone reads `"gameplane › Settings"` (collapsing both route segments like `/admin` does).
- Content column full width with no constrained max-width.
- Sidebar "Settings" nav item highlighted while on `/settings/theme`.

Composed entirely from HeroUI primitives (`Button`, `RadioGroup`, `Radio`, `Input`, `Textarea`, `Switch`, `Alert`, `Link`) inside the standard page layout. All sections live on **one scrollable page as stacked cards** (no sub-navigation, no tabs) — the simple-page convention used by screens like `Screen/Backups — Index`. Section order: **Preset theme**, **Appearance mode**, **Custom colors**, **Custom CSS**, **Export / Import**. Each section is a standard bordered settings card (title + subtitle + fields). The actions row sits at the bottom of the page: destructive **Reset to Defaults** on the left, primary **Save** on the right (changes apply live for preview; **Save** persists to the server).

```text
+-------------------------------------------------------------+
| gameplane > Settings                                        |
| Theme & Appearance                                          |
+-------------------------------------------------------------+
| +-- Preset theme ------------------------------------------+|
| | (o) Modern Pink [swatch]   ( ) Legacy Orange [swatch]    ||
| | "Your custom colors and CSS are kept ..."                ||
| +-----------------------------------------------------------+
| +-- Appearance mode ---------------------------------------+|
| | [ ( ) Light   (o) Dark   ( ) System ]                    ||
| +-----------------------------------------------------------+
| +-- Custom colors -----------------------------------------+|
| | Accent swatches (quick picks)                           ||
| | [■ picker]  [#RRGGBB]                                  ||
| | Contrast preview chip                                   ||
| | Surface Tone (quick picks)                              ||
| | [■ picker]  [#RRGGBB]                                  ||
| +-----------------------------------------------------------+
| +-- Custom CSS --------------------------------------------+|
| | [x] Enable custom CSS overlay   [textarea]  X / 32,768   ||
| +-----------------------------------------------------------+
| +-- Export / Import ---------------------------------------+|
| | [Copy] [Download]   paste area / file picker  [Apply]    ||
| +-----------------------------------------------------------+
| [Reset to Defaults]                                    [Save]|
+-------------------------------------------------------------+
```

---

## 3. Sections & Controls

### 3.1 Section 1: Presets (`preset`)

- **Preset theme** card shows three interactive radio cards:
  - **Modern Pink** (`presetId: "pink"`): Shows pink accent swatch (`#FF4FA3`) and dark preview swatch (`#1C1A20`).
  - **Legacy Orange** (`presetId: "legacy"`): Shows orange accent swatch (`#F97316`) and classic dark preview swatch (`#171717`).
  - **Custom colors** (`themeType: "custom_colors"`): Activates the user's stored custom colors (palette/spectrum swatch). Selecting it sets `themeType` to `custom_colors`; selecting either preset sets `themeType` back to `preset`.
- The **Appearance mode** selector (segmented control for Light / Dark / System) is its own card directly below Preset theme.
- Selecting a preset never discards stored custom colors or custom CSS (FR-012); a note says so: *"Your custom colors and CSS are kept and can be re-applied later."*

### 3.2 Section 2: Custom Colors (`custom_colors`)

- **Activation**: This card's controls are **disabled** until the "Custom colors" radio card in the Preset theme card is selected (`themeType: "custom_colors"`); selecting a preset disables them again. The stored values remain visible (greyed) while disabled.
- **Primary Accent**:
  - Palette swatches (Blue, Emerald, Purple, Amber, Cyan, Rose, Orange) as quick picks, plus a free color choice: a native color-picker input and a `#RRGGBB` text field, both bound to the same `customColors.accent` value (revised 2026-09-23, D2).
  - Preview chip showing button with accent color and computed contrast text.
- **Surface Tone**:
  - Radio cards (`"Dark Slate"`, `"Midnight"`, `"Charcoal"`, `"Crisp Light"`) as quick picks, plus a free color choice: a native color-picker input and a `#RRGGBB` text field, both bound to the same `customColors.surface` value (revised 2026-09-23, D2).
- Live preview: All dashboard elements underneath the page immediately show the updated colors.
- Contrast guard: warns when the chosen accent/surface pair fails WCAG AA.

### 3.3 Section 3: Custom CSS (overlay)

- **Overlay toggle**: a switch labeled `"Enable custom CSS overlay"` bound to `customCssEnabled`, with helper text: *"Your rules are applied on top of the active base theme and win over it; elements you don't target follow the base theme."*
- Textarea code input with monospace font (`font-mono`, `JetBrains Mono`).
- Placeholder text showing examples:
  ```css
  /* Example: customize typography or borders */
  :root {
    --radius: 14px;
  }
  ```
- Character count indicator (`X / 32,768`).
- Inline validation (mirroring server rules in `web/src/lib/theme-sanitize.ts`):
  - Error on `@import` or external `url()` references, naming the offending rule: *"External resource loads are not allowed (line 4: `@import ...`). Paste the content inline instead."*
  - Error on unparseable syntax; allowed `data:` URIs are not flagged.
- Warning alert: *"Custom CSS modifies application appearance directly. If the interface becomes unusable, recover via the `?safe-mode=1` URL parameter, the safe-mode keyboard shortcut, or the 'Sign in with safe mode' link on the login page."*
- Disabling the toggle keeps the saved stylesheet; only **Reset to Defaults** deletes it.

### 3.4 Section 4: Export / Import

- **Export**: buttons for `Copy to clipboard` and `Download gameplane-theme.json`, built from the saved profile state per `contracts/theme-export.md` (includes retained-but-inactive customs).
- **Import**: paste area + file picker accepting `gameplane-theme` v1 documents; shows a preview (preset name, accent swatch, CSS byte size) and an explicit `Apply import` confirmation. Validation and server `400` messages surface inline.
- Scope note: *"Sharing galleries and organization-enforced themes are not part of this feature."*

---

## 4. Safe Mode UI Banner

When safe mode is active — via the `?safe-mode=1` URL parameter, the safe-mode keyboard shortcut, or the login-page safe-mode link:
- A dismissible top alert banner (`SafeModeBanner.tsx`) appears:
  ```text
  [Alert Icon] Safe Mode Active: Custom CSS is suspended. 
               [Open Appearance Settings] | [Dismiss]
  ```
- The custom CSS overlay is not mounted (`data-custom-css="off"`); the base theme renders normally.
- The stored stylesheet is NOT deleted — the banner's *Open Appearance Settings* action navigates to `/settings/theme` so the user can edit or clear it.
- Safe mode applies for the session only; removing the parameter and reloading restores normal behavior.

---

## 5. Reset to Defaults

- **Placement**: Page actions row (at the bottom of the page), destructive-style button.
- **Confirmation**: Single confirmation dialog: *"This deletes your custom colors and custom CSS and restores the selected preset. Continue?"*
- **Effect**: Calls `POST /api/v1/users/me/preferences/reset` (contracts/user-preferences-api.md §1.3) — the only action that deletes stored customs.
