# Insider

Ein kleines, privates Wörterbuch für **Versprecher und Verhörer** — Wörter, die
beim Reden schiefgehen. Festhalten, gemeinsam lachen, später wiederentdecken.

> Tagline (Favorit): **„Heute schon verhört?"**

- **Einfach erklärt (für alle):** [ERKLAERUNG.md](./ERKLAERUNG.md)
- **Spezifikation:** [SPEC.md](./SPEC.md)
- **Fahrstunden-Nachweis:** [FAHRSTUNDEN.md](./FAHRSTUNDEN.md)
- **Roadmap (8 Phasen):** [ROADMAP.md](./ROADMAP.md)
- **Offene Punkte:** [TODO.md](./TODO.md)
- **Änderungen:** [CHANGELOG.md](./CHANGELOG.md)

## Tech-Stack

- **Backend:** Go + Gin, PostgreSQL, API-Prefix `/v1`, JWT-Auth (`uid`, `adm`)
- **DDL-Konvention:** Migrationen via `psql` als Superuser, App-User nur DML

## Projektstruktur

```
cmd/server/        Einstiegspunkt (main)
internal/
  config/          Konfiguration aus Umgebungsvariablen
  db/              PostgreSQL-Pool (pgx)
  auth/            Auth: Passwort, JWT, Tokens, Validierung, Service, Handler
  groups/          Gruppen, Mitglieder, Einladungen (Service, Handler)
  posts/           Beiträge + Feed (Service, Handler)
  fidolin/         KI-Worker: Analyzer (Interface + Heuristik), Worker-Pool, Store
  moderation/      Moderations-Schwellen (von Fidolin gelesen)
  fahrstunden/     Fahrstunden-Nachweis: Service, PDF, Oberfläche (web/)
  middleware/      Auth, Rate-Limit, Security-Header
  httpx/           Router (/v1)
migrations/        SQL-Migrationen (.up.sql / .down.sql)
scripts/           dev_db.sh (DB+Rolle), migrate.sh (Migrationen anwenden)
.github/workflows/ CI (Build, Vet, Format, Test, govulncheck)
```

## Schnellstart (lokal)

```bash
# 1) Dev-Datenbank + App-Rolle (nur DML) anlegen — als Postgres-Superuser
sudo -u postgres scripts/dev_db.sh

# 2) Migrationen anwenden (DDL als Superuser, Konvention §3)
sudo -u postgres DB_NAME=insider_dev bash scripts/migrate.sh

# 3) Umgebung setzen und starten
cp .env.example .env       # JWT_SECRET anpassen!
export $(grep -v '^#' .env | xargs)
make run                   # oder: go run ./cmd/server
```

Health-Check: `curl localhost:8080/healthz`

## Tests

```bash
make test     # Unit-Tests (ohne DB, laufen überall)
```

## Sicherheit (eingebaut, bewusst einfach gehalten)

- **Nur auf Einladung** — primäre Bot-Bremse; jeder Account hat `invited_by`
  (Einladungskette rückverfolgbar). Einladungen sind einmalig.
- **Passwörter** mit **bcrypt** gehasht; nie im Klartext gespeichert/geloggt.
- **JWT-Secret** kommt aus der Umgebung (`JWT_SECRET`), nie aus dem Code.
- **Rate-Limiting** pro IP auf den Auth-Endpoints (Brute-Force-Bremse).
- **Keine User-Enumeration**: Login-Fehler sind immer generisch.
- **E-Mail-Verifizierung Pflicht**; Sperren über `users.status`.
- **Security-Header** + **TrustedProxies** (echte Client-IP).

## Endpoints (Phase 1 + 2)

| Methode | Pfad | Zweck |
|---|---|---|
| POST | `/v1/auth/register` | Registrierung (Invite nötig; erster Account = Bootstrap-Admin) |
| POST | `/v1/auth/login` | Login → JWT |
| POST | `/v1/auth/verify-email` | E-Mail bestätigen |
| GET | `/v1/me` | eigenes Profil |
| POST | `/v1/groups` | Gruppe gründen |
| GET | `/v1/groups` | meine Gruppen |
| GET | `/v1/groups/:id` | Gruppen-Details (Rolle, Mitgliederzahl) |
| POST | `/v1/groups/:id/invite` | Einladung erzeugen |
| POST | `/v1/invitations/:token/accept` | Einladung annehmen |
| POST | `/v1/groups/:id/posts` | Beitrag anlegen (wird von Fidolin geprüft) |
| GET | `/v1/groups/:id/feed` | Feed (nur sichtbare Beiträge) |
| PATCH | `/v1/posts/:id` | „gemeint"/Sorte bestätigen (nur Autor) |
| GET | `/healthz` | Health-Check |

Dazu der **Fahrstunden-Nachweis** unter `/v1/fahrstunden/*` und seine Oberfläche
unter `/fahrstunden` — siehe [FAHRSTUNDEN.md](./FAHRSTUNDEN.md).

## Fahrstunden-Nachweis (Nebenbuch zum FS Manager)

Der FS Manager lässt pro Tag nur **495 Minuten** zu. Wird an einem Tag mehr
gefahren, muss die Stunde unter einem anderen Tag verbucht werden. Damit die
Dokumentation lückenlos bleibt, hält dieses Nebenbuch beide Daten getrennt fest:

- **gefahren am** — der Tag der tatsächlichen Fahrstunde
- **eingetragen am** — der Tag, unter dem sie im FS Manager verbucht ist

Dazu Art, Notiz und die **Unterschrift** der Fahrschüler:innen (mit dem Finger
auf dem Handy) — ausdruckbar als **PDF** je Fahrschüler:in. Das Tageslimit wird
auf `eingetragen_am` geprüft; passt eine Stunde nicht, schlägt der Dienst
konkrete Ausweichtage vor, statt still zu überbuchen.

Details, Endpoints und Konfiguration: [FAHRSTUNDEN.md](./FAHRSTUNDEN.md).

## Fidolin (KI-Moderation)

Fidolin läuft als Hintergrund-Worker (Goroutine-Pool mit Polling) und prüft neue
Beiträge. Sicherheit zuerst:

- Neue Beiträge sind **`pending_review`** (nicht im Feed), bis Fidolin sie prüft.
- Score `≥ 0.85` → `blocked`, `≥ 0.60` → `pending_review` (Mensch), sonst `visible`
  (Schwellen in `moderation_settings`).
- **Fail-closed:** Bei KI-Fehler bleibt der Beitrag beim Menschen (`pending_review`).
- Der `Analyzer` ist ein **Interface**: mitgeliefert ist eine offline-funktionierende
  Heuristik; ein LLM-Analyzer kann ohne Worker-Änderung eingesteckt werden.
- Mehrere Worker/Instanzen sicher dank `FOR UPDATE SKIP LOCKED`; hängengebliebene
  Jobs werden automatisch zurückgestellt (Stale-Reclaim).

## Stand der Umsetzung

- ✅ **Phase 1:** Scaffold, Auth, Invite-Registrierung.
- ✅ **Phase 2:** Gruppen, Mitglieder, Einladungen.
- ✅ **Phase 3:** Posts + Feed + Fidolin-Worker (Moderation + „gemeint"-Vorschlag).
- ⏳ Phasen 4–8: siehe [ROADMAP.md](./ROADMAP.md).
- ✅ **Fahrstunden-Nachweis** (eigenständig, unabhängig von den Phasen 1–8).

### Hinweise zu Phase 1

- **Bootstrap:** Der allererste Account wird ohne Einladung angelegt und ist
  `admin` (`invited_by = NULL`). Jeder weitere Account braucht einen gültigen
  Einladungs-Token.
- **Gruppenbeitritt** (`group_members`) folgt in Phase 2; in Phase 1 erstellt die
  Registrierung nur das Konto aus einer gültigen Einladung.
- **E-Mail-Verifizierung:** Das Token wird derzeit in der Registrierungs-Antwort
  zurückgegeben (`email_verification_token`). In Produktion wird es per E-Mail
  versandt (`TODO`).
