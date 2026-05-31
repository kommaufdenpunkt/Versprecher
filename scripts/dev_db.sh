#!/usr/bin/env bash
# dev_db.sh — legt lokale Dev-Datenbank + App-Rolle (nur DML) an.
# Lokal auszuführen als OS-User mit Postgres-Superuser-Zugriff:
#   sudo -u postgres scripts/dev_db.sh
set -euo pipefail

DB_NAME="${DB_NAME:-insider_dev}"
APP_USER="${APP_USER:-insider_app}"
APP_PW="${APP_PW:-insider_dev_pw}"

psql -v ON_ERROR_STOP=1 <<SQL
DO \$\$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '${APP_USER}') THEN
        CREATE ROLE ${APP_USER} LOGIN PASSWORD '${APP_PW}';
    END IF;
END
\$\$;
SELECT 'CREATE DATABASE ${DB_NAME} OWNER postgres'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = '${DB_NAME}')\gexec
SQL

psql -v ON_ERROR_STOP=1 -d "${DB_NAME}" <<SQL
GRANT CONNECT ON DATABASE ${DB_NAME} TO ${APP_USER};
GRANT USAGE ON SCHEMA public TO ${APP_USER};
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO ${APP_USER};
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO ${APP_USER};
SQL

echo "Dev-DB '${DB_NAME}' und Rolle '${APP_USER}' bereit."
