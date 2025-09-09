# Test payment verification with detailed logging
# This script will trigger the payment verification flow to debug the AWB/tracking URL issue

Write-Host "Testing Payment Verification with Debug Logging" -ForegroundColor Green
Write-Host "===============================================" -ForegroundColor Green

# Server URL
$baseUrl = "http://localhost:8080"

# First check if server is running
Write-Host "1. Checking server health..." -ForegroundColor Yellow
try {
    $healthResponse = Invoke-RestMethod -Uri "$baseUrl/ping" -Method GET
    Write-Host "✓ Server is running" -ForegroundColor Green
} catch {
    Write-Host "✗ Server is not running. Start the server first!" -ForegroundColor Red
    exit 1
}

# Check Delhivery configuration
Write-Host "`n2. Checking Delhivery configuration..." -ForegroundColor Yellow
try {
    $configResponse = Invoke-RestMethod -Uri "$baseUrl/debug/delhivery-config" -Method GET
    Write-Host "✓ Delhivery Config:" -ForegroundColor Green
    Write-Host "  - Base URL: $($configResponse.config.base_url)" -ForegroundColor Cyan
    Write-Host "  - Production: $($configResponse.config.is_production)" -ForegroundColor Cyan
    Write-Host "  - Warehouse: $($configResponse.config.warehouse_name)" -ForegroundColor Cyan
    Write-Host "  - Origin Pincode: $($configResponse.config.origin_pincode)" -ForegroundColor Cyan
} catch {
    Write-Host "✗ Failed to get Delhivery configuration" -ForegroundColor Red
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
}

# Sample test data for payment verification
Write-Host "`n3. Preparing test payment verification..." -ForegroundColor Yellow

# This is a sample payment verification request
# You'll need to replace these with actual values from a real payment
$paymentData = @{
    razorpay_payment_id = "pay_test123456"
    razorpay_order_id = "order_test123456"  
    razorpay_signature = "test_signature_123"
    order_id = "1"  # Replace with actual order ID
} | ConvertTo-Json

Write-Host "Test Payment Data:" -ForegroundColor Cyan
Write-Host $paymentData -ForegroundColor White

Write-Host "`n4. Attempting payment verification..." -ForegroundColor Yellow
Write-Host "Note: This will likely fail with invalid signature, but will show the logging flow" -ForegroundColor Yellow

try {
    $headers = @{
        'Content-Type' = 'application/json'
        'Authorization' = 'Bearer YOUR_TEST_TOKEN_HERE'  # Replace with actual token
    }
    
    $verifyResponse = Invoke-RestMethod -Uri "$baseUrl/api/v1/payments/verify" -Method POST -Body $paymentData -Headers $headers
    Write-Host "✓ Payment verification response:" -ForegroundColor Green
    Write-Host ($verifyResponse | ConvertTo-Json -Depth 3) -ForegroundColor White
} catch {
    $statusCode = $_.Exception.Response.StatusCode.value__
    $responseBody = $_.Exception.Response | ConvertFrom-Json -ErrorAction SilentlyContinue
    
    Write-Host "Payment verification failed (expected for test data):" -ForegroundColor Yellow
    Write-Host "  Status Code: $statusCode" -ForegroundColor Cyan
    Write-Host "  Error: $($_.Exception.Message)" -ForegroundColor Red
    
    if ($responseBody) {
        Write-Host "  Response: $($responseBody | ConvertTo-Json)" -ForegroundColor White
    }
}

Write-Host "`n5. Check server logs for detailed debugging information..." -ForegroundColor Yellow
Write-Host "Look for logs with the following patterns in your server output:" -ForegroundColor Cyan
Write-Host "  - 'Starting payment verification'" -ForegroundColor White
Write-Host "  - 'Shipment creation request'" -ForegroundColor White
Write-Host "  - 'Delhivery API response'" -ForegroundColor White
Write-Host "  - 'AWB number extraction'" -ForegroundColor White

Write-Host "`n6. Alternative: Test with a real order..." -ForegroundColor Yellow
Write-Host "To get real debugging data:" -ForegroundColor Cyan
Write-Host "  a. Create a real order through the app" -ForegroundColor White
Write-Host "  b. Make a real payment" -ForegroundColor White  
Write-Host "  c. Use the actual payment verification data" -ForegroundColor White
Write-Host "  d. Watch the server logs for detailed Delhivery API interaction" -ForegroundColor White

Write-Host "`nDebug script completed!" -ForegroundColor Green
