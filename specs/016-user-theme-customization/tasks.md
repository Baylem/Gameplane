# Tasks: User Theme Customization

**Input**: Design documents from `/specs/016-user-theme-customization/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ (user-preferences-api.md, theme-tokens-v2.md, theme-ui.md, theme-export.md), quickstart.md

**Tests**: Test tasks ARE included — Constitution Principle I makes E2E coverage non-negotiable, and plan.md's Testing section explicitly lists unit and E2E test files.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US4, mapping to spec.md priorities P1/P1/P2/P3)
- All file paths are repo-relative and verified against the current codebase layout

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Baseline verification and design-tool prerequisites

- [X] T001 Verify clean baseline on branch `016-user-theme-customization`: `go build ./...` in api/ and `npx tsc --noEmit` in web/ pass before any changes (plan.md Technical Context; Constitution VI permits local compile checks only)
- [X] T002 [P] Confirm the Pencil MCP server is available and `design.pen` opens, since Constitution Principle II requires design-first work in design.pen before any frontend code (blocks T003)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Design source, backend persistence/API, and client theme plumbing that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T003 Design the Theme & Appearance surfaces in design.pen via the Pencil MCP server per contracts/theme-ui.md — the `/settings/theme` settings page (stacked cards: Preset theme / Appearance mode / Custom colors / Custom CSS / Export-Import, with the settings nav column carrying a new Theme entry after General), TopBar and Sidebar triggers, SafeModeBanner, and the login-page "Sign in with safe mode" link — then re-export every touched node to design-export/json/ and design-export/screenshots/ (Constitution II)
- [X] T004 [P] Create migration api/internal/db/migrations/011_user_theme_preferences.sql per data-model.md §3: user_preferences table (user_id PK/FK cascade, theme_type, preset_id, appearance_mode, custom_accent, custom_surface, custom_css_enabled, custom_css, updated_at) and INSERT ... SELECT seeding all pre-existing users with preset_id='legacy'
- [X] T005 [P] Implement the DB layer in api/internal/db/preferences.go: GetPreferences (returns pink/system defaults when no row exists), UpsertPreferences (retention per contracts/user-preferences-api.md §1.2 — ordinary updates never null custom_accent/custom_surface/custom_css), ResetPreferences (nulls all customs, disables overlay, restores requested preset)
- [X] T006 [P] Write DB unit tests in api/internal/db/preferences_test.go: migration 011 seeds existing users with 'legacy', new users default to 'pink', custom_css_enabled defaults to 0, retention holds across preset switches and overlay toggles, reset nulls customs
- [X] T007 Implement preferences handlers in api/internal/handlers/users.go per contracts/user-preferences-api.md: GET/PUT /api/v1/users/me/preferences, POST /api/v1/users/me/preferences/reset, extend GET /api/v1/users/me with the preferences object; enforce FR-013 server-side sanitization with rule-identifying 400 messages (@import rejected, external url() incl. protocol-relative rejected, data: URIs allowed, syntax validated, 32,768-char cap, no <style/<script delimiters) and FR-012 retention semantics; wrap errors with %w (Constitution III)
- [X] T008 [P] Write handler tests in api/internal/handlers/users_preferences_test.go: full validation matrix from quickstart.md §3.2 (valid payloads, hex format, sanitization 400s naming the offending rule, 32 KB cap, retention after preset switch/overlay toggle, reset behavior, 401 unauthenticated, and cross-user isolation — User B's GET never returns User A's values, FR-008)
- [X] T009 [P] Add UserThemePreferences (themeType base-mode-only, customCssEnabled, customCss retained-when-disabled) and ThemeExport types to web/src/types.ts per data-model.md §2.3–§2.4
- [X] T010 [P] Add Users.getPreferences, Users.updatePreferences, and Users.resetPreferences client methods to web/src/lib/endpoints.ts per contracts/user-preferences-api.md
- [X] T011 Update the boot script in web/index.html per research.md R-05: read gameplane-theme-prefs from localStorage, set data-theme-preset / data-theme-type / data-custom-css and dark/light class synchronously before first paint, inject cached #gameplane-custom-css as the LAST <head> child only when customCssEnabled and URL lacks safe-mode=1, and apply the boot-time unauthenticated guard — never apply cached prefs on /login or /share/:token (FR-011). Scope: boot-time guard only; route-component enforcement lands in T025/T026
- [X] T012 Implement the useThemePreferences hook in web/src/lib/useThemePreferences.ts per research.md R-05 §4: exposes current preferences, optimistic DOM + localStorage updates, PUT mutation via endpoints.ts, revert-and-toast on error; on a failed PUT due to lost connectivity, retry when the browser fires the 'online' event and re-reconcile localStorage and DOM once the backend confirms (spec Edge Case: Offline / Transient Connectivity) (depends on T009, T010)
- [X] T013 Integrate theme provider and profile reconciliation in web/src/components/AppLayout.tsx per research.md R-05 §3: on useMe() (web/src/lib/auth.ts) load, reconcile backend preferences with localStorage and re-apply DOM attributes; ensure the overlay is unmounted on logout and on /login and /share/:token (depends on T011, T012)

**Checkpoint**: Foundation ready — migration, API, types, boot hydration, and provider are in place; user story implementation can now begin

---

## Phase 3: User Story 1 - Select and Switch Between Default Preset Themes (Priority: P1) 🎯 MVP

**Goal**: Users can switch between the Modern Pink and Legacy Orange presets from the UI, instantly and without page reload, with the choice surviving reloads

**Independent Test**: Open the appearance menu, select "Legacy", observe orange accents and classic dark surfaces without reload; select "Pink" and confirm return; reload and confirm the choice persists without flashing (spec.md US1)

### Tests for User Story 1

- [X] T014 [P] [US1] Update token assertions in web/src/__tests__/theme.test.tsx: light mode --accent #DB2777 (Pink) vs #EA580C (Legacy), dark mode #FF4FA3 vs #F97316, surface/border spot-checks per contracts/theme-tokens-v2.md §2, plus assertions that the forced-colors / prefers-contrast guards (T017) let system accessibility settings override theme and custom tokens
- [X] T015 [P] [US1] Create Playwright live spec web/e2e/specs/live/theme-customization.spec.ts per quickstart.md §5.1/§5.2.2: preset switch applies without reload (SC-003 <100ms), persists across page reload, light/dark toggle keeps each preset's brand accent; follow web/e2e live conventions (unique resource names, parallel-safe per Constitution I)

### Implementation for User Story 1

- [X] T016 [P] [US1] Add the Legacy preset token mappings to web/src/styles/globals.css under .dark[data-theme-preset="legacy"] and .light[data-theme-preset="legacy"] exactly per contracts/theme-tokens-v2.md §2 (research.md R-02)
- [X] T017 [US1] Add @media (forced-colors: active) and @media (prefers-contrast: more) guards to web/src/styles/globals.css so operating-system accessibility settings take precedence over preset and custom-color tokens (spec Edge Case: High Contrast / Accessibility Modes) (depends on T016 — same file)
- [X] T018 [P] [US1] Create the ThemeSettings page shell and Preset theme + Appearance mode cards in web/src/routes/ThemeSettings.tsx per contracts/theme-ui.md §2/§3.1 and the T003 design (Screen/Theme Settings, design-export lWvcv): register the /settings/theme route in web/src/router/tree.tsx, render inside AppLayout with the settings nav column (Theme active), stacked cards (preset radio cards with swatches, Light/Dark/System segmented control, retention note), actions row with Reset to Defaults placeholder (wired in T040)
- [X] T019 [P] [US1] Add the "Theme & Appearance" item (Palette icon) to the user avatar dropdown in web/src/components/ui/TopBar.tsx per contracts/theme-ui.md §1.1
- [X] T020 [P] [US1] Add the customize-theme icon button (aria-label="Customize theme") to the appearance footer in web/src/components/ui/Sidebar.tsx per contracts/theme-ui.md §1.2
- [X] T021 [US1] Write ThemeSettings page unit tests in web/src/routes/ThemeSettings.test.tsx: renders the five section cards, clicking "Legacy" sets data-theme-preset="legacy", save calls PUT /users/me/preferences via useThemePreferences, switching presets does not clear stored custom values (depends on T018)

**Checkpoint**: User Story 1 fully functional — preset switching works end-to-end and is independently testable

---

## Phase 4: User Story 2 - User Theme Migration and Preference Persistence (Priority: P1)

**Goal**: Existing users are migrated to Legacy, new users default to Pink, preferences persist across browsers/devices, and unauthenticated surfaces always render Pink with zero custom CSS

**Independent Test**: A pre-migration account loads Legacy on first login; a fresh account loads Pink; a change made in one browser context appears in a fresh context; login and share pages always render Pink (spec.md US2)

### Tests for User Story 2

- [X] T022 [P] [US2] Update boot hydration tests in web/src/__tests__/themeBoot.test.ts: localStorage gameplane-theme-prefs initializes data-theme-preset / data-theme-type / data-custom-css before first paint (no FOUC), backend preferences reconcile over stale cache after useMe() loads, and no prefs are applied on /login or /share/:token
- [X] T023 [US2] Append migration and isolation scenarios to web/e2e/specs/live/theme-customization.spec.ts per quickstart.md §5.2.1/§5.2.8: seeded pre-migration user sees Legacy, newly registered user sees Pink, theme change propagates to a fresh browser context (multi-device), /login and a public /share/:token render Pink with #gameplane-custom-css absent (depends on T015 — same file)
- [X] T024 [P] [US2] Create Go API-contract E2E in test/e2e/api_theme_preferences_e2e_test.go and register it in a bucket in test/e2e/buckets.sh per Constitution I: through the real API verify migration 011 defaults (pre-existing user → legacy, new user → pink), GET/PUT /users/me/preferences round-trip, POST /users/me/preferences/reset semantics, FR-013 sanitization 400s, and FR-012 retention; must call t.Parallel() and use per-test unique resource names per test/e2e conventions

### Implementation for User Story 2

- [X] T025 [US2] Add route-component enforcement on the login page in web/src/routes/Login.tsx: forces data-theme-preset="pink" / data-custom-css="off" on mount and never reads the prefs cache — this complements the T011 boot-time guard (which only prevents cache application before paint) by actively pinning unauthenticated chrome to Pink for the route's lifetime (FR-011, contracts/theme-tokens-v2.md §5)
- [X] T026 [P] [US2] Add the same route-component enforcement on the public share page in web/src/routes/Share.tsx: forces Pink preset attributes on mount and guarantees no custom CSS injection for previously logged-in sessions (FR-011, spec.md Edge Cases)

**Checkpoint**: User Stories 1 AND 2 both work independently — migration defaults, sync, API contract, and unauthenticated isolation verified

---

## Phase 5: User Story 3 - Personalize Dashboard with a Simple Custom Color Scheme (Priority: P2)

**Goal**: Users can pick a primary accent color and background surface tone without writing code, with automatic accessible contrast, live preview, and one-click revert to a preset

**Independent Test**: Choose a custom accent + surface in the Custom colors card, confirm the palette renders across navigation/buttons/panels, choose a light and then a dark surface tone and confirm the page relights itself and stays legible (D4 — Appearance mode is disabled while Custom colors is active), then revert to a preset (spec.md US3)

### Tests for User Story 3

- [X] T027 [P] [US3] Write derivation unit tests in web/src/lib/theme-derivation.test.ts per research.md R-03: WCAG AA (>= 4.5:1) accent-foreground selection, surface layer offsets, border calculation, muted-text contrast, and extreme pairs (accent identical to surface triggers the contrast guard)
- [X] T028 [US3] Append the custom-colors flow to web/e2e/specs/live/theme-customization.spec.ts per quickstart.md §5.2.3: pick Emerald accent (#10B981) + surface, verify controls adopt it live (SC-004 within 3 clicks), switch light/dark and verify contrast recalculation, reset restores the preset (depends on T023 — same file) — WITHDRAWN 2026-09-23 (D4): "switch light/dark" step is obsolete — Appearance mode is disabled while Custom colors is active; see T028a.
- [ ] T028a [US3] (D4, 2026-09-23) Rewrite the "custom colors ... light/dark" scenario in web/e2e/specs/live/theme-customization.spec.ts per spec.md D4: assert the Appearance mode control is disabled with the "Set by your surface color" note while Custom colors is active, and swap the light/dark-toggle assertions for a dark-surface-then-light-surface tone swap, checking data-theme and --accent-foreground after each (needs human sign-off before editing this test file, per CLAUDE.md Rule 1).

### Implementation for User Story 3

- [X] T029 [US3] Implement deriveCustomThemeTokens in web/src/lib/theme-derivation.ts per research.md R-03: pure function computing accent-foreground, accent-soft, surface layers, borders, and muted text from accent + surface hex inputs
- [X] T030 [US3] Add the Custom colors card to web/src/routes/ThemeSettings.tsx per contracts/theme-ui.md §3.2: accent picker with swatches and contrast preview chip, surface tone selector, live preview applied via #gameplane-custom-theme-vars (before #gameplane-custom-css per contracts/theme-tokens-v2.md §5), WCAG contrast guard warning, themeType="custom_colors" persistence (depends on T018, T029)

**Checkpoint**: All P1/P2 stories functional — custom colors work as a base theme and revert cleanly

---

## Phase 6: User Story 4 - Apply and Manage Custom CSS (Priority: P3)

**Goal**: Power users get a sanitized custom CSS overlay that wins over the base theme and survives base switches, three safe-mode recovery entry points, per-user isolation, and portable theme export/import

**Independent Test**: Save a custom CSS rule and see it applied; verify it persists across a preset switch; break the UI with CSS and recover via ?safe-mode=1, the keyboard shortcut, and the login-page link; export the full configuration and import it on a second account (spec.md US4 incl. scenario 5)

### Tests for User Story 4

- [X] T031 [P] [US4] Write sanitization unit tests in web/src/lib/theme-sanitize.test.ts per quickstart.md §4.2: @import rejected with the offending rule named, external url() (https:// and protocol-relative //host) rejected, data: URIs accepted, 32,768-char cap enforced, unbalanced braces flagged, <style/<script delimiters rejected
- [X] T032 [P] [US4] Write export/import unit tests in web/src/lib/theme-export.test.ts per contracts/theme-export.md: export builds a valid gameplane-theme v1 document including retained-but-inactive customs; import validation rejects unparseable JSON, wrong format/version, missing fields, bad enums/hex, oversized CSS; valid documents map to the correct PUT payload
- [X] T033 [US4] Append overlay, safe-mode, and export scenarios to web/e2e/specs/live/theme-customization.spec.ts per quickstart.md §5.2.4–§5.2.7: overlay rule persists across Pink↔Legacy switches while untargeted elements follow the base (FR-007 cascade), stored CSS retained after overlay toggle off/on (FR-012), safe mode via ?safe-mode=1 and via the login-page link suspends the overlay and shows the banner, editor rejects @import with the rule named, export → import round-trip on a second account reproduces the full setup, and a hand-edited import containing @import is rejected server-side (depends on T028 — same file)

### Implementation for User Story 4

- [X] T034 [P] [US4] Implement the FR-013 client-side sanitizer in web/src/lib/theme-sanitize.ts mirroring the server rules in api/internal/handlers/users.go: detect @import, external url() (allow data:), syntax errors, size cap, and HTML delimiters, returning rule-identifying messages for the editor
- [X] T035 [P] [US4] Implement export/import logic in web/src/lib/theme-export.ts per contracts/theme-export.md: buildGameplaneThemeExport (from loaded preferences), parse/validate imports with specific error messages, map valid documents to the updatePreferences payload
- [X] T036 [US4] Add the Custom CSS card to web/src/routes/ThemeSettings.tsx per contracts/theme-ui.md §3.3: overlay enable toggle bound to customCssEnabled with helper text, monospace editor, "X / 32,768" character count, inline sanitization errors via theme-sanitize.ts, safe-mode warning alert, disable-keeps-stylesheet behavior (depends on T018, T034)
- [X] T037 [US4] Add the Export / Import card to web/src/routes/ThemeSettings.tsx per contracts/theme-ui.md §3.4: copy-to-clipboard and gameplane-theme.json download, paste/file import with preview (preset name, accent swatch, CSS byte size), explicit Apply import confirmation, inline surfacing of server 400 messages (depends on T035, T036 — same file)
- [X] T038 [P] [US4] Create web/src/components/ui/SafeModeBanner.tsx and wire safe-mode detection in web/src/components/AppLayout.tsx per contracts/theme-ui.md §4: banner with "Open Appearance Settings" and "Dismiss", activated by ?safe-mode=1 in the URL, the Ctrl+Shift+Alt+T keyboard shortcut, or a session that arrived via the safe-mode login link; sets data-custom-css="off" without touching the stored stylesheet
- [X] T039 [P] [US4] Add the "Sign in with safe mode (custom styling disabled)" link below the sign-in form in web/src/routes/Login.tsx per contracts/theme-ui.md §1.3: standard Pink chrome (FR-011), carries the safe-mode flag into the authenticated session after successful sign-in
- [X] T040 [US4] Wire the Reset to Defaults confirmation in web/src/routes/ThemeSettings.tsx per contracts/theme-ui.md §5: single confirmation dialog, then POST /api/v1/users/me/preferences/reset via Users.resetPreferences, restoring the selected preset (depends on T036 — same file)

**Checkpoint**: All user stories independently functional — overlay, safe mode, and export/import verified end-to-end

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation sync, cross-preset parity coverage, and full-path validation on CI

- [X] T041 [P] Update api/specs.md: document the user_preferences table, the preferences/reset endpoints, FR-013 server-side sanitization, and FR-012 retention semantics (Constitution IV — same change as the behavior it documents)
- [X] T042 [P] Update web/specs.md: document the preset token system, custom-colors base derivation, custom CSS overlay cascade and injection order, safe-mode entry points, unauthenticated isolation, accessibility guards, and export/import format (Constitution IV)
- [X] T043 Add Legacy-preset parity E2E coverage for SC-006: parametrize web/e2e/screenshots/all-screens.spec.ts (or add a parity pass alongside it) to run the core workflow screens — server management, log viewing, module browsing, settings management — under data-theme-preset="legacy" and assert functional parity and legibility against the Pink baseline
- [ ] T044 Verify the full quickstart.md validation guide passes on CI (Constitution VI — no local suite runs): confirm the pipeline executes the migration/API tests (§3), frontend unit tests (§4), the Playwright live suite (§5.1), the Go API e2e bucket (T024), and the Legacy parity spec (T043); quickstart.md serves as the executable checklist the CI run is validated against
- [ ] T045 Push the branch and verify full CI green per Constitution I/VI: Go unit tests, web unit tests with coverage gates (web/vitest.config.ts, api/.testcoverage.yml), lint with zero suppressions, the Go e2e bucket including api_theme_preferences_e2e_test.go, and the Playwright suites including theme-customization and the Legacy parity run; confirm web/e2e/specs/live/theme-customization.spec.ts is picked up by the web e2e CI workflow

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS all user stories
- **User Stories (Phases 3–6)**: All depend on Foundational completion; recommended order is priority order US1 → US2 → US3 → US4 (see same-file notes below)
- **Polish (Phase 7)**: Depends on all desired user stories being complete

### User Story Dependencies

- **US1 (P1)**: Starts after Foundational — no dependencies on other stories
- **US2 (P1)**: Starts after Foundational — backend migration/defaults land in Foundational; US2 adds hydration tests, the Go API-contract e2e, and unauthenticated route guards. Playwright e2e task T023 appends to the spec file created by US1's T015 (sequence after US1 or coordinate the file)
- **US3 (P2)**: Starts after Foundational — reuses the ThemeSettings page shell from US1 (T018) and appends to the same Playwright spec file
- **US4 (P3)**: Starts after Foundational — reuses the ThemeSettings page shell (T018) and appends to the same Playwright spec file; server-side sanitization already landed in Foundational T007

### Within Each User Story

- Tests are written first (or alongside) and must fail before implementation makes them pass
- Library utilities (theme-derivation, theme-sanitize, theme-export) before the ThemeSettings page cards that consume them
- Story checkpoint validation before moving to the next priority

### Same-File Coordination (do NOT parallelize these)

- web/e2e/specs/live/theme-customization.spec.ts: T015 → T023 → T028 → T033 (sequential appends)
- web/src/routes/ThemeSettings.tsx: T018 → T021/T030 → T036 → T037 → T040
- web/src/styles/globals.css: T016 → T017 (sequential)
- web/index.html and web/src/components/AppLayout.tsx: T011/T013 (Foundational) → T038 (US4)
- web/src/routes/Login.tsx: T025 (US2) → T039 (US4)
- test/e2e/buckets.sh: T024 (only touch in this feature)

### Parallel Opportunities

- Foundational: T004+T005+T006 (backend files), T008, T009+T010+T011 (client files) in parallel; T003 (design) runs in parallel with all backend work
- US1: T014, T015 (tests), and T016, T018, T019, T020 (implementation) in parallel; T017 follows T016
- US2: T022, T024 in parallel; T026 in parallel with both
- US3: T027 in parallel with nothing else in-phase (test-first), then T029
- US4: T031, T032, T034, T035 in parallel; T038, T039 in parallel
- Polish: T041 and T042 in parallel; T043 independent of both

---

## Parallel Example: User Story 1

```bash
# Launch US1 tests together:
Task: "Update token assertions in web/src/__tests__/theme.test.tsx"
Task: "Create Playwright live spec web/e2e/specs/live/theme-customization.spec.ts"

# Launch US1 implementation slices together (different files):
Task: "Add Legacy preset tokens to web/src/styles/globals.css"
Task: "Create ThemeSettings page shell in web/src/routes/ThemeSettings.tsx"
Task: "Add dropdown item in web/src/components/ui/TopBar.tsx"
Task: "Add footer button in web/src/components/ui/Sidebar.tsx"
```

## Parallel Example: User Story 4

```bash
# Launch US4 utility slices together (different files, no interdependencies):
Task: "Implement sanitizer in web/src/lib/theme-sanitize.ts (+ tests)"
Task: "Implement export/import in web/src/lib/theme-export.ts (+ tests)"
Task: "Create SafeModeBanner + safe-mode wiring in web/src/components/ui/SafeModeBanner.tsx and AppLayout.tsx"
Task: "Add safe-mode login link in web/src/routes/Login.tsx"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (design export + backend + client plumbing)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: preset switching works end-to-end (T014/T015/T021 green on CI)
5. Demo: Pink ↔ Legacy switching with persistence is already a shippable increment

### Incremental Delivery

1. Setup + Foundational → foundation ready
2. US1 → preset switching (MVP) → validate → demo
3. US2 → migration defaults + cross-device sync + API-contract e2e + unauthenticated isolation → validate
4. US3 → custom color scheme → validate
5. US4 → custom CSS overlay + safe mode + export/import → validate
6. Polish → specs.md sync + SC-006 parity + CI green

### Subagent Fan-Out Strategy (Constitution V)

- Slice 1 (backend): T004–T008 — one agent, sequential within slice
- Slice 2 (design): T003 — Pencil MCP work, parallel with slice 1
- Slice 3 (client plumbing): T009–T013 — after T009/T010, parallel with slice 1
- Slices 4–7 (US1–US4): one agent each in priority order, respecting the same-file coordination list; the Go API e2e (T024) can join slice 1 if US2 starts early
- Tier-up review after each wave before acceptance

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to spec.md user stories for traceability
- Constitution II: no React UI code may be written before T003's design export is committed
- Constitution III: strict TypeScript, `%w` error wrapping, zero in-source lint suppressions
- Constitution VI: verification happens on CI; locally only `go build ./...` and `tsc --noEmit` (T044/T045 are CI-verification tasks, not local runs)
- Commit after each task or logical group with signed commits (`git commit -s`) and conventional prefixes
