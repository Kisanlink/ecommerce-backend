#!/usr/bin/env bash
#
# Helper script to copy the entire PostgreSQL schema/data into Neon.
# Usage:
#   ./scripts/neon-migrate.sh <source-dsn> <target-dsn> [dump-mode]
#
# Example (local pgAdmin DB -> Neon):
#   ./scripts/neon-migrate.sh \
#     "postgresql://postgres:12345678@localhost:5432/DemoDB?sslmode=disable" \
#     "postgresql://neondb_owner:npg_0MOY2LSjTiky@ep-orange-field-a1tfsvjg-pooler.ap-southeast-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require"
#
# dump-mode (optional) can be:
#   --schema-only  -> migrate schema without data
#   --data-only    -> migrate only data

set -euo pipefail

SOURCE_DSN="${1:-}"
TARGET_DSN="${2:-}"
DUMP_MODE="${3:-}"

if [[ -z "$SOURCE_DSN" || -z "$TARGET_DSN" ]]; then
  echo "Usage: $0 <source-dsn> <target-dsn> [--schema-only|--data-only]" >&2
  exit 1
fi

command -v pg_dump >/dev/null 2>&1 || {
  echo "pg_dump not found. Install PostgreSQL client tools." >&2
  exit 1
}
command -v pg_restore >/dev/null 2>&1 || {
  echo "pg_restore not found. Install PostgreSQL client tools." >&2
  exit 1
}

TMP_DUMP="neon_migration_$(date +%Y%m%d_%H%M%S).dump"

echo "🔄 Exporting source database..."
pg_dump --format=custom --no-owner --no-privileges $DUMP_MODE \
  --dbname="$SOURCE_DSN" \
  --file="$TMP_DUMP"

echo "🚀 Importing dump into Neon..."
pg_restore --clean --if-exists --no-owner $DUMP_MODE \
  --dbname="$TARGET_DSN" \
  "$TMP_DUMP"

echo "🧹 Cleaning up temporary dump..."
rm -f "$TMP_DUMP"

echo "✅ Migration finished successfully."



