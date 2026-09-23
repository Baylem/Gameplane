# Feature Specification: User Theme Customization

**Feature Branch**: `016-user-theme-customization`

**Created**: 2026-09-19

**Status**: Ready for Planning

**Input**: User description: "currently the app only has one theme the new pink one. i want different theme to be available for the user with 2 default option the legacy color scheme (the orange and dark one), and the current pink one. new user will default to the pink one, and existing user be migrated to the legacy one. the other option is custom css letting user specify a custom css for the app, and a simple custom color scheme"

## Context

The Gameplane dashboard recently transitioned to the modern HeroUI design system using a vibrant brand palette featuring a modern pink primary accent with deep neutral/violet dark surfaces and crisp light surfaces. Prior to this redesign, Gameplane shipped with an orange and dark color scheme (legacy theme).

Currently, all users interact exclusively with the modern pink theme, and the interface provides only a light/dark/system mode toggle without theme customization.

This feature introduces comprehensive theme customization for users:
1. **Default Presets**: Two curated out-of-the-box presets:
   - **Pink Theme (Default / Modern)**: The current modern HeroUI theme featuring pink primary accents.
   - **Legacy Theme (Orange & Dark)**: The original Gameplane theme featuring deep dark surfaces and warm orange accents.
2. **User Migration & Default Policy**:
   - Newly created users and new browser visitors automatically default to the modern Pink theme.
   - Existing users are seamlessly migrated in the database to retain their familiar Legacy theme.
3. **Simple Custom Color Scheme**:
   - A guided, code-free color picker allowing users to personalize key colors (primary accent color and background surface tone) to tailor their dashboard experience, with automatic contrast calculation.
4. **Custom CSS Overrides**:
   - An advanced custom styling option enabling users to inject custom CSS declarations directly into their personal application interface, paired with safety protections to prevent unusable UI states.

Export and import of a user's complete theme configuration (as portable JSON text) is included so users can move their setup between Gameplane instances. Theme sharing galleries, admin-enforced default themes, and per-server or per-page themes are explicitly out of scope for this feature.

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Select and Switch Between Default Preset Themes (Priority: P1)

An operator signs in to the dashboard and wants to choose their visual appearance. In the appearance settings, they see two default preset options: "Pink" (Modern) and "Legacy" (Orange & Dark). Selecting a preset immediately switches the interface's colors and chrome without refreshing the page.

**Why this priority**: Preset switching is the foundation of multi-theme support. It satisfies the core user requirement to access both the new pink branding and the classic orange and dark branding.

**Independent Test**: Can be fully tested by opening the appearance menu, selecting "Legacy", observing the UI transition to orange highlights and classic dark surfaces, then selecting "Pink" and confirming the UI transitions back to modern pink accents and tones.

**Acceptance Scenarios**:

1. **Given** a user viewing the dashboard in the Pink theme, **When** they select the "Legacy" preset in appearance settings, **Then** all primary accent elements (buttons, active navigation, focus rings, status highlights) change to the legacy orange tone, and surface backgrounds shift to the legacy dark/neutral palette.
2. **Given** a user viewing the dashboard in the Legacy theme, **When** they select the "Pink" preset in appearance settings, **Then** all accent elements change to the modern pink tone, and surfaces shift to the modern palette.
3. **Given** either preset theme is active, **When** the user toggles between Light and Dark appearance mode, **Then** the active theme renders with its respective light or dark palette variants while maintaining its brand accents (pink for Modern, orange for Legacy).
4. **Given** a user has changed their theme preset, **When** they navigate to different pages or reload the application, **Then** the chosen theme remains active without flashing or resetting.

---

### User Story 2 - User Theme Migration and Preference Persistence (Priority: P1)

Existing users returning to Gameplane are automatically migrated to the Legacy theme so their accustomed workflow is uninterrupted. New users joining Gameplane receive the modern Pink theme by default. The selected theme persists in the user's database profile across all browsers and devices.

**Why this priority**: Preserves continuity and expectations for existing users while ensuring all new users adopt the modern brand default across all touchpoints.

**Independent Test**: Can be tested by verifying that an existing user profile loads with the Legacy theme on initial login, a freshly created user profile loads with the Pink theme on initial login, and subsequent changes persist across sessions and multiple devices.

**Acceptance Scenarios**:

1. **Given** an existing user account created prior to this feature release, **When** the user logs in for the first time following release, **Then** their default theme is automatically set to the Legacy (Orange & Dark) theme via database migration.
2. **Given** a new user account registered after this feature release, **When** the user signs in for the first time, **Then** their default theme is set to the modern Pink theme.
3. **Given** a user updates their theme preference, **When** they log in on another browser or device, **Then** their saved theme preferences are fetched from their user profile and applied consistently.
4. **Given** an unauthenticated visitor viewing the login screen or a public server share link, **When** the page loads, **Then** the interface displays the standard modern Pink default theme, and user-defined custom CSS is strictly excluded.

---

### User Story 3 - Personalize Dashboard with a Simple Custom Color Scheme (Priority: P2)

An operator wants a personalized visual theme that matches their personal preference or organizational colors without writing code. They open the Theme Settings, select "Custom Color Scheme", and configure key color choices (primary accent color and background surface tone). The dashboard updates in real time to reflect these custom colors with automated contrast calculations.

**Why this priority**: Provides accessible, structured personalization for users who want distinct colors beyond the two default presets without having to write or maintain raw CSS.

**Independent Test**: Can be fully tested by selecting custom colors via the appearance controls, confirming the custom palette renders across navigation, buttons, and panels, and verifying that the user can revert to a preset with one click.

**Acceptance Scenarios**:

1. **Given** the user is in appearance settings, **When** they choose "Custom Color Scheme" and pick a custom accent color and surface tone, **Then** interactive components (primary action buttons, active navigation markers, badges, links) immediately adopt the chosen accent color, and text contrast is automatically calculated for legibility.
2. **Given** a user has a custom color scheme configured, **When** they switch between light and dark modes, **Then** the interface calculates appropriate contrast to ensure legibility and accessibility.
3. **Given** a custom color scheme is applied, **When** the user clicks "Reset to Default Preset", **Then** the custom colors are cleared and the selected preset (Pink or Legacy) is restored.

---

### User Story 4 - Apply and Manage Custom CSS (Priority: P3)

A power user or system administrator wants granular control over fonts, layout spacing, or specific component treatments. They navigate to Theme Settings, scroll to the Custom CSS section, input their custom CSS rules into an editor, and save. The custom stylesheet is saved to their profile and applied to their personal session.

**Why this priority**: Delivers maximum flexibility for advanced users and power operators while remaining optional and decoupled from the basic preset functionality.

**Independent Test**: Can be tested by entering a custom CSS rule (e.g. modifying a font-family or header background), saving, observing the style applied, and activating the safe-mode recovery mechanism to verify that broken CSS can be safely cleared.

**Acceptance Scenarios**:

1. **Given** the Custom CSS editor, **When** the user inputs valid CSS rules and saves, **Then** the rules are saved to their user profile, injected into the application, and take immediate effect.
2. **Given** a user enters invalid or syntax-broken CSS, **When** they save, **Then** the application does not crash, and a syntax warning or indicator informs the user.
3. **Given** custom CSS that accidentally hides critical user interface elements (such as navigation or the settings button), **When** the user enters safe mode via the URL query parameter, the keyboard shortcut, or the safe-mode sign-in link on the login page, **Then** custom CSS injection is temporarily suspended, allowing the user to edit or clear the broken stylesheet.
4. **Given** custom CSS configured by User A, **When** User B logs in on the same machine or another machine, **Then** User A's custom CSS is NOT applied to User B.
5. **Given** a user with a configured theme (preset, custom colors, and custom CSS), **When** they export their theme configuration and import it on another Gameplane instance, **Then** their full theme setup is reproduced on that instance, with the imported CSS sanitized per FR-013.

---

### Edge Cases

- **Broken or Malicious Custom CSS**: A user inputs CSS that makes text invisible (e.g. black text on black background), hides the settings page's action buttons, or includes disruptive animations. The system must provide an accessible escape hatch / safe recovery mechanism — a URL query parameter as the guaranteed path, a keyboard shortcut as a convenience, and a safe-mode sign-in link on the login page (FR-009) — to disable custom CSS and revert to defaults.
- **Remote Content in Custom CSS**: CSS referencing external resources (`@import`, external `url()`, remote fonts) is rejected at save time with a validation message identifying the offending rule; users may instead paste third-party CSS content inline within the 32 KB cap (FR-013) if they see fit.
- **Base Theme Switch with Active Custom CSS**: When a user switches the base theme (preset or custom colors) while the custom CSS overlay is enabled, elements not targeted by custom CSS adopt the new base styles immediately, while customized elements retain their custom appearance because the overlay remains last in the cascade.
- **Extreme Contrast / Incompatible Custom Colors**: A user picks an accent color identical to the background surface. The system should provide accessible contrast indicators or minimum contrast guards to prevent illegible buttons or text.
- **Unauthenticated Surfaces (Login and Share Links)**: Unauthenticated pages must remain consistent, predictable, and secure. Unauthenticated pages always use the modern Pink theme preset, and custom CSS from previously logged-in sessions must never execute on unauthenticated pages to prevent security spoofing or clickjacking risks. The login page may offer a safe-mode sign-in link (FR-009): it renders in the standard Pink theme like the rest of the page and carries no custom styling — it only passes the safe-mode flag into the authenticated session.
- **First Visit by Returning Users with Cleared Cache**: Because existing accounts are migrated directly in the user profile database, an existing user visiting from a fresh browser or with cleared cache still receives their migrated Legacy theme upon authentication.
- **Offline / Transient Connectivity**: If a user updates their theme while offline or during intermittent network connectivity, local client state updates immediately and synchronizes with the backend profile once connectivity resumes.
- **High Contrast / Accessibility Modes**: If the operating system or browser requests high contrast (`forced-colors` or `prefers-contrast`), system accessibility settings must take precedence over custom colors.

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST offer at least two predefined theme options: "Pink" (the modern HeroUI pink theme) and "Legacy" (the classic orange and dark theme).
- **FR-002**: System MUST automatically assign the "Pink" theme as the default for newly registered users and new visitor sessions.
- **FR-003**: System MUST migrate all pre-existing user accounts via a database migration so that their initial active theme in their user profile is the "Legacy" theme.
- **FR-004**: Users MUST be able to switch between available theme options at any time through the application interface.
- **FR-005**: System MUST persist the user's selected theme (preset choice, custom colors, and custom CSS) in their backend user profile in the database, syncing preferences across devices, with client-side caching for instant application boot.
- **FR-006**: System MUST provide a "Simple Custom Color Scheme" feature allowing users to customize their Primary Accent color and Background Surface tone without writing code, with automatic text contrast and border calculation.
- **FR-007**: System MUST provide a "Custom CSS" configuration allowing users to input and apply custom stylesheet rules to their dashboard interface. Custom CSS is an independent overlay layered on the active base theme (preset or custom colors), injected last in the cascade so user rules take precedence at equal specificity.
- **FR-008**: System MUST isolate custom CSS strictly to the user who configured it; custom styling MUST NOT affect other users or unauthenticated public screens.
- **FR-009**: System MUST provide a failsafe or safe-mode recovery mechanism allowing users to disable or reset broken custom CSS if interface usability is impaired, via three entry points: a URL query parameter (e.g. `?safe-mode=1`) as the guaranteed path, a keyboard shortcut as a convenience, and a safe-mode sign-in link on the login page. Safe mode suspends the custom CSS overlay for the session without deleting the stored stylesheet.
- **FR-010**: System MUST support Light, Dark, and System appearance modes in conjunction with theme presets.
- **FR-011**: Unauthenticated surfaces (including the Login page and public Server Share links) MUST always render with the modern Pink theme preset and MUST NOT execute user-defined custom CSS under any circumstances.
- **FR-012**: System MUST allow users to reset any custom color scheme or custom CSS back to a default preset with a single confirmation action. Switching presets or disabling the custom CSS overlay MUST NOT discard stored custom colors or CSS; only this explicit reset action deletes them.
- **FR-013**: Custom CSS MUST be sanitized before it is saved and injected: `@import` rules and external `url()` references (including remote fonts) are rejected, inline `data:` URIs are permitted, syntax is validated, and the stylesheet size is capped at 32 KB.
- **FR-014**: System MUST allow users to export their complete theme configuration (preset selection, appearance mode, custom colors, and custom CSS) as portable JSON text, and to import such an export on another Gameplane instance; imported custom CSS MUST pass the same sanitization as FR-013 before it is stored or applied.

### Key Entities

- **Theme Configuration**: The user's active styling preferences stored in their user profile, including:
  - `themeType`: The active base mode (`preset` | `custom_colors`).
  - `presetId`: The selected preset identifier (`pink` | `legacy`).
  - `appearanceMode`: The light/dark/system mode selection (`light` | `dark` | `system`).
  - `customColors`: Structured color settings (`primaryAccent`, `surfaceTone`) used when `themeType` is `custom_colors`. Retained (inactive) when the user switches back to a preset; deleted only via the FR-012 reset.
  - `customCssEnabled`: Whether the custom CSS overlay layer is active on top of the base theme.
  - `customCss`: Plain text containing user-specified CSS rules applied as an overlay on the active base theme when `customCssEnabled` is true. Sanitized per FR-013: no `@import` or external `url()` references, maximum size 32 KB. Retained when the overlay is disabled; deleted only via the FR-012 reset.
- **Theme Preset**: A named collection of semantic visual tokens defining colors for background, surface, foreground text, primary accent, borders, and status indicators.
- **User Migration Record**: Database schema migration flag or default assignment establishing the Legacy theme for existing users created prior to the migration timestamp.
- **Theme Export**: A portable JSON representation of a user's Theme Configuration used for manual transfer between Gameplane instances; imported content is validated and sanitized (FR-013) before storage.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of pre-existing user accounts in the database are migrated to the Legacy (Orange & Dark) theme upon running the release migration.
- **SC-002**: 100% of newly created user accounts default to the Pink theme upon account creation.
- **SC-003**: Switching between theme presets in the interface updates all visible colors across the dashboard within 100 milliseconds without requiring a full page refresh.
- **SC-004**: Users can customize their accent color and surface tone and observe live updates across primary controls within 3 clicks from any page.
- **SC-005**: 100% of users experiencing unreadable custom CSS can restore a functional default theme within 2 actions using the safe recovery mechanism.
- **SC-006**: Core dashboard workflows (server management, log viewing, module browsing, settings management) maintain 100% functional parity and legibility across both Pink and Legacy presets.
- **SC-007**: Unauthenticated public pages (login and share links) display zero visual distortion or injected custom code across all test cases.

---

## Clarifications

### Session 2026-09-19

- **Q1 (Persistence & Migration Scope)**: Where should user theme preferences and custom CSS/colors be persisted, and how should existing vs new users be distinguished?
  - **Decision**: Option A (Backend User Profile & Database). Theme settings, custom colors, and custom CSS are stored in the user database profile and synced across all devices/browsers. A database migration flags and migrates all pre-existing user accounts to `legacy`, while newly created accounts default to `pink`.
- **Q2 (Simple Custom Color Scheme)**: Which color attributes should be customizable in the simple custom color scheme editor?
  - **Decision**: Option A (Primary Accent + Background Surface). Users choose their primary accent color (for buttons, highlights, focus rings) and their preferred base dark/light surface tone; the system automatically calculates accessible text contrast and neutral borders.
- **Q3 (Unauthenticated Public Pages)**: What default theme should unauthenticated public pages (login screen and public share links) display?
  - **Decision**: Option A (Modern Pink default, strict custom CSS exclusion). Unauthenticated pages always display the modern Pink theme and never inject any custom CSS, ensuring consistent public branding and protection against CSS injection or spoofing.

### Session 2026-09-21

- Q: What safety restrictions should apply to user-supplied custom CSS before it is saved and injected? → A: Sanitized before save/injection (modified Option B): `@import` and external `url()` references (including remote fonts) are rejected, inline `data:` URIs are permitted, syntax is validated, and the stylesheet is capped at 32 KB so users can manually paste in content they would otherwise import.
- Q: Should custom CSS be a standalone third mode that replaces the other theme options, or an overlay that layers on top of the active base theme? → A: Layered overlay with user priority (modified Option A): custom CSS is an independent on/off layer stacked on the active base (Pink/Legacy preset or custom colors), injected last in the cascade so user rules take precedence — elements not targeted by custom CSS render per the base theme, while customized elements keep their custom color, rounding, positioning, etc., even when the base theme changes.
- Q: Which concrete mechanism should trigger the safe mode that disables broken custom CSS? → A: Option D plus login-page entry: a URL query parameter (e.g. `?safe-mode=1`) as the guaranteed recovery path, a keyboard shortcut as a convenience, and a safe-mode sign-in link on the login page — a guaranteed-clean surface because custom CSS never executes there (FR-011) — that starts the authenticated session with the custom CSS overlay suspended.
- Q: When a user switches away from a custom setup — changes the base preset, or toggles the custom CSS overlay off — should their custom colors and CSS text be kept in their profile or discarded? → A: Retain until explicit reset (Option A): switching presets or disabling the custom CSS overlay keeps `customColors`/`customCss` stored but inactive; only the explicit "Reset to Default Preset" action (FR-012) deletes them.
- Q: Which of these commonly associated capabilities should be explicitly declared out of scope for this feature? → A: Include export/import only (Option B): users can export their complete theme configuration (preset, custom colors, custom CSS) as portable JSON text and import it on another Gameplane instance; theme sharing galleries, admin-enforced default themes, and per-server or per-page themes are out of scope.

### Session 2026-09-22

- Q: Since custom colors have no on/off toggle, how are they activated and deactivated? → A: A third "Custom colors" radio card in the Preset theme card: selecting it sets `themeType` to `custom_colors`; selecting Modern Pink or Legacy Orange sets `themeType` back to `preset`. The Custom colors card's controls are disabled (stored values visible but greyed) unless the Custom colors radio is selected. Stored customs are retained across switches (FR-012).
- Q: Is the theme UI a modal or a page? → A (operator design decision): a full settings page at `/settings/theme` — app shell + settings navigation column (with a new "Theme" entry after "General") + stacked section cards (Preset theme / Appearance mode / Custom colors / Custom CSS / Export-Import) — replacing the originally specified `ThemeSettingsModal`.

### Session 2026-09-23

- Q: The Theme Settings page (`/settings/theme`) diverges visually from `/admin` (extra in-page breadcrumb row, misaligned header, narrower content column, sidebar "Settings" item not highlighted). What is the fix? → A (D1, operator decision): `/settings/theme` must match the `/admin` settings shell exactly — no in-page breadcrumb row (the app-shell TopBar breadcrumb alone reads "gameplane › Settings", collapsing the route's two path segments into the same single "Settings" crumb `/admin` produces), header and settings nav at the same vertical position as `/admin`, content column full width (no `max-w-3xl` constraint), and the sidebar "Settings" nav item highlighted while on `/settings/theme`. Section titles/subtitles ("Theme & Appearance") are unchanged.
- Q: Custom Colors only offers 7 preset accent swatches and 4 preset surface tones — how does a user pick an arbitrary color? → A (D2, operator decision): both Primary accent and Surface tone gain a free color choice — a native color-picker input plus a `#RRGGBB` text field — kept alongside (not replacing) the existing quick-pick swatches/radio tones. The contrast guard (WCAG AA) and the "disabled unless Custom colors is selected" activation rule (§3.2, Session 2026-09-22) apply identically to freely-chosen colors. No backend change: `custom_accent`/`custom_surface` were already validated only by `^#[0-9a-fA-F]{6}$` (data-model.md §2.3), not restricted to the UI's quick-pick list.
- Q: The login response omits `preferences`, so a returning user briefly sees the Pink default until the next `/users/me` refetch (up to the query's staleTime). Is this acceptable? → A (D3, operator decision): No — `POST /auth/login`'s response must embed `preferences` in the same shape GET `/users/me` returns, so the dashboard seeds the correct theme from the login response itself with no flash and no extra round-trip. The OIDC callback flow is unaffected (it redirects to a fresh page load, which always fetches `/users/me`).
- Q: While Custom colors is active, should Appearance mode (light/dark) still be user-selectable? → A (D4, operator decision): No — while `themeType` is `custom_colors` and `customColors` is set, light/dark is derived from the surface color's brightness (relative luminance vs. the WCAG ≈0.179 threshold: closer to white → light, closer to black → dark) instead of `appearanceMode`. The Appearance mode control on `/settings/theme` is disabled with the note *"Set by your surface color"* while Custom colors is selected; the stored `appearanceMode` is left untouched and reapplies once a preset is selected again. The sidebar footer appearance toggle receives the same disabled treatment. Reason: custom-color tokens only override 14 variables (`theme-derivation.ts` `CUSTOM_THEME_TOKEN_NAMES`) — everything else comes from the preset's `.light`/`.dark` block, so forcing a mode that fights the surface produces illegible black-on-black or light-on-light UI.

---

## Assumptions

- The modern Pink theme uses the HeroUI semantic tokens approved in `contracts/theme-tokens.md` (`#FF4FA3` dark / `#DB2777` light).
- The Legacy theme uses the original Gameplane palette (`#F97316` orange primary with dark neutral surfaces).
- Custom CSS is parsed and injected in the client browser during runtime and does not require server-side stylesheet compilation.
- The existing light / dark / system appearance toggle remains operational, providing light and dark variations of the chosen theme where supported.
- Visual surfaces and theme selection UI will be designed following Constitution Principle II before frontend implementation.
