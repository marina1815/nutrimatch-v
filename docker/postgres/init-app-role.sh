#!/usr/bin/env bash
set -euo pipefail

: "${APP_DB_USER:?APP_DB_USER is required}"
: "${APP_DB_PASSWORD:?APP_DB_PASSWORD is required}"

psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
  --set=app_user="$APP_DB_USER" \
  --set=app_password="$APP_DB_PASSWORD" <<'SQL'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'app_user', :'app_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'app_user')\gexec

SELECT format('ALTER ROLE %I LOGIN PASSWORD %L', :'app_user', :'app_password')
WHERE EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'app_user')\gexec
SQL
