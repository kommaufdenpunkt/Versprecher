-- 0006_moderation_settings.up.sql
-- Schwellen für Fidolin (§7). Genau eine Zeile (id = 1). Die Schreib-Endpoints
-- für Moderatoren folgen in Phase 7; hier wird die Zeile mit Defaults angelegt.

CREATE TABLE IF NOT EXISTS moderation_settings (
    id                    INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    auto_reject_threshold NUMERIC(3,2) NOT NULL DEFAULT 0.85,
    handoff_threshold     NUMERIC(3,2) NOT NULL DEFAULT 0.60
);

INSERT INTO moderation_settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING;
