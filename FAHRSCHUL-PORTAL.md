# Fahrschul-Portal „Untern Buchen" — Konzept (Entwurf)

> **Status:** Ideen-Entwurf / Brainstorming festgehalten. Noch kein Code.
> Ziel dieses Dokuments: das Gesamtbild greifbar machen, bevor wir uns auf
> einen Teil festlegen. Wir verfeinern es gemeinsam.
>
> _Zuletzt aktualisiert: 2026-06-28_

---

## 1. Idee in einem Satz

Ein **Portal** für die Fahrschule „Untern Buchen", auf das alle Beteiligten
zugreifen — aber **jede:r nur auf den Bereich, der ihn etwas angeht**.

Der Kerngedanke, den du am stärksten betont hast:

> „Jeden ist aber nicht alles interessant. … Somit ist von überall nur ein
> bestimmter Bereich verfügbar."

Genau das ist das Fundament: **rollenbasierte Zugriffskontrolle**
(jede Rolle sieht und darf nur ihren Ausschnitt — „Least Privilege").

---

## 2. Warum ein Portal? (statt E-Mails / Zettel / Excel)

- **Ein Ort der Wahrheit** — Schülerdaten, Fahrstunden, Zahlungen liegen nicht
  verteilt auf Zetteln und in Postfächern, sondern an einer Stelle.
- **Jede:r sieht live nur das Eigene** — keine Weiterleitungen, kein „schick mir
  mal die Zahlen".
- **Das Steuerbüro bekommt die Zahlen fertig „auf den Tisch"** — statt dass
  jemand sie mühsam zusammensucht. (Das war dir ausdrücklich wichtig: „Die Arbeit
  so machen, dass das Steuerbüro die Zahlen nur noch auf den Tisch hat, wäre brutal.")

---

## 3. Rollen & Zugänge (das Fundament)

Sechs Beteiligte: vier intern/Kunde, zwei extern.

| Rolle | Intern/Extern | Aufgabe / Interesse | Bereich im Portal | Zugriff |
|---|---|---|---|---|
| **Fahrlehrer** | intern | Fahrstunden abarbeiten, Diagrammkarte abhaken, **eigene Slots anbieten** | „Seine" Schüler + Ausbildung + eigener Kalender | lesen/schreiben (eigener Bereich) |
| **Bürokraft** | intern | Schüler-/Terminverwaltung, Stammdaten, Tagesgeschäft | Verwaltung | lesen/schreiben (operativ) |
| **Chef** | intern | Überblick & Leitung | Alles | Vollzugriff |
| **Fahrschüler** | Kunde | freie Slots **seines** Fahrlehrers sehen & Termine buchen | nur eigener Kalender + eigene Daten | sehr eng: eigene Termine buchen/sehen |
| **Steuerbüro** | extern | Finanzielle Auswertung | Nur Finanzen, fertig aufbereitet | **nur lesen** |
| **Dataport** | extern | Umsätze einholen, Ratenzahlung verwalten | Zahlungen/Umsatz je Schüler | lesen/schreiben (Zahlungsbereich) |

### Faustregel
- **Intern abgestuft:** Fahrlehrer (eng) ⊂ Bürokraft (operativ) ⊂ Chef (alles).
- **Extern eng & zweckgebunden:** Steuerbüro nur lesend auf Finanzen, Dataport nur
  auf den Zahlungsbereich. Keine externe Rolle sieht z. B. pädagogische Notizen
  oder interne Vermerke.

---

## 4. Bereiche / Module (Gesamtbild)

Grobe Aufteilung des Portals in Bereiche. Wer welchen sieht, steht in Klammern.

1. **Schüler & Stammdaten** _(Bürokraft, Chef; Fahrlehrer sieht seine Schüler)_
   - Schülerakte: Kontakt, Ausbildungsstand, zugeordneter Fahrlehrer.

2. **Ausbildung / Fahrstunden** _(Fahrlehrer, Chef; Bürokraft lesend)_
   - Fahrstunden erfassen und abarbeiten.
   - **Ausbildungsdiagrammkarte**: Pflicht-/Sonderfahrten per Haken abarbeiten.
   - Tarif-Basis: **80 Min = 130 €** (siehe §5).

3. **Zahlungen & Umsatz** _(Dataport, Chef; Bürokraft je nach Bedarf)_
   - Umsätze einholen, offene Beträge je Schüler.
   - **Ratenzahlung**: Standard **200 €/Monat** oder individuell.
   - Datensatz **je Schüler** sortiert.

4. **Finanz-Auswertung** _(Steuerbüro nur lesend, Chef)_
   - Fertig aufbereitete Zahlen — Umsätze, Zahlungen, offene Posten.
   - Ziel: das Steuerbüro hat die Zahlen „nur noch auf dem Tisch".

5. **Terminkalender & Slots** _(Fahrlehrer schreibt, Fahrschüler bucht; Büro/Chef sehen mit)_
   - **Der Fahrlehrer stellt seine eigenen Slots bereit** — ganz allein, volle Freiheit.
   - **Wochen-Vorlage:** typische Woche einmal als Vorlage speichern und
     **Woche für Woche übernehmen**. Pro Woche noch **ergänzen oder entfernen**.
   - **Manuell ein Termin per Hand** dazubuchen (außerhalb der Slots) ist möglich.
   - **Fahrschüler** sieht die **freien** Slots seines Fahrlehrers und bucht sich rein.
   - Leitprinzip: **dem Fahrlehrer die Freiheiten lassen** — Vorlage ist Angebot,
     kein Zwang. „Flexibel, quadratisch, praktisch, gut."

6. **Verwaltung / Konten & Rollen** _(Chef)_
   - Wer hat welchen Zugang? Nutzer anlegen/sperren.

---

## 5. Fachliche Fakten (aus deiner Beschreibung)

- **Fahrstunde:** 80 Minuten kosten **130 €**.
- **Ratenzahlung:** wenn jemand nicht alles auf einmal zahlen kann →
  **200 € monatlich** oder **individuell**.
- **Datenhaltung:** Zahlungs-/Umsatz-Datensatz wird **je Schüler** einsortiert.
- **Ausbildungsdiagrammkarte:** wird per **Haken** (abgehakt) abgearbeitet.

> Diese Werte halte ich erstmal genau so fest, wie du sie genannt hast — falls
> sich Tarife/Raten ändern oder differenzieren, ziehen wir sie hier nach.

---

## 6. Datenschutz / DSGVO (von Anfang an mitdenken)

Schülerdaten und Zahlungsdaten sind **personenbezogene Daten**, dazu kommen
**zwei externe Zugriffe** (Steuerbüro, Dataport). Das heißt:

- **Zweckbindung:** externe Rollen sehen nur, was sie für ihren Zweck brauchen.
- **Auftragsverarbeitung:** Verträge mit Steuerbüro / Dataport klären (wer darf
  was, wie lange).
- **Protokollierung:** wer hat wann welche Daten gesehen (besonders bei Extern).
- **Löschkonzept & Aufbewahrungsfristen** (steuerlich relevante Daten!).

_Das ist kein Bremsklotz, nur ein Punkt, den wir sauber halten, weil Externe
auf echte Personen-/Finanzdaten schauen._

---

## 7. Offene Fragen (klären wir gemeinsam)

| # | Frage |
|---|-------|
| 1 | Sieht ein Fahrlehrer **nur eigene** Schüler oder **alle**? |
| 2 | Darf die **Bürokraft** Zahlungen sehen/bearbeiten oder nur Stammdaten/Termine? |
| 3 | **Dataport** und **Steuerbüro** — eigene Logins ins Portal oder bekommen sie nur Exporte? |
| 4 | Ist **Dataport** ein externer Dienstleister mit eigener Software (dann eher Schnittstelle) oder arbeitet er im Portal? |
| 5 | Gibt es weitere Tarife außer 80 Min = 130 € (z. B. Theorie, Prüfungsgebühren)? |
| 6 | Müssen Fahrlehrer auch **Geld kassieren** dürfen, oder läuft alles Geld über Dataport/Büro? |
| 7 | Web-Portal für alle, oder zusätzlich **App** für Fahrlehrer (unterwegs abhaken)? |
| 8 | **Buchung:** bucht der Fahrschüler einen Slot **direkt verbindlich**, oder erst **anfragen** und der Fahrlehrer bestätigt? |
| 9 | Darf ein Fahrschüler einen gebuchten Termin selbst **stornieren/verschieben** — bis wie lange vorher (Frist)? |
| 10 | Sehen Fahrschüler **Namen** anderer Schüler in belegten Slots, oder nur „belegt/frei" (Datenschutz)? |
| 11 | Soll eine Buchung automatisch eine **Fahrstunde** (80 Min / 130 €) anlegen, oder erst beim Abhaken durch den Fahrlehrer? |

---

## 8. Nächste Schritte (Vorschlag)

1. **Offene Fragen §7 klären** — damit die Zugriffs-Matrix wirklich stimmt.
2. **Zugriffs-Matrix finalisieren** (wer sieht/darf was, je Bereich).
3. **Datenmodell skizzieren** — Schüler, Fahrstunde, Ausbildungskarte, Zahlung/Rate.
4. **Erstes Modul wählen** zum Anfangen (z. B. Fahrlehrer-Alltag oder Zahlungen).
5. Erst dann **Technik/Code**.

> Reihenfolge bewusst: erst das Bild, dann das Datenmodell, dann Code — sonst
> bauen wir an der falschen Stelle.
