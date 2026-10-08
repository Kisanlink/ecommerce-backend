# Recipes Migration Guide

This guide explains how to create the recipes table and seed initial data in NeonDB.

## Quick Start

### Option 1: Using the Automated Script (Recommended)

**Linux/Mac:**
```bash
chmod +x migrations/run_recipes_migration.sh
./migrations/run_recipes_migration.sh "postgresql://user:pass@host:5432/dbname?sslmode=require"
```

**Windows (PowerShell):**
```powershell
.\migrations\run_recipes_migration.ps1 -DatabaseUrl "postgresql://user:pass@host:5432/dbname?sslmode=require"
```

### Option 2: Using Environment Variable

**Linux/Mac:**
```bash
export DATABASE_URL="postgresql://user:pass@host:5432/dbname?sslmode=require"
./migrations/run_recipes_migration.sh
```

**Windows (PowerShell):**
```powershell
$env:DATABASE_URL = "postgresql://user:pass@host:5432/dbname?sslmode=require"
.\migrations\run_recipes_migration.ps1
```

### Option 3: Manual Execution

```bash
# Step 1: Create table
psql "your-connection-string" -f migrations/create_recipes_table.sql

# Step 2: Seed data
psql "your-connection-string" -f migrations/seed_recipes.sql
```

## What Gets Created

### 1. Recipes Table
- Full table structure with all columns
- Indexes for performance (category, rating, search, etc.)
- Constraints for data validation
- Support for soft deletes

### 2. Seeded Recipes
Four moringa recipes are automatically inserted:

1. **Dosa Moringa** (Breakfast)
   - ID: RECP00000001
   - Rating: 4.5/5
   - Reviews: 12

2. **Moringa Dessert Cake** (Dessert)
   - ID: RECP00000002
   - Rating: 4.7/5
   - Reviews: 18

3. **Moringa Smoothie** (Beverage)
   - ID: RECP00000003
   - Rating: 4.8/5
   - Reviews: 25

4. **Moringa Tea** (Beverage)
   - ID: RECP00000004
   - Rating: 4.6/5
   - Reviews: 20

## Table Structure

The recipes table includes:

- **Basic Info**: name, category, description, demo_description
- **Details**: prep_time, servings, difficulty, image
- **Content**: ingredients (array), instructions (array), pro_tips (array)
- **Nutrition**: nutrition_facts (JSONB)
- **Ratings**: rating, reviews
- **Audit**: created_at, updated_at, created_by, updated_by, deleted_at, deleted_by

## Verification

After running the migration, verify the data:

```sql
-- Count recipes
SELECT COUNT(*) FROM recipes;

-- List all recipes
SELECT recipe_id, name, category, rating, reviews FROM recipes ORDER BY created_at;

-- Check table structure
\d recipes
```

## Troubleshooting

### Error: "relation recipes already exists"
The table already exists. You can either:
- Drop and recreate: `DROP TABLE IF EXISTS recipes CASCADE;` then run migration again
- Or just run the seed script: `psql "connection-string" -f migrations/seed_recipes.sql`

### Error: "duplicate key value violates unique constraint"
The recipes are already seeded. You can:
- Delete existing recipes: `DELETE FROM recipes;` then run seed again
- Or skip seeding if data already exists

### Error: "psql: command not found"
Install PostgreSQL client tools:
- **Linux**: `sudo apt-get install postgresql-client` (Ubuntu/Debian) or `sudo yum install postgresql` (RHEL/CentOS)
- **Mac**: `brew install postgresql`
- **Windows**: Download from [PostgreSQL website](https://www.postgresql.org/download/windows/)

## Next Steps

After successful migration:

1. **Test the API**: See `docs/RECIPES_API_TESTING.md` for complete API testing guide
2. **Update Image URLs**: Replace placeholder S3 URLs in seed data with actual image URLs
3. **Add More Recipes**: Use the API endpoints to add more recipes

## Files

- `create_recipes_table.sql` - Creates the recipes table with indexes
- `seed_recipes.sql` - Inserts 4 initial moringa recipes
- `run_recipes_migration.sh` - Linux/Mac automation script
- `run_recipes_migration.ps1` - Windows PowerShell automation script

