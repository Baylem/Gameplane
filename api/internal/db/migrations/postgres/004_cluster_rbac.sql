-- PostgreSQL equivalent of migrations/sqlite/004_cluster_rbac.sql: add a
-- cluster dimension to user_role_bindings and make it part of the primary
-- key. Existing rows backfill to 'local' (the default cluster), NOT '*'.
--
-- Postgres can alter a primary key in place, so this skips SQLite's
-- create/copy/drop/rename rebuild; the resulting columns, defaults, key
-- and idx_user_role_bindings_user index are the same.
ALTER TABLE user_role_bindings ADD COLUMN cluster TEXT COLLATE "C" NOT NULL DEFAULT 'local';

ALTER TABLE user_role_bindings DROP CONSTRAINT user_role_bindings_pkey;

ALTER TABLE user_role_bindings ADD PRIMARY KEY (user_id, role_name, cluster, namespace);
