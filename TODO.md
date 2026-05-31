# Versprecher – Offene Punkte (TODO)

Offene Entscheidungen aus der Spezifikation (Abschnitt 15). Bitte vor bzw.
während der jeweiligen Roadmap-Phase klären.

| # | Thema | Entscheidung / Frage | Betrifft Phase |
|---|-------|----------------------|----------------|
| 1 | **Tagline** | Final wählen. Favorit: „Heute schon verhört?" | – (Branding) |
| 2 | **Wort des Monats** | Pro Gruppe (Kern) – zusätzlich **global** im öffentlichen Duden? | 8 (ggf. 6) |
| 3 | **Gruppenlimit** | Exaktes `max_members` „am Gefühl" justieren | 2 |
| 4 | **Ähnlichkeits-Matching** | Aggregation erst **exakt** über `word_normalized`, später ggf. **Fuzzy (Levenshtein)** | 6 |
| 5 | **Voice-Storage** | Bucket-Wahl + **TTL der presigned URLs** festlegen | 3 |

## Details

### 1. Tagline final wählen
- Favorit: **„Heute schon verhört?"**
- Bestätigung/Alternativen offen.

### 2. Wort des Monats – global?
- Kern: pro Gruppe.
- Offen: ob es **zusätzlich** ein globales „Wort des Monats" im öffentlichen Duden geben soll.

### 3. Gruppenlimit (`max_members`)
- Exakter Wert noch offen, „am Gefühl" zu justieren.
- Wirkt sich auf Phase 2 (Einladungen/Limit-Durchsetzung) aus.

### 4. Ähnlichkeits-Matching für Aggregation
- Start: **exaktes** Matching über `word_normalized`.
- Später: optional **Fuzzy-Matching (Levenshtein)** für Tippfehler/Varianten.

### 5. Voice-Storage
- Bucket-Wahl (welcher Objekt-Storage / welche Region).
- **TTL** der presigned URLs festlegen (Sicherheit vs. Caching).
