# Open Decisions: 017-share-link-expiry

Unsettled items not fixed by the maintainer's 2026-09-19 chat decisions. Do not treat any of the below as decided until a maintainer rules on it here.

Status of every item below: **Open** (2026-09-19). Record each ruling here, with its date, before or alongside the PR that implements it.

## Rulings (maintainer, 2026-09-19)

- **OD-1, OD-3, OD-5, OD-7 — Settled:** the create request carries exactly one of `expiresAt` (RFC3339 instant) or `neverExpires: true`; a request with neither (or both) is rejected with 400, so a missing field never creates a permanent link. The client computes preset instants (today + N days) and custom-date instants. In Go, `ShareLink.ExpiresAt` becomes `*time.Time` (nil = never), like `RevokedAt`/`LastUsed`; the response's `expiresAt` is an RFC3339 string or `null` (UI shows "Never"). The old `expiresIn` duration field keeps working for one release as deprecated (its existing 7-day default for an omitted value stays only on that legacy path), then is removed.
- **OD-2 — Settled:** a custom date is valid through the end of the chosen calendar day in the owner's local time (23:59:59 local), converted to an instant by the client.
- **OD-4 — Settled:** table rebuild in `010_share_links_expiry_nullable.sql` following `004_cluster_rbac.sql` (create `_new`, copy with named columns, drop, rename, recreate the three indexes), meeting every constraint listed under OD-4.
- **OD-6 — Settled:** the warning shows when the custom date is 365 days or more from today, with the text "Long-lived link — it stays valid for over a year unless you revoke it."

## OD-1: Request/response shape for expiry

Today `createShareReq.ExpiresIn *string` takes a Go duration string (`"24h"`, `"168h"`) and the handler computes an absolute `expiresAt` server-side, capped at 90 days. With presets up to 90 days, "no expiry", and an arbitrary custom date, a duration string is awkward for "no expiry" (what string means never?) and for a custom calendar date (a duration drifts with request latency; a date does not).

Options:
- **A**: Replace `expiresIn` with an optional absolute `expiresAt` (RFC3339 or date-only string) sent by the client; omitted/null means no expiry. The client computes the preset's absolute date itself (today + N days) or the picked custom date.
- **B**: Keep `expiresIn` for the four day-count presets (map to `"360h"`, `"720h"`, `"1440h"`, `"2160h"` or similar) and add a second optional field, e.g. `expiresAt`, used only for "Custom"; a third value (e.g. omit both, or `expiresIn: "never"`) signals "no expiry".
- **C**: Send a discriminated field, e.g. `expiry: { kind: "preset" | "custom" | "never", days?: number, date?: string }`.

This spec does not pick one, and neither does the implementer: the maintainer rules, and the ruling is recorded here before FR-009 is built. Also open under this item: whether a preset of N days is computed as N×24h from the request time or as a calendar date, and whether the client or the server computes it.

## OD-2: Custom date time-of-day cutover

The maintainer ruled "any date after today" but not which instant of that date the link stops working. Options: end of the selected day in the owner's local time (link valid through the whole calendar day), midnight local time at the start of that day, or midnight UTC on that date. Needs a maintainer ruling before FR-003/FR-004 are implemented, since it affects both the UI's date-to-timestamp conversion and any test fixtures that assert an exact `expiresAt`.

## OD-3: Representing "no expiry" on the wire and in SQL

FR-008 requires the DB column to accept NULL. Two sub-questions remain open:
- Whether `ShareLink.ExpiresAt` in Go becomes `*time.Time` (nil = never) or keeps `time.Time` with a documented zero-value sentinel. `*time.Time` is more idiomatic and matches `RevokedAt`/`LastUsed`'s existing `*time.Time` pattern in the same struct, but touches every call site that currently assumes `ExpiresAt` is always set (`shareResp.ExpiresAt string` formatting in `api/internal/handlers/shares.go`, `formatDate`/`getLinkStatus` in `web/src/routes/tabs/settings/ShareLinks.tsx`).
- Whether the JSON field is omitted, sent as `null`, or sent as an explicit sentinel string (e.g., `"never"`) when there is no expiry. `web/src/types.ts`'s `ShareLink.expiresAt` is currently `string` (non-optional); this needs to become `string | null` (or `string | undefined`) and every reader (`ShareLinks.tsx`'s `formatDate`, `getLinkStatus`) updated to handle the no-expiry case before rendering "Never" / suppressing "Expired".

## OD-4: Migration mechanics for a nullable `expires_at`

Facts (verified against the code):
- `006_share_links.sql:23` declares `expires_at TEXT NOT NULL`; `009_share_links_cluster.sql` added `cluster TEXT NOT NULL DEFAULT 'local'` as the last column and `idx_share_links_cluster_server`.
- The runner (`api/internal/db/db.go:68-104`) applies every `migrations/*.sql` file, as written, to whichever driver is configured; there are no per-driver files. Statements are split on `;\n` (`splitStatements`), and each file runs in one transaction.
- SQLite has no `ALTER TABLE ... ALTER COLUMN` at all, so Postgres's `ALTER COLUMN expires_at DROP NOT NULL` cannot be the migration; the file must be SQL both drivers accept.
- `004_cluster_rbac.sql` is the precedent: create `<table>_new`, `INSERT ... SELECT`, `DROP TABLE`, `ALTER TABLE ... RENAME TO`. No other table references `share_links`, so the drop is clean on both drivers.

Constraints any rebuild must meet: name the columns in both the `INSERT` and the `SELECT` (column order differs from `006` because of `009`); keep `created_by ... REFERENCES users(id) ON DELETE CASCADE`, `token_hash ... UNIQUE`, `cluster NOT NULL DEFAULT 'local'`, and every other `NOT NULL`; recreate `idx_share_links_token`, `idx_share_links_server` and `idx_share_links_cluster_server` after the rename (dropping the old table drops them on both drivers, and Postgres index names are schema-wide, so they cannot be created on `_new` while the old ones exist).

Open: the file name (next free number is `010`, e.g. `010_share_links_expiry_nullable.sql`, unless another migration reaches master first — none is on any remote branch today), and whether the maintainer accepts a table rebuild or prefers the runner to gain per-driver migration files.

## OD-5: Whether `expiresIn`/relative durations are removed from the public API entirely

Depends on OD-1. If Option A is chosen, `ExpiresIn` disappears from the wire contract, which is a breaking change to anyone scripting against the current API (there is no versioning scheme evident in `api/specs.md` for this endpoint). Needs a maintainer call on whether backward compatibility for the old field is required for one release.

## OD-6: The "1 year or more" warning for a custom date

The maintainer ruled that a warning appears when the custom date is 1 year or more away. Not ruled: whether "1 year" means 365 days, 366 in a leap span, or the same calendar date next year; and the warning's exact wording (only the No-expiry copy, "This link works until you revoke it.", was given).

## OD-7: API behaviour when a create request carries no expiry

Today an omitted or empty `expiresIn` gives a 7-day link (`api/internal/handlers/shares.go:97-98`), while the dialog's new pre-selected choice is 30 days. Not ruled: whether the API default moves to 30 days, stays 7 days, or whether an omitted field is rejected. Under OD-1 option A an omitted field would mean "no expiry", which turns a missing field into a never-expiring credential; that needs an explicit ruling rather than falling out of the wire shape.
