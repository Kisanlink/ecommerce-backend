# Quick test to verify the order item ID generation fix
Write-Host "Testing Order Item ID Generation Fix..." -ForegroundColor Green

# Test payment verification with proper headers
$headers = @{
    "Content-Type" = "application/json"
    "Authorization" = "Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6IjBiYTA3OGM4NjI3Y2Y2YTYzODEzMTgyZWViZjE5ZThlYjIwMjI5OGQiLCJ0eXAiOiJKV1QifQ.eyJpc3MiOiJodHRwczovL3NlY3VyZXRva2VuLmdvb2dsZS5jb20va2lzYW5saW5rLWVjb21tZXJjZS1kZXYiLCJhdWQiOiJraXNhbmxpbmstZWNvbW1lcmNlLWRldiIsImF1dGhfdGltZSI6MTczNzk3NjUyOCwidXNlcl9pZCI6InlBTExBQkNWNmxodFNRZHhCdGQ2dVl6YUEySzIiLCJzdWIiOiJ5QUxMQUJDVjZsaHRTUWR4QnRkNnVZemFBMksyIiwiaWF0IjoxNzM3OTc2NTI4LCJleHAiOjE3Mzc5ODAxMjgsImVtYWlsIjoibWFuaS5tYW1pZGFsYS5kZXZAZ21haWwuY29tIiwiZW1haWxfdmVyaWZpZWQiOnRydWUsImZpcmViYXNlIjp7ImlkZW50aXRpZXMiOnsiZW1haWwiOlsibWFuaS5tYW1pZGFsYS5kZXZAZ21haWwuY29tIl19LCJzaWduX2luX3Byb3ZpZGVyIjoicGFzc3dvcmQifX0.VztUvCK0vNEkcJrAoWNVuOvEPFh1kUXVfIcA4EfEgT1Dt_9SBNGelC1yCxQYfhGvkqKmSSfKAY5sPgXnpFOa8HvmLjhYXiUZnb_QLjbP4RzAkKNT8BPDqSk3U_-vY-1jfQnHPjkFIoqYEfOl7__5WFMCBSE9WZh_2e4H1XjOBV9XMhGXvLrN9r_JX4mfj2f4Ssk2dDfxL7gYXE_5b-NdwvIJhm7fAMK9-8zhjXWRTK8S7HpWlY3UdLY9_kfC9Wf3w5RhLZ2VwH9oL8f1xY2rN8g5Dz2ZnEd4KM_GgJ3vY2hU8qNK9oPl3wF7xQ5VdCf2zH1YgG9oKm3vPfW2nX5A"
}

$paymentData = @{
    razorpay_order_id = "order_NEW_TEST_" + (Get-Random -Minimum 100000 -Maximum 999999)
    razorpay_payment_id = "pay_NEW_TEST_" + (Get-Random -Minimum 100000 -Maximum 999999)  
    razorpay_signature = "test_signature_new_order_item_test"
    shipping_address_id = "1"
} | ConvertTo-Json

Write-Host "Testing payment verification with data: $paymentData" -ForegroundColor Yellow

try {
    $response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/payments/verify" -Method POST -Body $paymentData -Headers $headers -TimeoutSec 30
    Write-Host "✅ Payment verification successful!" -ForegroundColor Green
    Write-Host "Response: " ($response | ConvertTo-Json -Depth 3) -ForegroundColor White
} catch {
    Write-Host "❌ Payment verification failed:" -ForegroundColor Red
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.ErrorDetails) {
        Write-Host "Details: $($_.ErrorDetails.Message)" -ForegroundColor Red
    }
}

Write-Host "`nOrder Item ID Generation Fix Test Completed!" -ForegroundColor Green
