#!/usr/bin/env bash
# migrate.sh — wendet ausstehende .up.sql-Migrationen an (DDL als Superuser).
# Konvention (§3): Migrationen laufen als Superuser via psql, der App-User nur DML.
#
# Nutzung:
#   ADMIN_DATABASE_URL="postgres://postgres@127.0.0.1:5432/insider_dev" scripts/migrate.sh
# oder (lokal als OS-User postgres):
#   sudo -u postgres scripts/migrate.sh        # nutzt Peer-Auth auf insider_dev
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIG_DIR="$ROOT_DIR/migrations"

# psql-Aufruf: bevorzugt ADMIN_DATABASE_URL, sonst lokale DB insider_dev.
if [[ -n "${ADMIN_DATABASE_URL:-}" ]]; then
    PSQL=(psql -v ON_ERROR_STOP=1 "$ADMIN_DATABASE_URL")
else
    PSQL=(psql -v ON_ERROR_STOP=1 -d "${DB_NAME:-insider_dev}")
fi

# Versionsverwaltung.
"${PSQL[@]}" -c "CREATE TABLE IF NOT EXISTS schema_migrations (
    version    TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);" >/dev/null

applied="$("${PSQL[@]}" -tAc "SELECT version FROM schema_migrations;")"

for up in "$MIG_DIR"/*.up.sql; do
    [[ -e "$up" ]] || continue
    base="$(basename "$up")"
    version="${base%.up.sql}"
    if grep -qxF "$version" <<<"$applied"; then
        echo "skip   $version (bereits angewendet)"
        continue
    fi
    echo "apply  $version"
    # Migration + Versionseintrag in einer Transaktion.
    "${PSQL[@]}" --single-transaction \
        -f "$up" \
        -c "INSERT INTO schema_migrations (version) VALUES ('$version');"
done

echo "Migrationen aktuell."
