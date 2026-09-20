<!--
Draft only — NOT yet applied to docs/security.md. Per the maintainer's 2026-09-19
instructions, this section text is prepared here for review, to be spliced into
docs/security.md (placement: after "## Authorization"; security.md has no
existing share-link or API-token section) once the done_017-share-link-expiry
implementation lands.
-->

## Share links

Share links (`api/internal/db/shares.go`, schema in `api/internal/db/migrations/006_share_links.sql`) grant unauthenticated, token-bearing access to a single GameServer's status and connection address, optionally with permission to wake it. Because they are unauthenticated, their security rests entirely on the token being both hard to guess and hard to recover if the database or a backup leaks.

**Storage.** Only a SHA-256 hash of the token is persisted (`token_hash`, indexed for O(1) lookup); the raw 32-byte random token is generated at creation, returned exactly once in the create response, and never stored, logged, or recoverable afterwards.

**Why a fast hash is the right choice here, not a weakness.** SHA-256 is not a password-hashing function (no salt, no work factor), and that is deliberate:

- The input is a 32-byte cryptographically random value, not a low-entropy human-chosen secret. There is no dictionary or brute-force search over a keyspace of 2^256 that a fast hash makes newly feasible — the security margin comes entirely from the token's entropy, not from how expensive the hash is to compute.
- A database or backup leak exposes only hashes. Recovering a usable token from a hash would require reversing SHA-256 or brute-forcing a 256-bit random space; both are computationally infeasible regardless of the hash function's speed. A slow KDF (bcrypt/argon2/scrypt) would add no meaningful protection here, since the "weak secret" scenario those algorithms defend against does not apply.
- The token is looked up on every public request to `GET /shares/{token}` and `POST /shares/{token}/start`. A fast, indexed equality lookup keeps that path cheap; a deliberately slow KDF would turn every anonymous share-page load into an expensive hashing operation — the opposite of what a slow KDF is for (limiting an attacker's guesses per second), since here the attacker's limiting factor is token entropy, not hash speed.

**Indistinguishability.** An unknown token, an expired token, and a revoked token all produce the identical response (`ErrShareLinkInvalid` internally; a uniform 404 at the HTTP layer). This prevents an attacker from learning, by probing, whether a guessed token ever existed, expired, or was deliberately revoked.

**Token placement.** The token travels as a path segment (`/shares/{token}`), not a query parameter, keeping it out of query-string capture by proxies, access logs and analytics, and letting audit logging redact it from recorded request paths with a simple prefix rewrite (`redactShareToken` in `api/internal/audit/audit.go`). A path segment is still part of the URL, so it does not by itself keep the token out of `Referer` headers; API responses set `Referrer-Policy: no-referrer` (`secureHeaders` in `api/cmd/main.go`) for that.

**Expiry.** Every share link either has an expiry timestamp or is explicitly created with no expiry (owner's choice; see `specs/done_017-share-link-expiry/`). There is no platform-enforced maximum lifetime: a non-expiring or long-lived link is exactly as hard to guess on any given day as a short-lived one, because guessing difficulty comes from the token's entropy, not from its age. The tradeoff of a long-lived or non-expiring link is operational — a forgotten link stays live until the owner revokes it — not cryptographic, which is why the create-link UI warns the owner explicitly ("This link works until you revoke it." for no expiry; a long-lived-token warning for a custom date a year or more out) rather than the system silently capping the choice.

**Revocation.** Revocation sets `revoked_at` (never a delete, preserving the audit trail) and is checked independently of, and prior to, any expiry check, so it applies uniformly regardless of whether the link expires, expires far in the future, or never expires.

**Rate limiting.** The public resolve/start endpoints are rate-limited (`auth.ShareLimiter`) specifically because tokens are guessable-by-brute-force in principle (just computationally infeasible in practice); the rate limit is defense in depth against automated probing, not a substitute for token entropy.
