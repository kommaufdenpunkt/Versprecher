-- 0004_groups.down.sql
ALTER TABLE invitations DROP CONSTRAINT IF EXISTS fk_invitations_group;
DROP TABLE IF EXISTS group_members;
DROP TABLE IF EXISTS groups;
