-- PostgreSQL equivalent of migrations/sqlite/011_user_theme_preferences.sql.
-- Dialect changes only: BIGINT user_id (matches users.id) and a to_char
-- updated_at default producing the same "YYYY-MM-DD HH:MM:SS" UTC text.
-- custom_css_enabled stays INTEGER (0/1): the Go code reads and writes an int.
--
-- User theme preferences: per-user dashboard styling choices (preset theme,
-- custom colors, custom CSS overlay) for specs/done_016-user-theme-customization.
--
-- Migration rule: accounts created before this migration are explicitly
-- initialized with the legacy orange & dark theme (INSERT ... SELECT below)
-- so an upgrade preserves the pre-feature appearance. Accounts created
-- afterwards rely on the column defaults: pink preset, system appearance,
-- overlay off. custom_accent / custom_surface / custom_css are nullable and
-- are nulled only by the reset endpoint, never by ordinary updates (FR-012).
--
-- modernc-sqlite runs with foreign_keys OFF, so the REFERENCES / ON DELETE
-- clause is enforced only under Postgres; users.go deletes a user's rows
-- explicitly anyway, matching the convention noted in 003_roles.sql.
CREATE TABLE user_preferences (
    user_id            BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    theme_type         TEXT NOT NULL DEFAULT 'preset',
    preset_id          TEXT NOT NULL DEFAULT 'pink',
    appearance_mode    TEXT NOT NULL DEFAULT 'system',
    custom_accent      TEXT,
    custom_surface     TEXT,
    custom_css_enabled INTEGER NOT NULL DEFAULT 0,
    custom_css         TEXT,
    updated_at         TEXT NOT NULL DEFAULT (to_char(now() AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS'))
);

CREATE INDEX idx_user_preferences_user ON user_preferences(user_id);

-- Backfill every pre-existing user with the legacy preset.
INSERT INTO user_preferences (user_id, theme_type, preset_id, appearance_mode)
    SELECT id, 'preset', 'legacy', 'system'
    FROM users;
