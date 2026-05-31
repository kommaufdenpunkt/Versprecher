-- 0005_posts.up.sql
-- Phase 3: Beiträge (Versprecher/Verhörer) — §6.
-- ai_status/ai_claimed_at sind eine Implementierungs-Ergänzung für die
-- Fidolin-Pipeline (Polling-Worker, sicheres Claiming). Neue Posts starten als
-- 'pending_review' (NICHT im Feed sichtbar), bis Fidolin sie geprüft hat
-- (fail-closed). Erst eine unauffällige Prüfung schaltet sie auf 'visible'.

CREATE TABLE IF NOT EXISTS posts (
    id                  BIGSERIAL PRIMARY KEY,
    group_id            BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    author_id           BIGINT NOT NULL REFERENCES users(id),
    tagged_user_id      BIGINT REFERENCES users(id),
    word                TEXT NOT NULL,
    word_normalized     TEXT NOT NULL,
    explanation         TEXT NOT NULL DEFAULT '',
    kind                TEXT CHECK (kind IN ('versprecher', 'verhoerer')),
    ai_kind_suggestion  TEXT,
    ai_meant_suggestion TEXT,
    meant_confirmed     TEXT,
    voice_url           TEXT,
    ai_score            NUMERIC(3,2),
    status              TEXT NOT NULL DEFAULT 'pending_review'
                            CHECK (status IN ('visible', 'pending_review', 'blocked')),
    is_pinned           BOOLEAN NOT NULL DEFAULT false,
    -- Fidolin-Pipeline:
    ai_status           TEXT NOT NULL DEFAULT 'queued'
                            CHECK (ai_status IN ('queued', 'processing', 'analyzed', 'failed')),
    ai_claimed_at       TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Feed: sichtbare Beiträge je Gruppe, neueste zuerst (Keyset über id).
CREATE INDEX IF NOT EXISTS idx_posts_group_visible
    ON posts (group_id, id DESC) WHERE status = 'visible';

-- Fidolin: schnelle Suche nach offenen Jobs.
CREATE INDEX IF NOT EXISTS idx_posts_ai_open
    ON posts (created_at) WHERE ai_status IN ('queued', 'processing');

-- Aggregation (Phase 6) über das normalisierte Wort.
CREATE INDEX IF NOT EXISTS idx_posts_word_normalized ON posts (word_normalized);
