# Fahrstunden-Nachweis — das Nebenbuch zum FS Manager

Ein privates, lückenloses Verzeichnis der Fahrstunden: **gefahren am** und
**eingetragen am** getrennt festgehalten, mit Notiz und Unterschrift der
Fahrschülerin bzw. des Fahrschülers — ausdruckbar als PDF.

## Warum das nötig ist

Der FS Manager lässt pro Tag nur eine begrenzte Arbeitszeit zu (**495 Minuten**,
also 8 h 15 min). Wird an einem Tag mehr gefahren, muss die Stunde unter einem
anderen Tag verbucht werden — zum Beispiel *am Neunten gefahren, am Samstag
davor eingetragen*.

Damit die Dokumentation trotzdem lückenlos ist, hält dieses Nebenbuch **beide
Daten** fest:

| Feld | Bedeutung |
|---|---|
| `gefahren_am` | Der Tag, an dem die Fahrstunde tatsächlich stattgefunden hat |
| `eingetragen_am` | Der Tag, unter dem sie im FS Manager verbucht ist |

Das Tageslimit gilt auf **`eingetragen_am`** — das ist der Arbeitszeit-Tag im
FS Manager. Der Fahrtag selbst wird nicht begrenzt.

## Was es kann

- **Eintragen** mit Fahrschüler, Fahrtag, Dauer, Art und Notiz.
- **Automatische Wahl des Eintragetages**: Bleibt `eingetragen_am` leer, sucht
  der Dienst selbst den nächstgelegenen Tag mit genug freier Zeit. Bei gleichem
  Abstand gewinnt der frühere Tag — so wird in der Praxis verschoben.
- **Tageslimit statt Bauchgefühl**: Passt eine Stunde nicht mehr, kommt kein
  stilles Überbuchen, sondern eine Absage mit den konkreten Zahlen und
  Alternativtagen.
- **Bewusst überschreiten** ist möglich — aber nur mit Begründung, die an der
  Stunde gespeichert wird. Sonst wäre der Nachweis nicht mehr nachvollziehbar.
- **Unterschrift** direkt auf dem Gerät mit dem Finger, je Fahrstunde.
- **PDF-Nachweis** je Fahrschüler: beide Daten nebeneinander, Abweichung
  markiert, Summen, Unterschriften und ein Unterschriftsblock zum Gegenzeichnen.

## Oberfläche

Erreichbar unter dem konfigurierten Basispfad, standardmäßig:

```
http://localhost:8080/fahrstunden
```

Eine einzelne, eingebettete Seite — kein Build-Schritt, keine externen
Abhängigkeiten, für das Handy gebaut (große Bedienflächen, Unterschriftsfeld,
helles und dunkles Erscheinungsbild). Angemeldet wird sich mit dem normalen
Konto; alle Daten gehören ausschließlich diesem Konto.

## Konfiguration

Alles über Umgebungsvariablen — im Code steht nichts fest:

| Variable | Standard | Zweck |
|---|---|---|
| `FAHRSTUNDEN_TAGESLIMIT_MINUTEN` | `495` | Zeit, die der FS Manager pro Tag zulässt |
| `FAHRSTUNDEN_BASIS_PFAD` | `/fahrstunden` | URL der Oberfläche |
| `FAHRSTUNDEN_APP_NAME` | `Fahrstunden-Nachweis` | Name in Titelzeile und Kopf |
| `FAHRSTUNDEN_FAHRSCHULE` | *(leer)* | Kopfzeile im PDF |

**Eigener Name, eigene URL:** Der Basispfad ist frei wählbar. Soll der Nachweis
später zum Beispiel unter `/gino` laufen, reicht:

```bash
FAHRSTUNDEN_BASIS_PFAD=/gino
FAHRSTUNDEN_APP_NAME=Gino
```

Alles Weitere liegt dann unter diesem Pfad; die Oberfläche leitet ihre eigenen
Adressen daraus ab. Für eine eigene Domain (z. B. `gino.de`) genügt ein
Reverse-Proxy auf denselben Pfad — im Code ist dafür nichts zu ändern.

## Endpoints

Alle unter `/v1/fahrstunden`, alle mit gültigem JWT. Jede Route arbeitet
ausschließlich auf den Daten des angemeldeten Kontos.

| Methode | Pfad | Zweck |
|---|---|---|
| GET | `/stammdaten` | Arten, Tageslimit, heutiges Datum |
| GET | `/schueler` | Fahrschüler auflisten (`?nur_aktive=1`) |
| POST | `/schueler` | Fahrschüler anlegen |
| PATCH | `/schueler/:id` | Name, Klasse, Notiz, aktiv ändern |
| GET | `/schueler/:id/nachweis.pdf` | PDF-Nachweis (`?von=&bis=`) |
| GET | `/stunden` | Stunden auflisten (`?fahrschueler_id=&von=&bis=&limit=`) |
| POST | `/stunden` | Fahrstunde eintragen |
| PATCH | `/stunden/:id` | Fahrstunde ändern (Limit wird neu geprüft) |
| DELETE | `/stunden/:id` | Fahrstunde löschen |
| PUT | `/stunden/:id/unterschrift` | Unterschrift setzen (leer = entfernen) |
| GET | `/kapazitaet` | Auslastung je Eintragetag (`?von=&bis=`) |
| GET | `/kapazitaet/vorschlaege` | Passende Eintragetage (`?gefahren_am=&dauer_minuten=&fenster=`) |

Dazu ohne Anmeldung, nur für die Oberfläche:
`GET {basis}/konfig.json` — Anzeigename, API-Pfade, Tageslimit.

### Beispiel: eintragen, ohne den Eintragetag selbst zu wählen

```bash
curl -X POST localhost:8080/v1/fahrstunden/stunden \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"fahrschueler_id":2,"gefahren_am":"2026-08-10","dauer_minuten":90,
       "art":"ueberlandfahrt","notiz":"Überland B27"}'
```

Ist der 10.08. schon voll, antwortet der Dienst mit dem nächstgelegenen Tag:

```json
{"stunde":{"gefahren_am":"2026-08-10","eingetragen_am":"2026-08-09",
           "abweichung_tage":-1, "...":"..."}}
```

### Beispiel: das Tageslimit wird erreicht

Mit ausdrücklichem `eingetragen_am` auf einem vollen Tag kommt **409** mit den
Zahlen, aus denen die Oberfläche direkt einen Vorschlag machen kann:

```json
{
  "code": "tageslimit",
  "error": "Tageslimit überschritten: am 10.08.2026 sind noch 45 Min frei, gewünscht sind 90 Min (1:30 h)",
  "tageslimit": {"datum":"2026-08-10","limit_minuten":495,"belegt_minuten":450,
                 "frei_minuten":45,"wunsch_minuten":90}
}
```

Bewusst überschreiben geht mit `"limit_uebersteuern": true` **und**
`"limit_grund": "..."` — ohne Grund antwortet der Dienst mit 400.

## Arten einer Fahrstunde

`grundausbildung`, `uebungsstunde`, `ueberlandfahrt`, `autobahnfahrt`,
`nachtfahrt`, `pruefungsvorbereitung`, `pruefungsfahrt`, `sonstiges`

Die Sonderfahrten stehen einzeln, damit der Nachweis zeigt, was schon gefahren
wurde.

## Datenmodell

`migrations/0007_fahrstunden.up.sql` legt zwei Tabellen an:

- **`fahrschueler`** — Name, Klasse, Notiz, aktiv; Name je Fahrlehrer eindeutig.
- **`fahrstunden`** — `gefahren_am`, `eingetragen_am`, `dauer_minuten`, `art`,
  `notiz`, `unterschrift_png`, `unterschrieben_am`, `limit_uebersteuert`,
  `limit_grund`.

Beide hängen über `fahrlehrer_id` am Konto. Der Index
`(fahrlehrer_id, eingetragen_am)` trägt die Tageslimit-Prüfung.

## Wie die Prüfung sicher bleibt

Die Limit-Prüfung läuft in derselben Transaktion wie das Schreiben, abgesichert
durch eine Vorschlagssperre (`pg_advisory_xact_lock`) auf
`(fahrlehrer_id, eingetragen_am)`. Ohne sie könnten zwei gleichzeitige
Eintragungen beide die Prüfung bestehen und den Tag zusammen über das Limit
heben. Nachgemessen: von fünf gleichzeitigen Anfragen auf einen Tag mit 45
freien Minuten kommt genau eine durch.

## Tests

```bash
go test ./internal/fahrstunden/
```

Die Service-Tests laufen gegen ein Repository im Speicher, das dieselbe
Limit-Logik abbildet wie die Datenbank — sonst würden sie am Kern vorbeitesten.
