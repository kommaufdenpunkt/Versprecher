# Versprecher – Roadmap & Build-Reihenfolge

> Tagline (Favorit): **„Heute schon verhört?"** _(final noch zu bestätigen, siehe TODOs)_

Diese Roadmap leitet sich aus der empfohlenen Build-Reihenfolge der Spezifikation
(Abschnitt 14) ab. Die acht Phasen sind so geschnitten, dass jede Stufe auf der
vorherigen aufbaut und für sich genommen lauffähig/testbar bleibt.

Tech-Stack: **Go (Gin) + PostgreSQL** mit SQL-Migrationen, asynchroner
Moderations-Worker („Fidolin"), Objekt-Storage für Voice-Snippets.

---

## Phase 1 – Scaffold, Auth & Invite-Registrierung

Fundament: lauffähiger Service, Datenbank-Migrationen und sicherer Login per
Einladung.

- [ ] Projekt-Scaffold mit Go/Gin (Router, Config, Logging, Health-Check)
- [ ] PostgreSQL anbinden + Migrations-Tooling (z. B. `golang-migrate`) einrichten
- [ ] Basis-Migrationen: `users`, `sessions`/Tokens, `invites`
- [ ] Auth: Registrierung **nur per Invite-Code**, Login, Logout, Passwort-Hashing
- [ ] Session-/Token-Handling + Auth-Middleware
- [ ] CI: Build, Lint, Tests, Migration-Check

## Phase 2 – Gruppen, Mitglieder & Einladungen

- [ ] Migrationen: `groups`, `group_members`, Rollen (Admin/Mitglied)
- [ ] Gruppe erstellen / umbenennen / verlassen
- [ ] Mitglieder einladen (Invite-Code/-Link, an Phase 1 gekoppelt)
- [ ] `max_members` durchsetzen (Limit konfigurierbar, siehe TODOs)
- [ ] Berechtigungen: Wer darf einladen, entfernen, moderieren?

## Phase 3 – Posts & Feed + Fidolin-Worker

- [ ] Migrationen: `posts` (inkl. `word`, `word_normalized`, optional Voice-Ref)
- [ ] Post anlegen (Text + optional Voice-Snippet) und im Gruppen-Feed listen
- [ ] Feed-Endpoint (Pagination, Sortierung)
- [ ] **Fidolin-Worker** als asynchroner Job:
  - [ ] Moderations-Check beim Anlegen
  - [ ] „gemeint"-Vorschlag (was war wohl gemeint?) generieren
- [ ] Worker-Infrastruktur (Queue/Job-Tabelle, Retry, Status am Post)

## Phase 4 – Reaktionen & Kommentare (mit Moderation)

- [ ] Migrationen: `reactions`, `comments`
- [ ] Reaktionen setzen/entfernen
- [ ] Kommentare erstellen/löschen
- [ ] Moderation auch für Kommentare durch Fidolin
- [ ] Counts/Aggregation für Feed-Anzeige

## Phase 5 – Übersicht (Pinnen) & privater Duden

- [ ] Posts pinnen → Gruppen-Übersicht
- [ ] **Privater Duden**: pro Gruppe gesammelte Versprecher/Begriffe
- [ ] Duden-Einträge aus Posts ableiten (`word_normalized` als Schlüssel)
- [ ] Übersicht-/Duden-Endpoints + Sortierung

## Phase 6 – Einwilligungs-Flow, öffentlicher Duden & Aggregation

- [ ] **Einwilligungs-Flow**: Nutzer:innen geben Begriffe für die Öffentlichkeit frei
- [ ] **Öffentlicher Duden** (gruppenübergreifend)
- [ ] **Aggregation** gleicher Begriffe – zunächst **exakt über `word_normalized`**
      (Fuzzy/Levenshtein als späterer Ausbau, siehe TODOs)
- [ ] Datenschutz: nur freigegebene Begriffe veröffentlichen

## Phase 7 – Moderations-Tool

- [ ] Moderations-**Queue** (offene Fälle aus Fidolin)
- [ ] **Schwellen**/Thresholds konfigurierbar (Auto-Freigabe vs. manuell)
- [ ] **Duden-Freigabe** durch Moderation
- [ ] Audit/Log der Moderationsentscheidungen

## Phase 8 – Rituale

- [ ] **Wort des Monats / Wort des Jahres** (pro Gruppe; global TODO offen)
- [ ] **Erinnerungen / Push-Notifications**
- [ ] Scheduler/Cron für wiederkehrende Rituale

---

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

Siehe **[TODO.md](./TODO.md)** für offene Entscheidungen.
