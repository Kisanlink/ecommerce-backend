# Payment reconciliation & refund runbook

**Audience:** on-call engineer / payments admin.
**Purpose:** handle the cases where Razorpay has **captured money** but the system has
**no confirmed order** (or an order that must not ship). These are rare, deliberate
*fail-closed* outcomes — the money is safe, but a human must decide **refund** or **fulfil**.

> Refunds are **never** issued from application code. They are performed by an authorised
> admin in the **Razorpay Dashboard**. This runbook tells you how to detect each case,
> decide, and act.

---

## 1. The states that land here

Each of these logs loudly (ERROR/WARN) with the `razorpay_order_id` and `payment_id`.
Search logs for the quoted strings.

| # | Trigger | Client sees | Server log (grep) | Money captured? | Order created? |
|---|---------|-------------|-------------------|-----------------|----------------|
| A | **Cart swap** — cart total exceeds amount paid at verify | `409 CART_AMOUNT_MISMATCH` | `Cart-swap guard: cart total exceeds amount paid` / `Rejecting verify: cart total exceeds amount paid (possible cart swap)` | Maybe (if the widget captured) | **No** |
| B | **Gateway lookup failed** at verify, no existing order | `503 GATEWAY_LOOKUP_FAILED` | `Rejecting verify: Razorpay order lookup failed and no existing order` | Maybe | **No** |
| C | **Paid, verify never ran** (closed tab) and the webhook can't rebuild the order | — (no client) | `Webhook: paid order has no matching DB order - needs manual reconciliation` | **Yes** | **No** |
| D | **User-binding mismatch** at verify | `403` | `Rejecting verify: Razorpay order user binding missing or mismatched` | Maybe | **No** |
| E | **Placeholder address** — paid order can't ship | — | `refusing shipment to placeholder/invalid address` / `shipping address is placeholder/invalid - needs manual fulfillment` | **Yes** | **Yes** (unshipped) |

A, B and D are *pre-order* refusals: an order is **not** created and the cart is **not**
cleared, so the customer can simply retry checkout (which mints a fresh Razorpay order for
the current cart). The only open question is whether their **first** attempt actually
captured money that now has no order. C is the classic "paid but no order". E is a paid
order that is stuck unshipped (see the separate order-recovery runbook).

---

## 2. Triage — did money actually move?

1. Take the `razorpay_order_id` (`order_…`) and/or `payment_id` (`pay_…`) from the log line.
2. In the **Razorpay Dashboard → Transactions → Payments**, search the payment id / order id.
3. Read the payment **state**:
   - `authorized` only, or no payment → **no settled money**. Nothing to refund. Tell the
     customer to retry; close the alert.
   - `captured` → **money is held**. Continue to §3.
   - already `refunded` → already handled; close the alert.

---

## 3. Decide: refund or fulfil

Confirm there is **no** confirmed order for this payment before refunding:

- Admin API: look up by Razorpay order id / payment id (unshipped & order lists under
  `/api/v1/admin/...`), or query `orders` for `razorpay_order_id = '<order_…>'`.
- If an order **does** exist and is correct, do **not** refund — drive it through the
  order-recovery runbook (fulfil / ship) instead.

Then:

| Situation | Action |
|-----------|--------|
| Captured, **no** order, customer did not re-checkout | **Refund** (§4) |
| Captured, **no** order, customer already retried and has a *separate* successful order | **Refund** the orphan capture (§4); keep the good order |
| Captured, order exists but **unshipped** (case E / stuck) | **Do not refund** — fix the address and ship (order-recovery runbook) |
| Only `authorized`, never captured | No action; it auto-voids at Razorpay |

When in doubt, **refund** — a captured payment with no order is the customer's money with
nothing delivered.

---

## 4. Issue the refund (Razorpay Dashboard)

1. Razorpay Dashboard → **Transactions → Payments** → open the `captured` payment.
2. **Refund** → **full** amount (unless a partial refund has been explicitly agreed).
3. Add a note referencing the `razorpay_order_id`, the trigger (A–E) and this runbook.
4. Capture the `rfnd_…` id. Record it in the ops log / ticket next to the alert so the
   alert is not actioned twice.
5. Notify the customer (template: *"We spotted a checkout issue, your payment of ₹X has been
   fully refunded, please re-order."*).

> **Idempotency:** before refunding, re-check the payment is still `captured` (not already
> `refunded`). Never refund the same `pay_…` twice.

---

## 5. Prevent recurrence / reduce noise

- **Case A (cart swap):** expected to be rare and usually benign (customer edited the cart
  mid-checkout). A cluster of these from one account is a probable abuse signal — review that
  account. The guard tolerance is ₹1 (`ErrCartAmountMismatch`, `order_service.go`); widen only
  with care.
- **Case B (gateway lookup failed):** if frequent, check Razorpay status / our outbound
  connectivity. These usually self-heal on retry or via the webhook; refund only the ones that
  stayed captured with no order.
- **Case C (paid, no order):** the durable fix (persist a cart snapshot at create-order so the
  webhook can rebuild the order) is a **known open item** — until it ships, case C is handled
  manually here. Make sure the webhook is registered in the Razorpay dashboard so these at
  least alert.
- **Case E (placeholder address):** follow the order-recovery runbook; fix the address and ship,
  do not refund.

---

## 6. Pre-go-live checklist

- [ ] Razorpay **webhook** registered (so case C alerts at all).
- [ ] `RAZORPAY_MAGIC_CHECKOUT_SECRET` set in production (turns on 1CC callback signature
      enforcement; dormant until then).
- [ ] Log-based alerts wired for the grep strings in §1 (at minimum C, A, B).
- [ ] At least one admin has Razorpay Dashboard **refund** permission and has read this runbook.
- [ ] Ops log / ticket queue exists to record `rfnd_…` ids against alerts (dedupe).
