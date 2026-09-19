---
description: "Task list for Feature 017: Configurable Share Link Expiry"
---

# Tasks: Configurable Share Link Expiry

**Input**: Design documents from `specs/017-share-link-expiry/`: `spec.md`, `plan.md`, `OPEN-DECISIONS.md` (all rulings OD-1..OD-7 settled 2026-09-19), `docs-security-draft.md`.

**Prerequisites**: plan.md (required), spec.md (required for user stories), OPEN-DECISIONS.md (required — every ruling below cites its OD item; nothing here re-decides an OD).

**Worktree**: all tasks below operate only inside `/home/valgul/project/Gameplane-017` (branch `017-share-link-expiry`). Never touch `/home/valgul/project/Gameplane`. No task commits, checks out, restores, stashes, or resets; committing is a separate, later step outside this task list.

**Local verification (rule 8)**: `go build ./...` inside `api/` and `cd web && npx tsc --noEmit` are the only local checks any task below may run. No task runs `make test`, `make lint`, `make cover`, `go test`, `npm test`, envtest, or E2E locally — CI is the sole verification authority.

**Format**: `T### [P?] [Group] Description`
- **[P]**: Different files, no dependency on another incomplete task — safe to run concurrently with other `[P]` tasks in the same phase.
- **[Group]**: One of API, Contract, Web, Docs, matching the computed task's four touch-point groups.
- Every description carries a repo-relative path.

---

## Phase 1: Setup

**Purpose**: Confirm the migration slot and settled rulings before any file changes.

- [ ] T001 [P] Confirm `010` is still the next free migration number in `api/internal/db/migrations/` (no other branch has claimed it) — anchor file `api/internal/db/migrations/009_share_links_cluster.sql`. If `010_*.sql` already exists on this worktree from another concurrent feature, record the conflict in `specs/017-share-link-expiry/OPEN-DECISIONS.md` as a new open item and pick the next free number instead of overwriting.
- [ ] T002 [P] Re-read `specs/017-share-link-expiry/OPEN-DECISIONS.md` in full and confirm no item needed by Phases 2-4 (OD-1, OD-2, OD-4, OD-6) is still marked Open; only OD-3's two sub-questions are folded into OD-1's Settled ruling (`*time.Time`, JSON `null`) and need no separate re-check. Nothing to write; this is a go/no-go gate for Phase 2.

**Checkpoint**: migration slot confirmed, rulings re-verified as settled.

---

## Phase 2: API (migration, store, handler, Go tests)

**Purpose**: Implement FR-005 through FR-008, FR-010, FR-012 and the OD-1/OD-4 rulings end-to-end in the API tier. This phase is independently shippable: with the deprecated `expiresIn` path kept working, no web change is required to merge Phase 2.

### 2.1 Migration

- [ ] T003 [API] Author `api/internal/db/migrations/010_share_links_expiry_nullable.sql` per OD-4's settled mechanics: `CREATE TABLE share_links_new` with every column from `006_share_links.sql` + `009_share_links_cluster.sql` in their current order, `expires_at` changed from `TEXT NOT NULL` to `TEXT` (nullable), every other `NOT NULL` preserved (`token_hash UNIQUE`, `created_by ... REFERENCES users(id) ON DELETE CASCADE`, `cluster TEXT NOT NULL DEFAULT 'local'`); `INSERT INTO share_links_new (<named columns>) SELECT <same named columns> FROM share_links` (explicit column lists on both sides, per OD-4's constraint — column order differs from `006` because of `009`, so no `SELECT *`); `DROP TABLE share_links`; `ALTER TABLE share_links_new RENAME TO share_links`; recreate `idx_share_links_token`, `idx_share_links_server`, `idx_share_links_cluster_server` after the rename, exactly as `004_cluster_rbac.sql` does for its own rebuild. Do not recompute or touch any existing `expires_at` value (User Story 2, SC-003).
- [ ] T004 [API] Write a focused Go test proving migration correctness (add to `api/internal/db/shares_test.go` or a new `api/internal/db/migrations_010_test.go`, whichever the existing test file's structure favors — check how `004_cluster_rbac.sql`'s rebuild is tested, if at all, and follow that precedent): seed a `share_links` row with a non-null `expires_at` under the pre-010 schema shape (or run migrations up to `009` then insert, then run `010`), confirm the row's `expires_at` is byte-for-byte unchanged after `010` runs, and confirm a second row inserted post-migration with a NULL `expires_at` round-trips as NULL. Covers User Story 2's Independent Test and SC-003. Depends on T003.

### 2.2 Store (`api/internal/db/shares.go`)

- [ ] T005 [API] Change `ShareLink.ExpiresAt` from `time.Time` to `*time.Time` (nil = never), matching the existing `RevokedAt`/`LastUsed` `*time.Time` pattern already in the same struct (OD-1, OD-3). Update the doc comment on the field (currently "mandatory expiry timestamp") to state nil means no expiry.
- [ ] T006 [API] Change `CreateShareLink`'s signature from `expiresAt time.Time` to `expiresAt *time.Time` (nil = never); remove `MaxShareLinkExpiryDays` (the constant) and the `maxExpiry := now.AddDate(0, 0, MaxShareLinkExpiryDays)` check plus whatever conditional currently enforces it (FR-005). Keep the existing future-only validation (`ErrShareLinkExpiryInvalid` when `expiresAt != nil && !expiresAt.After(now)`), applied only when `expiresAt` is non-nil — a nil `expiresAt` bypasses the future check entirely (spec.md Clarifications). Depends on T005.
- [ ] T007 [API] Update `LookupShareLink`'s expiry check (`time.Now().After(link.ExpiresAt)`) to skip entirely when `link.ExpiresAt == nil` (FR-006); keep the revocation check (`revoked_at`) independent and evaluated regardless of expiry shape (FR-010, User Story 3). Depends on T005.
- [ ] T008 [API] Update `ListShareLinks` to scan a nullable `expires_at` column into `*time.Time` for both SQLite and Postgres drivers (check how the existing scan handles `RevokedAt`/`LastUsed` nullable columns and mirror that exact pattern) so a NULL row comes back as `ExpiresAt: nil` (FR-007). Depends on T005.
- [ ] T009 [API] Update every remaining `shares.go` call site that assumed `ExpiresAt` was always set (e.g. any `.Format(...)` or comparison outside the three functions above) to nil-check first. Depends on T005-T008.
- [ ] T010 [API] Update `api/internal/db/shares_test.go`: replace the two tests keyed on the removed cap — `shares_test.go:120` (90+1 days rejected) and `shares_test.go:136` (exactly 90 days accepted) — with tests of the new behavior per the maintainer's sign-off recorded in spec.md's Assumptions ("Tests that enforce today's cap... Replacing them needs maintainer sign-off... they are replaced by tests of the new behaviour, not deleted"): (a) a custom expiry far beyond 90 days (e.g. 200 days, per SC-002) is now accepted; (b) a nil `expiresAt` is accepted and persists as NULL; (c) a non-nil `expiresAt` at or before "now" is still rejected (`ErrShareLinkExpiryInvalid`), preserving the one assertion from the old tests that still holds. Add a dedicated test for `LookupShareLink` returning a link unexpired at year+ range with `ExpiresAt: nil` (SC-001), and one for `ListShareLinks` returning a mixed set (dated + nil) with both shapes intact. Depends on T005-T008.
- [ ] T011 [API] Add a Go test proving revocation is independent of expiry shape for all four shapes from User Story 3 (short, long/custom, and nil expiry) — create one link per shape, revoke each, assert `LookupShareLink` returns the same `ErrShareLinkInvalid` for all four. Depends on T007.

### 2.3 Handler (`api/internal/handlers/shares.go`)

- [ ] T012 [API] Add `NeverExpires *bool` (or equivalent) and an absolute `ExpiresAt *string` (RFC3339) to the create-request struct alongside the existing `ExpiresIn *string`, per OD-1's settled shape: the request carries exactly one of `expiresAt` or `neverExpires: true`. Reject with 400 when neither is present and `expiresIn` is also absent (see T013 for the deprecated-path interaction), and reject with 400 when both `expiresAt` and `neverExpires: true` are present together (FR-009, OD-1).
- [ ] T013 [API] Implement the three-way request handling in the POST create handler: (1) new path — `expiresAt` present and non-empty: parse RFC3339, validate strictly-future (reuse the store's validation path, do not duplicate the future-check logic — pass the parsed `*time.Time` straight to `CreateShareLink` and let T006's check fire), reject malformed RFC3339 with 400; (2) new path — `neverExpires: true`: pass `nil` to `CreateShareLink`; (3) deprecated path — `expiresIn` present (and neither of the above is): keep today's exact behavior for one release — parse the duration, compute `expiresAt` server-side, default to 7 days when the value is empty/omitted **only on this deprecated path** (OD-1's ruling: "its existing 7-day default for an omitted value stays only on that legacy path") — but remove the `MaxShareLinkExpiryDays` clamp (`shares.go:107-111`) from this path too, since FR-005 requires it gone from the handler entirely, not just the new fields. A request with none of `expiresAt`/`neverExpires`/`expiresIn` is rejected 400 (OD-7: a missing field never silently creates a permanent or defaulted link outside the explicitly-deprecated path). Depends on T012, T006.
- [ ] T014 [API] Update the response struct's `ExpiresAt string` field to `ExpiresAt *string` (or keep `string` and emit `""`/omit — follow OD-3's ruling: JSON `null` when there is no expiry, so `*string` with `omitempty` is wrong if it would omit rather than null; use a field that marshals explicit `null`). Update both response-construction sites (`shares.go:123` and `:176`) to emit `null` when `link.ExpiresAt == nil`, and the RFC3339 string otherwise. Depends on T005, T012.
- [ ] T015 [API] Add `api/internal/handlers/shares_test.go` (new file if none exists at this exact path, else extend) covering: 400 when the create request has neither `expiresAt` nor `neverExpires` nor `expiresIn`; 400 when both `expiresAt` and `neverExpires: true` are present; 201 with `expiresAt: null` in the response for `neverExpires: true`; 201 with the exact requested instant echoed back for `expiresAt`; the deprecated `expiresIn` path still returns 201 with a computed `expiresAt` and its 7-day-default-on-empty behavior preserved; a custom `expiresAt` beyond 90 days out (SC-002) succeeds with no clamp applied anywhere in the response. Depends on T013, T014.
- [ ] T016 [API] Run `go build ./...` inside `api/` (rule 8 sanctioned check) and fix any compile error surfaced by the `*time.Time` change across `api/internal/handlers/`, `api/internal/db/`, and any other package that imports `ShareLink`. Do not run `go test`. Depends on T003-T015.

**Checkpoint**: API tier fully implements FR-005 through FR-010, FR-012; deprecated `expiresIn` path works for one release; Go tests exist at store and handler tiers; the existing UI (unmodified) keeps working against this API since `expiresIn` still works.

---

## Phase 3: Contract & Types (Go/TS wire contract, spec docs)

**Purpose**: Make the new wire shape visible to the web client and keep `api/specs.md`/`web/specs.md` accurate per Constitution Principle IV (same changeset as the behavior they document). Independently mergeable after Phase 2; does not require the design-first UI slice.

- [ ] T020 [Contract] Update `api/specs.md`'s share-links section (~line 442-443, "006_share_links.sql: ... signed, expiring, revocable tokens") to describe the nullable expiry, the new `010` migration, and the `expiresAt`/`neverExpires` request fields with `expiresIn` marked deprecated-for-one-release. Depends on T003-T015 (documents the shipped behavior, not a proposal).
- [ ] T021 [Contract] Update `web/specs.md`'s Share links description (~line 884, "Create dialog opens to set expiry (24h/7d/30d/90d) and start permission") to describe the new six-choice set (15/30/60/90 days, No expiry, Custom) and the "Never" list-column behavior, explicitly flagging that the UI itself is not yet implemented (blocked, Phase 5) — this task documents the *target* contract so `web/specs.md` reads consistently once Phase 5 ships, matching how other blocked-but-specified sections are written elsewhere in that file. Depends on nothing beyond spec.md/OPEN-DECISIONS.md being settled (already true).

**Checkpoint**: `api/specs.md`/`web/specs.md` describe the shipped API behavior and the target UI behavior. `web/src/types.ts`'s widening, the `api.ts` client update, and the MSW mock update (formerly T017-T019 here) move to Phase 5 per OD-8: they ship with the UI slice, not here, since the API never returns `null` to a client that cannot yet request `neverExpires`.

---

## Phase 4: Docs

**Purpose**: FR-011 — record the token-hashing-remains-safe rationale in `docs/security.md` now that the cap is gone.

- [ ] T022 [Docs] Splice `specs/017-share-link-expiry/docs-security-draft.md`'s full "## Share links" section into `docs/security.md` immediately after the existing "## Authorization" section (per the draft file's own header comment), verbatim except for updating any code-line references (`api/internal/audit/audit.go:764` → `redactShareToken`'s current line, `api/cmd/main.go:640` → `secureHeaders`'s current line) if either has drifted since the draft was written — check both before pasting. Depends on nothing (can run any time; sequenced last only because the draft references "once implementation lands", so pasting after Phase 2 makes the described behavior already true).
- [ ] T023 [P] [Docs] Run a repo-wide check that `docs/security.md` has no duplicate "## Share links" heading and that the table of contents (if `docs/security.md` has one) is updated to include it. Depends on T022.

**Checkpoint**: `docs/security.md` documents the no-cap rationale; FR-011 satisfied.

---

## Phase 5: Web UI — BLOCKED on design-first (CLAUDE.md rule 1)

**Purpose**: FR-001 through FR-004, FR-007 (list rendering), User Story 1, SC-004. **Every task in this phase is blocked** and must not be started until, in order: (1) PR #378's design work merges to `master` (external precondition, not verified by this task list — check `gh pr view 378` for merge status before starting T024), and (2) `design.pen` nodes `atqRh` (create dialog) and `xCJlu` (Share links list) are redesigned via Pencil MCP and exported to `design-export/json/` and `design-export/screenshots/` (CLAUDE.md rule 1, `design-export` skill). No task below may touch `.pen` files directly (CLAUDE.md rule 2 — Pencil MCP only, never Read/Grep/cat/sed).

- [ ] T017 [Web] **BLOCKED** (moved from Phase 3, OD-8, 2026-09-20) — Update `web/src/types.ts`: change `ShareLink.expiresAt: string` (line ~785 and ~1010, the two nearby declarations noted in OD-3) to `expiresAt: string | null`, and add the new create-request fields (`expiresAt?: string`, `neverExpires?: boolean`) alongside the existing `expiresIn?: string` (line ~1017), with a comment noting `expiresIn` is deprecated-for-one-release per OD-1/OD-5. Held back with the rest of this phase per OD-8: `ShareLinks.tsx`'s `formatDate`/`getLinkStatus` need null handling before the wider type is safe to ship, and that's UI work blocked by design-first (CLAUDE.md rule 1).
- [ ] T018 [Web] **BLOCKED** — Update `web/src/lib/api.ts`'s `createShareLink` (or equivalent) client method signature to accept the new discriminated shape (an absolute date, or "never"), matching T017's types. Keep the method able to send the deprecated `expiresIn` only if any remaining caller needs it. Depends on T017.
- [ ] T019 [Web] **BLOCKED** (moved from Phase 3, OD-8, 2026-09-20) — Update `web/src/test/handlers.ts`'s MSW mock for `POST /servers/:name:shares` (~line 1438) to accept and honor the new `expiresAt`/`neverExpires` fields in the request body, returning `expiresAt: null` in the response when `neverExpires` was sent, and to keep honoring `expiresIn` on the deprecated path so any currently-passing component test keeps passing unmodified. Depends on T017.
- [ ] T024 [Web] **BLOCKED** — Redesign `atqRh` (create-link dialog) via Pencil MCP: remove the "Maximum 90 days" help text, replace the current expiry Select's four choices with six (15 days, 30 days, 60 days, 90 days, No expiry, Custom), 30 days pre-selected (FR-001); add the "This link works until you revoke it." warning shown on "No expiry" (FR-002); add a date-picker control shown only for "Custom", restricted to strictly-after-today dates (FR-003); add the long-lived-token warning ("Long-lived link — it stays valid for over a year unless you revoke it.") shown when the custom date is 365+ days out (FR-004, OD-6). Export touched nodes to `design-export/json/atqRh*.json` and `design-export/screenshots/atqRh*.png` in the same step (design-export skill). Depends on the external PR #378 merge and cannot start before it.
- [ ] T025 [Web] **BLOCKED** — Redesign `xCJlu` (Share links list) via Pencil MCP: Expires column renders "Never" for a NULL-expiry link and the row's status never shows "Expired" for it (FR-007, SC-004). Export per the same rule as T024. Depends on T024 (same design session, same `.pen` file).
- [ ] T026 [Web] **BLOCKED** [P] Update `web/src/routes/tabs/settings/ShareLinks.tsx`'s `formatDate` to render "Never" for `expiresAt === null` (SC-004), matching the exported design from T025. Depends on T025, T018.
- [ ] T027 [Web] **BLOCKED** [P] Update `getLinkStatus` (`ShareLinks.tsx` ~line 60) to accept `expiresAt: string | null` and never return "Expired" for `null` (FR-007). Depends on T025, T018.
- [ ] T028 [Web] **BLOCKED** — Rebuild the create-dialog form logic in `ShareLinks.tsx` (~line 101's `expiresIn: expiry` construction) against the T024 design: six-option selector defaulting to 30 days; computing an absolute `expiresAt` client-side for presets (today + N days) and for the custom date (end of chosen calendar day in local time, converted to a UTC instant — OD-2's exact ruling); sending `neverExpires: true` for "No expiry"; disabling submit (not just relying on server rejection) for a custom date that is today or earlier (Acceptance Scenario 4). Depends on T024, T018, T026, T027.
- [ ] T029 [Web] **BLOCKED** [P] Add/extend `web/src/routes/tabs/settings/ShareLinks.test.tsx` component tests: each of the six expiry choices produces the correct request shape sent to the MSW mock (T019); "No expiry" shows the warning and results in `neverExpires: true`; a custom date 400 days out shows the long-lived warning and still submits; a custom date of today or earlier is blocked client-side before any request is sent; the list renders "Never" and never "Expired" for a NULL-expiry mock row. Depends on T026, T027, T028, T019.
- [ ] T030 [Web] **BLOCKED** — Update `web/e2e/specs/live/share-links.spec.ts` (currently sends `expiresIn: "24h"` at line 77) to additionally cover at least one preset, "No expiry", and a custom date through the redesigned dialog, per spec.md's Assumption ("E2E tier" — this is the sole browser e2e surface for share links, no new Go `test/e2e/` bucket). Depends on T028.
- [ ] T031 [Web] **BLOCKED** — Once T024-T030 land, remove the deprecated `expiresIn` path's exclusivity assumption from `web/src/lib/api.ts`/`ShareLinks.tsx` if any residual reference remains (the dialog should no longer send `expiresIn` at all after T028) — verify with `grep -rn expiresIn web/src/routes/tabs/settings/ web/src/lib/api.ts` and remove any leftover. Depends on T028.

**Checkpoint**: entire phase remains unstarted until its two preconditions clear; `tasks.md` records it so the work is planned and not forgotten, per the computed task's explicit instruction to mark it blocked rather than skip it.

---

## Dependencies & Execution Order

- **Phase 1** has no dependencies; run first.
- **Phase 2** depends on Phase 1 (T002's go/no-go). Internally: T003 → T004; T005 → T006/T007/T008 → T009 → T010/T011; T012 → T013 → T014 → T015; T016 last (build check across the whole phase).
- **Phase 3** depends on Phase 2 being far enough along that the wire shape is fixed (T012-T014 specifically) — T020 waits on the full Phase 2 checkpoint since it documents shipped behavior. (T017-T019, the web type/mock updates, no longer run in this phase — see OD-8, moved to Phase 5.)
- **Phase 4** (T022) has no hard dependency on Phase 2/3 completing (the draft text is already maintainer-approved), but is sequenced last among the unblocked phases because the draft's own header says "once implementation lands."
- **Phase 5** is blocked as a whole on an external precondition (PR #378 merge) and then on the design-first Pencil MCP step; it is not scheduled relative to Phases 2-4 by task order, only by that external gate. Phases 2-4 fully complete and are independently mergeable without Phase 5.
- **MVP** for this feature's non-UI value (an API that supports no-cap and no-expiry, still served today by the unmodified UI via the deprecated `expiresIn` path) = Phases 1-4.
- **Full feature** (FR-001..FR-004 UI-visible) additionally requires Phase 5, which cannot be scheduled until its blocker clears.

## Notes on tests that change (per the computed task's explicit ask)

- `api/internal/db/shares_test.go:120` and `:136` — the two `MaxShareLinkExpiryDays`-cap assertions — **are replaced**, not deleted outright without trace, by T010, per the maintainer's ruling recorded in spec.md's Assumptions section (sign-off already given there for this specific pair of tests; CLAUDE.md rule 1's "never weaken a test without explicit sign-off" is satisfied by that recorded sign-off — no further human approval gate is needed for T010 specifically).
- Any handler test asserting the old silent clamp (`shares.go:107-111`) — none exists as a *dedicated* test today per spec.md's Assumptions ("The handler's silent clamp... has no dedicated handler test today") — so T015 authors new tests rather than modifying an existing clamp test.
- `web/e2e/specs/live/share-links.spec.ts:77` is updated (not replaced) by T030, and only once Phase 5 unblocks.

## Left open (not resolved by this task list)

- **PR #378 merge status** is an external precondition this task list cannot verify or wait on; whoever executes Phase 5 must check it first.
- **Exact file location for the migration-010 Go test** (T004) — whether it belongs in `api/internal/db/shares_test.go` or a new file — is left to the implementer to decide by following whatever precedent (if any) exists for testing `004_cluster_rbac.sql`'s rebuild; this was not independently researched to keep this task list from asserting a fact about `004`'s test coverage that hasn't been verified.
- **`expiresIn` removal timeline** ("for one release") has no task here — removing it is explicitly out of scope for this feature (OD-1's ruling) and belongs to a future feature/task list once one release has shipped.
