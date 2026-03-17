#!/usr/bin/env bash

set -euo pipefail

CONTAINER="${POSTGRES_CONTAINER:-infra-postgres-1}"
PG_USER="${POSTGRES_USER:-postgres}"
USE_LOCAL="${USE_LOCAL_PSQL:-0}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MIGRATIONS_DIR="$SCRIPT_DIR/../migrations"

green()  { printf '\033[0;32mâœ“ %s\033[0m\n' "$*"; }
red()    { printf '\033[0;31mâœ— %s\033[0m\n' "$*"; >&2 echo "ERROR: $*"; }
yellow() { printf '\033[1;33mÂ» %s\033[0m\n' "$*"; }


if [ "$USE_LOCAL" = "1" ]; then
  export PGPASSWORD="${DB_PASSWORD:-postgres}"
  PG_HOST="${DB_HOST:-localhost}"
  PG_PORT="${DB_PORT:-5432}"

  psql_exec_db() { psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$1" -q -c "$2"; }
  psql_exec()    { psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -q -c "$1"; }
  psql_file()    { psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$1" -q -f "$2"; }
  db_exists()    { psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -tAc \
                     "SELECT 1 FROM pg_database WHERE datname='$1'" | grep -q 1; }
else
  psql_exec_db() { docker exec "$CONTAINER" psql -U "$PG_USER" -d "$1" -q -c "$2"; }
  psql_exec()    { docker exec "$CONTAINER" psql -U "$PG_USER" -q -c "$1"; }
  psql_file()    { docker exec -i "$CONTAINER" psql -U "$PG_USER" -d "$1" -q < "$2"; }
  db_exists()    { docker exec "$CONTAINER" psql -U "$PG_USER" -tAc \
                     "SELECT 1 FROM pg_database WHERE datname='$1'" | grep -q 1; }
fi

yellow "Waiting for PostgreSQL to be ready..."
for i in $(seq 1 30); do
  if [ "$USE_LOCAL" = "1" ]; then
    pg_isready -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -q 2>/dev/null && break
  else
    docker exec "$CONTAINER" pg_isready -U "$PG_USER" -q 2>/dev/null && break
  fi
  printf "  Attempt %d/30...\r" "$i"
  sleep 2
done
green "PostgreSQL is ready"

DATABASES=(auth orders payments wallets products stores notifications fraud)

yellow "Creating databases..."
for db in "${DATABASES[@]}"; do
  if db_exists "$db"; then
    printf "  %-12s already exists\n" "'$db'"
  else
    psql_exec "CREATE DATABASE $db;"
    green "  Created database '$db'"
  fi
done


apply_migrations() {
  local db="$1" dir="$2"
  [ -d "$dir" ] || return 0

  yellow "  Migrating database '$db' from $dir..."
  local applied=0
  for file in "$dir"/*.sql; do
    [ -f "$file" ] || continue
    printf "    Applying %-40s" "$(basename "$file")..."
    psql_file "$db" "$file"
    printf " done\n"
    applied=$((applied+1))
  done
  [ "$applied" -eq 0 ] && echo "    (no .sql files found)" || green "  $applied migration(s) applied to '$db'"
}

apply_migrations auth      "$MIGRATIONS_DIR/auth"
apply_migrations orders    "$MIGRATIONS_DIR/order"
apply_migrations payments  "$MIGRATIONS_DIR/payment"
apply_migrations wallets   "$MIGRATIONS_DIR/wallet"
apply_migrations products  "$MIGRATIONS_DIR/product"
apply_migrations stores    "$MIGRATIONS_DIR/store"
apply_migrations notifications "$MIGRATIONS_DIR/notification"
apply_migrations fraud         "$MIGRATIONS_DIR/fraud"

OUTBOX_SQL="$MIGRATIONS_DIR/outbox/001_create_outbox.sql"
if [ -f "$OUTBOX_SQL" ]; then
  yellow "Applying outbox migration to all service databases..."
  for db in auth orders payments wallets products stores notifications fraud; do
    printf "    %-10s " "$db"
    psql_file "$db" "$OUTBOX_SQL"
    printf "done\n"
  done
  green "Outbox tables ready"
else
  red "Outbox migration not found at $OUTBOX_SQL â€” skipping"
fi

echo ""
echo "â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€"
green "All migrations completed successfully"
echo "â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€"
echo ""
echo "Databases ready:"
for db in "${DATABASES[@]}"; do
  printf "  âœ“ %s\n" "$db"
done
