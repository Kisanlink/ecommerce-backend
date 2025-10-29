# Session Cart Coupon Implementation

## Overview

This document describes the implementation of coupon functionality for session carts (anonymous users) in the ecommerce backend. Users can now apply, validate, and remove coupons from their session carts without requiring authentication.

## Features Implemented

### 1. Session Cart Coupon Service Methods

- `ApplyCouponToSessionCart(ctx, sessionID, couponCode)` - Applies a coupon to a session cart
- `RemoveCouponFromSessionCart(ctx, sessionID)` - Removes a coupon from a session cart  
- `ValidateCouponForSessionCart(ctx, sessionID, couponCode)` - Validates a coupon without applying it

### 2. Session Cart Coupon Handler Endpoints

- `POST /cart/apply-coupon` - Apply coupon to cart (works for both authenticated and anonymous users)
- `DELETE /cart/remove-coupon` - Remove coupon from cart (works for both authenticated and anonymous users)
- `GET /cart/validate-coupon?coupon_code=CODE` - Validate coupon for cart (works for both authenticated and anonymous users)

### 3. Key Features

- **Anonymous User Support**: Session carts can now use coupons without authentication
- **Coupon Validation**: Users can check if a coupon is valid before applying it
- **Discount Calculation**: Proper discount calculation with product restrictions
- **Session Persistence**: Coupon information is stored in the session cart
- **Error Handling**: Comprehensive error handling for invalid coupons, empty carts, etc.

## API Usage Examples

### Apply Coupon to Session Cart

```bash
# For anonymous users
curl -X POST http://localhost:8080/api/v1/cart/apply-coupon \
  -H "Content-Type: application/json" \
  -H "X-Session-ID: session_abc123" \
  -d '{"coupon_code": "WELCOME10"}'

# Response
{
  "success": true,
  "data": {
    "cart_total": 100.00,
    "discount_amount": 10.00,
    "discounted_total": 90.00,
    "session_id": "session_abc123"
  },
  "message": "Coupon applied successfully"
}
```

### Validate Coupon for Session Cart

```bash
# Check if coupon is valid before applying
curl -X GET "http://localhost:8080/api/v1/cart/validate-coupon?coupon_code=WELCOME10" \
  -H "X-Session-ID: session_abc123"

# Response
{
  "success": true,
  "data": {
    "cart_total": 100.00,
    "discount_amount": 10.00,
    "discounted_total": 90.00,
    "is_valid": true,
    "session_id": "session_abc123"
  },
  "message": "Coupon is valid"
}
```

### Remove Coupon from Session Cart

```bash
# Remove applied coupon
curl -X DELETE http://localhost:8080/api/v1/cart/remove-coupon \
  -H "X-Session-ID: session_abc123"

# Response
{
  "success": true,
  "data": {
    "cart_total": 100.00,
    "discount_amount": 0,
    "discounted_total": 100.00,
    "coupon": null,
    "session_id": "session_abc123"
  },
  "message": "Coupon removed successfully"
}
```

## Implementation Details

### Session Cart Service Updates

1. **Added Coupon Repository**: SessionCartService now includes `CouponRepositoryInterface` for coupon operations
2. **Coupon Methods**: Three new methods for coupon operations on session carts
3. **Validation Logic**: Same validation logic as authenticated carts, with simplified first-time user assumption
4. **Session Storage**: Coupon information is stored in the in-memory session cart storage

### Handler Updates

1. **Unified Endpoints**: Same endpoints work for both authenticated and anonymous users
2. **Session ID Support**: Anonymous users can use `X-Session-ID` header for cart operations
3. **Error Handling**: Proper error responses for invalid coupons, empty carts, etc.
4. **Response Format**: Consistent response format with session ID included for anonymous users

### Route Updates

1. **Public Routes**: Coupon endpoints are now available as public routes (no authentication required)
2. **Optional Auth**: Routes use optional authentication middleware to detect if user is authenticated
3. **Backward Compatibility**: Existing authenticated coupon routes remain unchanged

## Benefits

1. **Better User Experience**: Anonymous users can see discounts before creating an account
2. **Increased Conversions**: Users can apply coupons without authentication barriers
3. **Consistent API**: Same endpoints work for both authenticated and anonymous users
4. **Flexible Validation**: Users can validate coupons before applying them

## Error Handling

The implementation includes comprehensive error handling for:

- Invalid session IDs
- Invalid coupon codes
- Empty carts
- Coupon validation failures
- Product restriction violations
- Insufficient cart totals for minimum requirements

## Future Enhancements

1. **Redis Storage**: Move session cart storage from in-memory to Redis for scalability
2. **Coupon History**: Track coupon usage in session carts
3. **Advanced Validation**: More sophisticated first-time user detection
4. **Analytics**: Track coupon usage patterns for anonymous users
