# Test Review Functionality
$baseUrl = "http://localhost:8080"

# You'll need to get a valid Firebase token - for now testing without auth to see the errors
$headers = @{
    "Content-Type" = "application/json"
}

# Test getting reviews for a product
Write-Host "=== Testing Get Reviews ===" -ForegroundColor Yellow
$productId = "PROD00000004"
try {
    $reviewsResponse = Invoke-RestMethod -Uri "$baseUrl/api/v1/products/$productId/reviews" -Method GET -Headers $headers
    Write-Host "Reviews Response:" -ForegroundColor Green
    $reviewsResponse | ConvertTo-Json -Depth 3
} catch {
    Write-Host "Error getting reviews: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "Response: $($_.ErrorDetails.Message)" -ForegroundColor Red
}

Write-Host "`n=== Testing Create Review (without auth - should fail) ===" -ForegroundColor Yellow
$reviewData = @{
    rating = 5
    comment = "Test review for testing like functionality"
} | ConvertTo-Json

try {
    $createResponse = Invoke-RestMethod -Uri "$baseUrl/api/v1/products/$productId/reviews/" -Method POST -Headers $headers -Body $reviewData
    Write-Host "Create Response:" -ForegroundColor Green
    $createResponse | ConvertTo-Json -Depth 3
} catch {
    Write-Host "Expected error creating review without auth: $($_.Exception.Message)" -ForegroundColor Yellow
    Write-Host "Response: $($_.ErrorDetails.Message)" -ForegroundColor Yellow
}

# If you have a valid Firebase token, replace "your-firebase-token-here" with the actual token
Write-Host "`n=== To test with authentication, get a Firebase token and run: ===" -ForegroundColor Cyan
Write-Host '$headers = @{' -ForegroundColor Cyan
Write-Host '    "Content-Type" = "application/json"' -ForegroundColor Cyan
Write-Host '    "Authorization" = "Bearer your-firebase-token-here"' -ForegroundColor Cyan
Write-Host '}' -ForegroundColor Cyan
