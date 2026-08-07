-- 0008_fahrstunden_uhrzeiten.up.sql
-- Uhrzeiten zu beiden Daten: der Nachweis soll belegen können
--   „gefahren am 09.08.2026 um 14:00 Uhr, eingetragen am 08.08.2026 um 19:30 Uhr“.
--
-- Beide Felder sind bewusst optional (NULL erlaubt): Altbestand bleibt gültig,
-- und eine Stunde ohne notierte Uhrzeit ist immer noch ein gültiger Eintrag.
-- Die Endzeit wird nicht gespeichert, sondern aus gefahren_von + dauer_minuten
-- berechnet — so kann sie nie von der Dauer abweichen.

ALTER TABLE fahrstunden
    ADD COLUMN IF NOT EXISTS gefahren_von   TIME,
    ADD COLUMN IF NOT EXISTS eingetragen_um TIME;

COMMENT ON COLUMN fahrstunden.gefahren_von IS
    'Beginn der tatsächlichen Fahrstunde; Ende = gefahren_von + dauer_minuten';
COMMENT ON COLUMN fahrstunden.eingetragen_um IS
    'Uhrzeit, unter der die Stunde im FS Manager verbucht ist';
