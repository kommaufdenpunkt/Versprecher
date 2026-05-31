# Änderungshistorie

Alle Zeiten in UTC. Neueste Einträge oben.

## 2026-05-31

- **~10:10** (≈ 12:10 MESZ) – **Phase 1 umgesetzt** (Gerüst + Auth): Go/Gin-Projekt
  (`cmd/server`, `internal/{config,db,auth,middleware,httpx}`), Migrationen
  (`users`, `invitations`, `email_verifications`) inkl. `scripts/migrate.sh` und
  `scripts/dev_db.sh`. Endpoints `POST /v1/auth/register|login|verify-email`,
  `GET /v1/me`, `GET /healthz`. Sicherheit eingebaut: nur-auf-Einladung +
  Bootstrap-Admin, bcrypt, JWT (`uid`/`adm`), Rate-Limiting, Security-Header,
  TrustedProxies, keine User-Enumeration, E-Mail-Pflicht. Unit-Tests grün;
  End-to-End gegen lokales Postgres verifiziert (Bootstrap, Invite-Pflicht,
  Einmaligkeit, Rate-Limit, `invited_by`-Kette). `README.md` ergänzt.
- **09:59** (≈ 11:59 MESZ) – `SPEC.md` ergänzt: vollständige Insider-Spezifikation (Stand Mai 2026)
  übernommen. `ROADMAP.md` überarbeitet, sodass jede der 8 Phasen auf die
  konkreten Tabellen (§6) und Endpoints (§10) verweist. `TODO.md` an die Spec
  angeglichen (`max_members`-Default `30`, Verweise auf SPEC). Diese
  `CHANGELOG.md` angelegt; Stand-Datum in die Dokumente eingetragen.
- **~09:52** – Erste Fassung von `ROADMAP.md` und `TODO.md` aus den Abschnitten
  14 (Build-Reihenfolge) und 15 (offene Punkte) erstellt. Produktname zu diesem
  Zeitpunkt noch als „Versprecher" geführt.

## Nächste Schritte (geplant)

- **Phase 2:** Gruppen + Mitglieder + Einladungen (`groups`, `group_members`,
  FK auf `invitations.group_id`, `max_members`, Invite-Endpoint, `/invitations/:token/accept`).
- Danach weiter Phase für Phase gemäß `ROADMAP.md`.
- Offen aus Phase 1: CI-Pipeline (Build/Lint/Test/Migration-Check).
