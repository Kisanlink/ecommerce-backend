# PowerShell script to create recipes table and seed initial data
# Usage: .\migrations\run_recipes_migration.ps1 [database-url]
#
# Example:
#   .\migrations\run_recipes_migration.ps1 "postgresql://user:pass@host:5432/dbname?sslmode=require"
#
# Or set DATABASE_URL environment variable:
#   $env:DATABASE_URL = "postgresql://user:pass@host:5432/dbname?sslmode=require"
#   .\migrations\run_recipes_migration.ps1

param(
    [string]$DatabaseUrl = $env:DATABASE_URL
)

if ([string]::IsNullOrEmpty($DatabaseUrl)) {
    Write-Host "❌ Error: Database URL is required" -ForegroundColor Red
    Write-Host ""
    Write-Host "Usage: .\run_recipes_migration.ps1 -DatabaseUrl 'postgresql://...'"
    Write-Host "   or: `$env:DATABASE_URL = 'postgresql://...'; .\run_recipes_migration.ps1"
    Write-Host ""
    Write-Host "Example:"
    Write-Host "  .\run_recipes_migration.ps1 -DatabaseUrl 'postgresql://user:pass@host:5432/dbname?sslmode=require'"
    exit 1
}

# Check if psql is available
$psqlPath = Get-Command psql -ErrorAction SilentlyContinue
if (-not $psqlPath) {
    Write-Host "❌ Error: psql command not found. Please install PostgreSQL client tools." -ForegroundColor Red
    exit 1
}

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

Write-Host "🚀 Starting recipes migration..." -ForegroundColor Cyan
Write-Host ""

# Step 1: Create recipes table
Write-Host "📋 Step 1: Creating recipes table..." -ForegroundColor Yellow
$createTableScript = Join-Path $ScriptDir "create_recipes_table.sql"
$createResult = & psql $DatabaseUrl -f $createTableScript 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Recipes table created successfully" -ForegroundColor Green
} else {
    Write-Host "❌ Failed to create recipes table" -ForegroundColor Red
    Write-Host $createResult
    exit 1
}

Write-Host ""

# Step 2: Seed initial recipes
Write-Host "🌱 Step 2: Seeding initial recipes..." -ForegroundColor Yellow
$seedScript = Join-Path $ScriptDir "seed_recipes.sql"
$seedResult = & psql $DatabaseUrl -f $seedScript 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Recipes seeded successfully" -ForegroundColor Green
} else {
    Write-Host "❌ Failed to seed recipes" -ForegroundColor Red
    Write-Host $seedResult
    exit 1
}

Write-Host ""
Write-Host "✅ Migration completed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "📊 Summary:" -ForegroundColor Cyan
& psql $DatabaseUrl -c "SELECT COUNT(*) as total_recipes FROM recipes;"
Write-Host ""
Write-Host "🎉 You can now test the recipes API endpoints!" -ForegroundColor Green

