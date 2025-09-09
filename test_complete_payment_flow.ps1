# Complete Payment Flow Test Script with Detailed Debugging
# This script tests the complete flow: estimate shipping -> create order -> checkout -> verify payment
# It will capture all the detailed logs we added to identify the AWB/tracking URL issue

Write-Host "=== COMPREHENSIVE PAYMENT FLOW DEBUG TEST ===" -ForegroundColor Green
Write-Host "This script will test the complete payment flow to identify the AWB/tracking URL issue" -ForegroundColor Yellow

# Configuration
$baseUrl = "http://localhost:8080"
$headers = @{
    'Content-Type' = 'application/json'
    'User-Agent' = 'PaymentFlowTest/1.0'
}

# Test user authentication token (you'll need to replace this with a real Firebase token)
$authToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6ImUzZWU3ZTAyOGUzODg1YTM0NWNlMDcwNTVmODQ2ODYyMjU1YTcwNDYiLCJ0eXAiOiJKV1QifQ.eyJuYW1lIjoibWFuaSIsImlzcyI6Imh0dHBzOi8vc2VjdXJldG9rZW4uZ29vZ2xlLmNvbS9hbXJ0aS1mYTk4YiIsImF1ZCI6ImFtcnRpLWZhOThiIiwiYXV0aF90aW1lIjoxNzU3NDEzODE5LCJ1c2VyX2lkIjoiSVNHVmNnYllwd1A4Vlg0bzVNVXc2NUt1eDFHMyIsInN1YiI6IklTR1ZjZ2JZcHdQOFZYNG81TVV3NjVLdXgxRzMiLCJpYXQiOjE3NTc0MTM4MTksImV4cCI6MTc1NzQxNzQxOSwiZW1haWwiOiJtYW5pQGdtYWlsLmNvbSIsImVtYWlsX3ZlcmlmaWVkIjpmYWxzZSwiZmlyZWJhc2UiOnsiaWRlbnRpdGllcyI6eyJlbWFpbCI6WyJtYW5pQGdtYWlsLmNvbSJdfSwic2lnbl9pbl9wcm92aWRlciI6InBhc3N3b3JkIn19.OunfAWS17I3PqQmkU11KX2fohb6hoJcYoPd7ux1f6odXgxYljbBKFP2LI1ZwICNOlZHGcogpSLxpJyj-v94NWpmhYwTokfmfJveeXHjqlYzOhcPCOUBelQAvBVV-kJB1r2REprI6KLjvuO7dO-eR6qGrl4ZTnktpv_YUxJzr1az98J4EUq2G_JM0DR6zCytqzHW1iaFx3UD9rgtLs_JyhEvkFjySO_aFjr7fM0-LwtlM5pxf-bYND5JLZ9gUJAZc7yOIfJ5S8o3j1B6Lalj3JCgQfnWUPDMDG4MLQjIwrux0Mka-P4VS_y-7UEBnxFqzWmVTb2mw8kiKvDFOZ1jyPg"
$headersAuth = $headers.Clone()
$headersAuth['Authorization'] = $authToken

Write-Host "`n1. Testing Server Health..." -ForegroundColor Cyan
try {
    $ping = Invoke-RestMethod -Uri "$baseUrl/ping" -Method GET -Headers $headers
    Write-Host "✓ Server is healthy: $($ping.message)" -ForegroundColor Green
} catch {
    Write-Host "✗ Server health check failed!" -ForegroundColor Red
    exit 1
}

Write-Host "`n2. Testing Delhivery Configuration..." -ForegroundColor Cyan
try {
    $config = Invoke-RestMethod -Uri "$baseUrl/debug/delhivery-config" -Method GET -Headers $headers
    Write-Host "✓ Delhivery Configuration:" -ForegroundColor Green
    Write-Host "  Base URL: $($config.config.base_url)" -ForegroundColor White
    Write-Host "  Production: $($config.config.is_production)" -ForegroundColor White
    Write-Host "  Warehouse: $($config.config.warehouse_name)" -ForegroundColor White
    Write-Host "  Origin Pincode: $($config.config.origin_pincode)" -ForegroundColor White
} catch {
    Write-Host "✗ Delhivery config check failed!" -ForegroundColor Red
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`n3. Testing Shipping Estimation..." -ForegroundColor Cyan
$estimateRequest = @{
    address = @{
        street = "boduppal, east balaji hills , uppal , hyderabad"
        city = "Hyderabad"
        state = "Telangana"
        postal_code = "500092"
        country = "India"
    }
    shipping_method = "standard"
} | ConvertTo-Json -Depth 3

Write-Host "Estimate Request:" -ForegroundColor Yellow
Write-Host $estimateRequest -ForegroundColor White

try {
    $estimate = Invoke-RestMethod -Uri "$baseUrl/api/v1/cart/checkout/estimate-shipping" -Method POST -Body $estimateRequest -Headers $headersAuth
    Write-Host "✓ Shipping Estimation Response:" -ForegroundColor Green
    Write-Host ($estimate | ConvertTo-Json -Depth 3) -ForegroundColor White
} catch {
    $statusCode = $_.Exception.Response.StatusCode.value__
    Write-Host "✗ Shipping estimation failed with status code: $statusCode" -ForegroundColor Red
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    try {
        $errorResponse = $_.ErrorDetails.Message | ConvertFrom-Json
        Write-Host "Error Details: $($errorResponse | ConvertTo-Json)" -ForegroundColor Red
    } catch {
        Write-Host "Could not parse error response" -ForegroundColor Red
    }
}

Write-Host "`n4. Testing Payment Order Creation..." -ForegroundColor Cyan
$createOrderRequest = @{
    first_name = "Mnai"
    last_name = "Mmaidaala"
    mobile = "+919876543210"
    mobile_secondary = ""
    shipping_address_id = "1"  # Replace with actual address ID
} | ConvertTo-Json

Write-Host "Create Order Request:" -ForegroundColor Yellow
Write-Host $createOrderRequest -ForegroundColor White

try {
    $orderResponse = Invoke-RestMethod -Uri "$baseUrl/api/v1/payments/create-order" -Method POST -Body $createOrderRequest -Headers $headersAuth
    Write-Host "✓ Payment Order Creation Response:" -ForegroundColor Green
    Write-Host ($orderResponse | ConvertTo-Json -Depth 3) -ForegroundColor White
    
    $razorpayOrderId = $orderResponse.razorpay_order_id
    $orderIdFromResponse = $orderResponse.order_id
} catch {
    $statusCode = $_.Exception.Response.StatusCode.value__
    Write-Host "✗ Payment order creation failed with status code: $statusCode" -ForegroundColor Red
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    try {
        $errorResponse = $_.ErrorDetails.Message | ConvertFrom-Json
        Write-Host "Error Details: $($errorResponse | ConvertTo-Json)" -ForegroundColor Red
    } catch {
        Write-Host "Could not parse error response" -ForegroundColor Red
    }
    
    # Set dummy values for testing the verify endpoint anyway
    $razorpayOrderId = "order_test123"
    $orderIdFromResponse = "1"
}

Write-Host "`n5. Testing Checkout Page..." -ForegroundColor Cyan
if ($razorpayOrderId) {
    try {
        $checkoutUrl = "$baseUrl/api/v1/payments/checkout?order_id=$razorpayOrderId"
        Write-Host "Checkout URL: $checkoutUrl" -ForegroundColor White
        
        # Just test if the endpoint is accessible (don't expect HTML content in PowerShell)
        $checkoutResponse = Invoke-WebRequest -Uri $checkoutUrl -Method GET -Headers $headers
        Write-Host "✓ Checkout page accessible (Status: $($checkoutResponse.StatusCode))" -ForegroundColor Green
    } catch {
        Write-Host "✗ Checkout page test failed" -ForegroundColor Red
        Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    }
}

Write-Host "`n6. Testing Payment Verification (THIS IS WHERE THE ISSUE IS)..." -ForegroundColor Cyan
Write-Host "This is the critical step where AWB numbers should be generated!" -ForegroundColor Yellow

$verifyRequest = @{
    razorpay_payment_id = "pay_test123456789"
    razorpay_order_id = $razorpayOrderId
    razorpay_signature = "dummy_signature_for_testing"
    order_id = $orderIdFromResponse
} | ConvertTo-Json

Write-Host "Verify Payment Request:" -ForegroundColor Yellow
Write-Host $verifyRequest -ForegroundColor White

try {
    $verifyResponse = Invoke-RestMethod -Uri "$baseUrl/api/v1/payments/verify" -Method POST -Body $verifyRequest -Headers $headersAuth
    Write-Host "✓ Payment Verification Response:" -ForegroundColor Green
    Write-Host ($verifyResponse | ConvertTo-Json -Depth 3) -ForegroundColor White
    
    # Check if AWB and tracking URL are present
    if ($verifyResponse.awb_number -and $verifyResponse.tracking_url) {
        Write-Host "✓ SUCCESS: AWB Number and Tracking URL found!" -ForegroundColor Green
        Write-Host "  AWB Number: $($verifyResponse.awb_number)" -ForegroundColor Green
        Write-Host "  Tracking URL: $($verifyResponse.tracking_url)" -ForegroundColor Green
    } else {
        Write-Host "✗ ISSUE IDENTIFIED: Missing AWB Number or Tracking URL" -ForegroundColor Red
        Write-Host "  AWB Number: $($verifyResponse.awb_number)" -ForegroundColor Red
        Write-Host "  Tracking URL: $($verifyResponse.tracking_url)" -ForegroundColor Red
        Write-Host "  This is the problem we're trying to fix!" -ForegroundColor Red
    }
} catch {
    $statusCode = $_.Exception.Response.StatusCode.value__
    Write-Host "✗ Payment verification failed with status code: $statusCode" -ForegroundColor Red
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    try {
        $errorResponse = $_.ErrorDetails.Message | ConvertFrom-Json
        Write-Host "Error Details: $($errorResponse | ConvertTo-Json)" -ForegroundColor Red
    } catch {
        Write-Host "Could not parse error response" -ForegroundColor Red
    }
}

Write-Host "`n=== DEBUGGING INSTRUCTIONS ===" -ForegroundColor Yellow
Write-Host "Now check your server logs for the following patterns:" -ForegroundColor Cyan
Write-Host "1. Look for '=== STARTING DELHIVERY SHIPMENT CREATION FLOW ===' logs" -ForegroundColor White
Write-Host "2. Check for 'SHIPMENT CREATION SERVICE RESPONSE' logs" -ForegroundColor White
Write-Host "3. Look for any 'DELHIVERY API RETURNED FAILURE STATUS' errors" -ForegroundColor White
Write-Host "4. Check for 'EMPTY WAYBILL RECEIVED FROM DELHIVERY' errors" -ForegroundColor White
Write-Host "5. Look for 'DELHIVERY SHIPMENT CREATED SUCCESSFULLY' success logs" -ForegroundColor White

Write-Host "`n=== NEXT STEPS ===" -ForegroundColor Yellow
Write-Host "1. If you see authentication errors, replace YOUR_FIREBASE_TOKEN_HERE with a real token" -ForegroundColor White
Write-Host "2. Replace shipping_address_id with a real address ID from your database" -ForegroundColor White
Write-Host "3. Check server logs to see exactly where the Delhivery integration is failing" -ForegroundColor White
Write-Host "4. The detailed logging will show you if it's a config issue, API issue, or data processing issue" -ForegroundColor White

Write-Host "`nTest completed! Check the server logs for detailed debugging information." -ForegroundColor Green
