# Quick Start - Run Recipes Migration

## Option 1: Using PowerShell Script (Easiest)

### Step 1: Set your database connection variables

Open PowerShell and run:

```powershell
$env:DB_POSTGRES_HOST = "your-neon-host"
$env:DB_POSTGRES_USER = "your-username"
$env:DB_POSTGRES_PASSWORD = "your-password"
$env:DB_POSTGRES_DBNAME = "your-database-name"
$env:DB_POSTGRES_PORT = "5432"  # Optional, defaults to 5432
$env:DB_POSTGRES_SSLMODE = "require"  # Optional, defaults to require
```

### Step 2: Run the migration script

```powershell
.\migrations\run_migration_direct.ps1
```

---

## Option 2: Direct psql Commands (If script doesn't work)

### Step 1: Set password environment variable

```powershell
$env:PGPASSWORD = "your-password"
```

### Step 2: Run SQL files directly

```powershell
# Create table
& "C:\Program Files\PostgreSQL\17\bin\psql.exe" -h your-host -p 5432 -U your-user -d your-dbname -f migrations\create_recipes_table.sql

# Seed data
& "C:\Program Files\PostgreSQL\17\bin\psql.exe" -h your-host -p 5432 -U your-user -d your-dbname -f migrations\seed_recipes.sql
```

---

## Option 3: Using Connection String

If you have a full connection string:

```powershell
# Set connection string
$env:DATABASE_URL = "postgresql://user:password@host:5432/dbname?sslmode=require"

# Run migration
.\migrations\run_recipes_migration_simple.ps1
```

---

## Option 4: Manual psql Session

1. Connect to your database:
```powershell
& "C:\Program Files\PostgreSQL\17\bin\psql.exe" -h your-host -p 5432 -U your-user -d your-dbname
```

2. Once connected, run:
```sql
\i migrations/create_recipes_table.sql
\i migrations/seed_recipes.sql
```

3. Verify:
```sql
SELECT COUNT(*) FROM recipes;
SELECT recipe_id, name, category FROM recipes;
```

---

## Troubleshooting

### Error: "psql: FATAL: password authentication failed"
- Check your password is correct
- Make sure you're using the right username

### Error: "psql: FATAL: no pg_hba.conf entry"
- Check your host and port are correct
- Make sure SSL mode is set correctly (usually "require" for Neon)

### Error: "relation recipes already exists"
- The table already exists. You can either:
  - Drop it: `DROP TABLE IF EXISTS recipes CASCADE;` then run migration again
  - Or just run the seed script if you only need data

### Error: "duplicate key value violates unique constraint"
- Recipes are already seeded. You can:
  - Delete them: `DELETE FROM recipes;` then run seed again
  - Or skip seeding if data already exists

---

## Verify Migration Success

After running migration, verify with:

```powershell
$env:PGPASSWORD = "your-password"
& "C:\Program Files\PostgreSQL\17\bin\psql.exe" -h your-host -p 5432 -U your-user -d your-dbname -c "SELECT recipe_id, name, category, rating FROM recipes;"
```

You should see 4 recipes:
- RECP00000001 - Dosa Moringa
- RECP00000002 - Moringa Dessert Cake  
- RECP00000003 - Moringa Smoothie
- RECP00000004 - Moringa Tea

