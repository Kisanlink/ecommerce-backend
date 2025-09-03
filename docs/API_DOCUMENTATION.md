API Reference (v1)

This document lists all public APIs with auth requirements, and the end-to-end checkout flow your frontend should follow.

Base URL
- Local: http://localhost:8080

Auth model
- Firebase ID tokens (Bearer) for authenticated routes
- Currency is INR everywhere

Auth legend
- Public = no auth required
- Protected = Authorization: Bearer <id_token> required

Auth endpoints
- POST /api/v1/auth/signup — Public
    - Body: { name, email, password }
    - Returns Firebase ID token, refresh token, user
- POST /api/v1/auth/login — Public (alias: /signin)
    - Body: { email, password }
    - Returns ID token, refresh token, user
- POST /api/v1/auth/google — Public
    - Body: { id_token } from Google/Firebase SDK on client
    - Returns Firebase ID token, refresh token, user
- POST /api/v1/auth/verify-token — Public
    - Body: { id_token }
    - Verifies token validity
- POST /api/v1/auth/refresh-token — Public
    - Body: { refresh_token }
- GET /api/v1/auth/google — Public
    - Returns guidance for client-side Google Sign-In
- Protected auth utilities:
    - GET /api/v1/auth/profile — Protected
    - PUT /api/v1/auth/profile — Protected
    - POST /api/v1/auth/logout — Protected
    - DELETE /api/v1/auth/account — Protected

Profile endpoints
- POST /api/v1/profiles/auto — Protected
    - Auto-create a profile from the authenticated user
- GET /api/v1/profiles — Protected
- PUT /api/v1/profiles — Protected
- POST /api/v1/profiles/upload-image — Protected (multipart form; field: image)
- GET /api/v1/profiles/image-url — Protected

Product endpoints
- GET /api/v1/products — Public
- GET /api/v1/products/:id — Public
- GET /api/v1/products/category/:category — Public
- GET /api/v1/products/search?q=term — Public

Recipe endpoints
- GET /api/v1/recipes — Public
- GET /api/v1/recipes/:id — Public
- GET /api/v1/recipes/category/:category — Public
- Admin-only create/update/delete routes exist under /api/v1/recipes (Protected + Admin)

Favorites endpoints
- POST /api/v1/favorites — Protected (Body: { product_id })
- GET /api/v1/favorites — Protected
- DELETE /api/v1/favorites — Protected (Body: { product_id })
- GET /api/v1/favorites/check?product_id=PROD... — Protected

Cart endpoints
- All under /api/v1/cart — Protected
- GET /api/v1/cart — Get cart
- GET /api/v1/cart/summary — Summary
- GET /api/v1/cart/count — Count
- GET /api/v1/cart/validate — Validate
- POST /api/v1/cart/items — Add item (Body: { product_id, quantity })
- PUT /api/v1/cart/items/:product_id — Update quantity (Body: { quantity })
- POST /api/v1/cart/items/:product_id/increment — Increment
- POST /api/v1/cart/items/:product_id/decrement — Decrement
- DELETE /api/v1/cart/items/:product_id — Remove
- DELETE /api/v1/cart/clear — Clear cart

Checkout helper endpoints
- GET /api/v1/cart/checkout/prepare — Protected
- POST /api/v1/cart/checkout/estimate-shipping — Protected
    - Body: { address: { street, city, state, postal_code, country } }
    - Returns shipping_options (Delhivery Surface), subtotal, user_id
    - Implementation notes:
        - Serviceability via Delhivery /c/api/pin-codes
        - Charges via /api/kinko/v1/invoice/charges/.json with md=S (Surface), ss from config (default Delivered), cgm=grams
        - Weight = number_of_items * 100g, minimum 100g
        - Currency = INR

Delivery endpoints
- POST /api/v1/delivery/estimate — Protected (same body as estimate-shipping)

Payment endpoints
- All under /api/v1/payments — Protected (auth required)
- POST /api/v1/payments/create-order — Protected
    - Request (typical):
        - Prefer calling estimate-shipping first. Then call create-order with either:
            - shipping_address (string) or shipping_address_id (saved address)
            - subtotal (optional; ignored if last estimate exists)
    - Response:
        - { order_id, amount (rupees), currency: "INR", key_id, description, notes, checkout_url }
        - checkout_url: server-hosted GET /api/v1/payments/checkout?order_id=...
- GET /api/v1/payments/checkout?order_id=... — Protected
    - Minimal hosted checkout page. Server fetches amount/currency and injects into Razorpay options
- POST /api/v1/payments/verify — Protected
    - Body: { razorpay_order_id, razorpay_payment_id, razorpay_signature }
    - Verifies signature; creates DB order; clears cart; generates invoice
- GET /api/v1/payments/:payment_id/status — Protected
- Webhooks
    - POST /api/v1/webhooks/razorpay — Public (Razorpay calls this)

End-to-end checkout flow
1) Login/signup (Public); hold ID token in client
2) Manage cart (Protected) — add/update/remove items
3) Estimate shipping (Protected)
     - POST /api/v1/cart/checkout/estimate-shipping with address
     - Backend checks serviceability, computes grams, calls Delhivery, returns shipping cost and ETA (Surface)
4) Create payment order (Protected)
     - POST /api/v1/payments/create-order
     - Backend prefers the last estimate cached for this user to populate shipping_address and subtotal; currency forced to INR
     - Response contains checkout_url
5) Open hosted checkout (Protected)
     - GET the checkout_url returned above to open Razorpay Checkout
6) On success, verify payment (Protected)
     - POST /api/v1/payments/verify with ids and signature
     - Backend creates the database order, persists shipping metadata, generates invoice, and clears cart
7) Optional: rely on webhook to reconcile statuses (Public webhook endpoint)

Notes and guarantees
- Currency: INR only
- Amount rounding: rupees → paisa uses bankers’ rounding server-side to avoid off-by-one paise
- Shipping mode: Surface (md=S). Service status comes from config (default Delivered)
- Serviceability parser is tolerant to Delhivery’s pin type (string/number)
- Built-in checkout requires auth to mitigate abuse

Minimal examples
- Signup (Public)
    - POST /api/v1/auth/signup
    - Body: { "name": "User", "email": "user@example.com", "password": "••••••••" }
- Estimate shipping (Protected)
    - POST /api/v1/cart/checkout/estimate-shipping
    - Body: { "address": { "street": "...", "city": "...", "state": "...", "postal_code": "500092", "country": "India" } }
- Create order (Protected)
    - POST /api/v1/payments/create-order
    - Body: { "shipping_address": "...", "subtotal": 123.45 }
- Verify payment (Protected)
    - POST /api/v1/payments/verify
    - Body: { "razorpay_order_id": "order_...", "razorpay_payment_id": "pay_...", "razorpay_signature": "..." }

Admin-only endpoints
- Recipes admin CRUD (under /api/v1/recipes) — Protected + Admin
- Other admin areas exist and are guarded by Firebase auth plus admin checks

Troubleshooting
- If login fails with “ID token issued at future timestamp”, ensure the server time is in sync (NTP/time sync) and timezone is correct
- If checkout_url returns 404, confirm route: GET /api/v1/payments/checkout and include order_id; ensure you attach Authorization header

Security
- Do not expose live API keys or secrets in clients or docs
- Keep Authorization headers on all Protected endpoints
https://kisanlink-workspace.slack.com/docs/T08LB13ENMQ/F09D3PM3EBD
