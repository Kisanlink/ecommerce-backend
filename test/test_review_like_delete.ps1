# Test Review Like/Unlike with correct review ID
$baseUrl = "http://localhost:8080"

# Get a valid Firebase token first - you'll need to authenticate
# For testing purposes, let's see what happens without auth (should get 401)
$headers = @{
    "Content-Type" = "application/json"
}

$productId = "PROD00000005"
$reviewId = "REV1757340665267877700"  # Using the actual review ID from the response

Write-Host "=== Testing Like Review (without auth - should fail) ===" -ForegroundColor Yellow
$likeData = @{
    action = "like"
} | ConvertTo-Json

try {
    $likeResponse = Invoke-RestMethod -Uri "$baseUrl/api/v1/products/$productId/reviews/$reviewId/like" -Method POST -Headers $headers -Body $likeData
    Write-Host "Like Response:" -ForegroundColor Green
    $likeResponse | ConvertTo-Json -Depth 3
} catch {
    Write-Host "Expected error liking review without auth: $($_.Exception.Message)" -ForegroundColor Yellow
    Write-Host "Response: $($_.ErrorDetails.Message)" -ForegroundColor Yellow
}

Write-Host "`n=== Testing Delete Review (without auth - should fail) ===" -ForegroundColor Yellow
try {
    $deleteResponse = Invoke-RestMethod -Uri "$baseUrl/api/v1/products/$productId/reviews/$reviewId" -Method DELETE -Headers $headers
    Write-Host "Delete Response:" -ForegroundColor Green
    $deleteResponse | ConvertTo-Json -Depth 3
} catch {
    Write-Host "Expected error deleting review without auth: $($_.Exception.Message)" -ForegroundColor Yellow
    Write-Host "Response: $($_.ErrorDetails.Message)" -ForegroundColor Yellow
}

Write-Host "`nActual Review ID that exists: $reviewId" -ForegroundColor Cyan
Write-Host "Review ID you were trying: EV1757342645891782000 (doesn't exist)" -ForegroundColor Red
