# Contract: Theme Customization UI

**Feature**: `016-user-theme-customization`  
**Binding Modules**: `web/src/components/ui/ThemeSettingsModal.tsx`, `web/src/components/ui/TopBar.tsx`, `web/src/components/ui/Sidebar.tsx`, `web/src/components/ui/SafeModeBanner.tsx`, `web/src/routes/Login.tsx`  
**Status**: Binding (revised 2026-09-21 after clarification session 2026-09-21)  

---

## 1. Triggers & Navigation

### 1.1 TopBar User Menu

The user avatar dropdown in `TopBar.tsx` gains a dedicated item:
- **Label**: `"Theme & Appearance"`
- **Icon**: `Palette` (from `lucide-react`)
- **Action**: Opens `ThemeSettingsModal`

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
- Adds a small settings icon button (`Palette` or `Sliders`) with `aria-label="Customize theme"` that also opens `ThemeSettingsModal`.

### 1.3 Login Page Safe-Mode Link

`web/src/routes/Login.tsx` gains a secondary link below the sign-in form:
- **Label**: `"Sign in with safe mode (custom styling disabled)"`
- **Behavior**: Submits the same credentials flow but carries the safe-mode flag into the authenticated session (equivalent to landing with `?safe-mode=1`).
- **Rendering**: The login page always renders in the modern Pink preset with no custom CSS (FR-011); the link itself is standard pink-themed chrome and carries no custom styling — it is only an entry point.

---

## 2. Theme Settings Modal Structure

Composed entirely from HeroUI primitives (`Modal`, `ModalHeader`, `ModalBody`, `ModalFooter`, `Tabs`, `Tab`, `Button`, `RadioGroup`, `Radio`, `Input`, `Textarea`, `Alert`):

```text
+-------------------------------------------------------------+
| Theme & Appearance                                      [X] |
+-------------------------------------------------------------+
| [ Presets ]  [ Custom Colors ]  [ Custom CSS ]  [ Export ]  |
|-------------------------------------------------------------|
| Choose a preset theme:                                      |
|                                                             |
| +-------------------------+     +-------------------------+ |
| | (o) Modern Pink         |     | ( ) Legacy Orange       | |
| | [Pink Swatch]           |     | [Orange Swatch]         | |
| | Modern HeroUI brand     |     | Original Gameplane      | |
| +-------------------------+     +-------------------------+ |
|                                                             |
| Appearance Mode:                                            |
| [ ( ) Light  (o) Dark  ( ) System ]                         |
+-------------------------------------------------------------+
| [Reset to Defaults]                       [Cancel]  [Save]  |
+-------------------------------------------------------------+
```

---

## 3. Tabs & Controls

### 3.1 Tab 1: Presets (`preset`)

- Shows two interactive radio cards:
  - **Modern Pink** (`presetId: "pink"`): Shows pink accent swatch (`#FF4FA3`) and dark preview swatch (`#1C1A20`).
  - **Legacy Orange** (`presetId: "legacy"`): Shows orange accent swatch (`#F97316`) and classic dark preview swatch (`#171717`).
- Appearance mode selector: Segmented control for Light / Dark / System.
- Selecting a preset never discards stored custom colors or custom CSS (FR-012); a note says so: *"Your custom colors and CSS are kept and can be re-applied later."*

### 3.2 Tab 2: Custom Colors (`custom_colors`)

- **Primary Accent**:
  - Color picker input or palette swatches (Blue, Emerald, Purple, Amber, Cyan, Rose, Orange).
  - Preview chip showing button with accent color and computed contrast text.
- **Surface Tone**:
  - Dropdown or Radio cards: `"Dark Slate"`, `"Midnight"`, `"Charcoal"`, `"Crisp Light"`.
- Live preview: All dashboard elements underneath the modal immediately show the updated colors.
- Contrast guard: warns when the chosen accent/surface pair fails WCAG AA.

### 3.3 Tab 3: Custom CSS (overlay)

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

### 3.4 Tab 4: Export / Import

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
- The stored stylesheet is NOT deleted — the banner's *Open Appearance Settings* action opens the modal so the user can edit or clear it.
- Safe mode applies for the session only; removing the parameter and reloading restores normal behavior.

---

## 5. Reset to Defaults

- **Placement**: Modal footer, destructive-style button.
- **Confirmation**: Single confirmation dialog: *"This deletes your custom colors and custom CSS and restores the selected preset. Continue?"*
- **Effect**: Calls `POST /api/v1/users/me/preferences/reset` (contracts/user-preferences-api.md §1.3) — the only action that deletes stored customs.
