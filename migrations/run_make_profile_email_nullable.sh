#!/bin/bash

# Script to make email nullable in profiles table
# This allows phone-only users to have profiles without email

# Connection string - update this with your actual connection string
CONNECTION_STRING="${DB_CONNECTION_STRING:-postgresql://neondb_owner:npg_0MOY2LSjTiky@ep-orange-field-a1tfsvjg-pooler.ap-southeast-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require}"

echo "=========================================="
echo "Making email nullable in profiles table"
echo "=========================================="
echo ""

# Run the migration
echo "Step 1: Running migration to make email nullable..."
psql "$CONNECTION_STRING" -f migrations/make_profile_email_nullable.sql

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Migration completed successfully!"
    echo ""
    echo "Verification:"
    echo "Run this query to check:"
    echo "  SELECT COUNT(*) FROM profiles WHERE email IS NULL;"
    echo "  SELECT COUNT(*) FROM profiles WHERE email IS NOT NULL AND email != '';"
else
    echo ""
    echo "❌ Migration failed. Please check the error above."
    exit 1
fi

