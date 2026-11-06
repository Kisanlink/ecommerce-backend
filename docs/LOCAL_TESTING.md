# Local testing — step by step

Run and test the whole backend on your own machine. **You will not change anything in the
Razorpay dashboard, the WhatsApp number, or any live setting. No real money moves and no real
OTP is sent.**

This guide is written to be followed top to bottom the first time. Later you only repeat
**Part C (Run it)**.

---

## Why this is safe (read once, 60 seconds)

- **Razorpay:** every account has **two sets of keys at the same time** — test (`rzp_test_…`)
  and live (`rzp_live_…`). Both always work. The "Test / Live" switch in the dashboard only
  changes what *you look at*; it does **not** turn keys on or off. Locally we use the **test
  keys**, so you never touch the dashboard mode and real customers are unaffected.
- **WhatsApp:** the OTP code is made by *our own backend*, not by WhatsApp. The only thing
  WhatsApp does is *deliver* it. We skip delivery entirely by logging in through a local
  Firebase emulator instead, so no WhatsApp message is ever sent.
- **Firebase:** we run a local "Auth emulator" (a fake Firebase on your machine). Your real
  Firebase key is loaded only so the app starts; with the emulator turned on it is never used
  to contact Google.

---

## What you'll have running

Four things, in four terminals, all on your machine:

```
Terminal 1:  Docker          → Postgres (DB) + Redis + DynamoDB
Terminal 2:  Firebase          → the Auth emulator (fake login)
Terminal 3:  Delhivery mock    → a fake courier API (so checkout can confirm delivery)
Terminal 4:  the backend       → http://localhost:8080
```

> **About the Delhivery mock:** checkout refuses to take payment unless it can confirm the
> pincode is deliverable, and fulfilment calls Delhivery to create the shipment. Locally you
> usually don't have Delhivery access, so we run a tiny fake courier
> (`scripts/local/mock-delhivery.mjs`) that always says "serviceable" and hands back a fake
> tracking number. **The backend code is not changed at all** — you just point it at the mock
> instead of the real Delhivery. Nothing about this can leak into production.

---

## Part 0 — Install these once

| Tool | Install | Check it works |
|------|---------|----------------|
| Docker Desktop | https://www.docker.com/products/docker-desktop/ | `docker --version` |
| Go | https://go.dev/dl/ (match `go.mod`) | `go version` |
| Node.js 18+ | https://nodejs.org | `node --version` |
| Firebase CLI | `npm install -g firebase-tools` | `firebase --version` |

> **Which terminal?** On Windows, use **Git Bash** for the `curl`/`node` commands in this
> guide (it comes with Git for Windows) and **PowerShell** for the run script in Part C. If a
> block is marked PowerShell or bash, use that one.

---

## Part A — Collect your keys

You need four things. Here is exactly where each one comes from.

### A1. Razorpay TEST key id + secret

1. Go to **https://dashboard.razorpay.com** and log in.
2. Top-left, flip the mode switch to **Test Mode**. *(This only changes your view — it does
   not affect live payments.)*
3. Left menu → **Account & Settings** → under "Website and app settings" → **API Keys**.
4. Click **Generate Test Key** (if you already made one, it's shown here — regenerate if you
   don't have the secret saved).
5. Copy two values:
   - **Key Id** — looks like `rzp_test_AbCd1234EfGh56`
   - **Key Secret** — shown **once**; copy it now.

### A2. A webhook secret (you invent this one)

For local testing you do **not** need a Razorpay-dashboard webhook. The webhook *simulator*
in this guide signs its fake webhook with a secret that you choose, and the backend checks
against the same value. So just pick any string and reuse it in both places. This guide uses:

```
local_webhook_secret
```

### A3. Firebase project id + service account file

1. Go to **https://console.firebase.google.com** and open your project.
2. Click the ⚙ gear (top-left) → **Project settings**.
3. On the **General** tab, copy the **Project ID** (e.g. `amrti-app`).
4. Open the **Service accounts** tab → click **Generate new private key** → **Generate key**.
   A `.json` file downloads.
5. Save that file into the backend folder as:
   ```
   ecommerce-backend/firebase_service_account.json
   ```
   *(This file is gitignored. With the emulator on, it's only used so the app boots — it never
   contacts Google.)*

### A4. Delhivery — nothing to get, use the local mock

**Why this exists.** `POST /payments/create-order` checks that the pincode is deliverable and
**refuses if it can't check** (so a broken courier can't produce an undeliverable paid order),
and fulfilment calls Delhivery to create the shipment. Locally you don't have Delhivery access,
so instead of a real account you run a tiny fake courier that always says "serviceable" and
hands back a fake tracking number. **The backend is not modified** — you only point it at the
mock with an env var, so this can never affect production.

**What it is.** `scripts/local/mock-delhivery.mjs` — a small Node script (no `npm install`,
uses only built-in modules). It answers the four Delhivery endpoints the backend calls:
serviceability (every pincode → serviceable), charges, TAT, and shipment creation (returns a
`MOCK…` waybill).

**How to run it.** Open a terminal in the `ecommerce-backend` folder and run:

```bash
node scripts/local/mock-delhivery.mjs
```

You should see:

```
[mock-delhivery] listening on http://localhost:4100
[mock-delhivery] set DELHIVERY_BASE_URL=http://localhost:4100 in .env.local
```

Leave this terminal running while you test — it logs every call it answers (e.g.
`serviceability 500081 -> serviceable`). To stop it, press `Ctrl-C`. Want a different port?
`PORT=4200 node scripts/local/mock-delhivery.mjs` (then use that port in `DELHIVERY_BASE_URL`).

> This is **Terminal 3** in the run order (Part C). It is listed here too so you have the whole
> Delhivery story in one place; you don't run it twice.

**How to point the backend at it.** In your `.env.local` (set up in Part B1):

```dotenv
DELHIVERY_BASE_URL=http://localhost:4100
DELHIVERY_API_TOKEN=dummy-local-token   # any value — the mock ignores it
DELHIVERY_PRODUCTION=
```

**Check it works** (optional, in another terminal):

```bash
curl "http://localhost:4100/c/api/pin-codes/json/?filter_codes=500081"
```

A healthy mock returns JSON containing `"pre_paid":"Y"` and `"remarks":""` — that's what makes
the backend treat the pincode as serviceable.

> **Prefer the real Delhivery staging API instead?** If you do have staging credentials, skip
> the mock entirely: set `DELHIVERY_BASE_URL` to the staging base URL and `DELHIVERY_API_TOKEN`
> to your staging token, leave `DELHIVERY_PRODUCTION` empty, and don't start Terminal 3. Use a
> known serviceable pincode (e.g. `500081`) in your test address.

---

## Part B — One-time setup

### B1. Create `ecommerce-backend/.env.local`

Make a new file called `.env.local` in the `ecommerce-backend` folder and paste this,
replacing every `[...]` with the value you collected in Part A:

```dotenv
# ---------- Local database (Docker) — NOT production ----------
DB_POSTGRES_HOST=localhost
DB_POSTGRES_PORT=5555
DB_POSTGRES_USER=postgres
DB_POSTGRES_PASSWORD=12345678
DB_POSTGRES_DBNAME=DemoDB
DB_POSTGRES_SSLMODE=disable

# ---------- Local Redis (Docker) ----------
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# ---------- Razorpay TEST keys (from A1) ----------
RAZORPAY_KEY_ID=[your rzp_test_... key id]
RAZORPAY_KEY_SECRET=[your test key secret]
RAZORPAY_LIVE=
RAZORPAY_WEBHOOK_SECRET=local_webhook_secret
# Leave the 1CC secret unset so Magic Checkout signatures stay off locally:
# RAZORPAY_MAGIC_CHECKOUT_SECRET=

# ---------- Firebase (from A3) ----------
FIREBASE_PROJECT_ID=[your firebase project id]
FIREBASE_SERVICE_ACCOUNT_PATH=firebase_service_account.json
FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:9099   # use 127.0.0.1, NOT localhost (IPv6 ::1 breaks it)

# ---------- WhatsApp: leave everything unset → no real messages ----------

# ---------- Delhivery: point at the local mock (from A4) ----------
DELHIVERY_PRODUCTION=
DELHIVERY_BASE_URL=http://localhost:4100
DELHIVERY_API_TOKEN=dummy-local-token
# (Using real Delhivery staging instead? Put its base URL + token here and skip Terminal 3.)

PORT=8080
GIN_MODE=debug
```

> **Important safety point:** this file sets the database to the **local** Docker Postgres.
> The run script in Part C loads it in a way that **overrides** your real `.env`, so the local
> server can never accidentally connect to the production database. Keep `.env.local` out of
> git (it already is).

### B2. Start the database + cache (Terminal 1)

Start **only the infrastructure** — Postgres, Redis, DynamoDB. Do **not** start the whole
stack: `docker-compose up -d` on its own also launches a containerised `ecommerce-backend` and
`nginx` that grab port **8080**, and then the backend you run with `go run` (Part C) can't bind
it. Name the three services explicitly:

```bash
cd ecommerce-backend
docker-compose up -d postgres redis dynamodb
docker-compose ps        # postgres and redis should show "healthy"
```

> Already ran a bare `docker-compose up -d` and have the backend/nginx containers running? Free
> the port with `docker-compose stop ecommerce-backend nginx` (your DB/Redis/Dynamo keep
> running). `dynamodb` showing "unhealthy" is normal — it has no health check and isn't used by
> this flow.

### B3. Put some products in the local database

A fresh local database is empty, so your cart would have nothing to add. Seed it.

> **The seed does not read `.env.local` by itself.** If you just run `go run ./cmd/seed`, it
> falls back to its defaults — port `5432` with SSL **required** — and fails with
> `pq: SSL is not enabled on the server`. Your Docker Postgres is on **5555** with SSL **off**,
> so pass those two settings on the command line:

Git Bash:
```bash
DB_POSTGRES_PORT=5555 DB_POSTGRES_SSLMODE=disable go run ./cmd/seed
```

PowerShell:
```powershell
$env:DB_POSTGRES_PORT="5555"; $env:DB_POSTGRES_SSLMODE="disable"; go run ./cmd/seed
```

The seed's other DB defaults (host `localhost`, user `postgres`, password `12345678`, db
`DemoDB`) already match the container, so only the port and SSL mode need overriding. It loads
products, inventory, coupons and an admin user into the **local** database — you should see it
finish with a success summary instead of the SSL error.

---

## Part C — Run it (do this every time)

Open **four** terminals.

**Terminal 1 — infrastructure** (if not already up from B2):
```bash
cd ecommerce-backend
docker-compose up -d postgres redis dynamodb
```
*(Only these three — not a bare `docker-compose up -d`, which would also start a backend
container on port 8080 and block Terminal 4.)*

**Terminal 2 — Firebase Auth emulator** (use your real project id from A3):
```bash
cd ecommerce-backend
firebase emulators:start --only auth --project YOUR_FIREBASE_PROJECT_ID
```
Leave it running. (Emulator UI: http://localhost:4000)

**Terminal 3 — Delhivery mock** (the fake courier, so checkout can confirm delivery):
```bash
cd ecommerce-backend
node scripts/local/mock-delhivery.mjs
```
Leave it running. It prints each call it answers. *(Skip this terminal only if you set
`DELHIVERY_BASE_URL` to a real Delhivery staging URL in A4.)*

**Terminal 4 — the backend.** This loads `.env.local` and starts the server.

PowerShell (Windows) — save as `scripts/local/run-local.ps1` and run `./scripts/local/run-local.ps1`:
```powershell
Get-Content .env.local | Where-Object { $_ -match '^\s*[^#].*=' } | ForEach-Object {
  $kv = $_ -split '=', 2
  [Environment]::SetEnvironmentVariable($kv[0].Trim(), $kv[1].Trim(), 'Process')
}
Write-Host "DB -> $($env:DB_POSTGRES_HOST):$($env:DB_POSTGRES_PORT)/$($env:DB_POSTGRES_DBNAME)  (must say localhost:5555)"
go run ./cmd/server/main.go
```

Git Bash:
```bash
set -a; . ./.env.local; set +a
echo "DB -> $DB_POSTGRES_HOST:$DB_POSTGRES_PORT/$DB_POSTGRES_DBNAME   (must say localhost:5555)"
go run ./cmd/server/main.go
```

**Check it started correctly:** in the startup logs find the line `NeonDB Configuration` — it
must show **host=localhost, port=5555** and **is_neondb=false**. If it shows a `neon.tech`
host, stop immediately — your production DB leaked in; fix the env and relaunch.

Then open **http://localhost:8080/health** — you should get an OK response, and
**http://localhost:8080/docs** opens the API explorer (Swagger).

---

## Part D — Log in and place a test order

### D1. Get a login token (no WhatsApp, no Google)

In Git Bash (replace the project id with yours from A3):
```bash
FIREBASE_PROJECT_ID=YOUR_FIREBASE_PROJECT_ID \
  node scripts/local/mint-emulator-token.mjs tester@local.test Test123!
```
It prints a **UID** and an **ID TOKEN**. Copy the ID TOKEN — that's your login.

> The first time you use this token on a payment call, the backend automatically creates the
> matching local user for you. You don't need to insert anything by hand.

### D2. Use the token in Swagger (easiest, no curl)

1. Open **http://localhost:8080/docs**.
2. Click the green **Authorize** button (top right).
3. In the value box, type the word **`Bearer`**, then a **space**, then paste the token — the
   whole thing on one line:
   ```
   Bearer eyJhbGciOi...<the rest of your token>...fQ.
   ```
   The scheme is a header API key, so Swagger sends exactly what you type — you **must** include
   the `Bearer ` prefix yourself. Pasting the bare token (no `Bearer `) is the #1 cause of a
   `401` here. If the dialog shows two boxes (`BearerAuth` and `FirebaseAuth`), fill either one
   the same way. Click **Authorize** → **Close**.
4. Now every request you send from this page is logged in as your test user.

> **Copy the token as one line.** The minter prints it on a single line, but your terminal may
> *wrap* it across several rows — if a real newline gets copied in, the header breaks and you
> get a 401. Paste into a plain text editor first and confirm it's one unbroken line if unsure.

Do these in order, using the "Try it out" button on each:

1. **GET `/api/v1/users/profile`** → in the response copy your internal **`id`** (looks like
   `USER...`). You need it for the next step.
2. **Create a shipping address** (one-time per user). There is **no address API** — the
   address CRUD routes are not implemented (they're commented out in `user_routes.go`), so you
   won't find them in Swagger. In the live app, Magic Checkout collects the address through the
   Razorpay widget; the standard `create-order` instead needs a `shipping_address_id` that
   already exists. For local testing, insert one directly, putting **your** internal `id` from
   step 1 in `user_id`:
   ```bash
   docker exec ecommerce-postgres psql -U postgres -d DemoDB -c \
   "INSERT INTO addresses (id, created_at, updated_at, user_id, type, first_name, last_name, address_line1, city, state, postal_code, country, phone, is_default) \
    VALUES ('ADDRLOCALTEST01', now(), now(), 'PASTE_YOUR_USER_ID', 'home', 'Test', 'User', '123 Test Street', 'Hyderabad', 'Telangana', '500081', 'India', '9999999999', true);"
   ```
   Your `shipping_address_id` is then `ADDRLOCALTEST01`. The pincode `500081` is serviceable via
   the Delhivery mock. *(The address must belong to your user — `create-order` checks
   `user_id` ownership and returns `Invalid shipping_address_id` otherwise.)*
3. **POST `/api/v1/cart/items`** → body `{ "product_id": "PROD00000005", "quantity": 1 }`.
   Get a product id from **GET `/api/v1/products`**. (If this returns *insufficient stock*,
   re-seed — see the note in B3 / troubleshooting.)
4. **POST `/api/v1/payments/create-order`** → body:
   ```json
   { "first_name": "Test", "mobile": "9999999999", "shipping_address_id": "ADDRLOCALTEST01" }
   ```
   The response has an **`order_id`** like `order_...`. A payment is now ready — and a recovery
   **snapshot row has been saved** for it.

### D3. Pay with a Razorpay TEST card

Open the hosted checkout in your browser:
```
http://localhost:8080/api/v1/payments/checkout?order_id=<order_id>
```
Pay with a **test** instrument — card `4111 1111 1111 1111`, any future expiry, any CVV; or
test UPI `success@razorpay`. No real money moves. The page finishes by calling `/verify`,
which creates the confirmed order.

---

## Part E — Test the "paid but tab closed" recovery (the important one)

This proves the new safety net: a buyer pays, then closes the tab **before** the confirm step
runs. The webhook must rebuild the order on its own from the saved snapshot.

1. Do **D1** and **D2** to get a fresh `order_id`. **Do not** open the checkout / do not pay
   through the browser this time — we'll simulate it.
2. Fire the webhook exactly as Razorpay would (Git Bash), using the same secret from your env:
   ```bash
   RAZORPAY_WEBHOOK_SECRET=local_webhook_secret \
     node scripts/local/sim-razorpay-webhook.mjs <order_id>
   ```
   You should get `200`, and the backend log shows
   **`Webhook: rebuilt paid order from cart snapshot`**.
3. Confirm the order now exists: in Swagger, **GET `/api/v1/orders`** (still authorized as your
   user) shows the new order.
4. Confirm the snapshot was marked done (in a DB tool / `psql`):
   ```sql
   SELECT status, consumed_at FROM payment_cart_snapshots WHERE razorpay_order_id = '<order_id>';
   -- status should be 'consumed'
   ```
5. **Run it twice** (run the simulator again): you must still have **exactly one** order:
   ```sql
   SELECT count(*) FROM orders WHERE razorpay_order_id = '<order_id>';   -- expect 1
   ```

Want the real round trip instead of the simulator? Run `ngrok http 8080`, then in the Razorpay
dashboard (**Test Mode**) add a webhook pointing at
`https://<your-ngrok>.ngrok.io/api/v1/webhooks/razorpay` with secret `local_webhook_secret`,
pay with a test card, and close the tab before it finishes. (A test-mode webhook is separate
from your live webhooks and changes nothing in production.)

---

## Part F — If something goes wrong

| Symptom | Cause / fix |
|---------|-------------|
| Backend won't start — **port 8080 already in use** / bind error | A containerised backend is holding it. Run `docker-compose stop ecommerce-backend nginx`, then start the backend again. Start infra with `docker-compose up -d postgres redis dynamodb`, never a bare `docker-compose up -d`. |
| Seed fails with `SSL is not enabled on the server` | You ran `go run ./cmd/seed` without the DB overrides. Use `DB_POSTGRES_PORT=5555 DB_POSTGRES_SSLMODE=disable go run ./cmd/seed` (B3). |
| Log shows a `neon.tech` DB host | `.env.local` didn't override. Launch via the Part C script (don't just `go run`), and confirm the printed `DB -> localhost:5555` line. |
| `create-order` returns **503** "could not confirm delivery" | The Delhivery mock (Terminal 3) isn't running, or `DELHIVERY_BASE_URL` in `.env.local` doesn't point at it (`http://localhost:4100`). Start the mock and confirm it logs a `serviceability … -> serviceable` line when you retry. (On real staging: check the token/URL and use a serviceable pincode.) |
| `create-order` returns **503** "could not start checkout" | The snapshot couldn't be saved (DB issue). Check Postgres is up and the `payment_cart_snapshots` table exists (it auto-creates on boot). |
| Any authed call returns **401** | Most often the Authorize value is missing the `Bearer ` prefix, or the token got a line-break when copied (D2) — it must be `Bearer <token>` on one line. Otherwise: token expired (re-mint, D1), the emulator isn't running, or its `--project` id doesn't match `FIREBASE_PROJECT_ID`. |
| Token mint script: `ECONNREFUSED ::1:9099` | Node resolved `localhost` to IPv6 but the emulator is IPv4-only. The script now uses `127.0.0.1`; if you set `FIREBASE_AUTH_EMULATOR_HOST` yourself, use `127.0.0.1:9099`, not `localhost:9099`. Also confirm the Auth emulator (Terminal 2) is actually running. |
| Webhook simulator returns **400** "Invalid webhook" | The `RAZORPAY_WEBHOOK_SECRET` you ran the script with doesn't match the one the backend started with. Make them identical. |
| Cart "product not found" | Run the seed (B3), or copy a real id from `GET /api/v1/products`. |
| No address endpoint in Swagger / `create-order` says `Invalid shipping_address_id` | There is no address API (routes are commented out). Insert an address row for your user and use its id — see D2 step 2. The row's `user_id` must equal your internal `id` from `GET /users/profile`. |

---

## Teardown

```bash
docker-compose down       # stop DB/Redis/Dynamo (keeps data)
docker-compose down -v    # also wipe the local data
# Ctrl-C the emulator (T2), the Delhivery mock (T3), and the backend (T4)
```

---

## Quick reference

| Thing | Value |
|------|-------|
| Backend | http://localhost:8080 (API explorer `/docs`, health `/health`) |
| Postgres | `localhost:5555` db `DemoDB` user `postgres` pass `12345678` (sslmode `disable`) |
| Redis | `localhost:6379` |
| Firebase Auth emulator | `localhost:9099` (UI `localhost:4000`) |
| Delhivery mock | `node scripts/local/mock-delhivery.mjs` → `http://localhost:4100` |
| Mint a login token | `node scripts/local/mint-emulator-token.mjs [email] [password]` |
| Simulate a paid webhook | `node scripts/local/sim-razorpay-webhook.mjs <order_id>` |
| Razorpay test card | `4111 1111 1111 1111`, any future expiry + CVV |

**Nothing in this guide changes a provider dashboard mode or sends a real OTP or payment.**
