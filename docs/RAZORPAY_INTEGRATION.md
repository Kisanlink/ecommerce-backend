# Razorpay Payment Integration

Important: Do not commit live credentials to source control or documentation. Use environment variables and secure secret stores.

Configuration
- Base URL: https://api.razorpay.com/v1/
- Key ID: RZP_KEY_ID (env)
- Key Secret: RZP_KEY_SECRET (env)
- Webhook Secret: RZP_WEBHOOK_SECRET (env)

Server endpoints used by frontend
- POST /api/v1/payments/create-order (Protected)
	- Request: { shipping_address? string, shipping_address_id? string, subtotal? number }
	- Response: { order_id, amount (rupees), currency: "INR", key_id, checkout_url, notes }
- GET /api/v1/payments/checkout?order_id=... (Protected)
	- Minimal hosted checkout page; injects amount and currency server-side
- POST /api/v1/payments/verify (Protected)
	- Body: { razorpay_order_id, razorpay_payment_id, razorpay_signature }
	- Verifies HMAC, creates DB order, clears cart, generates invoice
- GET /api/v1/payments/:payment_id/status (Protected)
- POST /api/v1/webhooks/razorpay (Public)

Frontend flow
1) Prepare cart and estimate shipping on server
2) Call create-order; use checkout_url from response
3) Open hosted checkout URL or embed your own Razorpay Checkout with order_id + key_id
4) On success, POST /payments/verify with the IDs/signature from Razorpay

Implementation notes
- Currency is INR everywhere
- Amount rupees → paisa conversion uses math.Round to avoid off-by-one paise
- Notes include user_id, shipping details, cart totals; used after verification
- Built-in checkout requires auth to mitigate abuse

Security
- Keep Key Secret and Webhook Secret only on server
- Verify payment signature on server before creating orders
- Validate order amount and currency before fulfillment

Supported payment methods
- Cards, UPI, Net Banking, Wallets, EMI (as per Razorpay account configuration)
