# CORS Testing Script for Ecommerce Backend (PowerShell)
# This script tests CORS configuration with different origins and methods

$BASE_URL = "http://localhost:8080"
$API_BASE = "$BASE_URL/api/v1"

Write-Host "Testing CORS Configuration for Ecommerce Backend" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""

# Function to test CORS for a given origin and endpoint
function Test-CORS {
    param(
        [string]$Origin,
        [string]$Endpoint,
        [string]$Method,
        [string]$Description
    )
    
    Write-Host "Testing $Description... " -NoNewline
    
    # Test preflight request
    $preflightResponse = try {
        Invoke-WebRequest -Uri $Endpoint -Method OPTIONS -Headers @{
            "Origin" = $Origin
            "Access-Control-Request-Method" = $Method
            "Access-Control-Request-Headers" = "Content-Type,Authorization"
        } -UseBasicParsing
        $preflightResponse.StatusCode
    } catch {
        $_.Exception.Response.StatusCode.Value__
    }
    
    if ($preflightResponse -eq 204) {
        Write-Host "✓ Preflight OK" -ForegroundColor Green
    } else {
        Write-Host "✗ Preflight Failed (HTTP $preflightResponse)" -ForegroundColor Red
    }
    
    # Test actual request
    $actualResponse = try {
        Invoke-WebRequest -Uri $Endpoint -Method $Method -Headers @{
            "Origin" = $Origin
            "Content-Type" = "application/json"
        } -UseBasicParsing
        $actualResponse.StatusCode
    } catch {
        $_.Exception.Response.StatusCode.Value__
    }
    
    if ($actualResponse -in @(200, 401, 404)) {
        Write-Host "  ✓ Request OK (HTTP $actualResponse)" -ForegroundColor Green
    } else {
        Write-Host "  ✗ Request Failed (HTTP $actualResponse)" -ForegroundColor Red
    }
    Write-Host ""
}

# Test allowed origins
Write-Host "Testing Allowed Origins:" -ForegroundColor Yellow
Test-CORS "http://localhost:3000" "$API_BASE/cart" "GET" "Localhost:3000"
Test-CORS "http://localhost:5173" "$API_BASE/cart" "GET" "Localhost:5173"
Test-CORS "https://amrti.com" "$API_BASE/cart" "GET" "Production Domain"
Test-CORS "https://www.amrti.com" "$API_BASE/cart" "GET" "WWW Production Domain"

Write-Host ""

# Test blocked origins
Write-Host "Testing Blocked Origins:" -ForegroundColor Yellow
Test-CORS "http://malicious-site.com" "$API_BASE/cart" "GET" "Malicious Site"
Test-CORS "https://evil.com" "$API_BASE/cart" "GET" "Evil Site"

Write-Host ""

# Test different HTTP methods
Write-Host "Testing HTTP Methods:" -ForegroundColor Yellow
Test-CORS "http://localhost:3000" "$API_BASE/cart" "GET" "GET Method"
Test-CORS "http://localhost:3000" "$API_BASE/cart" "POST" "POST Method"
Test-CORS "http://localhost:3000" "$API_BASE/cart" "PUT" "PUT Method"
Test-CORS "http://localhost:3000" "$API_BASE/cart" "DELETE" "DELETE Method"

Write-Host ""

# Test cart-specific headers
Write-Host "Testing Cart-Specific Headers:" -ForegroundColor Yellow
Write-Host "Testing X-Session-ID header... " -NoNewline
$cartHeadersResponse = try {
    Invoke-WebRequest -Uri "$API_BASE/cart" -Method OPTIONS -Headers @{
        "Origin" = "http://localhost:3000"
        "Access-Control-Request-Method" = "POST"
        "Access-Control-Request-Headers" = "X-Session-ID,X-Cart-ID"
    } -UseBasicParsing
    $cartHeadersResponse.StatusCode
} catch {
    $_.Exception.Response.StatusCode.Value__
}

if ($cartHeadersResponse -eq 204) {
    Write-Host "✓ Cart Headers OK" -ForegroundColor Green
} else {
    Write-Host "✗ Cart Headers Failed (HTTP $cartHeadersResponse)" -ForegroundColor Red
}

Write-Host ""

# Test API endpoints
Write-Host "Testing API Endpoints:" -ForegroundColor Yellow
Test-CORS "http://localhost:3000" "$API_BASE/products" "GET" "Products API"
Test-CORS "http://localhost:3000" "$API_BASE/orders" "GET" "Orders API"
Test-CORS "http://localhost:3000" "$API_BASE/favorites" "GET" "Favorites API"

Write-Host ""
Write-Host "CORS Testing Complete!" -ForegroundColor Cyan
Write-Host ""
Write-Host "Note: Some requests may return 401 (Unauthorized) or 404 (Not Found) which is expected" -ForegroundColor Yellow
Write-Host "The important thing is that CORS headers are properly set and preflight requests work." -ForegroundColor Yellow
