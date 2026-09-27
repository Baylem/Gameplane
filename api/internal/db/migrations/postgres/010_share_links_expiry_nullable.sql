-- PostgreSQL equivalent of migrations/sqlite/010_share_links_expiry_nullable.sql:
-- make share_links.expires_at nullable so a link can have no expiry at all
-- (NULL = never expires), per specs/done_017-share-link-expiry.
--
-- SQLite has to rebuild the table to loosen NOT NULL; Postgres can do it in
-- place. Existing expires_at values and all three indexes are untouched.
ALTER TABLE share_links ALTER COLUMN expires_at DROP NOT NULL;
