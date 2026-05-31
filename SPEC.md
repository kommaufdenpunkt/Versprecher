# Insider — Spezifikation (Stand Mai 2026)

> Arbeitsdokument für die Umsetzung mit Claude Code. Identifier (Tabellen, Spalten, Endpoints) in Englisch, Inhalte/Erklärungen in Deutsch. `TODO:`-Stellen sind bewusst offen.
>
> _Im Repo erfasst: 2026-05-31 09:59 UTC · Änderungen: [CHANGELOG.md](./CHANGELOG.md)_

-----

## 1. Was ist Insider

Ein kleines, privates Wörterbuch für **Versprecher und Verhörer** — Wörter, die beim Reden schiefgehen. Festhalten, gemeinsam lachen, später wiederentdecken. **Keine Sprichwörter.** Zwei Sorten Wörter:

- **Versprecher** — falsch *gesagt* (z. B. „Spabl” aus Spatzl + Baby).
- **Verhörer** — falsch *gehört* (z. B. „Fernsehturnier” statt Fernsehturm).

## 2. Kernprinzipien (nicht verhandelbar)

1. **Privat per Default.** Nichts wird ohne ausdrückliche Zustimmung öffentlich.
2. **Doppelte Einwilligung** für die Öffentlichkeit: Verfasser **und** markierte Person.
3. **Ohne Hate.** KI-Filter + menschliche Moderation. Keine Downvotes.
4. **Nähe statt Reichweite.** Gruppen klein halten. Reichweite läuft nur über den anonymen öffentlichen Duden.
5. **Echte Menschen.** Nur auf Einladung, ein Mensch = ein Profil. **Kein** Login über Dritte (Apple/Google/Facebook bewusst nicht).
6. **DSGVO ernst nehmen.** Es geht um echte Stimmen und Namen.

## 3. Tech-Stack (Vorschlag, orientiert am bestehenden VibeVibo-Setup)

- **Backend:** Go + Gin, PostgreSQL. API-Prefix `/v1`. JWT-Auth (Claims wie gewohnt, z. B. `uid`, `adm`).
- **Frontend (App):** Flutter, als PWA-fähig gedacht (Push, Sperrbildschirm-Erinnerungen).
- **Moderations-Tool:** separate Web-Oberfläche (React + Vite + TS), getrennt von der App — analog zu admin.vibevibo.de.
- **Sprachnotizen:** Object Storage (z. B. Hetzner Bucket) mit presigned URLs, kurze TTL.
- **KI (Fidolin):** Worker-Pool (Goroutines) mit Polling, der neue Beiträge prüft — analog zum bestehenden Fidolin-Muster, nur für Text statt Bilder.
- **DDL-Konvention:** Migrationen via `psql` als Superuser, App-User nur DML.

-----

## 4. Rollen

| Rolle            | Beschreibung                                                                           |
|------------------|----------------------------------------------------------------------------------------|
| `user`           | Normaler Nutzer in einer oder mehreren Gruppen.                                        |
| Verfasser        | Der/die einen Beitrag erstellt (kein eigenes Feld, ergibt sich aus `posts.author_id`). |
| Markierte Person | Wer sich versprochen hat (`posts.tagged_user_id`, optional).                            |
| `moderator`      | Zugriff aufs Moderations-Tool; gibt Wörter für den öffentlichen Duden frei.            |
| `admin`          | Vollzugriff, Einstellungen, Sperren.                                                   |

Gruppen-interne Rolle separat: `group_members.role` ∈ `owner | member`.

-----

## 5. Die drei Ebenen

- **Feed** — laufender Stream pro Gruppe, alles Spontane.
- **Übersicht** — Highlights der Gruppe. Insider landen hier direkt; andere Beiträge kann man **anpinnen** (`posts.is_pinned`). Übersicht ist *innerhalb* der Gruppe, nicht global.
- **Duden** — kuratiertes Wörterbuch, A–Z:
  - **Privater Duden** (pro Gruppe): volles Wort + Erklärung + O-Ton + wer’s war.
  - **Öffentlicher Duden** (global): **nur das Wort** (= das Hashtag), optional Namen (siehe §8). Gleiche/ähnliche Versprecher werden zu einem Eintrag mit Zähler aggregiert.

**Hashtag-Regel:** Ein Hashtag = genau ein Wort, max. 12 Buchstaben. Das geteilte Wort *ist* das Hashtag. Validierung: `^#?[A-Za-zÄÖÜäöüß]{1,12}$`.

-----

## 6. Datenmodell (PostgreSQL)

### users

| Spalte            | Typ                | Notiz                                           |
|-------------------|--------------------|-------------------------------------------------|
| id                | BIGSERIAL PK       |                                                 |
| email             | TEXT UNIQUE        | verifiziert                                     |
| email_verified_at | TIMESTAMPTZ        |                                                 |
| password_hash     | TEXT               | bcrypt                                          |
| display_name      | TEXT               |                                                 |
| role              | TEXT               | `user | moderator | admin`, default `user`      |
| status            | TEXT               | `active | suspended | banned`, default `active` |
| invited_by        | BIGINT FK→users.id | NULL nur beim allerersten Account               |
| created_at        | TIMESTAMPTZ        |                                                 |

### groups (Kreise)

| Spalte      | Typ                | Notiz                                         |
|-------------|--------------------|-----------------------------------------------|
| id          | BIGSERIAL PK       |                                               |
| name        | TEXT               |                                               |
| owner_id    | BIGINT FK→users.id |                                               |
| max_members | INT                | default `30` — `TODO: finalen Wert festlegen` |
| created_at  | TIMESTAMPTZ        |                                               |

### group_members

| Spalte     | Typ                 | Notiz                                  |
|------------|---------------------|----------------------------------------|
| group_id   | BIGINT FK→groups.id |                                        |
| user_id    | BIGINT FK→users.id  |                                        |
| role       | TEXT                | `owner | member`                       |
| invited_by | BIGINT FK→users.id  | für Rückverfolgung der Einladungskette |
| joined_at  | TIMESTAMPTZ         |                                        |
|            |                     | PK (group_id, user_id)                 |

### invitations

| Spalte     | Typ                | Notiz                                    |
|------------|--------------------|------------------------------------------|
| id         | BIGSERIAL PK       |                                          |
| group_id   | BIGINT FK          |                                          |
| inviter_id | BIGINT FK→users.id |                                          |
| token      | TEXT UNIQUE        | sha256, einmalig                         |
| email      | TEXT               | optional vorab                           |
| status     | TEXT               | `pending | accepted | expired | revoked` |
| expires_at | TIMESTAMPTZ        |                                          |
| created_at | TIMESTAMPTZ        |                                          |

### posts (die Versprecher/Verhörer)

| Spalte              | Typ                | Notiz                                  |
|---------------------|--------------------|----------------------------------------|
| id                  | BIGSERIAL PK       |                                        |
| group_id            | BIGINT FK          |                                        |
| author_id           | BIGINT FK→users.id | Verfasser                              |
| tagged_user_id      | BIGINT FK→users.id | wer sich versprochen hat, NULL möglich |
| word                | TEXT               | das verdrehte Wort                     |
| word_normalized     | TEXT               | lowercase/trim, für Aggregation        |
| explanation         | TEXT               | Bedeutung/Herkunft                     |
| kind                | TEXT               | `versprecher | verhoerer`              |
| ai_kind_suggestion  | TEXT               | Fidolins Vorschlag                     |
| ai_meant_suggestion | TEXT               | „gemeint war: …” (Vorschlag)           |
| meant_confirmed     | TEXT               | vom Verfasser bestätigt/geändert       |
| voice_url           | TEXT               | optional, presigned key                |
| ai_score            | NUMERIC(3,2)       | Moderationswert 0–1                    |
| status              | TEXT               | `visible | pending_review | blocked`   |
| is_pinned           | BOOLEAN            | in Übersicht, default false            |
| created_at          | TIMESTAMPTZ        |                                        |

### publications (Einwilligung für öffentlichen Duden)

| Spalte                  | Typ                | Notiz                            |
|-------------------------|--------------------|----------------------------------|
| id                      | BIGSERIAL PK       |                                  |
| post_id                 | BIGINT FK UNIQUE   |                                  |
| requested_by            | BIGINT FK→users.id |                                  |
| author_consent          | BOOLEAN            | default false                    |
| tagged_consent          | BOOLEAN            | NULL = noch keine Antwort        |
| name_visible_author     | BOOLEAN            | default false                    |
| name_visible_tagged     | BOOLEAN            | default false                    |
| state                   | TEXT               | `pending | public | revoked`     |
| public_word             | TEXT               | normalisiert, was im Duden steht |
| created_at / decided_at | TIMESTAMPTZ        |                                  |

### public_duden_entries (aggregiert)

| Spalte           | Typ                | Notiz                     |
|------------------|--------------------|---------------------------|
| id               | BIGSERIAL PK       |                           |
| word_normalized  | TEXT UNIQUE        | Aggregations-Schlüssel    |
| display_word     | TEXT               |                           |
| kind             | TEXT               | `versprecher | verhoerer` |
| occurrence_count | INT                | „auch X anderen passiert” |
| approved         | BOOLEAN            | von Moderator freigegeben |
| approved_by      | BIGINT FK→users.id |                           |
| first_seen_at    | TIMESTAMPTZ        |                           |

### reactions

| Spalte  | Typ       | Notiz                                            |
|---------|-----------|--------------------------------------------------|
| post_id | BIGINT FK |                                                  |
| user_id | BIGINT FK |                                                  |
| emoji   | TEXT      | nur `😂 ❤️ 😭` (CHECK)                             |
|         |           | PK (post_id, user_id) — eine Reaktion pro Person |

### comments

| Spalte     | Typ          | Notiz                                |
|------------|--------------|--------------------------------------|
| id         | BIGSERIAL PK |                                      |
| post_id    | BIGINT FK    |                                      |
| author_id  | BIGINT FK    |                                      |
| text       | TEXT         |                                      |
| ai_score   | NUMERIC(3,2) |                                      |
| status     | TEXT         | `visible | pending_review | blocked` |
| created_at | TIMESTAMPTZ  |                                      |

### word_of_month / word_of_month_votes

- `word_of_month`: id, group_id, year INT, month INT, winning_post_id FK, decided_at. UNIQUE (group_id, year, month).
- `word_of_month_votes`: group_id, year, month, user_id, post_id. UNIQUE (group_id, year, month, user_id).

### word_of_year / word_of_year_votes

- `word_of_year`: id, group_id, year INT, winning_post_id FK. UNIQUE (group_id, year).
- `word_of_year_votes`: group_id, year, user_id, post_id (Auswahl nur unter den 12 Monatssiegern).

### moderation_settings

| Spalte                | Typ          | Notiz                             |
|-----------------------|--------------|-----------------------------------|
| id                    | INT PK       | nur eine Zeile (oder pro Instanz) |
| auto_reject_threshold | NUMERIC(3,2) | default `0.85`                    |
| handoff_threshold     | NUMERIC(3,2) | default `0.60`                    |

### moderation_queue (oder Sicht auf posts/comments mit `pending_review`)

- Einträge, die Fidolin zur Prüfung markiert hat, mit Bezug auf post_id/comment_id, Score, Grund.

-----

## 7. Fidolin — KI-Vertrag

Fidolin ist **keine** Bot-Persona im Feed (taucht nie als Nutzer auf, schreibt keine Kommentare). Er läuft still im Hintergrund und hat **zwei Aufgaben**:

**A) Moderation** — prüft Beitrag/Kommentar auf Hass/Übergriffiges. Die lustigen Versprecher selbst werden **nicht** rausgefiltert.

**B) Verstehen** — schlägt vor, *was gemeint war* und welche Sorte (Versprecher/Verhörer). **Nur Vorschlag**, der Verfasser bestätigt/ändert. Die KI rät beim Verdrehten auch mal daneben → niemals automatisch final.

**Ein-/Ausgabe (LLM-Call, reines JSON):**

```json
// IN:  { "word": "...", "explanation": "..." }
// OUT:
{
  "score": 0.07,                 // 0–1, Problemgrad
  "reason": "harmloser Versprecher",
  "kind_suggestion": "versprecher",
  "meant_suggestion": "Spatzl + Baby"
}
```

**Schwellen (konfigurierbar via moderation_settings):**

- `score ≥ auto_reject_threshold (0.85)` → `status = blocked`
- `score ≥ handoff_threshold (0.60)` → `status = pending_review` (in Queue)
- sonst → `status = visible`

Fallback bei KI-Fehler: `pending_review` (sicherheitshalber Mensch drüber).

-----

## 8. Einwilligungs-Logik (genau so umsetzen)

Default: jeder Post ist **privat** (nur Gruppe). Der Weg in den öffentlichen Duden:

```
darf_öffentlich =
    author_consent == true
    AND ( tagged_user_id IS NULL? NEIN → siehe Sonderfälle
          OR author_id == tagged_user_id        // eigener Versprecher
          OR tagged_consent == true )           // markierte Person stimmt zu
```

Regeln:

1. **Beide stimmen zu** (Verfasser + markierte Person) → `state = public`, Wort wandert in `public_duden_entries` (nach Moderator-Freigabe, §9).
2. **Eigener Versprecher** (author == tagged) → Zustimmung des Verfassers allein reicht.
3. **Markierte Person nicht in der Gruppe / nicht vorhanden** → kann nicht zustimmen → bleibt privat.
4. **Keine Antwort / Schweigen** → gilt als **Nein**. (kein Auto-Approve nach Timeout)
5. **Widerruf jederzeit:** Verfasser *oder* markierte Person setzt `state = revoked` → Eintrag verschwindet aus dem öffentlichen Duden, bleibt in der Gruppe.

**Namen im öffentlichen Duden:**

- Standard = **nur das Wort**, anonym.
- Name wird nur gezeigt, wenn die jeweilige Person ihn freigibt: Verfasser über `name_visible_author`, markierte Person über `name_visible_tagged`. Jede:r entscheidet nur über den **eigenen** Namen.

-----

## 9. Öffentlicher Duden — Aggregation & Freigabe

- Beim Veröffentlichen wird über `word_normalized` aggregiert: existiert der Eintrag schon, `occurrence_count += 1` („auch X anderen passiert”); sonst neu anlegen.
- Geteilt wird **nur** das anonyme Wort + Zähler (+ optional freigegebene Namen). Niemals der private Inhalt/O-Ton aus den Gruppen.
- **Sich anschließen ist opt-in.** Wer sich nicht anschließt, dessen Wort bleibt in der kleineren Gruppe.
- Ein Eintrag erscheint öffentlich erst, wenn ein **Moderator** ihn freigibt (`approved = true`). So bleibt der öffentliche Duden sauber.

-----

## 10. API-Endpoints (Prefix `/v1`)

| Methode | Pfad                                | Zweck                                                   | Wer                |
|---------|-------------------------------------|---------------------------------------------------------|--------------------|
| POST    | `/auth/register`                    | Registrierung (nur mit gültigem Invite-Token)           | offen              |
| POST    | `/auth/login`                       | Login → JWT                                             | offen              |
| POST    | `/auth/verify-email`                | Mail bestätigen                                         | offen              |
| POST    | `/groups`                           | Gruppe gründen                                          | user               |
| GET     | `/groups`                           | meine Gruppen                                           | user               |
| GET     | `/groups/:id`                       | Gruppen-Details                                         | member             |
| POST    | `/groups/:id/invite`                | Einladungslink erzeugen                                 | member/owner       |
| POST    | `/invitations/:token/accept`        | Einladung annehmen                                      | offen (eingeloggt) |
| GET     | `/groups/:id/feed`                  | Feed                                                    | member             |
| GET     | `/groups/:id/overview`              | Übersicht (gepinnt + Insider)                           | member             |
| POST    | `/groups/:id/posts`                 | Beitrag anlegen → Fidolin prüft + schlägt „gemeint” vor | member             |
| PATCH   | `/posts/:id`                        | „gemeint”/kind bestätigen/ändern                        | author             |
| POST    | `/posts/:id/pin`                    | in Übersicht pinnen/entpinnen                           | member             |
| POST    | `/posts/:id/react`                  | Reaktion (😂❤️😭)                                         | member             |
| GET     | `/posts/:id/comments`               | Kommentare                                              | member             |
| POST    | `/posts/:id/comments`               | Kommentieren → Fidolin prüft                            | member             |
| POST    | `/posts/:id/publish-request`        | Veröffentlichung anstoßen, benachrichtigt Beteiligte    | author             |
| POST    | `/posts/:id/consent`                | Zustimmung + Namens-Sichtbarkeit setzen                 | author / tagged    |
| POST    | `/posts/:id/revoke`                 | zurück in die Gruppe holen                              | author / tagged    |
| GET     | `/groups/:id/duden`                 | privater Duden der Gruppe (A–Z)                         | member             |
| GET     | `/duden`                            | öffentlicher Duden (Wort + Zähler)                      | offen              |
| POST    | `/groups/:id/word-of-month/vote`    | Monatswort wählen                                       | member             |
| GET     | `/groups/:id/word-of-month/:yyyymm` | Monatswort ansehen                                      | member             |
| POST    | `/groups/:id/word-of-year/vote`     | Jahreswort (aus 12 Siegern)                             | member             |
| GET     | `/groups/:id/word-of-year/:yyyy`    | Jahreswort + Rückblick                                  | member             |
| GET     | `/me`                               | Profil + Einstellungen                                  | user               |

**Moderations-Tool (eigene Oberfläche/Routen, `moderator`/`admin`):**

| Methode   | Pfad                         | Zweck                                         |
|-----------|------------------------------|-----------------------------------------------|
| GET       | `/mod/queue`                 | markierte Beiträge/Kommentare                 |
| POST      | `/mod/posts/:id/decision`    | freigeben / blockieren                        |
| POST      | `/mod/comments/:id/decision` | freigeben / blockieren                        |
| POST      | `/mod/duden/:word/approve`   | Wort für öffentlichen Duden freigeben         |
| GET / PUT | `/mod/settings`              | Schwellen (auto_reject, handoff) lesen/ändern |
| POST      | `/mod/users/:id/suspend`     | Nutzer sperren                                |

-----

## 11. Sicherheit

- **Nur auf Einladung** ist die primäre Bot-Bremse. Jeder Account hat `invited_by` → ganze Einladungskette rückverfolgbar und kappbar.
- E-Mail-Verifizierung Pflicht. Ein Mensch = ein Profil; Doppel-Accounts untersagt.
- Rate-Limiting auf Schreib-Endpoints; echte Client-IP über TrustedProxies.
- Sperren über `users.status` (`suspended/banned`). IP-Bann nur als grober Zusatz — wegen CGNAT/VPN kein verlässliches Einzelsignal.
- DSGVO: Sprachnotizen = personenbezogene Daten → Einwilligung, Löschkonzept, Export. Impressum + Datenschutz von Anfang an.

-----

## 12. Rituale & Erinnerungen

- **Wort des Monats → Wort des Jahres:** monatliche Abstimmung pro Gruppe; aus den 12 Siegern das Jahreswort.
- **Geburtstag eines Wortes:** Push „X wird heute 1 Jahr alt 🎂”.
- **Jahresrückblick pro Gruppe:** Wort des Jahres, fleißigste Versprecher-Person.
- **Sperrbildschirm-Erinnerung (PWA):** „Vor 5 Monaten hat … gesagt”.
- **Nur Reaktionen** (😂 ❤️ 😭), **keine Downvotes**.

## 13. Bewusst weglassen

- Login über Dritte. Follower-Zahlen / Reichweiten-Ranking. Downvotes. Große öffentliche Gruppen.

-----

## 14. Empfohlene Build-Reihenfolge

1. Scaffold (Go/Gin + Postgres-Migrationen), Auth + Invite-Registrierung.
2. Gruppen + Mitglieder + Einladungen (inkl. `max_members`).
3. Posts anlegen + Feed; Fidolin-Worker (Moderation + „gemeint”-Vorschlag).
4. Reaktionen + Kommentare (mit Moderation).
5. Übersicht (Pinnen) + privater Duden.
6. Einwilligungs-Flow + öffentlicher Duden + Aggregation.
7. Moderations-Tool (Queue, Schwellen, Duden-Freigabe).
8. Rituale (Wort des Monats/Jahres, Erinnerungen/Push).

## 15. Offene Punkte (`TODO`)

- **Tagline** final wählen (Favorit: „Heute schon verhört?”).
- **Wort des Monats:** pro Gruppe (Kern) — zusätzlich global im öffentlichen Duden? `TODO`
- **Exaktes Gruppenlimit** (`max_members`) am Gefühl justieren.
- **Ähnlichkeits-Matching** für Aggregation: erst exakt über `word_normalized`, später ggf. Fuzzy (Levenshtein) — `TODO`.
- **Voice-Storage**: Bucket-Wahl + TTL der presigned URLs festlegen.
