# Test Magic Checkout CORS and Endpoints
# Run this script to verify all Magic Checkout endpoints are working

$API_BASE = "https://api.amrti.com/api/v1"
$ORIGIN = "http://127.0.0.1:5500"

Write-Host "=== Testing Magic Checkout CORS and Endpoints ===" -ForegroundColor Cyan
Write-Host ""

# Test 1: Health Check
Write-Host "1. Testing Health Check..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "https://api.amrti.com/health" `
        -Method GET `
        -Headers @{"Origin" = $ORIGIN}
    Write-Host "✅ Health Check: $($response.StatusCode)" -ForegroundColor Green
    Write-Host "   CORS Headers: $($response.Headers['access-control-allow-origin'])" -ForegroundColor Gray
} catch {
    Write-Host "❌ Health Check Failed: $_" -ForegroundColor Red
}
Write-Host ""

# Test 2: Preflight for Create Order
Write-Host "2. Testing Preflight (OPTIONS) for Create Order..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$API_BASE/payments/magic-checkout/create-order" `
        -Method OPTIONS `
        -Headers @{
            "Origin" = $ORIGIN
            "Access-Control-Request-Method" = "POST"
            "Access-Control-Request-Headers" = "Content-Type,Authorization,X-Session-ID"
        }
    Write-Host "✅ Preflight: $($response.StatusCode)" -ForegroundColor Green
    Write-Host "   Allowed Methods: $($response.Headers['access-control-allow-methods'])" -ForegroundColor Gray
    Write-Host "   Allowed Headers: $($response.Headers['access-control-allow-headers'])" -ForegroundColor Gray
} catch {
    Write-Host "❌ Preflight Failed: $_" -ForegroundColor Red
}
Write-Host ""

# Test 3: Preflight for Magic Checkout Endpoints
Write-Host "3. Testing Preflight for Magic Checkout endpoints..." -ForegroundColor Yellow
$endpoints = @(
    "/magic-checkout/get-promotions",
    "/magic-checkout/apply-promotion",
    "/magic-checkout/shipping-info",
    "/magic-checkout/addresses-callback",
    "/magic-checkout/customer/test_order"
)

foreach ($endpoint in $endpoints) {
    try {
        $response = Invoke-WebRequest -Uri "$API_BASE$endpoint" `
            -Method OPTIONS `
            -Headers @{
                "Origin" = $ORIGIN
                "Access-Control-Request-Method" = "POST"
            }
        Write-Host "✅ $endpoint : $($response.StatusCode)" -ForegroundColor Green
    } catch {
        Write-Host "❌ $endpoint : Failed" -ForegroundColor Red
    }
}
Write-Host ""

# Test 4: Verify Payment Endpoint
Write-Host "4. Testing Preflight for Verify Payment..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$API_BASE/payments/verify" `
        -Method OPTIONS `
        -Headers @{
            "Origin" = $ORIGIN
            "Access-Control-Request-Method" = "POST"
            "Access-Control-Request-Headers" = "Content-Type,Authorization"
        }
    Write-Host "✅ Verify Payment Preflight: $($response.StatusCode)" -ForegroundColor Green
} catch {
    Write-Host "❌ Verify Payment Preflight Failed: $_" -ForegroundColor Red
}
Write-Host ""

# Test 5: Webhook Endpoint (should allow CORS for Razorpay)
Write-Host "5. Testing Webhook Endpoint..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$API_BASE/webhooks/razorpay" `
        -Method OPTIONS `
        -Headers @{
            "Origin" = "https://api.razorpay.com"
            "Access-Control-Request-Method" = "POST"
        }
    Write-Host "✅ Webhook Preflight: $($response.StatusCode)" -ForegroundColor Green
} catch {
    Write-Host "⚠️  Webhook Preflight: $_" -ForegroundColor Yellow
}
Write-Host ""

Write-Host "=== CORS Testing Complete ===" -ForegroundColor Cyan
Write-Host ""
Write-Host "Next Steps:" -ForegroundColor Yellow
Write-Host "1. Open magic-checkout-demo.html in your browser" -ForegroundColor White
Write-Host "2. Enter your Firebase Auth Token" -ForegroundColor White
Write-Host "3. Click 'Launch Magic Checkout'" -ForegroundColor White
Write-Host "4. Check browser console for any CORS errors" -ForegroundColor White


