#!/bin/bash
# Quick migration script for pending_recipe_reviews table
# Just run: bash migrations/run_pending_recipe_reviews_migration.sh

set -e

# Your connection string
CONNECTION_STRING="postgresql://neondb_owner:npg_0MOY2LSjTiky@ep-orange-field-a1tfsvjg-pooler.ap-southeast-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CREATE_TABLE_SCRIPT="$SCRIPT_DIR/create_pending_recipe_reviews_table.sql"

echo "Starting pending_recipe_reviews migration..."
echo "Connecting to NeonDB..."
echo ""

# Create pending_recipe_reviews table
echo "Creating pending_recipe_reviews table..."
psql "$CONNECTION_STRING" -f "$CREATE_TABLE_SCRIPT"

if [ $? -eq 0 ]; then
    echo "pending_recipe_reviews table created successfully"
else
    echo "Failed to create pending_recipe_reviews table"
    exit 1
fi

echo ""

# Verify
echo "Verifying table creation..."
psql "$CONNECTION_STRING" -c "\d pending_recipe_reviews"
psql "$CONNECTION_STRING" -c "SELECT COUNT(*) as total_pending_reviews FROM pending_recipe_reviews;"
echo ""

echo "Migration completed successfully!"
echo "You can now submit recipes for review using the API!"

