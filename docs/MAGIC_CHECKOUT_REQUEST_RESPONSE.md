# Razorpay Magic Checkout - Migration Guide
## For Existing Razorpay Integration

**Version:** 1.0  
**Date:** November 2025  
**Audience:** Developers with existing Razorpay Standard Checkout

---

## Quick Overview

You already have Razorpay integrated. To enable Magic Checkout, you need:

### 3 Changes:
1. **Update frontend**: Change script + add `one_click_checkout: true`
2. **Modify order creation**: Add `line_items_total` and `line_items`
3. **Create 3 new API endpoints** for Razorpay to call

### Time Estimate: 2-4 hours

---

## Step 1: Enable Magic Checkout (1-2 days wait time)

### Contact Razorpay
- Email your Razorpay SPOC (Sales Point of Contact)
- Request: "Please enable Magic Checkout feature on my account"
- Wait for confirmation email

### Configure Dashboard
Once enabled:

1. **Login to Dashboard** → Navigate to **Magic Checkout**
2. Select **Custom E-Commerce Platform**
3. Configure **3 API endpoints**:
   - **Get Promotions URL:** `https://yourwebsite.com/api/get-promotions`
   - **Apply Promotion URL:** `https://yourwebsite.com/api/apply-promotion`
   - **Shipping Info URL:** `https://yourwebsite.com/api/shipping-info`
4. Click **Save**

**Important:** These URLs must be publicly accessible (no authentication).

---

## Step 2: Frontend Changes (15 minutes)

### Change 1: Update Script URL

**OLD:**
```html
<script src="https://checkout.razorpay.com/v1/checkout.js"></script>
```

**NEW:**
```html
<script src="https://checkout.razorpay.com/v1/magic-checkout.js"></script>
```

### Change 2: Add `one_click_checkout: true`

**OLD Code:**
```javascript
const options = {
  key: 'YOUR_KEY_ID',
  order_id: orderId,
  name: 'Your Store',
  handler: function(response) {
    // your handler
  }
};
```

**NEW Code:**
```javascript
const options = {
  key: 'YOUR_KEY_ID',
  one_click_checkout: true, // ← ADD THIS LINE
  show_coupons: true,       // ← ADD THIS (optional)
  order_id: orderId,
  name: 'Your Store',
  handler: function(response) {
    // your handler - NO CHANGES
  }
};
```

**That's it for frontend!** Your existing payment verification and webhook handlers remain unchanged.

---

## Step 3: Modify Order Creation API (30 minutes)

You need to add `line_items_total` and `line_items` to your order creation.

### Current Code (Example):
```javascript
// Your current order creation
const orderOptions = {
  amount: 50000,  // ₹500
  currency: 'INR',
  receipt: 'receipt_123',
  notes: { customer: 'John' }
};

const order = await razorpay.orders.create(orderOptions);
```

### New Code (Add line items):
```javascript
// Calculate total
const line_items_total = cartItems.reduce((sum, item) => 
  sum + (item.price * item.quantity), 0
);

// Transform cart to Razorpay format
const line_items = cartItems.map(item => ({
  sku: item.sku,
  name: item.name,
  price: item.price,
  offer_price: item.price,
  tax_amount: 0,
  quantity: item.quantity,
  description: item.description || '',
  image_url: item.image_url || '',
  product_url: item.product_url || '',
  weight: item.weight || 0,
  dimensions: {
    length: item.length || 0,
    width: item.width || 0,
    height: item.height || 0
  },
  variant_id: item.variant_id || item.sku
}));

// Create order with line items
const orderOptions = {
  amount: line_items_total,
  currency: 'INR',
  receipt: 'receipt_123',
  notes: { customer: 'John' },
  
  // ADD THESE TWO FIELDS (MANDATORY):
  line_items_total: line_items_total,
  line_items: line_items
};

const order = await razorpay.orders.create(orderOptions);
```

**Critical:** Without `line_items_total`, Magic Checkout won't activate!

---

## Step 4: Create 3 New API Endpoints (2-3 hours)

Razorpay will call these endpoints during checkout. You need to implement them.

---

### Endpoint 1: Shipping Info API

**Purpose:** Tell Razorpay if you deliver to customer's location + shipping/COD fees

**URL:** `POST /api/shipping-info`

**Request from Razorpay:**
```json
{
  "order_id": "receipt_123",
  "razorpay_order_id": "order_abc",
  "email": "user@example.com",
  "contact": "+919876543210",
  "addresses": [{
    "id": "0",
    "zipcode": "560001",
    "state_code": "KA",
    "country": "IN"
  }]
}
```

**Your Response:**
```json
{
  "addresses": [{
    "id": "0",
    "serviceable": true,
    "cod": true,
    "shipping_fee": 5000,
    "cod_fee": 2000
  }]
}
```

**Implementation:**
```javascript
app.post('/api/shipping-info', async (req, res) => {
  const { addresses } = req.body;
  
  const result = await Promise.all(addresses.map(async (addr) => {
    // YOUR BUSINESS LOGIC
    const canDeliver = await checkDelivery(addr.zipcode);
    const codAllowed = await checkCOD(addr.zipcode);
    const shippingCost = await getShippingCost(addr.zipcode);
    
    return {
      id: addr.id,
      serviceable: canDeliver,    // true/false
      cod: codAllowed,             // true/false
      shipping_fee: shippingCost,  // in paise (₹50 = 5000)
      cod_fee: codAllowed ? 2000 : 0
    };
  }));
  
  res.json({ addresses: result });
});
```

**Notes:**
- Amounts in **paise** (1 rupee = 100 paise)
- `serviceable: false` will prevent checkout
- Response must be < 3 seconds

---

### Endpoint 2: Get Promotions API

**Purpose:** Show available coupon codes to customers

**URL:** `POST /api/get-promotions`

**Request from Razorpay:**
```json
{
  "order_id": "receipt_123",
  "contact": "+919876543210",
  "email": "user@example.com"
}
```

**Your Response:**
```json
{
  "promotions": [
    {
      "code": "SAVE100",
      "value": 10000,
      "type": "cart_value",
      "description": "Get ₹100 off",
      "reference_id": "promo_1"
    }
  ]
}
```

**Implementation:**
```javascript
app.post('/api/get-promotions', async (req, res) => {
  const { order_id, email } = req.body;
  
  // YOUR BUSINESS LOGIC
  const activeCoupons = await getActiveCoupons(email, order_id);
  
  const promotions = activeCoupons.map(coupon => ({
    code: coupon.code,
    value: coupon.discount_amount, // in paise
    type: 'cart_value',
    description: coupon.description,
    reference_id: coupon.id
  }));
  
  res.json({ promotions });
});
```

**Notes:**
- Return empty array `[]` if no coupons available
- `type` can be `'cart_value'` or `'percentage'`

---

### Endpoint 3: Apply Promotion API

**Purpose:** Validate coupon code when customer applies it

**URL:** `POST /api/apply-promotion`

**Request from Razorpay:**
```json
{
  "order_id": "receipt_123",
  "contact": "+919876543210",
  "email": "user@example.com",
  "code": "SAVE100"
}
```

**Success Response:**
```json
{
  "promotion": {
    "code": "SAVE100",
    "value": 10000,
    "type": "cart_value",
    "description": "Get ₹100 off",
    "reference_id": "promo_1"
  }
}
```

**Error Response:**
```json
{
  "error": {
    "code": "INVALID_PROMOTION",
    "description": "Coupon code is invalid or expired"
  }
}
```

**Implementation:**
```javascript
app.post('/api/apply-promotion', async (req, res) => {
  const { code, order_id, email } = req.body;
  
  // YOUR BUSINESS LOGIC
  const coupon = await validateCoupon(code, order_id, email);
  
  if (!coupon.valid) {
    return res.status(400).json({
      error: {
        code: 'INVALID_PROMOTION',
        description: coupon.reason || 'Invalid coupon'
      }
    });
  }
  
  res.json({
    promotion: {
      code: coupon.code,
      value: coupon.discount_amount, // in paise
      type: 'cart_value',
      description: coupon.description,
      reference_id: coupon.id
    }
  });
});
```

**Error Codes:**
- `INVALID_PROMOTION` - Code doesn't exist/expired
- `LOGIN_REQUIRED` - User must login to use this coupon
- `REQUIREMENT_NOT_MET` - Conditions not met (e.g., min cart value)

---

## Complete Code Examples

### Node.js/Express Example

```javascript
const express = require('express');
const Razorpay = require('razorpay');
const app = express();

app.use(express.json());

const razorpay = new Razorpay({
  key_id: process.env.RAZORPAY_KEY_ID,
  key_secret: process.env.RAZORPAY_KEY_SECRET
});

// MODIFIED: Create Order with Line Items
app.post('/api/create-order', async (req, res) => {
  const { cart_items, customer } = req.body;
  
  const line_items_total = cart_items.reduce((sum, item) => 
    sum + (item.price * item.quantity), 0
  );
  
  const line_items = cart_items.map(item => ({
    sku: item.sku,
    name: item.name,
    price: item.price,
    offer_price: item.price,
    tax_amount: 0,
    quantity: item.quantity,
    description: item.description || '',
    image_url: item.image_url || '',
    product_url: item.product_url || '',
    weight: item.weight || 0,
    dimensions: { length: 0, width: 0, height: 0 },
    variant_id: item.sku
  }));
  
  const order = await razorpay.orders.create({
    amount: line_items_total,
    currency: 'INR',
    receipt: `receipt_${Date.now()}`,
    line_items_total: line_items_total,
    line_items: line_items
  });
  
  res.json({
    order_id: order.id,
    amount: order.amount,
    key_id: process.env.RAZORPAY_KEY_ID
  });
});

// NEW: Shipping Info API
app.post('/api/shipping-info', async (req, res) => {
  const { addresses } = req.body;
  
  const result = addresses.map(addr => ({
    id: addr.id,
    serviceable: true,  // Your logic here
    cod: true,          // Your logic here
    shipping_fee: 5000, // ₹50 in paise
    cod_fee: 2000       // ₹20 in paise
  }));
  
  res.json({ addresses: result });
});

// NEW: Get Promotions API
app.post('/api/get-promotions', async (req, res) => {
  // Your business logic to fetch active coupons
  const promotions = [
    {
      code: 'SAVE100',
      value: 10000,
      type: 'cart_value',
      description: 'Get ₹100 off',
      reference_id: 'promo_1'
    }
  ];
  
  res.json({ promotions });
});

// NEW: Apply Promotion API
app.post('/api/apply-promotion', async (req, res) => {
  const { code } = req.body;
  
  // Your business logic to validate coupon
  if (code === 'SAVE100') {
    res.json({
      promotion: {
        code: 'SAVE100',
        value: 10000,
        type: 'cart_value',
        description: 'Get ₹100 off',
        reference_id: 'promo_1'
      }
    });
  } else {
    res.status(400).json({
      error: {
        code: 'INVALID_PROMOTION',
        description: 'Invalid coupon code'
      }
    });
  }
});

app.listen(3000);
```

### Python/Flask Example

```python
from flask import Flask, request, jsonify
import razorpay
import os

app = Flask(__name__)

razorpay_client = razorpay.Client(
    auth=(os.getenv('RAZORPAY_KEY_ID'), os.getenv('RAZORPAY_KEY_SECRET'))
)

# MODIFIED: Create Order with Line Items
@app.route('/api/create-order', methods=['POST'])
def create_order():
    data = request.json
    cart_items = data['cart_items']
    
    line_items_total = sum(item['price'] * item['quantity'] for item in cart_items)
    
    line_items = [{
        'sku': item['sku'],
        'name': item['name'],
        'price': item['price'],
        'offer_price': item['price'],
        'tax_amount': 0,
        'quantity': item['quantity'],
        'description': item.get('description', ''),
        'image_url': item.get('image_url', ''),
        'product_url': item.get('product_url', ''),
        'weight': item.get('weight', 0),
        'dimensions': {'length': 0, 'width': 0, 'height': 0},
        'variant_id': item['sku']
    } for item in cart_items]
    
    order = razorpay_client.order.create({
        'amount': line_items_total,
        'currency': 'INR',
        'receipt': f'receipt_{int(time.time())}',
        'line_items_total': line_items_total,
        'line_items': line_items
    })
    
    return jsonify({
        'order_id': order['id'],
        'amount': order['amount'],
        'key_id': os.getenv('RAZORPAY_KEY_ID')
    })

# NEW: Shipping Info API
@app.route('/api/shipping-info', methods=['POST'])
def shipping_info():
    data = request.json
    addresses = data['addresses']
    
    result = [{
        'id': addr['id'],
        'serviceable': True,  # Your logic
        'cod': True,          # Your logic
        'shipping_fee': 5000, # ₹50
        'cod_fee': 2000       # ₹20
    } for addr in addresses]
    
    return jsonify({'addresses': result})

# NEW: Get Promotions API
@app.route('/api/get-promotions', methods=['POST'])
def get_promotions():
    promotions = [
        {
            'code': 'SAVE100',
            'value': 10000,
            'type': 'cart_value',
            'description': 'Get ₹100 off',
            'reference_id': 'promo_1'
        }
    ]
    return jsonify({'promotions': promotions})

# NEW: Apply Promotion API
@app.route('/api/apply-promotion', methods=['POST'])
def apply_promotion():
    code = request.json['code']
    
    if code == 'SAVE100':
        return jsonify({
            'promotion': {
                'code': 'SAVE100',
                'value': 10000,
                'type': 'cart_value',
                'description': 'Get ₹100 off',
                'reference_id': 'promo_1'
            }
        })
    else:
        return jsonify({
            'error': {
                'code': 'INVALID_PROMOTION',
                'description': 'Invalid coupon code'
            }
        }), 400

if __name__ == '__main__':
    app.run()
```

---

## Testing

### Test in Browser

1. Add items to cart
2. Click checkout
3. **Look for Magic Checkout UI:**
   - Razorpay "Magic" branding
   - Address selection interface
   - Coupon code section

4. **Check Network Tab:**
   - Should see calls to your 3 new endpoints
   - `/api/shipping-info`
   - `/api/get-promotions`
   - `/api/apply-promotion` (if coupon used)

### Test Cards

**Success:**
- Card: `4111 1111 1111 1111`
- CVV: `123`
- Expiry: Any future date

**Failure:**
- Card: `4111 1111 1111 1234`

---

## Troubleshooting

### Issue: Standard Checkout Shows Instead of Magic

**Cause:** `line_items_total` missing in order creation

**Fix:**
```javascript
// Make sure you have this in order creation:
const orderOptions = {
  amount: totalAmount,
  currency: 'INR',
  receipt: 'receipt_123',
  line_items_total: totalAmount, // ← THIS IS MANDATORY
  line_items: [...] // ← THIS TOO
};
```

### Issue: "Endpoint not responding"

**Check:**
- Endpoints are publicly accessible (no auth)
- Endpoints are HTTPS
- Response time < 3 seconds
- Correct URLs configured in Dashboard

### Issue: Coupons not showing

**Check:**
- `/api/get-promotions` is returning array (even if empty)
- `show_coupons: true` in frontend options
- Response format matches exactly

### Issue: Shipping fee not calculating

**Check:**
- `/api/shipping-info` returning correct format
- Amounts in paise (₹50 = 5000)
- `serviceable: true` for valid addresses

---

## Go-Live Checklist

### Before Going Live

- [ ] Magic Checkout enabled by Razorpay (confirmed via email)
- [ ] All 3 endpoints created and tested
- [ ] Endpoints configured in Dashboard (Live Mode)
- [ ] Frontend updated (script URL + one_click_checkout)
- [ ] Order creation includes line_items_total
- [ ] Tested end-to-end in Test Mode
- [ ] Tested with ₹1 payment in Live Mode
- [ ] Endpoints respond in < 3 seconds
- [ ] HTTPS enabled on all endpoints
- [ ] Error logging in place

### After Going Live

- [ ] Monitor API endpoint response times
- [ ] Check for errors in logs
- [ ] Verify coupon applications working
- [ ] Confirm shipping calculations correct
- [ ] Monitor payment success rate

---

## Summary

### What Changed:

1. **Script:** `checkout.js` → `magic-checkout.js`
2. **Frontend:** Added `one_click_checkout: true`
3. **Order Creation:** Added `line_items_total` + `line_items`
4. **New APIs:** Created 3 endpoints (shipping, promotions)

### What Stayed Same:

- Payment verification logic
- Webhook handlers
- Order management
- Existing business logic

### Key Points:

- **Without `line_items_total`** → Standard Checkout shows
- **Endpoints must be public** → No authentication
- **Amounts in paise** → ₹1 = 100 paise
- **Response time** → Must be < 3 seconds

---

## Need Help?

**Razorpay Support:**
- Email: support@razorpay.com
- Dashboard: Settings → Support

**Common Errors:**
- Standard checkout showing → Missing `line_items_total`
- Endpoint errors → Check Dashboard logs
- Coupon not applying → Check error response format

---

**End of Migration Guide**

This document contains everything needed to migrate from Standard Checkout to Magic Checkout. Keep this file for reference during implementation and testing.