-- 0007_fahrstunden.up.sql
-- Fahrstunden-Nachweis: das Nebenbuch zum FS Manager.
--
-- Hintergrund: Der FS Manager lässt pro Tag nur eine begrenzte Arbeitszeit zu
-- (Standard 495 Minuten = 8 h 15 min). Eine Fahrstunde wird deshalb manchmal an
-- einem anderen Tag eingetragen als sie tatsächlich gefahren wurde. Damit die
-- Dokumentation trotzdem lückenlos bleibt, hält diese Tabelle BEIDE Daten fest:
--   gefahren_am    = Tag der tatsächlichen Fahrstunde
--   eingetragen_am = Tag, unter dem die Stunde im FS Manager verbucht ist
-- Das Tageslimit gilt auf eingetragen_am (das ist der Arbeitszeit-Tag im FS Manager).

CREATE TABLE IF NOT EXISTS fahrschueler (
    id            BIGSERIAL PRIMARY KEY,
    fahrlehrer_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    klasse        TEXT NOT NULL DEFAULT '',   -- Führerscheinklasse, z. B. B, BE, A
    notiz         TEXT NOT NULL DEFAULT '',
    aktiv         BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Jeder Fahrlehrer sieht nur die eigenen Schüler:innen; Name pro Fahrlehrer
-- eindeutig (verhindert versehentliche Doppelanlage beim schnellen Eintragen).
CREATE UNIQUE INDEX IF NOT EXISTS idx_fahrschueler_lehrer_name
    ON fahrschueler (fahrlehrer_id, lower(name));

CREATE TABLE IF NOT EXISTS fahrstunden (
    id                 BIGSERIAL PRIMARY KEY,
    fahrlehrer_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fahrschueler_id    BIGINT NOT NULL REFERENCES fahrschueler(id) ON DELETE RESTRICT,

    gefahren_am        DATE NOT NULL,
    eingetragen_am     DATE NOT NULL,
    -- Obergrenze 1440 ist reine Plausibilität (ein Kalendertag). Das echte
    -- Tageslimit (495) steckt in der Anwendung, weil es sich ändern kann.
    dauer_minuten      INT NOT NULL CHECK (dauer_minuten > 0 AND dauer_minuten <= 1440),
    art                TEXT NOT NULL DEFAULT 'uebungsstunde'
                           CHECK (art IN ('grundausbildung', 'uebungsstunde', 'ueberlandfahrt',
                                          'autobahnfahrt', 'nachtfahrt', 'pruefungsvorbereitung',
                                          'pruefungsfahrt', 'sonstiges')),
    notiz              TEXT NOT NULL DEFAULT '',

    -- Unterschrift des Fahrschülers, direkt auf dem Gerät erfasst (PNG als data-URL).
    unterschrift_png   TEXT,
    unterschrieben_am  TIMESTAMPTZ,

    -- Wurde das Tageslimit bewusst überschritten, wird das mit Grund festgehalten
    -- statt still erlaubt — sonst wäre der Nachweis nicht mehr lückenlos.
    limit_uebersteuert BOOLEAN NOT NULL DEFAULT false,
    limit_grund        TEXT NOT NULL DEFAULT '',

    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Trägt die Tageslimit-Prüfung: Summe der Minuten je Fahrlehrer und Eintragetag.
CREATE INDEX IF NOT EXISTS idx_fahrstunden_lehrer_eingetragen
    ON fahrstunden (fahrlehrer_id, eingetragen_am);

-- Nachweisliste je Fahrschüler, neueste Fahrt zuerst.
CREATE INDEX IF NOT EXISTS idx_fahrstunden_schueler_gefahren
    ON fahrstunden (fahrschueler_id, gefahren_am DESC, id DESC);

-- Gesamtübersicht des Fahrlehrers nach Fahrtag.
CREATE INDEX IF NOT EXISTS idx_fahrstunden_lehrer_gefahren
    ON fahrstunden (fahrlehrer_id, gefahren_am DESC, id DESC);
