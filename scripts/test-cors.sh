#!/bin/bash

# CORS Testing Script for Ecommerce Backend
# This script tests CORS configuration with different origins and methods

BASE_URL="http://localhost:8080"
API_BASE="$BASE_URL/api/v1"

echo "Testing CORS Configuration for Ecommerce Backend"
echo "================================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to test CORS for a given origin and endpoint
test_cors() {
    local origin="$1"
    local endpoint="$2"
    local method="$3"
    local description="$4"
    
    echo -n "Testing $description... "
    
    # Test preflight request
    response=$(curl -s -o /dev/null -w "%{http_code}" \
        -X OPTIONS \
        -H "Origin: $origin" \
        -H "Access-Control-Request-Method: $method" \
        -H "Access-Control-Request-Headers: Content-Type,Authorization" \
        "$endpoint")
    
    if [ "$response" = "204" ]; then
        echo -e "${GREEN}✓ Preflight OK${NC}"
    else
        echo -e "${RED}✗ Preflight Failed (HTTP $response)${NC}"
    fi
    
    # Test actual request
    response=$(curl -s -o /dev/null -w "%{http_code}" \
        -X "$method" \
        -H "Origin: $origin" \
        -H "Content-Type: application/json" \
        "$endpoint")
    
    if [ "$response" = "200" ] || [ "$response" = "401" ] || [ "$response" = "404" ]; then
        echo -e "  ${GREEN}✓ Request OK (HTTP $response)${NC}"
    else
        echo -e "  ${RED}✗ Request Failed (HTTP $response)${NC}"
    fi
    echo ""
}

# Test allowed origins
echo -e "${YELLOW}Testing Allowed Origins:${NC}"
test_cors "http://localhost:3000" "$API_BASE/cart" "GET" "Localhost:3000"
test_cors "http://localhost:5173" "$API_BASE/cart" "GET" "Localhost:5173"
test_cors "https://amrti.com" "$API_BASE/cart" "GET" "Production Domain"
test_cors "https://www.amrti.com" "$API_BASE/cart" "GET" "WWW Production Domain"

echo ""

# Test blocked origins
echo -e "${YELLOW}Testing Blocked Origins:${NC}"
test_cors "http://malicious-site.com" "$API_BASE/cart" "GET" "Malicious Site"
test_cors "https://evil.com" "$API_BASE/cart" "GET" "Evil Site"

echo ""

# Test different HTTP methods
echo -e "${YELLOW}Testing HTTP Methods:${NC}"
test_cors "http://localhost:3000" "$API_BASE/cart" "GET" "GET Method"
test_cors "http://localhost:3000" "$API_BASE/cart" "POST" "POST Method"
test_cors "http://localhost:3000" "$API_BASE/cart" "PUT" "PUT Method"
test_cors "http://localhost:3000" "$API_BASE/cart" "DELETE" "DELETE Method"

echo ""

# Test cart-specific headers
echo -e "${YELLOW}Testing Cart-Specific Headers:${NC}"
echo -n "Testing X-Session-ID header... "
response=$(curl -s -o /dev/null -w "%{http_code}" \
    -X OPTIONS \
    -H "Origin: http://localhost:3000" \
    -H "Access-Control-Request-Method: POST" \
    -H "Access-Control-Request-Headers: X-Session-ID,X-Cart-ID" \
    "$API_BASE/cart")

if [ "$response" = "204" ]; then
    echo -e "${GREEN}✓ Cart Headers OK${NC}"
else
    echo -e "${RED}✗ Cart Headers Failed (HTTP $response)${NC}"
fi

echo ""

# Test API endpoints
echo -e "${YELLOW}Testing API Endpoints:${NC}"
test_cors "http://localhost:3000" "$API_BASE/products" "GET" "Products API"
test_cors "http://localhost:3000" "$API_BASE/orders" "GET" "Orders API"
test_cors "http://localhost:3000" "$API_BASE/favorites" "GET" "Favorites API"

echo ""
echo "CORS Testing Complete!"
echo ""
echo "Note: Some requests may return 401 (Unauthorized) or 404 (Not Found) which is expected"
echo "The important thing is that CORS headers are properly set and preflight requests work."
