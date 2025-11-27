# Neon Database Migration Guide

## Overview

This document explains how to move the existing PostgreSQL schema and
data to a fully managed [Neon](https://neon.tech/) cluster and point the
`ecommerce-backend` services at the new database.

The migration process has four major steps:

1. Provision a Neon project and gather its connection string.
2. Export the current PostgreSQL database.
3. Import the dump into Neon.
4. Update application environment variables to use the Neon endpoint.

## 1. Prerequisites

- `pg_dump`, `pg_restore`, and `psql` installed (bundled with PostgreSQL
  CLI tools).
- Network access to both the source database and Neon.
- The Neon connection string
  (`postgresql://user:password@host/db?sslmode=require`).
- Optional: Docker or Render credentials if the app runs there.

## 2. Provision Neon

1. Create a Neon project and branch (Neon gives you a default `main`
   branch).
2. Under **Connection Details**, copy:
   - **Host** (e.g. `ep-orange-field-a1tfsvjg-pooler.ap-southeast-1.aws.neon.tech`)
   - **Port** (`5432`)
   - **Database** (`neondb`)
   - **User** (`neondb_owner`)
   - **Password** (`npg_...`)
3. Neon enforces TLS. Keep `sslmode=require` (and optionally
   `channel_binding=require`) in every connection string.

## 3. Run the migration script

Use the helper script to dump the current database and restore it into
Neon in one go:

```bash
chmod +x scripts/neon-migrate.sh
./scripts/neon-migrate.sh \
  "postgresql://postgres:YOUR_LOCAL_PASSWORD@localhost:5432/DemoDB?sslmode=disable" \
  "postgresql://neondb_owner:npg_0MOY2LSjTiky@ep-orange-field-a1tfsvjg-pooler.ap-southeast-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require"
```

What the script does:

1. Runs `pg_dump --format=custom` against the source DSN.
2. Streams the dump into `pg_restore --clean --if-exists` against Neon.
3. Cleans up the temporary dump file.

You can pass `--schema-only` or `--data-only` as a third argument if you
want to migrate just the schema or data.

### 3.1 Picking the right DSNs

- **Source DSN**: Build it from the pgAdmin connection details (user,
  password, host, port, DB). Example:
  `postgresql://postgres:12345678@localhost:5432/DemoDB?sslmode=disable`.
- **Neon DSN**: Copy the `psql` string from the Neon dashboard, e.g.
  `postgresql://neondb_owner:npg_0MOY2LSjTiky@ep-orange-field-a1tfsvjg-pooler.ap-southeast-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require`.
- Windows PowerShell example:

  ```powershell
  cd D:\amrti1\ecommerce-backend
  chmod +x scripts/neon-migrate.sh
  ./scripts/neon-migrate.sh `
    "postgresql://postgres:12345678@localhost:5432/DemoDB?sslmode=disable" `
    "postgresql://neondb_owner:npg_0MOY2LSjTiky@ep-orange-field-a1tfsvjg-pooler.ap-southeast-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require"
  ```

## 4. Verify data in Neon

```bash
psql "postgresql://neondb_owner:***@ep-orange-field-a1tfsvjg-pooler.ap-southeast-1.aws.neon.tech/neondb?sslmode=require" \
  -c "\dt"
```

Confirm that all tables exist and spot-check critical rows, e.g.

```sql
SELECT COUNT(*) FROM users;
SELECT COUNT(*) FROM orders;
```

## 5. Update application configuration

Set the following environment variables in every runtime environment
(Docker, Render, GitHub Actions, etc.):

```bash
POSTGRES_DSN="postgresql://neondb_owner:npg_0MOY2LSjTiky@ep-orange-field-a1tfsvjg-pooler.ap-southeast-1.aws.neon.tech/neondb?sslmode=require"

DB_POSTGRES_HOST="ep-orange-field-a1tfsvjg-pooler.ap-southeast-1.aws.neon.tech"
DB_POSTGRES_PORT="5432"
DB_POSTGRES_USER="neondb_owner"
DB_POSTGRES_PASSWORD="npg_0MOY2LSjTiky"
DB_POSTGRES_DBNAME="neondb"
DB_POSTGRES_SSLMODE="require"
DB_POSTGRES_MAX_CONNS="20"   # Neon starter tiers cap concurrent conns
DB_POSTGRES_IDLE_CONNS="5"
DB_POSTGRES_CHANNEL_BINDING="require"  # optional but recommended
```

Tips:

- Render: add the variables in the **Environment** tab; remove any
  `localhost` DSNs.
- Docker Compose: either remove the local `postgres` service or leave it
  for development but override the env vars when deploying.
- CLI testing: run
  `psql "$POSTGRES_DSN"` to verify the container can reach Neon.

## 6. Cutover checklist

- [ ] Application pods/containers log `Connected to Postgres` without
  retry loops.
- [ ] `SELECT current_setting('server_version');` in Neon returns the
  expected version (≥15).
- [ ] Automated migrations (GORM auto-migrate + SQL files) succeed.
- [ ] Background jobs or scripts (`cmd/seed`, `cmd/migrate-users-profiles`,
  etc.) use the same Neon DSN.
- [ ] Monitoring dashboards / alerting are updated to point at Neon’s
  metrics if applicable.

## 7. Fallback & rollback

- Keep the original database snapshot (`.dump` file) until Neon runs in
  production for a few days.
- If you need to roll back, point the env vars back to the old DSN and
  re-import the dump.

## 8. Troubleshooting

| Symptom | Fix |
| ------- | --- |
| `no pg_hba.conf entry` | Ensure you are hitting the `*-pooler` host and `sslmode=require`. |
| `certificate verify failed` | Install CA certificates inside your container or set `PGSSLROOTCERT` (Alpine images need `ca-certificates`). |
| `password authentication failed` | Reset the Neon password and update every environment variable. |
| `channel binding required but not supported` | Upgrade `libpq` to 14+ or drop `channel_binding=require` from the DSN (Neon allows it to be optional). |

## 9. Field notes from the first migration

This captures the live journey so future deploys can avoid the same hits.

1. **Render deploy kept failing**  
   - Logs: `failed to connect to user=postgres database=DemoDB ... 127.0.0.1:5432 ... connection refused`.  
   - Cause: Render container still pointed at a local Postgres instance.
2. **Neon DSN confirmed**  
   - Copied the exact URI:
     `postgresql://neondb_owner:npg_0MOY2LSjTiky@ep-orange-field-a1tfsvjg-pooler.ap-southeast-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require`.
   - Noted that `sslmode=require` and channel binding must stay in place.
3. **Source DSN assembled from pgAdmin**  
   - Local DB details → `postgresql://postgres:12345678@localhost:5432/DemoDB?sslmode=disable`.
4. **PowerShell hurdles**  
   - `&&` not a valid separator → switched to `;`.  
   - Script still produced no output because `.sh` needs Bash.  
   - Resolution: run it through Git Bash/WSL:
     ```bash
     bash ./scripts/neon-migrate.sh \
       "postgresql://postgres:12345678@localhost:5432/DemoDB?sslmode=disable" \
       "postgresql://neondb_owner:npg_0MOY2LSjTiky@ep-orange-field-a1tfsvjg-pooler.ap-southeast-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require"
     ```
5. **Successful run output**  
   ```
   🔄 Exporting source database...
   🚀 Importing dump into Neon...
   🧹 Cleaning up temporary dump...
   ✅ Migration finished successfully.
   ```
6. **Pending actions**  
   - Update Render env vars to the Neon DSN.  
   - Redeploy and verify logs show a healthy DB connection.  
   - Spot-check Neon with `psql "\dt"` and `SELECT COUNT(*) FROM users;`.

Keep adding to this section after each migration/cutover so the tribal
knowledge stays written down.

Following this guide migrates all existing tables to Neon and ensures the
Go services connect exclusively to the managed database.



1st psql path setup c->programfiles->PostgreSQ->bin copy path go to the edit env variables and click o path ad them click on edit adn click o new and add path and click on ok on all , then get the dsn connection strings of source adn tagret db for migration adn runthe script with surce and destinamtion and 