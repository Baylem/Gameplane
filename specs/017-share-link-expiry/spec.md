# Feature Specification: Configurable Share Link Expiry

**Feature Branch**: `017-share-link-expiry`

**Created**: 2026-09-19

**Status**: Draft

**Input**: Maintainer decision (2026-09-19, in chat): replace the fixed 90-day maximum on share links with a richer set of expiry choices, including an option that never expires and a custom date with no upper bound.

## Clarifications

### Session 2026-09-19 (maintainer, in chat)

- Q: What expiry choices should the create-link dialog offer? → A: 15 days, 30 days, 60 days, 90 days, No expiry, Custom (date picker). 30 days is pre-selected.
- Q: Is "No expiry" allowed, and how is it surfaced? → A: Allowed. Selecting it shows the warning "This link works until you revoke it." The Share links list shows "Never" in the Expires column for such links.
- Q: What are the bounds on a custom date? → A: Any date after today, no maximum. A warning is shown when the chosen date is 1 year or more away, flagging it as a long-lived token. (How "1 year" is measured and the warning's wording were not ruled on — see OD-6.)
- Q: Does `MaxShareLinkExpiryDays` (today's hard cap of 90 days, `api/internal/db/shares.go`) survive? → A: No — it is removed from both the handler and the store. Expiry validation is limited to "must be in the future when set"; NULL (no expiry) bypasses that check entirely.
- Q: Why is it safe to keep hashing tokens with SHA-256 with no expiry cap? → A: See "Background: token storage" below; this spec does not reopen that decision, only records the rationale for docs/security.md.

## Background: token storage (unchanged, recorded for docs/security.md)

Share tokens are stored only as their SHA-256 hash (`api/internal/db/migrations/006_share_links.sql`, introduced in commit `1e1ecc20`). The raw token is generated, returned exactly once in the create response, and never persisted or logged. This spec does not change that design; it only removes the maximum-lifetime cap that today complements it. The rationale, to be copied into `docs/security.md`:

- The share token is a bearer credential: anyone holding the raw string gets the access the link grants (view, and optionally start), with no separate password or session.
- Because only the hash is stored, a database or backup leak does not by itself expose usable tokens — the attacker would still need to reverse a SHA-256 hash of a high-entropy random value, which is computationally infeasible.
- A fast, unsalted hash (rather than a slow KDF like bcrypt/argon2) is an acceptable and correct choice here specifically *because* the input is a 32-byte cryptographically random token, not a low-entropy user-chosen secret. Fast hashing also enables an indexed equality lookup (`idx_share_links_token`) for O(1) resolution on every public request; a slow KDF would make that lookup path expensive by design (which is precisely why it is right for passwords and wrong here).
- Unknown, expired, and revoked tokens are made indistinguishable to the caller (`ErrShareLinkInvalid`, uniform 404 responses) so that probing the endpoint reveals nothing about whether a given token ever existed.
- The token travels as a path segment (`/shares/{token}`), not a query parameter, keeping it out of query-string capture (proxy/access logs, analytics) and letting audit logging redact it from recorded paths with a prefix rewrite (`redactShareToken`, `api/internal/audit/audit.go:764`). A path segment is still part of the URL, so it is not by itself a `Referer` defence; that comes from `Referrer-Policy: no-referrer`, which `secureHeaders` sets on API responses (`api/cmd/main.go:640`).
- Removing the fixed 90-day cap does not change any of the above: a "no expiry" link is exactly as hard to guess on day 3650 as it is on day 1, because guessing difficulty comes from token entropy, not from how long the token remains valid. The tradeoff of a long-lived or non-expiring link is operational (a forgotten link stays live until revoked), not cryptographic, which is why the UI carries the burden of warning the owner instead of the system enforcing a cap.

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Choose an expiry that matches how the link will be used (Priority: P1)

A server owner creating a share link wants an expiry that matches the situation: a quick 15-day link for a weekend event, a 30-day default for a friend's ongoing visit, a 90-day link for a longer arrangement, a link with no expiry for a link they intend to manage manually, or a specific end date tied to something else (e.g., a LAN party six weeks out).

**Why this priority**: This is the entire feature — without it, the owner is stuck with the current fixed menu and 90-day ceiling.

**Independent Test**: Open the create-link dialog, select each of the six expiry choices in turn, and verify the created link's `expiresAt` (or absence of one) matches the choice, and that the correct warning text appears for "No expiry" and for a custom date 1 year or more out (threshold per OD-6).

**Acceptance Scenarios**:

1. **Given** the create-link dialog, **When** the owner opens the expiry selector, **Then** they see exactly: 15 days, 30 days, 60 days, 90 days, No expiry, Custom, with 30 days pre-selected.
2. **Given** the owner selects "No expiry", **When** they view the form, **Then** the warning "This link works until you revoke it." is shown, and submitting creates a link with no expiry.
3. **Given** the owner selects "Custom" and picks a date 400 days from today, **When** they view the form, **Then** a long-lived-token warning is shown alongside the chosen date, and submitting still succeeds.
4. **Given** the owner selects "Custom" and picks today's date or an earlier date, **When** they attempt to submit, **Then** the UI rejects it before the request is sent (date must be strictly after today).
5. **Given** a link created with no expiry, **When** the owner views the Share links list, **Then** the Expires column reads "Never" and the link never reports "Expired" status.

---

### User Story 2 - Existing links are unaffected (Priority: P1)

An operator upgrading Gameplane must not have any existing share link's behavior change: links created under the old fixed-menu, 90-day-capped system keep exactly the expiry timestamp they were given.

**Why this priority**: Silent expiry changes on already-distributed links would be a correctness and trust regression.

**Independent Test**: Seed a share link under the pre-migration schema shape (non-null `expires_at`), run the new migration, and confirm `LookupShareLink`/`ListShareLinks` return the same `expires_at` unchanged and the link still resolves or expires exactly as before.

**Acceptance Scenarios**:

1. **Given** a share link row with a non-null `expires_at` created before this feature, **When** the migration runs, **Then** the row's `expires_at` value is preserved exactly (new append-only migration; existing values are copied verbatim, never recomputed).
2. **Given** that pre-existing link is now past its original `expires_at`, **When** it is looked up, **Then** it still resolves as expired (`ErrShareLinkInvalid`), unchanged from today's behavior.

---

### User Story 3 - Revocation still works for every expiry shape (Priority: P2)

An owner must be able to revoke any link — expiring, custom-dated, or one with no expiry — at any time, and revocation must take priority over expiry state.

**Independent Test**: Create one link of each expiry shape (short, long, custom, none), revoke each, and confirm all four immediately fail lookup with the same invalid-link response.

**Acceptance Scenarios**:

1. **Given** a link with no expiry, **When** the owner revokes it, **Then** subsequent lookups return the same indistinguishable invalid response as any other revoked link.

---

### Edge Cases

- **Existing links keep their expiry.** Covered by User Story 2. The migration only widens the column's nullability; if it rebuilds the table (see FR-008 / OD-4) it copies every existing `expires_at` value verbatim and never recomputes one.
- **NULL expiry never expires.** `LookupShareLink`'s expiry check (`time.Now().After(link.ExpiresAt)`) must be skipped entirely when `ExpiresAt` is unset/NULL, on both SQLite and Postgres.
- **Revocation still works.** Revocation is a separate column (`revoked_at`) and is checked independently of expiry; a NULL-expiry link is revoked exactly the same way as any other link (User Story 3).
- **Clock skew.** Expiry comparisons continue to use the API server's own clock (`time.Now()`), consistent with today's behavior; this feature does not introduce a new skew source since NULL expiry has no comparison to skew.
- **API rejects past dates.** `CreateShareLink` must continue to reject a non-NULL `expiresAt` that is not strictly after "now" (`ErrShareLinkExpiryInvalid`), for both preset and custom choices. There is no longer a maximum-date check.
- **UI local-time vs UTC.** The custom date picker collects a date in the browser's local time zone; the API stores and compares UTC RFC3339 timestamps (unchanged convention, see `006_share_links.sql` header comment). Which instant of the chosen calendar day becomes the stored UTC `expiresAt` (end of day local, midnight local, midnight UTC, ...) is not ruled on — see OD-2. Whatever the ruling, the conversion must happen once and be covered by tests that pin the exact timestamp.
- **"No expiry" and the public Share page.** The public resolve/start endpoints do not change: they already treat "not expired" as one of several validity conditions checked in the store, so a NULL expiry simply always satisfies that condition.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The create-link dialog MUST offer exactly six expiry choices: 15 days, 30 days, 60 days, 90 days, No expiry, and Custom (date), with 30 days selected by default.
- **FR-002**: Selecting "No expiry" MUST display the warning "This link works until you revoke it." and MUST create a link whose expiry is unset (NULL / never).
- **FR-003**: Selecting "Custom" MUST present a date picker restricted to dates strictly after today, with no maximum date.
- **FR-004**: When the custom date chosen is 1 year or more from today, the UI MUST display a long-lived-token warning in addition to the chosen date. The exact threshold measurement and warning copy are open (OD-6).
- **FR-005**: The store (`CreateShareLink`) MUST accept a NULL/unset expiry and persist it as such, and MUST continue to reject any provided expiry that is not strictly in the future. `MaxShareLinkExpiryDays` and its associated maximum-expiry check MUST be removed from both the handler and the store.
- **FR-006**: `LookupShareLink` MUST treat a NULL expiry as "never expires" — it MUST NOT be compared against the current time — while continuing to reject the link if it is revoked.
- **FR-007**: `ListShareLinks` MUST return links with a NULL expiry with that fact represented in a way the API response and the UI can render as "Never" in the Expires column, and MUST NOT report such a link as "Expired".
- **FR-008**: A new, append-only migration (next free number, today `010_*.sql`; `001`–`009` are never edited) MUST widen `share_links.expires_at` to allow NULL without altering any existing non-null `expires_at` value. The migration runner applies each `.sql` file unchanged to both SQLite and Postgres (`api/internal/db/db.go:68-104`, no per-driver files), and SQLite has no `ALTER COLUMN`, so the one file MUST use SQL both drivers accept — in practice the create/copy/drop/rename rebuild that `004_cluster_rbac.sql` already uses. Mechanics and constraints are in OD-4.
- **FR-009**: The API request/response contract (`api/internal/handlers/shares.go`, `web/src/types.ts`, `web/src/lib/api.ts`) MUST represent "no expiry" and a caller-chosen absolute date, not only a relative duration string. The existing `expiresIn` relative-duration field's fate (kept alongside a new field, or replaced by an absolute-date field) is an open decision (OD-1, OD-5), as is what the API does when the request carries no expiry at all (today: 7 days server-side, `api/internal/handlers/shares.go:97-98`; see OD-7).
- **FR-010**: Revocation (`RevokeShareLink`) MUST behave identically regardless of the link's expiry shape (short, long, custom, or none).
- **FR-011**: `docs/security.md` MUST gain a "Share links" section recording the token-hashing rationale (see Background above) so the removal of the expiry cap is documented alongside why it remains safe.
- **FR-012**: The audit trail's existing behavior of redacting the token from recorded paths MUST be unaffected by this change (no new logging of expiry values is required, but none may leak the token).

### Key Entities

- **ShareLink** (`api/internal/db/shares.go`): gains the ability for `ExpiresAt` to represent "no expiry" (a nil/zero-value sentinel in Go, NULL in the database) instead of always holding a mandatory future timestamp.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An owner can create a share link with no expiry, and it remains resolvable (not reported invalid due to expiry) indefinitely until explicitly revoked.
- **SC-002**: An owner can create a share link with a custom date more than 90 days out (e.g., 200 days), which was previously impossible (hard 90-day cap), and it resolves correctly up to and expires after that date.
- **SC-003**: 100% of share links created before this feature ships continue to expire at their original timestamp, unchanged.
- **SC-004**: The Share links list renders "Never" for every no-expiry link and never labels one "Expired".

## Assumptions

- **Feature number.** `016-user-theme-customization` is already on master, so this feature is `017`. Branch `017-share-link-expiry`, worktree `/home/valgul/project/Gameplane-017` (maintainer decision).
- **Design first.** The create dialog `atqRh` (its "Maximum 90 days" help text and the expiry Select) and the Share links screen `xCJlu` (Expires column showing "Never") are changed in `design.pen` through the Pencil MCP and exported to `design-export/` before any React change (CLAUDE.md rule 1). The original UI contract is `specs/014-heroui-web-rebuild/contracts/share-link-ui.md`.
- **Tests that enforce today's cap.** `api/internal/db/shares_test.go:120` (90+1 days rejected) and `:136` (exactly 90 days accepted) reference `MaxShareLinkExpiryDays`. Replacing them needs maintainer sign-off (CLAUDE.md rule 1); they are replaced by tests of the new behaviour, not deleted. The handler's silent clamp (`api/internal/handlers/shares.go:107-111`) has no dedicated handler test today.
- **E2E tier.** Browser E2E for share links is the Playwright live spec `web/e2e/specs/live/share-links.spec.ts` (it sends `expiresIn: "24h"` at line 77), per feature 014's OD-3 ruling; no Go `test/e2e/` bucket is added.
- **Out of scope.** The public Share page (`web/src/routes/Share.tsx`) and `ShareLinkPublic` carry no expiry field and do not change. The token-hashing design (`006_share_links.sql`, commit `1e1ecc20`) does not change.
