-- 0008_fahrstunden_uhrzeiten.down.sql
ALTER TABLE fahrstunden
    DROP COLUMN IF EXISTS gefahren_von,
    DROP COLUMN IF EXISTS eingetragen_um;
