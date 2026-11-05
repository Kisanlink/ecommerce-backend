#!/usr/bin/env node
/*
 * LOCAL-ONLY Razorpay webhook simulator.
 *
 * Posts a signature-valid `order.paid` (or `payment.captured`) webhook to the local
 * backend so you can exercise the server-side fulfilment path WITHOUT a browser, a real
 * payment, or a Razorpay dashboard webhook. This is how you test the paid-but-no-order
 * recovery (ProcessWebhook -> fulfillFromWebhook -> RebuildOrderFromSnapshot): create a
 * Razorpay order, DO NOT call /verify, then run this against that order id.
 *
 * The signature is HMAC-SHA256(body, RAZORPAY_WEBHOOK_SECRET) in hex, in the
 * `X-Razorpay-Signature` header — exactly what PaymentService.ParseWebhook checks. The JSON
 * shape matches the backend's RazorpayWebhookPayload parser (payload.order.order /
 * payload.payment.payment), which is what matters locally.
 *
 * NEVER point this at a production URL. It is a test tool only.
 *
 * Usage:
 *   RAZORPAY_WEBHOOK_SECRET=<your local webhook secret> \
 *     node scripts/local/sim-razorpay-webhook.mjs <razorpay_order_id> [payment_id] [event]
 *
 *   WEBHOOK_URL   default http://localhost:8080/api/v1/webhooks/razorpay
 *   event         default order.paid  (or payment.captured)
 */
import crypto from 'node:crypto';

const orderId = process.argv[2];
const paymentId = process.argv[3] || 'pay_LOCAL' + Date.now();
const event = process.argv[4] || 'order.paid';

const url = process.env.WEBHOOK_URL || 'http://localhost:8080/api/v1/webhooks/razorpay';
const secret = process.env.RAZORPAY_WEBHOOK_SECRET;

if (!orderId) {
  console.error('Usage: RAZORPAY_WEBHOOK_SECRET=... node sim-razorpay-webhook.mjs <order_id> [payment_id] [event]');
  process.exit(1);
}
if (!secret) {
  console.error('RAZORPAY_WEBHOOK_SECRET must be set to the SAME value the backend runs with.');
  process.exit(1);
}

const payload = {
  entity: 'event',
  event, // "order.paid" | "payment.captured"
  contains: ['order', 'payment'],
  payload: {
    order: { order: { id: orderId, status: 'paid', amount: 0 } },
    payment: { payment: { id: paymentId, order_id: orderId, status: 'captured', amount: 0 } },
  },
  created_at: Math.floor(Date.now() / 1000),
};

const body = JSON.stringify(payload);
const signature = crypto.createHmac('sha256', secret).update(body).digest('hex');

const res = await fetch(url, {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'X-Razorpay-Signature': signature,
    'X-Razorpay-Event-Id': 'evt_LOCAL' + Date.now(),
  },
  body,
});

console.log(`POST ${url}`);
console.log(`${event}  order=${orderId}  payment=${paymentId}  ->  ${res.status} ${res.statusText}`);
const text = await res.text();
if (text) console.log(text);
console.log('\nNow check: the order exists (GET /api/v1/orders with the buyer token or the admin orders list),');
console.log('and the payment_cart_snapshots row for this order id has status = "consumed".');
