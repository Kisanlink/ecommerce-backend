# Address-Based Order Creation Flow Test

## Overview
This document demonstrates the proper flow for creating orders using saved addresses instead of typing addresses repeatedly.

## Flow Steps

### 1. User Authentication
First, the user must be authenticated with Firebase auth token.

### 2. Get User's Saved Addresses
```bash
# Get all saved addresses for the user
curl -X GET "http://localhost:8080/api/v1/addresses" \
  -H "Authorization: Bearer YOUR_FIREBASE_TOKEN" \
  -H "Content-Type: application/json"
```

**Expected Response:**
```json
{
  "data": [
    {
      "id": "address-uuid-1",
      "user_id": "user-uuid",
      "type": "home",
      "first_name": "John",
      "last_name": "Doe", 
      "address_line_1": "123 Main Street",
      "address_line_2": "Apt 4B",
      "city": "Mumbai",
      "state": "Maharashtra", 
      "postal_code": "400001",
      "country": "India",
      "phone": "9876543210",
      "is_default": true
    },
    {
      "id": "address-uuid-2",
      "user_id": "user-uuid",
      "type": "work",
      "first_name": "John",
      "last_name": "Doe",
      "address_line_1": "456 Office Complex",
      "city": "Mumbai", 
      "state": "Maharashtra",
      "postal_code": "400002",
      "country": "India",
      "phone": "9876543210",
      "is_default": false
    }
  ]
}
```

### 3. Add Items to Cart
```bash
# Add items to cart (if not already done)
curl -X POST "http://localhost:8080/api/v1/cart/items" \
  -H "Authorization: Bearer YOUR_FIREBASE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": "product-uuid",
    "quantity": 2
  }'
```

### 4. Estimate Shipping (Optional)
User can estimate shipping costs using their saved address:

```bash
# Estimate shipping using saved address ID
curl -X POST "http://localhost:8080/api/v1/cart/checkout/estimate-shipping" \
  -H "Authorization: Bearer YOUR_FIREBASE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "address_id": "address-uuid-1"
  }'
```

**Expected Response:**
```json
{
  "data": {
    "estimate_id": "est_123", 
    "subtotal": 1250.00,
    "shipping_charges": 0.00,
    "total": 1250.00,
    "free_delivery": true,
    "free_delivery_threshold": 500,
    "options": [
      {
        "courier": "Delhivery",
        "service": "Surface",
        "price": 0.00,
        "eta_min_days": 2,
        "eta_max_days": 4
      }
    ],
    "address": {
      "formatted": "123 Main Street, Apt 4B, Mumbai, Maharashtra, 400001, India"
    }
  }
}
```

### 5. Create Payment Order with Address ID
This is the key step - user selects an address ID and creates the payment order:

```bash
# Create payment order using saved address ID
curl -X POST "http://localhost:8080/api/v1/payments/create-order" \
  -H "Authorization: Bearer YOUR_FIREBASE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "shipping_address_id": "address-uuid-1",
    "first_name": "John",
    "last_name": "Doe",
    "mobile": "9876543210"
  }'
```

**Expected Response:**
```json
{
  "order_id": "order_razorpay_123",
  "amount": 1250.00,
  "currency": "INR", 
  "key_id": "rzp_test_key",
  "description": "Kisanlink Order Payment - Free Delivery",
  "notes": {
    "user_id": "user-uuid",
    "shipping_address": "123 Main Street, Apt 4B, Mumbai, Maharashtra, 400001, India",
    "shipping_address_id": "address-uuid-1",
    "cart_total": 1250.00,
    "shipping_charges": 0.00,
    "subtotal": 1250.00,
    "free_delivery": true,
    "first_name": "John",
    "last_name": "Doe", 
    "mobile": "9876543210"
  },
  "checkout_url": "http://localhost:8080/api/v1/payments/checkout?order_id=order_razorpay_123"
}
```

### 6. Complete Payment
User completes payment through Razorpay and verifies:

```bash
# Verify payment after completion
curl -X POST "http://localhost:8080/api/v1/payments/verify" \
  -H "Authorization: Bearer YOUR_FIREBASE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "razorpay_order_id": "order_razorpay_123",
    "razorpay_payment_id": "pay_razorpay_456", 
    "razorpay_signature": "signature_hash"
  }'
```

## Benefits of This Flow

### ✅ **User Experience Benefits:**
- **No Repeated Typing:** User saves address once, reuses multiple times
- **Fast Checkout:** Just select address ID, no form filling
- **Address Management:** Users can manage all addresses in one place
- **Default Address:** System remembers preferred address

### ✅ **Technical Benefits:**
- **Data Consistency:** Address stored in structured database format
- **Validation:** Address validated once during creation
- **Audit Trail:** Track which address was used for each order
- **Performance:** No need to parse/validate address strings repeatedly

### ✅ **Business Benefits:**
- **Reduced Cart Abandonment:** Faster checkout process
- **Better Data Quality:** Structured address data for analytics
- **Customer Insights:** Understanding delivery preferences
- **Shipping Optimization:** Accurate address data for carrier APIs

## Error Handling

### Invalid Address ID
```json
{
  "error": "Invalid shipping_address_id. Please call GET /api/v1/addresses to get valid address IDs",
  "hint": "Use GET /api/v1/addresses to see your saved addresses and their IDs"
}
```

### No Address Provided
```json
{
  "error": "Shipping address is required",
  "hint": "Provide either 'shipping_address_id' (recommended) or 'shipping_address' (deprecated)", 
  "recommendation": "Use GET /api/v1/addresses to get your saved address IDs, then use 'shipping_address_id' field"
}
```

## API Endpoints Summary

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/addresses` | List user's saved addresses |
| POST | `/api/v1/addresses` | Create new address | 
| PUT | `/api/v1/addresses/:id` | Update existing address |
| DELETE | `/api/v1/addresses/:id` | Delete address |
| PUT | `/api/v1/addresses/:id/default` | Set as default address |
| POST | `/api/v1/cart/checkout/estimate-shipping` | Estimate shipping with address_id |
| POST | `/api/v1/payments/create-order` | Create order with address_id |

## Complete Flow Sequence

```
1. User Authentication 
   ↓
2. GET /api/v1/addresses (get saved addresses)
   ↓  
3. [Optional] POST /api/v1/cart/checkout/estimate-shipping (with address_id)
   ↓
4. POST /api/v1/payments/create-order (with shipping_address_id) 
   ↓
5. Complete Payment via Razorpay
   ↓
6. POST /api/v1/payments/verify
   ↓
7. Order Created with Address from Database
```

This flow eliminates the need for users to type addresses repeatedly and provides a much better user experience while maintaining data integrity.
