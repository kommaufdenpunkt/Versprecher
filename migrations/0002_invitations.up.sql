-- 0002_invitations.up.sql
-- Tabelle: invitations (§6). group_id ist in Phase 1 noch ohne FK auf groups
-- (die Tabelle entsteht in Phase 2). Der Fremdschlüssel auf groups wird in einer
-- Phase-2-Migration nachgezogen. token speichert den sha256-Hash des Rohtokens.

CREATE TABLE IF NOT EXISTS invitations (
    id          BIGSERIAL PRIMARY KEY,
    group_id    BIGINT,
    inviter_id  BIGINT NOT NULL REFERENCES users(id),
    token       TEXT NOT NULL UNIQUE,
    email       TEXT,
    status      TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'accepted', 'expired', 'revoked')),
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_invitations_status ON invitations (status);
