# Implementation Plan: User Theme Customization

**Branch**: `016-user-theme-customization` | **Date**: 2026-09-19 (revised 2026-09-21 after clarification session 2026-09-21) | **Spec**: ./spec.md

**Input**: Feature specification from `specs/016-user-theme-customization/spec.md`

## Summary

This feature delivers user-customizable visual themes to Gameplane, providing two curated presets ("Modern Pink" and "Legacy Orange"), a simple code-free color scheme editor as an alternative base theme, and an advanced custom CSS overlay with sanitization, safe-mode recovery, and portable export/import.

Key architecture points:
1. **User Migration & Backend Persistence**: A new migration (`api/internal/db/migrations/011_user_theme_preferences.sql`) creates `user_preferences` and migrates all pre-existing user accounts to the `legacy` orange & dark theme. New users default to the modern `pink` theme. Preferences sync across devices via `GET/PUT /api/v1/users/me/preferences` and hydrate directly within `useMe()`.
2. **CSS Token System**: `web/src/styles/globals.css` extends HeroUI semantic tokens with `data-theme-preset="legacy"`, mapping the orange accent (`#F97316`) and dark neutrals (`#0F0F0F`, `#171717`, `#1C1C1C`) to HeroUI's base tokens.
3. **Simple Custom Colors**: A TypeScript derivation utility (`web/src/lib/theme-derivation.ts`) calculates accessible semantic tokens from user-selected Primary Accent and Surface tones; custom colors act as the base theme when `themeType = "custom_colors"`.
4. **Custom CSS Overlay (FR-007/FR-013)**: Custom CSS is an independent overlay (`customCssEnabled` flag), not a third theme mode. It is injected via `<style id="gameplane-custom-css">` as the **last** child of `<head>` so user rules win over the base at equal specificity, and it survives base theme switches. Sanitization per FR-013: `@import` and external `url()` references rejected (inline `data:` URIs allowed), syntax validated, 32 KB cap, HTML delimiter rejection — enforced server-side, mirrored client-side in `web/src/lib/theme-sanitize.ts`.
5. **Safe Mode (FR-009)**: Three entry points suspend the overlay for the session without deleting it: the `?safe-mode=1` URL parameter (guaranteed path), a keyboard shortcut, and a "Sign in with safe mode" link on the login page (a guaranteed-clean surface per FR-011). A `SafeModeBanner` offers quick access to fix or clear the stylesheet.
6. **Retention & Reset (FR-012)**: Switching presets or disabling the overlay never deletes stored custom colors/CSS. Only the explicit "Reset to Defaults" confirmation (`POST /api/v1/users/me/preferences/reset`) clears them.
7. **Export / Import (FR-014)**: Client-generated versioned JSON (`gameplane-theme` v1, `web/src/lib/theme-export.ts`) exported via copy/download; import is validated client-side and applied through the PUT endpoint so server sanitization stays authoritative. Sharing galleries, admin-enforced defaults, and per-page themes are out of scope.
8. **UI & Design**: A dedicated settings page at route `/settings/theme` with settings nav column and stacked section cards (Preset theme / Appearance mode / Custom colors / Custom CSS / Export-Import) built from HeroUI primitives inside the standard app shell (sidebar + top bar + page header), with triggers in the TopBar user avatar dropdown and Sidebar appearance footer that navigate to the page.

---

## Technical Context

**Language/Version**: Go 1.25 (backend API), TypeScript 6.0.3 strict (frontend web), React 19.2.8 (versions per web/package.json).

**Primary Dependencies**:
- Backend: `chi/v5`, standard library `database/sql`, `modernc.org/sqlite`.
- Frontend: `@heroui/react`, `@heroui/styles`, `@tanstack/react-query`, `lucide-react`.

**Storage**: SQLite and PostgreSQL (opt-in) via `user_preferences` table with cascading foreign key to `users(id)`; `custom_css` capped at 32 KB (FR-013); `custom_css_enabled` boolean for the overlay. Client-side caching in `localStorage` under `gameplane-theme-prefs` (alongside the existing `gameplane-theme` mode key).

**Testing**:
- Go unit tests: `internal/db` (migration verification), `internal/handlers` (preferences + reset endpoints, sanitization rejections, retention).
- Web unit tests: Vitest 5 (`theme.test.tsx`, `themeBoot.test.ts`, `theme-sanitize.test.ts`, `theme-export.test.ts`, `theme-derivation.test.ts`, `ThemeSettings.test.tsx`).
- E2E tests: Playwright live specs (`web/e2e/specs/live/theme-customization.spec.ts`) for dashboard flows.
- Go API-contract E2E: `test/e2e/api_theme_preferences_e2e_test.go`, registered in `test/e2e/buckets.sh` per Constitution I (preferences/reset endpoints, migration defaults, sanitization rejections, retention through the real API).
- Legacy-preset parity E2E: `web/e2e/screenshots/all-screens.spec.ts` parametrized to run under `data-theme-preset="legacy"` (SC-006).

**Target Platform**: Evergreen desktop (1440px) and mobile (390px) browsers served by the Gameplane API.

**Project Type**: Full-stack web application (Go API service + React web frontend).

**Performance Goals**:
- Client-side theme switching completes in < 100ms without full page reload (SC-003).
- Zero Flash of Unstyled Content (FOUC) on application boot via synchronous `index.html` boot script.

**Constraints**:
- Constitution Principle I: E2E coverage for theme switching, overlay cascade, safe-mode recovery (URL param + login link), and export/import round-trip (Playwright live), plus a Go API-contract E2E registered in `test/e2e/buckets.sh`, plus Legacy-preset parity coverage of core workflow screens (SC-006).
- Constitution Principle II: Theme Settings page (route `/settings/theme`), Safe Mode banner, export/import controls, and login-page safe-mode link designed in `design.pen` via Pencil MCP server before code implementation, with matching `design-export/` snapshots.
- Constitution Principle III: Strict TypeScript, Go `%w` error wrapping, no `//nolint` or `// @ts-ignore`.
- FR-003: 100% of pre-existing accounts migrated to Legacy theme.
- FR-011: Unauthenticated public pages (login and share links) strictly render with the default Pink theme preset and never execute custom CSS (the login-page safe-mode link is plain pink chrome).
- FR-013: CSS sanitization enforced server-side; client checks are UX mirrors, never the gate.

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | How this plan satisfies it |
|---|---|---|
| **I. E2E-Tested Delivery** | PASS | Playwright live specs (`web/e2e/specs/live/theme-customization.spec.ts`) verify preset switching, migration defaults, persistence across reloads, custom colors, overlay cascade priority across base switches, retention, safe-mode recovery via URL parameter and login-page link, sanitization rejection, and export/import round-trip — per project e2e conventions (unique resource names, parallel-safe). Additionally, a Go API-contract E2E (`test/e2e/api_theme_preferences_e2e_test.go`) is registered in `test/e2e/buckets.sh` per the constitution's bucket requirement, and `web/e2e/screenshots/all-screens.spec.ts` is parametrized under the Legacy preset for SC-006 parity. |
| **II. Design-First** | PASS | The theme settings page at `/settings/theme` (Presets / Custom Colors / Custom CSS / Export sections), `SafeModeBanner`, export/import section, and login-page safe-mode link will be created in `design.pen` via the Pencil MCP server and exported to `design-export/{json,screenshots}/` before React code is merged. |
| **III. Language & Ecosystem** | PASS | Strict TypeScript enabled; error wrapping with `%w`; zero in-source linter suppressions (`//nolint`, `// @ts-ignore`). Coverage gates remain intact. |
| **IV. Spec-Driven** | PASS | Specification, clarifications, plan, research, data model, contracts, and quickstart guide precede implementation. `web/specs.md` and `api/specs.md` will be updated in the same change as the implementation. |
| **V. Delegate to Workflows** | PASS | Implementation tasks will be fanned out across independent slices (API/DB slice, CSS token slice, overlay/sanitization slice, UI settings-page slice, export/import slice, E2E slice) with tier-appropriate review. |
| **VI. CI Bears the Heavy Lifting** | PASS | All unit tests, migrations, linting, and Playwright E2E suites run on GitHub Actions CI; only local `go build` and `tsc --noEmit` checks. |

*Post-design re-check (after Phase 1, 2026-09-21): All principles continue to pass. The overlay model reduces DOM-surface complexity (one flag instead of a third mode); sanitization adds a server validation slice but no architectural violation. No Complexity Tracking entries required.*

---

## Project Structure

### Documentation (this feature)

```text
specs/016-user-theme-customization/
├── spec.md                  # Feature specification with clarification resolutions
├── plan.md                  # This implementation plan
├── research.md              # Phase 0: Technical decisions (R-01 to R-07)
├── data-model.md            # Phase 1: Entity model, DDL, TypeScript interfaces
├── quickstart.md            # Phase 1: Runnable end-to-end verification guide
├── contracts/
│   ├── user-preferences-api.md  # REST contract: preferences GET/PUT + reset POST
│   ├── theme-tokens-v2.md       # Semantic token mappings + overlay injection/ordering rules
│   ├── theme-ui.md              # UI contract: settings-page sections, safe-mode link/banner, reset, export
│   └── theme-export.md          # gameplane-theme v1 export/import JSON schema
├── checklists/
│   └── requirements.md      # Specification quality checklist (validated)
└── tasks.md                 # Phase 2 output (/speckit-tasks command)
```

### Source Code Layout

```text
api/
├── internal/
│   ├── db/
│   │   ├── migrations/
│   │   │   └── 011_user_theme_preferences.sql  # NEW: table & existing user migration (010 is taken)
│   │   ├── preferences.go                      # NEW: DB queries for user_preferences
│   │   └── preferences_test.go                 # NEW: unit tests for migration and queries
│   └── handlers/
│       ├── users.go                            # UPDATE: mount /users/me/preferences (+ /reset) & extend /users/me
│       └── users_preferences_test.go           # NEW: tests incl. sanitization 400s & retention semantics

web/
├── index.html                                  # UPDATE: boot script reads gameplane-theme-prefs, injects
│                                               #   cached overlay last, honors ?safe-mode=1
├── src/
│   ├── types.ts                                # UPDATE: UserThemePreferences (+ customCssEnabled), ThemeExport
│   ├── styles/
│   │   └── globals.css                         # UPDATE: add [data-theme-preset="legacy"] tokens
│   ├── lib/
│   │   ├── endpoints.ts                        # UPDATE: Users.getPreferences / updatePreferences / resetPreferences
│   │   ├── useThemePreferences.ts              # NEW: hook for reading & updating theme state (lib convention)
│   │   ├── theme-derivation.ts                 # NEW: derive custom color tokens from accent & surface
│   │   ├── theme-derivation.test.ts            # NEW: unit tests for contrast and color math
│   │   ├── theme-sanitize.ts                   # NEW: FR-013 CSS validation/sanitization rules (client mirror)
│   │   ├── theme-sanitize.test.ts              # NEW: unit tests for sanitization rules
│   │   ├── theme-export.ts                     # NEW: gameplane-theme v1 export/import build & validation
│   │   └── theme-export.test.ts                # NEW: unit tests for export/import round-trip
│   ├── components/
│   │   ├── ui/
│   │   │   ├── TopBar.tsx                      # UPDATE: add "Theme & Appearance" item in user dropdown
│   │   │   ├── Sidebar.tsx                     # UPDATE: add settings trigger button in footer
│   │   │   └── SafeModeBanner.tsx              # NEW: floating banner when safe mode is active
│   │   └── AppLayout.tsx                       # UPDATE: integrate theme provider & safe mode banner
│   ├── routes/
│   │   ├── ThemeSettings.tsx                   # NEW: settings page at /settings/theme with stacked section cards
│   │   ├── ThemeSettings.test.tsx              # NEW: unit tests for page interactions
│   │   └── Login.tsx                           # UPDATE: add "Sign in with safe mode" link (Pink chrome)
│   └── __tests__/
│       ├── theme.test.tsx                      # UPDATE: assert Pink vs Legacy token values
│       └── themeBoot.test.ts                   # UPDATE: assert preset/overlay init + safe-mode skip
└── e2e/
    └── specs/
        └── live/
            └── theme-customization.spec.ts     # NEW: Playwright live tests (see quickstart §5.1)

test/
└── e2e/
    ├── api_theme_preferences_e2e_test.go       # NEW: Go API-contract E2E — preferences/reset endpoints,
    │                                           #   migration defaults, FR-013 rejections, FR-012 retention
    └── buckets.sh                              # UPDATE: register the theme preferences e2e in a bucket
```

---

## Complexity Tracking

*No constitutional or architectural violations. No entry required.*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| None | N/A | N/A |
