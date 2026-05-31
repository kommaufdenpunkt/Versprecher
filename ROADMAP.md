# Insider – Roadmap & Build-Reihenfolge

> Privates Wörterbuch für **Versprecher und Verhörer**.
> Tagline (Favorit): **„Heute schon verhört?"** _(final noch zu bestätigen, siehe [TODO.md](./TODO.md))_

Diese Roadmap setzt die empfohlene Build-Reihenfolge aus der
[Spezifikation](./SPEC.md) (Abschnitt 14) in konkrete, abhakbare Aufgaben um.
Jede Phase verweist auf die zugehörigen Tabellen (§6) und Endpoints (§10) und
baut auf der vorherigen auf.

**Tech-Stack:** Go + Gin, PostgreSQL, API-Prefix `/v1`, JWT-Auth (`uid`, `adm`).
Flutter-App (PWA-fähig). Separates Moderations-Tool (React + Vite + TS).
Sprachnotizen im Object Storage (presigned URLs, kurze TTL). KI „Fidolin" als
Goroutine-Worker-Pool mit Polling.
**DDL-Konvention:** Migrationen via `psql` als Superuser, App-User nur DML.

---

## Phase 1 – Scaffold, Auth & Invite-Registrierung

Fundament: lauffähiger Service, Migrationen, sicherer Login **nur per Einladung**.

- [ ] Projekt-Scaffold Go/Gin (Router unter `/v1`, Config, Logging, Health-Check)
- [ ] PostgreSQL + Migrations-Setup (`psql`-Superuser für DDL, App-User nur DML)
- [ ] Migrationen: `users`, `invitations`
- [ ] JWT-Auth mit Claims `uid`, `adm`; Auth-Middleware
- [ ] `POST /auth/register` – nur mit gültigem Invite-Token; `invited_by` setzen
- [ ] `POST /auth/login` → JWT
- [ ] `POST /auth/verify-email` – E-Mail-Verifizierung (Pflicht, §11)
- [ ] Passwort-Hashing (bcrypt), `users.status`-Handling (`active/suspended/banned`)
- [ ] Rate-Limiting auf Schreib-Endpoints, TrustedProxies für echte Client-IP
- [ ] CI: Build, Lint, Tests, Migration-Check

**Tabellen:** `users`, `invitations` · **Endpoints:** `/auth/*`

## Phase 2 – Gruppen, Mitglieder & Einladungen

- [ ] Migrationen: `groups` (inkl. `max_members` default 30), `group_members`
- [ ] `POST /groups` (gründen), `GET /groups` (meine), `GET /groups/:id` (Details)
- [ ] `POST /groups/:id/invite` – Einladungslink/Token erzeugen (sha256, einmalig)
- [ ] `POST /invitations/:token/accept` – Einladung annehmen, Mitglied anlegen
- [ ] `max_members` durchsetzen (finaler Wert offen, siehe TODOs)
- [ ] Rollen `owner | member`; `invited_by`-Kette für Rückverfolgung (§11)

**Tabellen:** `groups`, `group_members`, `invitations` · **Endpoints:** `/groups`, `/groups/:id`, `/groups/:id/invite`, `/invitations/:token/accept`

## Phase 3 – Posts anlegen + Feed + Fidolin-Worker

- [ ] Migration: `posts` (inkl. `word`, `word_normalized`, `kind`, `voice_url`, `ai_*`, `status`)
- [ ] Hashtag-Validierung beim Wort: `^#?[A-Za-zÄÖÜäöüß]{1,12}$` (max. 12 Buchstaben)
- [ ] `word_normalized` (lowercase/trim) für spätere Aggregation berechnen
- [ ] `POST /groups/:id/posts` – Beitrag anlegen, Voice-Snippet optional (presigned)
- [ ] `GET /groups/:id/feed` – Feed (Pagination, Sortierung)
- [ ] `PATCH /posts/:id` – „gemeint"/`kind` durch Verfasser bestätigen/ändern
- [ ] **Fidolin-Worker** (Goroutine-Pool, Polling) – JSON-Vertrag §7:
  - [ ] IN `{word, explanation}` → OUT `{score, reason, kind_suggestion, meant_suggestion}`
  - [ ] Felder füllen: `ai_score`, `ai_kind_suggestion`, `ai_meant_suggestion`
  - [ ] Schwellen aus `moderation_settings`: `≥0.85` → `blocked`, `≥0.60` → `pending_review`, sonst `visible`
  - [ ] Fallback bei KI-Fehler → `pending_review`
- [ ] Wichtig: Fidolin filtert **nicht** die lustigen Versprecher raus; nur Hass/Übergriffiges

**Tabellen:** `posts` (+ liest `moderation_settings`) · **Endpoints:** `/groups/:id/posts`, `/groups/:id/feed`, `/posts/:id`

## Phase 4 – Reaktionen & Kommentare (mit Moderation)

- [ ] Migrationen: `reactions`, `comments`
- [ ] `POST /posts/:id/react` – nur `😂 ❤️ 😭` (CHECK), eine Reaktion pro Person (PK)
- [ ] `GET /posts/:id/comments`, `POST /posts/:id/comments`
- [ ] Kommentar-Moderation durch Fidolin (`comments.ai_score`, `comments.status`)
- [ ] **Keine Downvotes** (Kernprinzip §2/§12)

**Tabellen:** `reactions`, `comments` · **Endpoints:** `/posts/:id/react`, `/posts/:id/comments`

## Phase 5 – Übersicht (Pinnen) + privater Duden

- [ ] `POST /posts/:id/pin` – an-/abpinnen (`posts.is_pinned`)
- [ ] `GET /groups/:id/overview` – gepinnte Beiträge **+** Insider (landen direkt hier)
- [ ] `GET /groups/:id/duden` – privater Duden A–Z (volles Wort + Erklärung + O-Ton + wer's war)

**Endpoints:** `/posts/:id/pin`, `/groups/:id/overview`, `/groups/:id/duden`

## Phase 6 – Einwilligungs-Flow + öffentlicher Duden + Aggregation

- [ ] Migrationen: `publications`, `public_duden_entries`
- [ ] `POST /posts/:id/publish-request` – Veröffentlichung anstoßen, Beteiligte benachrichtigen
- [ ] `POST /posts/:id/consent` – Zustimmung + Namens-Sichtbarkeit setzen (author / tagged)
- [ ] `POST /posts/:id/revoke` – zurück in die Gruppe (`state = revoked`)
- [ ] **Doppelte Einwilligung** exakt nach §8 umsetzen:
  - [ ] `author_consent` **und** (`author == tagged` ODER `tagged_consent`)
  - [ ] Markierte Person nicht in Gruppe/nicht vorhanden → bleibt privat
  - [ ] Schweigen = **Nein** (kein Auto-Approve nach Timeout)
  - [ ] Widerruf jederzeit durch Verfasser *oder* markierte Person
- [ ] Namens-Sichtbarkeit getrennt: `name_visible_author` / `name_visible_tagged` (jede:r nur über eigenen Namen)
- [ ] **Aggregation** über `word_normalized` (exakt): existiert → `occurrence_count += 1`, sonst neu
- [ ] `GET /duden` – öffentlicher Duden: **nur Wort + Zähler** (+ optional freigegebene Namen), nie privater Inhalt/O-Ton
- [ ] Fuzzy-Matching (Levenshtein) bewusst später (siehe TODOs)

**Tabellen:** `publications`, `public_duden_entries` · **Endpoints:** `/posts/:id/publish-request`, `/posts/:id/consent`, `/posts/:id/revoke`, `/duden`

## Phase 7 – Moderations-Tool

Separate Oberfläche (React + Vite + TS), Routen für `moderator`/`admin`.

- [ ] Migrationen: `moderation_settings` (Defaults `0.85` / `0.60`), `moderation_queue` (oder Sicht auf `pending_review`)
- [ ] `GET /mod/queue` – markierte Beiträge/Kommentare (Score, Grund)
- [ ] `POST /mod/posts/:id/decision`, `POST /mod/comments/:id/decision` – freigeben/blockieren
- [ ] `POST /mod/duden/:word/approve` – Wort für öffentlichen Duden freigeben (`approved = true`)
- [ ] `GET / PUT /mod/settings` – Schwellen lesen/ändern
- [ ] `POST /mod/users/:id/suspend` – Nutzer sperren (`users.status`)

**Tabellen:** `moderation_settings`, `moderation_queue` · **Endpoints:** `/mod/*`

## Phase 8 – Rituale (Wort des Monats/Jahres, Erinnerungen/Push)

- [ ] Migrationen: `word_of_month`(+`_votes`), `word_of_year`(+`_votes`)
- [ ] `POST /groups/:id/word-of-month/vote`, `GET /groups/:id/word-of-month/:yyyymm`
- [ ] `POST /groups/:id/word-of-year/vote` (nur aus 12 Monatssiegern), `GET /groups/:id/word-of-year/:yyyy`
- [ ] **Geburtstag eines Wortes:** Push „X wird heute 1 Jahr alt 🎂"
- [ ] **Jahresrückblick** pro Gruppe (Wort des Jahres, fleißigste Versprecher-Person)
- [ ] **Sperrbildschirm-Erinnerung (PWA):** „Vor 5 Monaten hat … gesagt"
- [ ] Scheduler/Cron für wiederkehrende Rituale
- [ ] Offen: globales „Wort des Monats" im öffentlichen Duden? (siehe TODOs)

**Tabellen:** `word_of_month(_votes)`, `word_of_year(_votes)` · **Endpoints:** `/groups/:id/word-of-month/*`, `/groups/:id/word-of-year/*`

---

## Querschnittsthemen (durchgehend beachten)

- **Privat per Default** – nichts ohne ausdrückliche Zustimmung öffentlich (§2).
- **Echte Menschen** – nur auf Einladung, ein Mensch = ein Profil, kein Login über Dritte (§2/§11).
- **DSGVO** – Sprachnotizen sind personenbezogene Daten: Einwilligung, Löschkonzept, Export; Impressum + Datenschutz von Anfang an (§11).
- **Bewusst weglassen** – Drittanbieter-Login, Follower-/Reichweiten-Ranking, Downvotes, große öffentliche Gruppen (§13).

## Abhängigkeiten (Kurzüberblick)

```
1 Scaffold/Auth ─▶ 2 Gruppen ─▶ 3 Posts+Feed+Fidolin ─▶ 4 Reaktionen/Kommentare
                                        │
                                        ▼
                          5 Übersicht + privater Duden
                                        │
                                        ▼
                 6 Einwilligung + öffentlicher Duden + Aggregation
                                        │
                                        ▼
                              7 Moderations-Tool
                                        │
                                        ▼
                                  8 Rituale
```

Siehe **[SPEC.md](./SPEC.md)** für die vollständige Spezifikation und
**[TODO.md](./TODO.md)** für offene Entscheidungen.
