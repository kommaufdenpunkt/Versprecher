# Eigenes Programm? — Ideen

Zurück zur [Übersicht](./README.md)

Du baust schon einen Go-Service mit Postgres. Genau die Bausteine, die du dafür
gelernt hast, lösen dein Buchhaltungsproblem — und zwar besser als jedes
Standardtool, weil kein Standardtool Bigo-Bohnen kennt.

---

## Warum überhaupt selbst bauen?

Drei Dinge kann keine Standardsoftware:

1. **Bigo-Abrechnungen verstehen.** Bohnen/Diamanten, USD-Auszahlung,
   Agenturprovision, Streamer-Splits — sevdesk kennt davon nichts.
2. **Die Agentur-Seite.** Wer von deinen Streamern hat wie viel verdient, was
   davon ist deine Provision, was leitest du weiter, wer hat schon eine
   Gutschrift bekommen?
3. **Dich rechtzeitig warnen.** Die 25.000-€-Grenze, die
   Reverse-Charge-Rechnung, die vergessene Rücklage.

Und ein vierter Grund: Was du hier baust, kannst du später **anderen
Bigo-Agenturen verkaufen**. Die haben exakt dasselbe Problem.

---

## Ideen-Katalog

| # | Idee | Was es dir bringt | Aufwand |
|---|---|---|---|
| 1 | **Belegerfassung mit Foto + OCR** | Bon abfotografieren → Betrag, Datum, Händler automatisch erkannt, Kategorie vorgeschlagen. Ende der Schuhkarton-Buchhaltung. | mittel |
| 2 | **Bigo-Import** | CSV/Screenshot der Abrechnung rein → Bohnen, USD, Auszahlungsdatum, EUR-Umrechnung zum Kurs des Zuflusstags. | mittel |
| 3 | **EÜR-Ansicht** | Alle Buchungen auf die amtlichen Zeilen der Anlage EÜR gemappt, jederzeit als Zwischenstand abrufbar. | mittel |
| 4 | **Kleinunternehmer-Ampel** | Laufender Umsatz gegen 25.000 € / 100.000 €. Grün, gelb ab 80 %, rot mit konkreter Handlungsanweisung. | klein |
| 5 | **Reverse-Charge-Detektor** | Erkennt Rechnungen ausländischer Anbieter (Adobe, Meta, Google, OpenAI …), rechnet die 19 % aus, erinnert an die USt-Voranmeldung. | klein |
| 6 | **Rücklagen-Rechner** | Schätzt deine Steuerlast aus Hauptjob-Brutto + laufendem Gewinn und sagt dir: „lege von dieser Auszahlung 412 € zurück". | klein |
| 7 | **Creator-Verwaltung** | Streamer, Verträge, Splits, Auszahlungen, offene Posten. Das Herz des Agenturteils. | mittel |
| 8 | **Gutschriften-Generator** | PDF-Gutschrift pro Streamer und Monat, fortlaufend nummeriert, § 19-Hinweis, Pflichtangaben. Spart dir jeden Monat Stunden. | mittel |
| 9 | **KSK-Zähler** | Summiert Honorare an Kreative; Alarm bei Überschreiten der Bagatellgrenze (ab 2026: 1.000 €). | klein |
| 10 | **AfA-/GWG-Assistent** | Erkennt: unter 800 € netto → sofort; Computer → 1 Jahr; sonst Abschreibungsplan. Führt automatisch das Anlagenverzeichnis. | mittel |
| 11 | **Fahrtenliste** | Fahrt erfassen (Datum, Ziel, Zweck, km) → 0,30 €/km automatisch als Buchung. | klein |
| 12 | **Homeoffice-Tagezähler** | Ein Klick pro Tag → zählt die 6-€-Tage bis zum Deckel von 1.260 €. | klein |
| 13 | **Belegarchiv, GoBD-tauglich** | Unveränderbare Ablage mit Hash + Zeitstempel, 8 Jahre, Volltextsuche, Export als ZIP für den Steuerberater. | mittel |
| 14 | **Steuerberater-Export** | Ein Knopf: CSV/DATEV-ähnlicher Export + alle Belege. Senkt deine Beraterrechnung spürbar. | klein |
| 15 | **Jahres-Checkliste** | Fristen, offene Belege, fehlende Gegenkonten — als abhakbare Liste wie deine ROADMAP. | klein |

---

## Empfehlung: MVP „Kassenwart" in drei Phasen

Nicht alles auf einmal. Was dir **dieses Jahr** am meisten Geld und Nerven
spart, sind #2, #4, #6 und #14 — der Rest kann warten.

### Phase 1 — Erfassen (das Fundament)

- [ ] Migrationen: `accounts`, `transactions`, `categories`, `receipts`
- [ ] `POST /v1/tx` — Buchung anlegen (Einnahme/Ausgabe, Datum, Betrag, Kategorie)
- [ ] `POST /v1/receipts` — Beleg hochladen, Hash speichern, an Buchung hängen
- [ ] `GET /v1/euer?year=2025` — Zwischenstand: Einnahmen, Ausgaben, Gewinn
- [ ] Kategorien fest verdrahtet nach den EÜR-Zeilen
- [ ] Zufluss-/Abflussprinzip: gebucht wird auf das **Zahlungsdatum**

### Phase 2 — Bigo & Rücklage (der Alltag)

- [ ] `POST /v1/imports/bigo` — Abrechnung einlesen, Bohnen → USD → EUR
- [ ] Wechselkurstabelle (monatliche BMF-Umrechnungskurse) als `fx_rates`
- [ ] Kleinunternehmer-Ampel auf `GET /v1/status`
- [ ] Rücklagenvorschlag pro Einnahme (Grenzsteuersatz als Konfigwert)
- [ ] Reverse-Charge-Flag auf Ausgaben + monatliche Summe

### Phase 3 — Agentur (der Hebel)

- [ ] `creators`, `payouts`, `splits`
- [ ] Auszahlung pro Streamer berechnen, Provision automatisch abziehen
- [ ] Gutschrift als PDF, fortlaufende Nummer aus `credit_note_seq`
- [ ] KSK-Zähler über alle Honorare
- [ ] Export-Knopf für den Steuerberater

---

## Datenmodell-Skizze

```sql
-- Kern
transactions (id, booked_on, direction, amount_cents, currency,
              amount_eur_cents, fx_rate, category_id, counterparty,
              note, reverse_charge, created_at)
categories   (id, code, label, euer_line)       -- Mapping auf Anlage EÜR
receipts     (id, transaction_id, file_path, sha256, uploaded_at)
assets       (id, name, acquired_on, cost_cents, method, useful_life_years)

-- Bigo / Agentur
creators     (id, display_name, bigo_id, contract_url, share_percent, status)
payouts      (id, creator_id, period, beans, gross_usd_cents, eur_cents,
              commission_cents, paid_on, credit_note_no)
fx_rates     (month, currency, rate)
```

Zwei Konventionen, die dir später viel Ärger ersparen:

- **Beträge immer als `bigint` in Cent.** Niemals `float` für Geld.
- **`amount_eur_cents` und `fx_rate` mitspeichern**, nicht bei Bedarf neu
  rechnen — der Kurs von damals muss auch in fünf Jahren noch nachvollziehbar
  sein. Das verlangen die GoBD.

---

## Architektur — passt auf deinen Stack

Dieselbe Struktur wie im Versprecher-Projekt, du müsstest nichts Neues lernen:

```
cmd/kassenwart/
internal/
  config/     wie gehabt
  db/         pgx-Pool, wie gehabt
  auth/       kannst du 1:1 übernehmen
  ledger/     Buchungen, Kategorien, EÜR-Berechnung
  receipts/   Upload, Hash, Archiv
  bigo/       Import-Parser + Wechselkurse
  agency/     Creator, Splits, Gutschriften
  warnings/   Ampel, Fristen, Reverse-Charge  (als Worker wie Fidolin!)
  httpx/      Router unter /v1
```

`internal/warnings` ist praktisch derselbe Bauplan wie dein Fidolin-Worker:
ein Goroutine-Pool, der regelmäßig Regeln über die Daten laufen lässt und
Hinweise erzeugt. Der Code, den du dort schon geschrieben hast, ist zu großen
Teilen wiederverwendbar.

**Als eigenes Repository anlegen**, nicht in Versprecher hineinbauen — andere
Domäne, andere Daten, anderer Lebenszyklus.

---

## Was du NICHT selbst bauen solltest

| Finger weg von | Warum |
|---|---|
| Direkte ELSTER-Übermittlung | Braucht ERiC, Zertifizierung und Pflege bei jeder Formularänderung. Exportiere lieber sauber und übertrage in „Mein ELSTER" bzw. gib es dem Berater. |
| Eigene Steuerberechnung als „Ergebnis" | Der Splittingtarif, Progressionsvorbehalt, Anlage N — das wird nie richtig genug. Schätzen für die Rücklage: ja. Als Wahrheit ausgeben: nein. |
| Zertifizierte GoBD-Archivierung als Verkaufsargument | „GoBD-konform" ist eine Zusage mit Haftung. Für dich selbst reicht saubere Praxis. |
| Zahlungsabwicklung für deine Streamer | Erlaubnispflichtiges Zahlungsdienstgeschäft. Überweisen, nicht durchleiten. |

---

## Kaufen statt bauen?

Ehrliche Gegenrechnung: Ein Buchhaltungsabo kostet ~10–20 €/Monat und kann
alles außer Bigo und Agentur. Dein Tool kostet dich Wochenenden.

**Der pragmatische Mittelweg:** Standardtool für die normale Buchhaltung
(Belege, EÜR, Export zum Berater) — und dein eigenes Programm nur für das,
was keins davon kann: **Bigo-Import, Creator-Splits, Gutschriften, Ampel**.
Das ist ein Wochenendprojekt statt eines Jahresprojekts, und du behältst den
Teil, der wirklich weh tut.

---

## Und die Geschäftsidee dahinter

Es gibt in Deutschland viele Bigo-, TikTok- und Twitch-Agenturen mit exakt
deinem Problem. Ein kleines SaaS für „Creator-Agentur-Abrechnung" hat einen
echten Markt — und es liegt genau auf deinem angemeldeten Gewerbe (Feld 18
deckt Agentur- und Onlinedienstleistung ab; für den reinen Softwareverkauf
würde ich Feld 18 bei Gelegenheit trotzdem erweitern lassen).

⚠️ Eine harte Grenze: **Steuerberatung ist erlaubnispflichtig (§ 5 StBerG).**
Ein Tool darf rechnen, sortieren, erinnern und exportieren. Es darf Nutzern
nicht sagen, *wie sie einen Sachverhalt steuerlich zu behandeln haben*. In der
Praxis: neutrale Formulierungen, klarer Hinweis „ersetzt keine Steuerberatung",
keine individuellen Empfehlungen. Vor dem ersten zahlenden Kunden einmal
anwaltlich prüfen lassen.
