# Test Review Functionality with Correct Review ID
$baseUrl = "http://localhost:8080"

# Test data
$productId = "PROD00000005"  
$validReviewId = "REV1757340665267877700"  # This is the actual review that exists
$invalidReviewId = "EV1757342645891782000"  # This is what you were trying (doesn't exist)

Write-Host "=== Testing with INVALID Review ID (should fail) ===" -ForegroundColor Red
Write-Host "URL: $baseUrl/api/v1/products/$productId/reviews/$invalidReviewId/like" -ForegroundColor Red

$headers = @{ "Content-Type" = "application/json" }
$likeData = @{ action = "like" } | ConvertTo-Json

try {
    $response = Invoke-RestMethod -Uri "$baseUrl/api/v1/products/$productId/reviews/$invalidReviewId/like" -Method POST -Headers $headers -Body $likeData
    Write-Host "Unexpected success:" -ForegroundColor Red
    $response | ConvertTo-Json
} catch {
    Write-Host "Expected Error:" -ForegroundColor Yellow
    Write-Host "Status: $($_.Exception.Message)" -ForegroundColor Yellow
    if ($_.ErrorDetails -and $_.ErrorDetails.Message) {
        Write-Host "Details: $($_.ErrorDetails.Message)" -ForegroundColor Yellow
    }
}

Write-Host "`n=== Testing with VALID Review ID (should work if authenticated) ===" -ForegroundColor Green
Write-Host "URL: $baseUrl/api/v1/products/$productId/reviews/$validReviewId/like" -ForegroundColor Green

try {
    $response = Invoke-RestMethod -Uri "$baseUrl/api/v1/products/$productId/reviews/$validReviewId/like" -Method POST -Headers $headers -Body $likeData
    Write-Host "Unexpected success (should need auth):" -ForegroundColor Red
    $response | ConvertTo-Json
} catch {
    Write-Host "Expected Auth Error:" -ForegroundColor Yellow
    Write-Host "Status: $($_.Exception.Message)" -ForegroundColor Yellow
    if ($_.ErrorDetails -and $_.ErrorDetails.Message) {
        Write-Host "Details: $($_.ErrorDetails.Message)" -ForegroundColor Yellow
    }
}

Write-Host "`n=== Summary ===" -ForegroundColor Cyan
Write-Host "✓ Invalid Review ID: $invalidReviewId (doesn't exist)" -ForegroundColor Red
Write-Host "✓ Valid Review ID: $validReviewId (exists)" -ForegroundColor Green
Write-Host "✓ Both should fail due to missing authentication" -ForegroundColor Yellow
Write-Host "`n📝 To test with authentication:" -ForegroundColor Cyan
Write-Host "1. Get a Firebase JWT token" -ForegroundColor Cyan
Write-Host "2. Add it to headers: 'Authorization' = 'Bearer YOUR_TOKEN'" -ForegroundColor Cyan
Write-Host "3. Use the valid review ID: $validReviewId" -ForegroundColor Cyan
