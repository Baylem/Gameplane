-- Make share_links.expires_at nullable so a link can have no expiry at all
-- (NULL = never expires), per specs/done_017-share-link-expiry OD-1/OD-3/OD-4.
--
-- SQLite has no ALTER TABLE ... ALTER COLUMN, so the column can't be
-- loosened from NOT NULL to nullable in place on that driver. Rebuild the
-- table instead, following the same create/copy/drop/rename sequence as
-- 004_cluster_rbac.sql, which is portable to Postgres too. Nothing
-- FK-references share_links, so the drop is clean on both drivers.
--
-- Column order matches 006_share_links.sql with 009_share_links_cluster.sql's
-- trailing `cluster` column appended, and both the INSERT and SELECT name
-- every column explicitly (no SELECT *) so the copy is correct regardless of
-- column order. Existing expires_at values are copied verbatim, never
-- recomputed.
CREATE TABLE share_links_new (
    id           TEXT PRIMARY KEY,
    namespace    TEXT NOT NULL,
    server_name  TEXT NOT NULL,
    created_by   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    can_start    INTEGER NOT NULL DEFAULT 0,
    token_hash   TEXT NOT NULL UNIQUE,
    expires_at   TEXT,
    revoked_at   TEXT,
    created_at   TEXT NOT NULL,
    last_used    TEXT,
    cluster      TEXT NOT NULL DEFAULT 'local'
);

INSERT INTO share_links_new (id, namespace, server_name, created_by, can_start, token_hash, expires_at, revoked_at, created_at, last_used, cluster)
    SELECT id, namespace, server_name, created_by, can_start, token_hash, expires_at, revoked_at, created_at, last_used, cluster FROM share_links;

DROP TABLE share_links;

ALTER TABLE share_links_new RENAME TO share_links;

CREATE INDEX idx_share_links_token ON share_links(token_hash);
CREATE INDEX idx_share_links_server ON share_links(namespace, server_name);
CREATE INDEX idx_share_links_cluster_server ON share_links(cluster, namespace, server_name);
