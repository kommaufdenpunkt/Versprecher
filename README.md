# Insider

Ein kleines, privates Wörterbuch für **Versprecher und Verhörer** — Wörter, die
beim Reden schiefgehen. Festhalten, gemeinsam lachen, später wiederentdecken.

> Tagline (Favorit): **„Heute schon verhört?"**

- **Spezifikation:** [SPEC.md](./SPEC.md)
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
  middleware/      Auth, Rate-Limit, Security-Header
  httpx/           Router (/v1)
migrations/        SQL-Migrationen (.up.sql / .down.sql)
scripts/           dev_db.sh (DB+Rolle), migrate.sh (Migrationen anwenden)
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

## Stand der Umsetzung

- ✅ **Phase 1:** Scaffold, Auth, Invite-Registrierung
  (`/v1/auth/register`, `/login`, `/verify-email`, `/me`).
- ⏳ Phasen 2–8: siehe [ROADMAP.md](./ROADMAP.md).

### Hinweise zu Phase 1

- **Bootstrap:** Der allererste Account wird ohne Einladung angelegt und ist
  `admin` (`invited_by = NULL`). Jeder weitere Account braucht einen gültigen
  Einladungs-Token.
- **Gruppenbeitritt** (`group_members`) folgt in Phase 2; in Phase 1 erstellt die
  Registrierung nur das Konto aus einer gültigen Einladung.
- **E-Mail-Verifizierung:** Das Token wird derzeit in der Registrierungs-Antwort
  zurückgegeben (`email_verification_token`). In Produktion wird es per E-Mail
  versandt (`TODO`).
