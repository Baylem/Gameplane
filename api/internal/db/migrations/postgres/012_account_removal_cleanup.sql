-- PostgreSQL equivalent of migrations/sqlite/012_account_removal_cleanup.sql.
-- Dialect change only: revoked_at comes from to_char(now() AT TIME ZONE 'UTC')
-- in the same RFC3339 UTC text ("YYYY-MM-DDTHH:MM:SSZ") that SQLite's
-- strftime('%Y-%m-%dT%H:%M:%SZ', 'now') writes and shares.go parses.
--
-- One-off cleanup of rows that belong to users who no longer exist. On
-- Postgres the ON DELETE CASCADE foreign keys already removed the account
-- rows when the user was deleted, and share_links.created_by has no foreign
-- key (see 006_share_links.sql), so its links are revoked, not deleted,
-- exactly as on SQLite. Every statement normally matches nothing here. The
-- file exists so both drivers record the same schema_migrations versions.
DELETE FROM oidc_links WHERE user_id NOT IN (SELECT id FROM users);

DELETE FROM user_preferences WHERE user_id NOT IN (SELECT id FROM users);

DELETE FROM sessions WHERE user_id NOT IN (SELECT id FROM users);

DELETE FROM api_tokens WHERE user_id NOT IN (SELECT id FROM users);

DELETE FROM user_role_bindings WHERE user_id NOT IN (SELECT id FROM users);

UPDATE share_links SET revoked_at = to_char(now() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
    WHERE revoked_at IS NULL AND created_by NOT IN (SELECT id FROM users);
