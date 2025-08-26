# Product API Test Script
# This script tests all product CRUD operations with proper authentication

$baseUrl = "http://localhost:8080/api/v1"
$headers = @{
    "Content-Type" = "application/json"
}

Write-Host "=== Product API Testing ==="

# 1. Test GET all products (public endpoint)
Write-Host "`n1. Testing GET /products (public)"
try {
    $response = Invoke-RestMethod -Uri "$baseUrl/products" -Method GET -Headers $headers
    Write-Host "✅ GET all products successful"
    Write-Host "Total products: $($response.data.Count)"
} catch {
    Write-Host "❌ GET all products failed: $($_.Exception.Message)"
}

# 2. Test GET product by ID (public endpoint)
Write-Host "`n2. Testing GET /products/:id (public)"
try {
    # Get the first product ID from the previous response
    $products = Invoke-RestMethod -Uri "$baseUrl/products" -Method GET -Headers $headers
    if ($products.data -and $products.data.Count -gt 0) {
        $firstProductId = $products.data[0].id
        $product = Invoke-RestMethod -Uri "$baseUrl/products/$firstProductId" -Method GET -Headers $headers
        Write-Host "✅ GET product by ID successful"
        Write-Host "Product: $($product.data.name)"
    } else {
        Write-Host "⚠️  No products found to test GET by ID"
    }
} catch {
    Write-Host "❌ GET product by ID failed: $($_.Exception.Message)"
}

# 3. Test search products (public endpoint)
Write-Host "`n3. Testing GET /products/search (public)"
try {
    $searchResponse = Invoke-RestMethod -Uri "$baseUrl/products/search?q=powder" -Method GET -Headers $headers
    Write-Host "✅ Search products successful"
    Write-Host "Search results: $($searchResponse.data.Count)"
} catch {
    Write-Host "❌ Search products failed: $($_.Exception.Message)"
}

# 4. Test products by category (public endpoint)
Write-Host "`n4. Testing GET /products/category/:category (public)"
try {
    $categoryResponse = Invoke-RestMethod -Uri "$baseUrl/products/category/health" -Method GET -Headers $headers
    Write-Host "✅ Get products by category successful"
    Write-Host "Category products: $($categoryResponse.data.Count)"
} catch {
    Write-Host "❌ Get products by category failed: $($_.Exception.Message)"
}

# 5. Test CREATE product without authentication (should fail)
Write-Host "`n5. Testing POST /products without auth (should fail)"
$newProduct = @{
    name = "Test Product"
    description = "A test product"
    category = "test"
    price = 29.99
    actual_price = 39.99
    discount_percent = 25
    stock = 100
    review_count = 0
    rating = 0.0
} | ConvertTo-Json

try {
    $createResponse = Invoke-RestMethod -Uri "$baseUrl/products" -Method POST -Body $newProduct -Headers $headers
    Write-Host "❌ CREATE without auth should have failed but succeeded"
} catch {
    if ($_.Exception.Response.StatusCode -eq 401) {
        Write-Host "✅ CREATE without auth correctly failed with 401 Unauthorized"
    } else {
        Write-Host "⚠️  CREATE without auth failed with unexpected status: $($_.Exception.Response.StatusCode)"
    }
}

# 6. Test UPDATE product without authentication (should fail)
Write-Host "`n6. Testing PUT /products/:id without auth (should fail)"
try {
    if ($products.data -and $products.data.Count -gt 0) {
        $firstProductId = $products.data[0].id
        $updateProduct = @{
            name = "Updated Test Product"
            description = "Updated description"
            category = "test"
            price = 34.99
            actual_price = 44.99
            discount_percent = 22
            stock = 50
            review_count = 5
            rating = 4.5
        } | ConvertTo-Json

        $updateResponse = Invoke-RestMethod -Uri "$baseUrl/products/$firstProductId" -Method PUT -Body $updateProduct -Headers $headers
        Write-Host "❌ UPDATE without auth should have failed but succeeded"
    }
} catch {
    if ($_.Exception.Response.StatusCode -eq 401) {
        Write-Host "✅ UPDATE without auth correctly failed with 401 Unauthorized"
    } else {
        Write-Host "⚠️  UPDATE without auth failed with unexpected status: $($_.Exception.Response.StatusCode)"
    }
}

# 7. Test DELETE product without authentication (should fail)
Write-Host "`n7. Testing DELETE /products/:id without auth (should fail)"
try {
    if ($products.data -and $products.data.Count -gt 0) {
        $firstProductId = $products.data[0].id
        $deleteResponse = Invoke-RestMethod -Uri "$baseUrl/products/$firstProductId" -Method DELETE -Headers $headers
        Write-Host "❌ DELETE without auth should have failed but succeeded"
    }
} catch {
    if ($_.Exception.Response.StatusCode -eq 401) {
        Write-Host "✅ DELETE without auth correctly failed with 401 Unauthorized"
    } else {
        Write-Host "⚠️  DELETE without auth failed with unexpected status: $($_.Exception.Response.StatusCode)"
    }
}

Write-Host "`n=== Test Summary ==="
Write-Host "✅ Public endpoints (GET operations) should work"
Write-Host "🔒 Protected endpoints (POST/PUT/DELETE) should require admin authentication"
Write-Host "`nTo test authenticated operations:"
Write-Host "1. Sign up/login to get a Firebase token"
Write-Host "2. Ensure your user has 'admin' role in the database"
Write-Host "3. Add 'Authorization: Bearer [token]' header to requests"
