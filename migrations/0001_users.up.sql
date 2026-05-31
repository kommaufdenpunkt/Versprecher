-- 0001_users.up.sql
-- Tabelle: users (§6). DDL wird als Superuser via psql ausgeführt; der App-User
-- erhält nur DML-Rechte (siehe scripts/dev_db.sh).

CREATE TABLE IF NOT EXISTS users (
    id                BIGSERIAL PRIMARY KEY,
    email             TEXT NOT NULL UNIQUE,
    email_verified_at TIMESTAMPTZ,
    password_hash     TEXT NOT NULL,
    display_name      TEXT NOT NULL,
    role              TEXT NOT NULL DEFAULT 'user'
                          CHECK (role IN ('user', 'moderator', 'admin')),
    status            TEXT NOT NULL DEFAULT 'active'
                          CHECK (status IN ('active', 'suspended', 'banned')),
    invited_by        BIGINT REFERENCES users(id),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Suche per E-Mail beim Login (case-insensitiv wird im Code über lower() gelöst).
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
