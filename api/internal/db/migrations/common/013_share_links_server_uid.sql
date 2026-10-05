-- Bind share links to the UID of the GameServer they were minted for, so a
-- delete+recreate under the same name inside one second (creationTimestamp has
-- 1s precision) cannot be mistaken for the original server.
-- Existing rows get the empty string: legacy links, validated by the
-- creationTimestamp check only. No backfill.
ALTER TABLE share_links ADD COLUMN server_uid TEXT NOT NULL DEFAULT '';
