-- 0003_email_verifications.up.sql
-- Implementierungs-Ergänzung zu Phase 1 (nicht explizit im §6-Modell):
-- Tokens für die E-Mail-Verifizierung. Gespeichert wird nur der sha256-Hash.

CREATE TABLE IF NOT EXISTS email_verifications (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_email_verifications_user ON email_verifications (user_id);
