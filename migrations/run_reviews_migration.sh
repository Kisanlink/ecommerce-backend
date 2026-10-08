#!/bin/bash
# Migration script for reviews table
# Usage: bash migrations/run_reviews_migration.sh

set -e

# Your connection string
CONNECTION_STRING="postgresql://neondb_owner:npg_0MOY2LSjTiky@ep-orange-field-a1tfsvjg-pooler.ap-southeast-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CREATE_TABLE_SCRIPT="$SCRIPT_DIR/create_reviews_table.sql"
SEED_SCRIPT="$SCRIPT_DIR/seed_reviews.sql"

echo "Starting reviews migration..."
echo "Connecting to NeonDB..."
echo ""

# Step 1: Create reviews table
echo "Step 1: Creating reviews table..."
psql "$CONNECTION_STRING" -f "$CREATE_TABLE_SCRIPT"

if [ $? -eq 0 ]; then
    echo "Reviews table created successfully"
else
    echo "Failed to create reviews table"
    exit 1
fi

echo ""

# Step 2: Seed reviews for PROD00000004
echo "Step 2: Seeding reviews for PROD00000004 (Moringa Powder)..."
psql "$CONNECTION_STRING" -f "$SEED_SCRIPT"

if [ $? -eq 0 ]; then
    echo "Reviews seeded successfully"
else
    echo "Failed to seed reviews"
    exit 1
fi

echo ""

# Verify
echo "Verifying data..."
psql "$CONNECTION_STRING" -c "SELECT COUNT(*) as total_reviews FROM reviews WHERE product_id = 'PROD00000004';"
psql "$CONNECTION_STRING" -c "SELECT id, user_name, rating, title, likes_count, is_verified FROM reviews WHERE product_id = 'PROD00000004' ORDER BY review_date DESC LIMIT 5;"
echo ""

echo "Migration completed successfully!"
echo "You can now test the reviews API endpoints!"
echo "See docs/RECIPES_API_TESTING.md for API testing guide"

