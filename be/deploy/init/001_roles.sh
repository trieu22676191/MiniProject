#!/bin/sh
set -eu

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
  -v owner_password="$MINI_OWNER_PASSWORD" \
  -v app_password="$MINI_APP_PASSWORD" \
  -v relay_password="$MINI_RELAY_PASSWORD" <<'SQL'
CREATE ROLE mini_owner LOGIN PASSWORD :'owner_password' NOSUPERUSER NOBYPASSRLS;
CREATE ROLE mini_app LOGIN PASSWORD :'app_password' NOSUPERUSER NOBYPASSRLS;
CREATE ROLE mini_relay LOGIN PASSWORD :'relay_password' NOSUPERUSER BYPASSRLS;
CREATE SCHEMA mini_order AUTHORIZATION mini_owner;
CREATE SCHEMA mini_stock AUTHORIZATION mini_owner;
GRANT USAGE ON SCHEMA mini_order, mini_stock TO mini_app;
GRANT USAGE ON SCHEMA mini_order, mini_stock TO mini_relay;
ALTER DEFAULT PRIVILEGES FOR ROLE mini_owner IN SCHEMA mini_order, mini_stock
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO mini_app;
ALTER DEFAULT PRIVILEGES FOR ROLE mini_owner IN SCHEMA mini_order, mini_stock
  GRANT USAGE, SELECT ON SEQUENCES TO mini_app;
SQL
