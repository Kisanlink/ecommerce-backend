# Test Payment Verification API with comprehensive error handling
# This script tests the enhanced payment verification system

# Test server
$BaseURL = "http://localhost:8080"

Write-Host "Starting Payment Verification Test..." -ForegroundColor Green

# First check if server is running
try {
    $healthCheck = Invoke-RestMethod -Uri "$BaseURL/health" -Method GET
    Write-Host "✅ Server is running: $($healthCheck | ConvertTo-Json)" -ForegroundColor Green
} catch {
    Write-Host "❌ Server is not running. Please start the server first." -ForegroundColor Red
    exit 1
}

# Test payment verification with comprehensive logging
Write-Host "`nTesting Payment Verification..." -ForegroundColor Yellow

# Mock payment verification data
$paymentData = @{
    razorpay_order_id = "order_test_123456789"
    razorpay_payment_id = "pay_test_987654321"
    razorpay_signature = "test_signature_mock"
    shipping_address_id = "1"
} | ConvertTo-Json

$headers = @{
    "Content-Type" = "application/json"
    "Authorization" = "Bearer test_token_for_user"
}

try {
    Write-Host "Sending payment verification request..." -ForegroundColor Blue
    Write-Host "Request Data: $paymentData" -ForegroundColor Gray
    
    $response = Invoke-RestMethod -Uri "$BaseURL/api/v1/payments/verify" -Method POST -Body $paymentData -Headers $headers
    Write-Host "✅ Payment verification response received:" -ForegroundColor Green
    Write-Host ($response | ConvertTo-Json -Depth 5) -ForegroundColor White
    
    # Check if AWB and tracking URL are present
    if ($response.awb -and $response.tracking_url) {
        Write-Host "✅ AWB and Tracking URL are present in response!" -ForegroundColor Green
        Write-Host "AWB: $($response.awb)" -ForegroundColor Cyan
        Write-Host "Tracking URL: $($response.tracking_url)" -ForegroundColor Cyan
    } else {
        Write-Host "⚠️ AWB or Tracking URL missing from response" -ForegroundColor Yellow
    }
    
} catch {
    Write-Host "❌ Payment verification failed:" -ForegroundColor Red
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response) {
        $responseStream = $_.Exception.Response.GetResponseStream()
        $reader = New-Object System.IO.StreamReader($responseStream)
        $responseBody = $reader.ReadToEnd()
        Write-Host "Response Body: $responseBody" -ForegroundColor Red
    }
}

# Test payment status endpoint
Write-Host "`nTesting Payment Status..." -ForegroundColor Yellow

try {
    $statusResponse = Invoke-RestMethod -Uri "$BaseURL/api/v1/payments/pay_test_987654321/status" -Method GET -Headers $headers
    Write-Host "✅ Payment status response:" -ForegroundColor Green
    Write-Host ($statusResponse | ConvertTo-Json -Depth 3) -ForegroundColor White
} catch {
    Write-Host "❌ Payment status check failed:" -ForegroundColor Red
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`nPayment Verification Test Completed!" -ForegroundColor Green
Write-Host "Note: This test uses mock data. For real testing, use actual Razorpay credentials and signatures." -ForegroundColor Gray
