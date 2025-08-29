# Test script for favorites API
$baseUrl = "http://localhost:8080"
$token = "Bearer zg49P9SVhsghkiSueEldz4BVKUM2"

# Test adding a favorite
$addFavoriteBody = @{
    product_id = "PROD00000001"
} | ConvertTo-Json

$headers = @{
    "Authorization" = $token
    "Content-Type" = "application/json"
}

Write-Host "Testing Add Favorite..."
$response = try {
    Invoke-RestMethod -Uri "$baseUrl/api/v1/favorites" -Method POST -Body $addFavoriteBody -Headers $headers
} catch {
    $_.Exception.Response.StatusCode
    $_ | Out-String
}

Write-Host "Response: $response"
