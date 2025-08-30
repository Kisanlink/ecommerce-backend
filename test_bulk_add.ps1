# PowerShell script to test bulk cart addition
# This will test the enhanced bulk add functionality with duplicate detection

$baseUrl = "http://localhost:8080"

# First, get a test token (you may need to adjust this based on your auth system)
Write-Host "Testing bulk cart addition with duplicate detection..."

# Test data with some duplicates within the request
$testData = @{
    items = @(
        @{
            product_id = "PROD0000000001"  # First product
            quantity = 2
        },
        @{
            product_id = "PROD0000000002"  # Second product
            quantity = 1
        },
        @{
            product_id = "PROD0000000001"  # Duplicate - should be consolidated
            quantity = 3
        }
    )
} | ConvertTo-Json -Depth 3

Write-Host "Test data with duplicates:"
Write-Host $testData

# You'll need to add proper authentication header here
$headers = @{
    "Content-Type" = "application/json"
    # "Authorization" = "Bearer YOUR_JWT_TOKEN"
}

try {
    $response = Invoke-RestMethod -Uri "$baseUrl/api/v1/cart/bulk" -Method POST -Body $testData -Headers $headers
    Write-Host "Success! Bulk add response:"
    $response | ConvertTo-Json -Depth 5
} catch {
    Write-Host "Error occurred:"
    $_.Exception.Message
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseText = $reader.ReadToEnd()
        Write-Host "Error response: $responseText"
    }
}
