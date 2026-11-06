# Complete E-commerce Testing Guide 🛒

This guide walks you through testing the entire e-commerce flow from user signup to order tracking.

## 🚀 Prerequisites

1. **Start the server**: `go run cmd/server/main.go`
2. **Server URL**: `http://localhost:8080`
3. **Use Postman/curl** for testing API endpoints

---

## 📋 Step-by-Step Testing Flow

### 1. 👤 User Registration/Signup

**Endpoint**: `POST /api/v1/auth/register`

```json
{
  "email": "test@kisanlink.com",
  "password": "TestPassword123",
  "name": "Test User",
  "phone": "+919876543210"
}
```

**Expected Response**:
```json
{
  "message": "User registered successfully",
  "user_id": "USER_xxxxxxxxxxxxx"
}
```

---

### 2. 🔐 User Login (Get Firebase Token)

**Endpoint**: `POST /api/v1/auth/login`

```json
{
  "email": "test@kisanlink.com",
  "password": "TestPassword123"
}
```

**Expected Response**:
```json
{
  "token": "firebase_jwt_token_here",
  "user": {
    "uid": "firebase_uid",
    "email": "test@kisanlink.com",
    "name": "Test User"
  }
}
```

**⚠️ Important**: Save the `token` from the response - you'll need it for all authenticated requests as:
```
Authorization: Bearer <token>
```

---

### 3. 🏪 Browse Available Products

**Endpoint**: `GET /api/v1/products`

**Headers**:
```
Authorization: Bearer <your_firebase_token>
```

**Expected Response**:
```json
{
  "products": [
    {
      "id": "PROD_xxxxxxxxxxxxx",
      "name": "Organic Tomatoes",
      "price": 50.00,
      "description": "Fresh organic tomatoes",
      "category": "vegetables",
      "stock": 100,
      "weight": 500
    }
  ]
}
```

---

### 4. 🛒 Add Products to Cart

**Endpoint**: `POST /api/v1/cart/add`

**Headers**:
```
Authorization: Bearer <your_firebase_token>
```

**Body**:
```json
{
  "product_id": "PROD_xxxxxxxxxxxxx",
  "quantity": 2
}
```

**Expected Response**:
```json
{
  "cart": {
    "id": "CART_xxxxxxxxxxxxx",
    "user_id": "USER_xxxxxxxxxxxxx",
    "total_items": 2,
    "total_price": 100.00,
    "items": [
      {
        "product_id": "PROD_xxxxxxxxxxxxx",
        "quantity": 2,
        "unit_price": 50.00,
        "total_price": 100.00
      }
    ]
  }
}
```

---

### 5. 🛒 Add More Products (Optional)

Repeat step 4 with different `product_id` values to add multiple items.

---

### 6. 👀 View Cart Summary

**Endpoint**: `GET /api/v1/cart`

**Headers**:
```
Authorization: Bearer <your_firebase_token>
```

**Expected Response**:
```json
{
  "cart": {
    "id": "CART_xxxxxxxxxxxxx",
    "user_id": "USER_xxxxxxxxxxxxx",
    "total_items": 4,
    "total_price": 200.00,
    "coupon_code": "",
    "discount_amount": 0,
    "items": [...]
  }
}
```

---

### 7. 🎟️ Apply Coupon (Optional)

**Endpoint**: `POST /api/v1/cart/apply-coupon`

**Headers**:
```
Authorization: Bearer <your_firebase_token>
```

**Body**:
```json
{
  "coupon_code": "SAVE10"
}
```

**Expected Response**:
```json
{
  "success": true,
  "message": "Coupon applied successfully",
  "cart": {
    "coupon_code": "SAVE10",
    "discount_amount": 20.00,
    "total_price": 180.00
  }
}
```

---

### 8. 🚚 Estimate Shipping Cost

**Endpoint**: `POST /api/v1/cart/estimate-shipping`

**Headers**:
```
Authorization: Bearer <your_firebase_token>
```

**Body**:
```json
{
  "destination_pincode": "110001",
  "destination_city": "New Delhi",
  "destination_state": "Delhi"
}
```

**Expected Response**:
```json
{
  "shipping_cost": 50.00,
  "estimated_delivery_days": "3-5",
  "serviceable": true,
  "courier_partner": "Delhivery",
  "total_with_shipping": 230.00
}
```

---

### 9. 💳 Create Order and Payment

**Endpoint**: `POST /api/v1/payments/create-order`

**Headers**:
```
Authorization: Bearer <your_firebase_token>
```

**Body**:
```json
{
  "cart_id": "CART_xxxxxxxxxxxxx",
  "amount": 230.00,
  "currency": "INR",
  "shipping_address": "123 Main Street, New Delhi, Delhi 110001"
}
```

**Expected Response**:
```json
{
  "order_id": "order_razorpay_xxxxx",
  "amount": 230.00,
  "currency": "INR",
  "key_id": "rzp_live_xxxxxxxxxxxxxx",
  "description": "Kisanlink Order Payment",
  "notes": {
    "user_id": "USER_xxxxxxxxxxxxx",
    "cart_id": "CART_xxxxxxxxxxxxx"
  }
}
```

---

### 10. ✅ Verify Payment (Simulate Successful Payment)

**Endpoint**: `POST /api/v1/payments/verify`

**Headers**:
```
Authorization: Bearer <your_firebase_token>
```

**Body**:
```json
{
  "razorpay_order_id": "order_razorpay_xxxxx",
  "razorpay_payment_id": "pay_razorpay_xxxxx",
  "razorpay_signature": "generated_signature_hash"
}
```

**Expected Response**:
```json
{
  "success": true,
  "message": "Payment verified and order confirmed successfully",
  "payment_id": "pay_razorpay_xxxxx"
}
```

---

### 11. 📋 View Order History

**Endpoint**: `GET /api/v1/orders`

**Headers**:
```
Authorization: Bearer <your_firebase_token>
```

**Expected Response**:
```json
{
  "orders": [
    {
      "id": "ORDR_xxxxxxxxxxxxx",
      "user_id": "USER_xxxxxxxxxxxxx",
      "total_amount": 230.00,
      "status": "confirmed",
      "shipping_address": "123 Main Street, New Delhi, Delhi 110001",
      "payment_method": "razorpay:pay_razorpay_xxxxx",
      "payment_status": "completed",
      "created_at": "2025-09-02T10:00:00Z"
    }
  ]
}
```

---

### 12. 📦 Get Order Details with Items

**Endpoint**: `GET /api/v1/orders/{order_id}`

**Headers**:
```
Authorization: Bearer <your_firebase_token>
```

**Expected Response**:
```json
{
  "order": {
    "id": "ORDR_xxxxxxxxxxxxx",
    "user_id": "USER_xxxxxxxxxxxxx",
    "total_amount": 230.00,
    "status": "confirmed",
    "payment_status": "completed",
    "items": [
      {
        "id": "OITM_xxxxxxxxxxxxx",
        "order_id": "ORDR_xxxxxxxxxxxxx",
        "product_id": "PROD_xxxxxxxxxxxxx",
        "quantity": 2,
        "unit_price": 50.00,
        "total_price": 100.00
      }
    ]
  }
}
```

---

### 13. 🚚 Track Shipping Status

**Endpoint**: `GET /api/v1/delivery/track/{order_id}`

**Headers**:
```
Authorization: Bearer <your_firebase_token>
```

**Expected Response**:
```json
{
  "tracking_number": "DLV123456789",
  "status": "in_transit",
  "estimated_delivery": "2025-09-05",
  "current_location": "Delhi Hub",
  "tracking_history": [
    {
      "status": "booked",
      "timestamp": "2025-09-02T10:00:00Z",
      "location": "Origin"
    },
    {
      "status": "picked_up",
      "timestamp": "2025-09-02T14:00:00Z",
      "location": "Hyderabad Hub"
    },
    {
      "status": "in_transit",
      "timestamp": "2025-09-03T08:00:00Z",
      "location": "Delhi Hub"
    }
  ]
}
```

---

## 🔧 Testing Tools

### Using curl:

```bash
# Example: Add item to cart
curl -X POST http://localhost:8080/api/v1/cart/add \
  -H "Authorization: Bearer YOUR_FIREBASE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id":"PROD_xxxxxxxxxxxxx","quantity":2}'
```

### Using Postman:

1. Create a new collection
2. Set base URL: `http://localhost:8080`
3. Add Authorization header to collection: `Bearer {{token}}`
4. Create requests for each endpoint above

---

## 📊 Test Data

### Sample Product IDs:
```
PROD_123456789ABC (if you have seeded data)
```

### Sample Coupon Codes:
```
SAVE10 - 10% discount
FLAT20 - ₹20 flat discount
NEWUSER - 15% discount for new users
```

### Sample Addresses:
```
"123 Main Street, New Delhi, Delhi 110001"
"456 Park Avenue, Mumbai, Maharashtra 400001"
"789 Brigade Road, Bangalore, Karnataka 560001"
```

---

## ⚠️ Important Notes

1. **Firebase Token**: Make sure to get a valid Firebase token from login
2. **Product IDs**: Use actual product IDs from your database
3. **Cart ID**: Save the cart ID from add-to-cart responses
4. **Order Creation**: The system will automatically create a database order before creating the Razorpay order
5. **Payment Simulation**: For testing, you can use Razorpay test credentials
6. **Shipping**: Estimates are based on real Delhivery API integration

---

## 🎯 Success Indicators

✅ **User registered and logged in**  
✅ **Products added to cart**  
✅ **Shipping cost estimated**  
✅ **Order created in database**  
✅ **Payment processed successfully**  
✅ **Cart cleared after successful payment**  
✅ **Order status updated to confirmed**  
✅ **All order details persisted in PostgreSQL**

**Your complete e-commerce flow is working end-to-end!** 🎉
