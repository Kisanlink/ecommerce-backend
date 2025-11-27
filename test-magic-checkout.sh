#!/bin/bash

# Magic Checkout Test Script
# This script tests all the Magic Checkout endpoints to verify they're working

BASE_URL="http://localhost:8080"  # Change this to your server URL
ORDER_ID="order_test123"
KEY_ID="rzp_test_RbZdqWTb0nVG4u"

echo "🧪 Testing Magic Checkout Endpoints..."
echo "Base URL: $BASE_URL"
echo "========================================="

# Test 1: Test endpoint
echo "1. Testing configuration endpoint..."
curl -s -X GET "$BASE_URL/api/v1/magic-checkout/test" | jq '.'
echo ""

# Test 2: Customer GET endpoint
echo "2. Testing customer GET endpoint..."
curl -s -X GET "$BASE_URL/api/v1/magic-checkout/customer/$ORDER_ID?key_id=$KEY_ID" | jq '.'
echo ""

# Test 3: Customer PATCH endpoint (the failing one)
echo "3. Testing customer PATCH endpoint (the failing one)..."
curl -s -X PATCH "$BASE_URL/api/v1/magic-checkout/customer/$ORDER_ID?key_id=$KEY_ID" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "customer_details[contact]=+919876543210&customer_details[email]=test@example.com&customer_details[shipping_address][line1]=Test Address&customer_details[shipping_address][city]=Test City&customer_details[shipping_address][state]=Test State&customer_details[shipping_address][zipcode]=500001" | jq '.'
echo ""

# Test 4: Shipping info endpoint
echo "4. Testing shipping info endpoint..."
curl -s -X POST "$BASE_URL/api/v1/magic-checkout/shipping-info" \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "test123",
    "razorpay_order_id": "'$ORDER_ID'",
    "email": "test@example.com",
    "contact": "+919876543210",
    "addresses": [{
      "id": "1",
      "zipcode": "500001",
      "city": "Hyderabad",
      "state": "Telangana",
      "address_line_1": "Test Address",
      "country": "IN"
    }]
  }' | jq '.'
echo ""

# Test 5: Debug endpoint
echo "5. Testing debug catch-all endpoint..."
curl -s -X POST "$BASE_URL/api/v1/magic-checkout/debug/test-path" \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}' | jq '.'
echo ""

echo "✅ All tests completed!"
echo "Check your server logs for detailed request information."
echo ""
echo "🔍 Look for these log messages:"
echo "- MAGIC CHECKOUT CUSTOMER REQUEST (CRITICAL)"
echo "- HANDLING PATCH REQUEST FOR MAGIC CHECKOUT"
echo "- EXTRACTED CUSTOMER DATA"
echo "- ORDER ADDRESS CREATED FROM FORM DATA"