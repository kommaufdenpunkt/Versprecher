# Änderungshistorie

Alle Zeiten in UTC. Neueste Einträge oben.

## 2026-08-07

- **Fahrstunden-Nachweis** (Nebenbuch zum FS Manager) hinzugefügt: Paket
  `internal/fahrstunden`, Migration `0007_fahrstunden` (`fahrschueler`,
  `fahrstunden`). Hält **„gefahren am“ und „eingetragen am“ getrennt** fest,
  dazu Art, Notiz und die Unterschrift der Fahrschüler:innen.
  - **Tageslimit** (Standard 495 Min) gilt auf `eingetragen_am`. Passt eine
    Stunde nicht, kommt **409** mit den konkreten Zahlen statt stillem
    Überbuchen; bewusstes Überschreiten nur **mit Begründung** (sonst 400).
  - **Automatische Wahl des Eintragetages**, wenn `eingetragen_am` fehlt:
    nächstgelegener Tag mit genug freier Zeit, bei Gleichstand der frühere.
  - Prüfung und Insert in **einer Transaktion** mit Vorschlagssperre
    (`pg_advisory_xact_lock`) je (Fahrlehrer, Eintragetag).
  - **PDF-Nachweis** je Fahrschüler:in (`go-pdf/fpdf`): beide Daten
    nebeneinander, Abweichung markiert, eingebettete Unterschriften, Summen je
    Art und Unterschriftsblock zum Gegenzeichnen.
  - **Oberfläche** als eingebettete Einzelseite unter frei wählbarem Basispfad
    (`FAHRSTUNDEN_BASIS_PFAD`, Standard `/fahrstunden`), mobiltauglich, mit
    Unterschriftsfeld für den Finger und Live-Anzeige der freien Minuten.
  - Endpoints unter `/v1/fahrstunden/*`; alle Daten hängen am angemeldeten
    Konto. Doku: [FAHRSTUNDEN.md](./FAHRSTUNDEN.md).
  - Unit-Tests grün; end-to-end gegen Postgres verifiziert (Limit greift,
    automatische Tageswahl, Übersteuerung mit Grund, Unterschrift aus dem
    Browser bis ins PDF, 5 gleichzeitige Eintragungen → genau eine durch).

## 2026-05-31

- **~15:11** (≈ 17:11 MESZ) – **Phase 3 umgesetzt** (Posts + Feed + Fidolin):
  Pakete `internal/posts`, `internal/fidolin`, `internal/moderation`; Migrationen
  `0005_posts`, `0006_moderation_settings`. Endpoints `POST /v1/groups/:id/posts`,
  `GET /v1/groups/:id/feed` (Keyset-Pagination), `PATCH /v1/posts/:id`.
  **Fidolin** als Goroutine-Worker-Pool mit Polling, sicherem Claiming
  (`FOR UPDATE SKIP LOCKED`) und Stale-Reclaim; Analyzer als Interface
  (Offline-Heuristik mitgeliefert, LLM einsteckbar); Schwellen aus
  `moderation_settings`; **fail-closed** (neue Posts erst `pending_review`,
  KI-Fehler → Mensch). Hashtag-Regel (max. 12 Buchstaben) durchgesetzt.
  Graceful Shutdown (HTTP + Fidolin) in `main`. Unit-Tests grün; end-to-end
  gegen Postgres verifiziert: harmlos → `visible`, Hass → `blocked`,
  Verhörer-Vorschlag, Autor-only `PATCH`, Mitglieds-Autorisierung.
- **~10:25** (≈ 12:25 MESZ) – **Phase 2 umgesetzt** (Gruppen + Mitglieder +
  Einladungen): Paket `internal/groups` (Service, Repository, Handler), Migration
  `0004_groups` (`groups`, `group_members`, FK `invitations.group_id`). Endpoints
  `POST /v1/groups`, `GET /v1/groups`, `GET /v1/groups/:id`,
  `POST /v1/groups/:id/invite`, `POST /v1/invitations/:token/accept`. Registrierung
  mit Gruppen-Einladung tritt direkt bei (verbindet Phase 1+2 über die kleine
  Schnittstelle `auth.GroupJoiner`). `max_members` (Default 30) wird durchgesetzt.
  Außerdem: **Einfach-Erklärung** `ERKLAERUNG.md`, **CI** unter
  `.github/workflows/ci.yml` (Build, Vet, gofmt, Tests, **govulncheck**).
  Unit-Tests grün; Phase 2 end-to-end gegen lokales Postgres verifiziert
  (Gründen, Liste, Details, Einladen, Beitritt per Registrierung & per Accept,
  Einmaligkeit, bereits-Mitglied 409, kein-Mitglied 403, Rate-Limit).
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

- **Phase 4:** Reaktionen (😂 ❤️ 😭) + Kommentare (mit Moderation durch Fidolin).
- Danach weiter Phase für Phase gemäß `ROADMAP.md`.
