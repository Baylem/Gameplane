-- Sessions now store a SHA-256 digest of the browser session value instead
-- of the value itself (see auth.sessionDigest). Rows written before this
-- migration hold raw cookie values, which can no longer be matched, so they
-- are deliberately invalidated: every user signs in again once after the
-- upgrade. Rows created afterwards hold digests.
DELETE FROM sessions;
