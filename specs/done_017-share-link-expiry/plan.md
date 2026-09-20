# Implementation Plan: Configurable Share Link Expiry

**Branch**: `017-share-link-expiry` | **Date**: 2026-09-20 | **Spec**: [./spec.md](./spec.md)

**Input**: Feature specification from `specs/done_017-share-link-expiry/spec.md`, settled by maintainer rulings in `specs/done_017-share-link-expiry/OPEN-DECISIONS.md` (OD-1/3/5/7, OD-2, OD-4, OD-6 all **Settled** as of 2026-09-19).

## Summary

Replace the fixed 90-day maximum lifetime on share links with six create-dialog choices (15/30/60/90 days, No expiry, Custom date with no upper bound). The wire contract carries exactly one of `expiresAt` (RFC3339 instant, computed client-side) or `neverExpires: true`; a request with neither or both is rejected 400. `ShareLink.ExpiresAt` becomes `*time.Time` (nil = never), matching the existing `RevokedAt`/`LastUsed` pattern. The legacy `expiresIn` duration field is kept for one release as a deprecated path (its 7-day default applies only there), then removed in a follow-up feature. A new append-only migration (`010_share_links_expiry_nullable.sql`) rebuilds `share_links` (SQLite has no `ALTER COLUMN`) to make `expires_at` nullable without touching any existing row's value. The web UI change (dialog redesign, "Never" list column) is design-first per CLAUDE.md rule 1 and is **blocked** until `design.pen` nodes `atqRh`/`xCJlu` are redesigned via Pencil MCP — itself blocked until PR #378's design work merges (see Constitution Check and Phase 5 below). `docs/security.md` gains a "Share links" section from the already-drafted `docs-security-draft.md`.

---

## Technical Context

**Language/Version**: Go 1.25 (`api/`), TypeScript (React 18 + Vite, `web/`), SQL (SQLite + Postgres dual-driver migrations)

**Primary Dependencies**: `api/internal/db` (SQLite/Postgres store), `api/internal/handlers` (chi), `web/src/lib/api.ts` (fetch client), HeroUI v3 components, MSW (component tests), Playwright (live e2e)

**Storage**: `share_links` table (`api/internal/db/migrations/006_share_links.sql`, `009_share_links_cluster.sql`); this feature adds `010_share_links_expiry_nullable.sql`

**Testing**: Go unit/store tests (`api/internal/db/shares_test.go`), Go handler tests (new, `api/internal/handlers/shares_test.go` if absent, else extended), web component tests (Vitest + Testing Library + MSW), no new `test/e2e/` Go bucket (per spec.md Assumption "E2E tier" — the existing Playwright live spec `web/e2e/specs/live/share-links.spec.ts` covers browser e2e once the design lands)

**Target Platform**: API server (Go, `api/cmd/main.go serve`), dashboard SPA (`web/`)

**Project Type**: Web application (Go API + React SPA) — existing feature area, no new component type

**Performance Goals**: No change; lookup remains the existing indexed `token_hash` equality query (`idx_share_links_token`)

**Constraints**: Migration must be one SQL file valid on both SQLite and Postgres (no `ALTER COLUMN`); existing non-null `expires_at` values must be copied verbatim, never recomputed; revocation must remain independent of expiry shape; no new token logging.

**Scale/Scope**: 1 migration, 1 store file, 1 handler file, 2 web contract files, 1 web route component + dialog, 1 MSW mock file, `docs/security.md`; web-UI-visible task tree is blocked pending design-first prerequisites (see Constitution Check).

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Justification & Verification Plan |
|---|---|---|
| **I. E2E-Tested Delivery** | **PASS (deferred slice)** | No new Go `test/e2e/` bucket per spec.md's own Assumption ("E2E tier"); the existing Playwright live spec (`web/e2e/specs/live/share-links.spec.ts`) is the E2E surface, and it is explicitly blocked (Phase 5) until the redesigned dialog ships. API-tier behavior (migration, store, handler) is covered by Go tests at every tier without needing a live browser. |
| **II. Design-First for User-Facing Change** | **BLOCKED, tracked** | `design.pen` nodes `atqRh` (create dialog) and `xCJlu` (Share links list) must be redesigned via Pencil MCP and exported to `design-export/` (CLAUDE.md rule 1) before any React change in `web/src/routes/tabs/settings/ShareLinks.tsx`. This itself is blocked on PR #378's design work merging first (stated in the computed task; not independently verified by this plan — see OPEN-DECISIONS.md addition below). Phase 5 tasks are written and sequenced but marked blocked; they are not started by this plan. |
| **III. Language & Ecosystem Best Practice** | **PASS** | Go: `*time.Time` matches existing `RevokedAt`/`LastUsed` idiom; errors wrapped with `%w`. TypeScript: strict, `expiresAt: string \| null`, no unjustified `any`. |
| **IV. Spec-Driven Development** | **PASS** | `spec.md` + `OPEN-DECISIONS.md` are complete and settled per maintainer ruling; this plan and `tasks.md` are the next required artifacts; `api/specs.md` and `web/specs.md` are updated in Phase 2 tasks as part of the same changeset that alters their documented behavior. |
| **V. Delegate to Workflows & Subagents** | **N/A here** | This plan/tasks artifact is produced directly; implementation delegation (tiering, review-at-tier+1) is the orchestrator's concern when `tasks.md` is executed, not this document's. |
| **VI. CI Bears the Heavy Lifting** | **PASS** | Local verification limited to `go build ./...` and `npx tsc --noEmit` per rule 8; `make test`/`make lint` are never run locally; CI validates coverage thresholds (`api` 80%, `web` 92%L/76%F/82%B/92%S). |

**Post-Design Re-check**: Re-run once Phase 5's design-first blocker clears; until then, Phases 1-4 (API/contract/docs) can complete and merge independently of the UI slice, since `expiresIn` staying supported for one release means the existing UI keeps working unmodified against the new API.

---

## Project Structure

### Documentation (this feature)

```text
specs/done_017-share-link-expiry/
├── plan.md                      # This file
├── tasks.md                     # Phase breakdown (this feature)
├── OPEN-DECISIONS.md            # Settled rulings (OD-1..OD-7) + any newly surfaced opens
├── spec.md                      # Requirements (already complete)
└── docs-security-draft.md       # Drafted docs/security.md section, spliced in by T0xx
```

No `data-model.md`/`research.md`/`contracts/` directory is added: the single entity change (`ShareLink.ExpiresAt`) and the wire contract are fully specified in `spec.md` Key Entities and `OPEN-DECISIONS.md` OD-1/OD-3, so a separate contracts file would only restate them. `specs/014-heroui-web-rebuild/contracts/share-link-ui.md` remains the UI contract of record and is updated in place, not duplicated.

### Source Code (repository root)

```text
api/
├── internal/db/
│   ├── migrations/
│   │   └── 010_share_links_expiry_nullable.sql   # NEW — table rebuild, nullable expires_at
│   ├── shares.go                                  # ExpiresAt -> *time.Time; drop MaxShareLinkExpiryDays
│   └── shares_test.go                             # Updated: replace cap-assertion tests (see tasks.md)
├── internal/handlers/
│   ├── shares.go                                  # expiresAt | neverExpires validation; deprecated expiresIn path
│   └── shares_test.go                             # NEW or extended: 400 on neither/both, deprecated-path test
└── specs.md                                        # Share-links section updated

web/src/
├── types.ts                                        # ShareLinkCreateRequest / ShareLink.expiresAt: string | null
├── lib/api.ts                                       # createShareLink signature (expiresAt | neverExpires)
├── routes/tabs/settings/ShareLinks.tsx              # BLOCKED on design-first (Phase 5)
├── test/handlers.ts                                 # MSW mocks updated for new request/response shape
└── specs.md                                         # Share-links UI section updated

docs/
└── security.md                                      # NEW "Share links" section (after "## Authorization")

design.pen                                           # atqRh, xCJlu — Pencil MCP, BLOCKED (Phase 5)
design-export/{json,screenshots}/atqRh*, xCJlu*.{json,png}  # BLOCKED, exported alongside the .pen change
```

---

## Complexity Tracking

| Aspect | Justification | Simpler Alternative Rejected Because |
|---|---|---|
| Table rebuild instead of `ALTER COLUMN` | SQLite has no `ALTER TABLE ... ALTER COLUMN`; the migration runner applies one file unchanged to both drivers (`api/internal/db/db.go:68-104`) | Per-driver migration files were considered and explicitly left open in OD-4 as a maintainer option, but the settled ruling picked the `004_cluster_rbac.sql`-style rebuild, so this plan follows the ruling rather than reopening it |
| Keeping deprecated `expiresIn` for one release | Avoids a breaking change to any existing caller/script (OD-5) and keeps the current UI functional while the design-first UI slice is blocked | Removing it immediately would break `web/e2e/specs/live/share-links.spec.ts:77` and any external scripts with no migration window |
