# Test script for shipping-info endpoint
# This works WITHOUT real order IDs - just needs addresses

$baseUrl = "https://api.amrti.com"
$endpoint = "$baseUrl/api/v1/magic-checkout/shipping-info"

Write-Host "Testing Shipping Info Endpoint" -ForegroundColor Cyan
Write-Host "=================================" -ForegroundColor Cyan
Write-Host ""

# Test 1: Minimal test (just addresses - no order IDs)
Write-Host "Test 1: Minimal test (addresses only)" -ForegroundColor Yellow
$body1 = @{
    addresses = @(
        @{
            id = "0"
            zipcode = "560001"
            state_code = "KA"
            country = "IN"
        }
    )
} | ConvertTo-Json -Depth 10

try {
    $response1 = Invoke-RestMethod -Uri $endpoint -Method POST -Headers @{ "Content-Type" = "application/json" } -Body $body1
    Write-Host "Success!" -ForegroundColor Green
    $response1 | ConvertTo-Json -Depth 10
} catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.ErrorDetails.Message) {
        Write-Host "Details: $($_.ErrorDetails.Message)" -ForegroundColor Red
    }
}
Write-Host ""

# Test 2: With mock order IDs (simulates what Razorpay sends)
Write-Host "Test 2: With mock order IDs" -ForegroundColor Yellow
$body2 = @{
    order_id = "mc_test_1234567890"
    razorpay_order_id = "order_test123"
    email = "test@example.com"
    contact = "+919876543210"
    addresses = @(
        @{
            id = "0"
            zipcode = "560001"
            state_code = "KA"
            country = "IN"
            city = "Bangalore"
            state = "Karnataka"
        }
    )
} | ConvertTo-Json -Depth 10

try {
    $response2 = Invoke-RestMethod -Uri $endpoint -Method POST -Headers @{ "Content-Type" = "application/json" } -Body $body2
    Write-Host "Success!" -ForegroundColor Green
    $response2 | ConvertTo-Json -Depth 10
} catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.ErrorDetails.Message) {
        Write-Host "Details: $($_.ErrorDetails.Message)" -ForegroundColor Red
    }
}
Write-Host ""

# Test 3: Test with cart amount (simulates order with known cart total)
Write-Host "Test 3: With cart amount (for free delivery test)" -ForegroundColor Yellow
$body3 = @{
    order_id = "mc_test_1234567890"
    razorpay_order_id = "order_test123"
    amount = 60000  # Rs.600 in paise (should get free delivery)
    addresses = @(
        @{
            id = "0"
            zipcode = "110001"
            state_code = "DL"
            country = "IN"
            city = "New Delhi"
            state = "Delhi"
        }
    )
} | ConvertTo-Json -Depth 10

try {
    $response3 = Invoke-RestMethod -Uri $endpoint -Method POST -Headers @{ "Content-Type" = "application/json" } -Body $body3
    Write-Host "Success!" -ForegroundColor Green
    Write-Host "Note: Cart total Rs.600 should qualify for FREE delivery" -ForegroundColor Cyan
    $response3 | ConvertTo-Json -Depth 10
} catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.ErrorDetails.Message) {
        Write-Host "Details: $($_.ErrorDetails.Message)" -ForegroundColor Red
    }
}
Write-Host ""

# Test 4: Test with small cart amount (should charge shipping)
Write-Host "Test 4: Small cart amount (should charge shipping)" -ForegroundColor Yellow
$body4 = @{
    order_id = "mc_test_1234567890"
    razorpay_order_id = "order_test123"
    amount = 20000  # Rs.200 in paise (below Rs.500, should charge shipping)
    addresses = @(
        @{
            id = "0"
            zipcode = "400001"
            state_code = "MH"
            country = "IN"
            city = "Mumbai"
            state = "Maharashtra"
        }
    )
} | ConvertTo-Json -Depth 10

try {
    $response4 = Invoke-RestMethod -Uri $endpoint -Method POST -Headers @{ "Content-Type" = "application/json" } -Body $body4
    Write-Host "Success!" -ForegroundColor Green
    Write-Host "Note: Cart total Rs.200 should charge shipping fee" -ForegroundColor Cyan
    $response4 | ConvertTo-Json -Depth 10
} catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.ErrorDetails.Message) {
        Write-Host "Details: $($_.ErrorDetails.Message)" -ForegroundColor Red
    }
}
Write-Host ""

# Test 5: Invalid zipcode (should return serviceable: false)
Write-Host "Test 5: Invalid zipcode (should fail serviceability)" -ForegroundColor Yellow
$body5 = @{
    addresses = @(
        @{
            id = "0"
            zipcode = "123"  # Invalid - not 6 digits
            state_code = "XX"
            country = "IN"
        }
    )
} | ConvertTo-Json -Depth 10

try {
    $response5 = Invoke-RestMethod -Uri $endpoint -Method POST -Headers @{ "Content-Type" = "application/json" } -Body $body5
    Write-Host "Success!" -ForegroundColor Green
    Write-Host "Note: Invalid zipcode should return serviceable: false" -ForegroundColor Cyan
    $response5 | ConvertTo-Json -Depth 10
} catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.ErrorDetails.Message) {
        Write-Host "Details: $($_.ErrorDetails.Message)" -ForegroundColor Red
    }
}
Write-Host ""

Write-Host "=================================" -ForegroundColor Cyan
Write-Host "All tests completed!" -ForegroundColor Green
Write-Host ""
Write-Host "Notes:" -ForegroundColor Yellow
Write-Host "  - order_id and razorpay_order_id are OPTIONAL for testing" -ForegroundColor White
Write-Host "  - Only addresses array is REQUIRED" -ForegroundColor White
Write-Host "  - If order_id is provided, handler tries to get cart total from cache" -ForegroundColor White
Write-Host "  - If amount is provided, it is used as cart total in paise" -ForegroundColor White
Write-Host "  - Cart >= Rs.500 gets FREE delivery with shipping_fee: 0" -ForegroundColor White
Write-Host "  - Cart below Rs.500 gets shipping charges calculated from Delhivery API" -ForegroundColor White

