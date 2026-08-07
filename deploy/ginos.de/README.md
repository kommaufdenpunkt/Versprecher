# Betrieb unter ginos.de

Vorlagen, um den [Fahrstunden-Nachweis](../../FAHRSTUNDEN.md) unter einer
eigenen Domain zu betreiben. Die Oberfläche liegt dann direkt auf
`https://ginos.de` — kein Unterpfad nötig.

| Datei | Zweck |
|---|---|
| `Caddyfile` | Reverse-Proxy mit automatischem HTTPS |
| `ginos.env.beispiel` | Umgebung (nach `/etc/ginos.env`, `chmod 600`) |
| `ginos.service` | systemd-Dienst |

## Einrichten

```bash
# 1) Domain: A-/AAAA-Eintrag für ginos.de und www.ginos.de auf den Server

# 2) Anwendung bauen und ablegen
go build -o /usr/local/bin/ginos-server ./cmd/server
useradd --system --shell /usr/sbin/nologin ginos

# 3) Umgebung anlegen — JWT_SECRET und DB-Passwort ersetzen!
cp deploy/ginos.de/ginos.env.beispiel /etc/ginos.env
chmod 600 /etc/ginos.env
openssl rand -base64 48        # Ergebnis als JWT_SECRET eintragen

# 4) Datenbank und Migrationen (DDL als Superuser, §3)
sudo -u postgres DB_NAME=insider scripts/dev_db.sh
sudo -u postgres DB_NAME=insider bash scripts/migrate.sh

# 5) Dienst starten
cp deploy/ginos.de/ginos.service /etc/systemd/system/
systemctl daemon-reload && systemctl enable --now ginos

# 6) Caddy davor
cp deploy/ginos.de/Caddyfile /etc/caddy/Caddyfile
systemctl reload caddy
```

Prüfen: `curl -sf https://ginos.de/healthz` muss `{"status":"ok"}` liefern.

## Das erste Konto

Der allererste Account wird ohne Einladung angelegt und ist Admin:

```bash
curl -X POST https://ginos.de/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"...","password":"...","display_name":"Vorname Nachname"}'
```

Der `display_name` steht später als „Fahrlehrer:in“ im PDF-Kopf.
Ist `REQUIRE_EMAIL_VERIFICATION` aktiv (Standard), muss das zurückgegebene
`email_verification_token` einmal an `POST /v1/auth/verify-email` geschickt
werden.

Danach: `https://ginos.de` im Handy-Browser öffnen, anmelden und
„Zum Home-Bildschirm hinzufügen“ — dann startet der Nachweis wie eine App.

## Worauf zu achten ist

- **`TRUSTED_PROXIES=127.0.0.1`** muss gesetzt sein. Sonst sieht die Anwendung
  nur die IP von Caddy, und das Rate-Limit auf den Anmelde-Endpunkten wäre
  wirkungslos.
- **`JWT_SECRET` niemals** aus dem Beispiel übernehmen — wer es kennt, kann
  sich beliebige Anmeldungen ausstellen.
- **Sicherung:** In den Fahrstunden stecken Unterschriften. Ein regelmäßiger
  `pg_dump` gehört dazu, sonst ist der Nachweis beim Plattenausfall weg.
- **Personenbezogene Daten:** Namen und Unterschriften von Fahrschüler:innen
  sind personenbezogen. Der Zugang gehört hinter das eigene Konto, und die
  Sicherungen gehören verschlüsselt abgelegt.
