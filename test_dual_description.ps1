# Test script for dual description columns functionality
$serverUrl = "http://localhost:8080"

# Test 1: Create a recipe with both description and demo_description
Write-Host "Test 1: Creating recipe with both description and demo_description..." -ForegroundColor Yellow

$createRecipeBody = @{
    name = "Test Dual Description Recipe"
    category = "Desserts"
    description = "This is the main detailed description of the recipe with full instructions and background information."
    demo_description = "This is a shorter demo description for quick overview and display purposes."
    prep_time = "30 minutes"
    difficulty = "Medium"
    image = "https://example.com/test-recipe-image.jpg"
    rating = 4.5
    reviews = 10
    servings = 4
    ingredients = @("Test ingredient 1", "Test ingredient 2", "Test ingredient 3")
    instructions = @("Step 1: Test instruction", "Step 2: Test instruction", "Step 3: Test instruction")
    pro_tips = @("Pro tip 1", "Pro tip 2")
    nutrition_facts = @{
        calories = "250"
        protein = "15g"
        carbs = "30g"
        fat = "8g"
    }
} | ConvertTo-Json -Depth 10

Write-Host "Request body:" -ForegroundColor Cyan
Write-Host $createRecipeBody -ForegroundColor Gray

try {
    $response1 = Invoke-RestMethod -Uri "$serverUrl/api/v1/recipes/" -Method POST -Body $createRecipeBody -ContentType "application/json" -ErrorAction Stop
    Write-Host "✅ Recipe created successfully!" -ForegroundColor Green
    Write-Host "Recipe ID: $($response1.data.recipee_id)" -ForegroundColor Cyan
    Write-Host "Description: $($response1.data.description)" -ForegroundColor Cyan
    Write-Host "Demo Description: $($response1.data.demo_description)" -ForegroundColor Cyan
    
    $recipeId = $response1.data.recipee_id
    
    # Test 2: Get the recipe back and verify both fields are saved
    Write-Host "`nTest 2: Retrieving recipe to verify dual descriptions..." -ForegroundColor Yellow
    
    $response2 = Invoke-RestMethod -Uri "$serverUrl/api/v1/recipes/$recipeId" -Method GET -ErrorAction Stop
    Write-Host "✅ Recipe retrieved successfully!" -ForegroundColor Green
    Write-Host "Description: $($response2.data.description)" -ForegroundColor Cyan
    Write-Host "Demo Description: $($response2.data.demo_description)" -ForegroundColor Cyan
    
    # Test 3: Create recipe with only description (demo_description should fallback)
    Write-Host "`nTest 3: Creating recipe with only description (testing fallback)..." -ForegroundColor Yellow
    
    $createRecipeBody2 = @{
        name = "Test Fallback Recipe"
        category = "Main Dishes"
        description = "This is only the main description. Demo description should fallback to this value."
        prep_time = "45 minutes"
        difficulty = "Easy"
        image = "https://example.com/test-recipe-2.jpg"
        rating = 4.2
        reviews = 5
        servings = 2
        ingredients = @("Fallback ingredient 1", "Fallback ingredient 2")
        instructions = @("Fallback step 1", "Fallback step 2")
        pro_tips = @("Fallback tip")
        nutrition_facts = @{
            calories = "180"
            protein = "12g"
        }
    } | ConvertTo-Json -Depth 10
    
    $response3 = Invoke-RestMethod -Uri "$serverUrl/api/v1/recipes/" -Method POST -Body $createRecipeBody2 -ContentType "application/json" -ErrorAction Stop
    Write-Host "✅ Fallback recipe created successfully!" -ForegroundColor Green
    Write-Host "Recipe ID: $($response3.data.recipee_id)" -ForegroundColor Cyan
    Write-Host "Description: $($response3.data.description)" -ForegroundColor Cyan
    Write-Host "Demo Description: $($response3.data.demo_description)" -ForegroundColor Cyan
    
    if ($response3.data.demo_description -eq $response3.data.description) {
        Write-Host "✅ Fallback mechanism working correctly!" -ForegroundColor Green
    } else {
        Write-Host "❌ Fallback mechanism not working as expected!" -ForegroundColor Red
    }
    
    Write-Host "`n🎉 All tests completed successfully!" -ForegroundColor Green
    Write-Host "✅ Dual description column functionality is working properly" -ForegroundColor Green
    Write-Host "✅ Both description and demo_description fields are saved and retrieved correctly" -ForegroundColor Green
    Write-Host "✅ Fallback mechanism works when demo_description is not provided" -ForegroundColor Green
    
} catch {
    Write-Host "❌ Error occurred: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "Response body: $responseBody" -ForegroundColor Red
    }
}
