#!/bin/bash
# Quick migration script with your connection string
# Just run: bash migrations/run_migration_now.sh

set -e

# Your connection string
CONNECTION_STRING="postgresql://neondb_owner:npg_0MOY2LSjTiky@ep-orange-field-a1tfsvjg-pooler.ap-southeast-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CREATE_TABLE_SCRIPT="$SCRIPT_DIR/create_recipes_table.sql"
SEED_SCRIPT="$SCRIPT_DIR/seed_recipes.sql"

echo "Starting recipes migration..."
echo "Connecting to NeonDB..."
echo ""

# Step 1: Create recipes table
echo "Step 1: Creating recipes table..."
psql "$CONNECTION_STRING" -f "$CREATE_TABLE_SCRIPT"

if [ $? -eq 0 ]; then
    echo "Recipes table created successfully"
else
    echo "Failed to create recipes table"
    exit 1
fi

echo ""

# Step 2: Seed data
echo "Step 2: Seeding initial recipes..."
psql "$CONNECTION_STRING" -f "$SEED_SCRIPT"

if [ $? -eq 0 ]; then
    echo "Recipes seeded successfully"
else
    echo "Failed to seed recipes"
    exit 1
fi

echo ""

# Verify
echo "Verifying data..."
psql "$CONNECTION_STRING" -c "SELECT COUNT(*) as total_recipes FROM recipes;"
psql "$CONNECTION_STRING" -c "SELECT recipe_id, name, category, rating FROM recipes ORDER BY recipe_id;"
echo ""

echo "Migration completed successfully!"
echo "You can now test the recipes API endpoints!"
echo "See docs/RECIPES_API_TESTING.md for API testing guide"

