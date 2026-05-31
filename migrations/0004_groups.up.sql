-- 0004_groups.up.sql
-- Phase 2: Gruppen (Kreise) + Mitglieder. Außerdem wird der in Phase 1 bewusst
-- ausgelassene Fremdschlüssel invitations.group_id -> groups.id nachgezogen (§6).

CREATE TABLE IF NOT EXISTS groups (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    owner_id    BIGINT NOT NULL REFERENCES users(id),
    max_members INT NOT NULL DEFAULT 30,  -- TODO: finalen Wert festlegen (§15)
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS group_members (
    group_id   BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT NOT NULL CHECK (role IN ('owner', 'member')),
    invited_by BIGINT REFERENCES users(id),
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (group_id, user_id)
);

-- Für "meine Gruppen" (Suche per user_id).
CREATE INDEX IF NOT EXISTS idx_group_members_user ON group_members (user_id);

-- Fremdschlüssel auf groups nachziehen (war in Phase 1 noch offen).
ALTER TABLE invitations
    ADD CONSTRAINT fk_invitations_group
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE;
