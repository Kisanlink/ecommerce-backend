#!/bin/bash
#
# Script to create recipes table and seed initial data
# Usage: ./migrations/run_recipes_migration.sh [database-url]
#
# Example:
#   ./migrations/run_recipes_migration.sh "postgresql://user:pass@host:5432/dbname?sslmode=require"
#
# Or set DATABASE_URL environment variable:
#   export DATABASE_URL="postgresql://user:pass@host:5432/dbname?sslmode=require"
#   ./migrations/run_recipes_migration.sh

set -euo pipefail

# Get database URL from argument or environment variable
DB_URL="${1:-${DATABASE_URL:-}}"

if [[ -z "$DB_URL" ]]; then
    echo "❌ Error: Database URL is required"
    echo ""
    echo "Usage: $0 <database-url>"
    echo "   or: export DATABASE_URL='postgresql://...' && $0"
    echo ""
    echo "Example:"
    echo "  $0 'postgresql://user:pass@host:5432/dbname?sslmode=require'"
    exit 1
fi

# Check if psql is available
command -v psql >/dev/null 2>&1 || {
    echo "❌ Error: psql command not found. Please install PostgreSQL client tools."
    exit 1
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🚀 Starting recipes migration..."
echo ""

# Step 1: Create recipes table
echo "📋 Step 1: Creating recipes table..."
psql "$DB_URL" -f "$SCRIPT_DIR/create_recipes_table.sql"
if [ $? -eq 0 ]; then
    echo "✅ Recipes table created successfully"
else
    echo "❌ Failed to create recipes table"
    exit 1
fi

echo ""

# Step 2: Seed initial recipes
echo "🌱 Step 2: Seeding initial recipes..."
psql "$DB_URL" -f "$SCRIPT_DIR/seed_recipes.sql"
if [ $? -eq 0 ]; then
    echo "✅ Recipes seeded successfully"
else
    echo "❌ Failed to seed recipes"
    exit 1
fi

echo ""
echo "✅ Migration completed successfully!"
echo ""
echo "📊 Summary:"
psql "$DB_URL" -c "SELECT COUNT(*) as total_recipes FROM recipes;"
echo ""
echo "🎉 You can now test the recipes API endpoints!"

