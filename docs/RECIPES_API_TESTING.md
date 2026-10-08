# Recipes API Testing Guide

This document provides comprehensive testing instructions for the Recipes API endpoints.

## Table of Contents
1. [Setup](#setup)
2. [API Endpoints](#api-endpoints)
3. [Testing Examples](#testing-examples)
4. [Sample Requests](#sample-requests)

## Setup

### 1. Run Migration Script

First, create the recipes table and seed initial data:

**Linux/Mac:**
```bash
chmod +x migrations/run_recipes_migration.sh
./migrations/run_recipes_migration.sh "your-neondb-connection-string"
```

**Windows (PowerShell):**
```powershell
.\migrations\run_recipes_migration.ps1 -DatabaseUrl "your-neondb-connection-string"
```

**Or manually:**
```bash
psql "your-neondb-connection-string" -f migrations/create_recipes_table.sql
psql "your-neondb-connection-string" -f migrations/seed_recipes.sql
```

### 2. Start the Server

```bash
go run cmd/server/main.go
```

The server should start on `http://localhost:8080` (or your configured port).

## API Endpoints

### Base URL
```
http://localhost:8080/api/v1
```

### Public Endpoints (No Authentication Required)

#### 1. Get All Recipes
**GET** `/recipes`

Query Parameters:
- `page` (optional): Page number (default: 1)
- `page_size` (optional): Items per page (default: 10, max: 100)
- `category` (optional): Filter by category (e.g., "Breakfast", "Dessert", "Beverage")
- `search` (optional): Search term for name/description
- `ingredient` (optional): Filter by ingredient name

**Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/recipes?page=1&page_size=10"
```

**Response:**
```json
{
  "recipes": [
    {
      "id": "RECP00000001",
      "name": "Dosa Moringa",
      "demo_description": "A healthy South Indian dosa with moringa leaves - perfect for breakfast!",
      "reviews": 12,
      "rating": 4.5,
      "prep_time": "30 mins",
      "difficulty": "Medium",
      "image": "https://your-s3-bucket.s3.amazonaws.com/recipes/dosa-moringa.jpg"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 10,
    "total": 4,
    "total_pages": 1
  }
}
```

#### 2. Get Recipe by ID
**GET** `/recipes/:id`

**Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/recipes/RECP00000001"
```

**Response:**
```json
{
  "recipe": {
    "id": "RECP00000001",
    "recipe_id": "RECP00000001",
    "name": "Dosa Moringa",
    "category": "Breakfast",
    "description": "A nutritious and delicious South Indian dosa...",
    "demo_description": "A healthy South Indian dosa with moringa leaves...",
    "prep_time": "30 mins",
    "servings": 4,
    "difficulty": "Medium",
    "image": "https://your-s3-bucket.s3.amazonaws.com/recipes/dosa-moringa.jpg",
    "ingredients": [
      "1 cup rice",
      "1/2 cup urad dal",
      "1/4 cup moringa leaves (fresh)",
      ...
    ],
    "instructions": [
      "Soak rice and urad dal separately for 4-6 hours",
      "Grind rice and dal separately to a smooth batter",
      ...
    ],
    "nutrition_facts": {
      "calories": "180",
      "protein": "6g",
      "carbs": "32g",
      ...
    },
    "pro_tips": [
      "Add moringa leaves just before fermenting for best flavor",
      ...
    ],
    "rating": 4.5,
    "reviews": 12,
    "created_at": "2025-01-XX...",
    "updated_at": "2025-01-XX..."
  }
}
```

#### 3. Get Recipes by Category
**GET** `/recipes/category/:category`

**Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/recipes/category/Breakfast"
```

#### 4. Search Recipes
**GET** `/recipes?search=moringa`

**Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/recipes?search=moringa"
```

#### 5. Filter by Ingredient
**GET** `/recipes?ingredient=banana`

**Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/recipes?ingredient=banana"
```

### User Endpoints (Authentication Required)

#### 6. Submit Recipe for Review
**POST** `/recipes/submit`

**Headers:**
```
Authorization: Bearer <firebase-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "New Moringa Recipe",
  "category": "Breakfast",
  "description": "A delicious new recipe...",
  "prep_time": "20 mins",
  "servings": 2,
  "difficulty": "Easy",
  "ingredients": [
    "1 cup ingredient1",
    "2 tbsp ingredient2"
  ],
  "instructions": [
    "Step 1: Do this",
    "Step 2: Do that"
  ],
  "nutrition_facts": {
    "calories": "150",
    "protein": "5g"
  },
  "pro_tips": [
    "Tip 1",
    "Tip 2"
  ]
}
```

**Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/recipes/submit" \
  -H "Authorization: Bearer YOUR_FIREBASE_TOKEN" \
  -H "Content-Type: application/json" \
  -d @recipe_submission.json
```

#### 7. Get User's Submitted Recipes
**GET** `/recipes/my-submissions`

**Headers:**
```
Authorization: Bearer <firebase-token>
```

**Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/recipes/my-submissions" \
  -H "Authorization: Bearer YOUR_FIREBASE_TOKEN"
```

#### 8. Upload Recipe Image
**POST** `/recipes/images/upload`

**Headers:**
```
Authorization: Bearer <firebase-token>
Content-Type: multipart/form-data
```

**Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/recipes/images/upload" \
  -H "Authorization: Bearer YOUR_FIREBASE_TOKEN" \
  -F "image=@/path/to/image.jpg"
```

#### 9. Delete Recipe Image
**DELETE** `/recipes/images/:s3_key`

**Headers:**
```
Authorization: Bearer <firebase-token>
```

**Example:**
```bash
curl -X DELETE "http://localhost:8080/api/v1/recipes/images/recipes/image-key.jpg" \
  -H "Authorization: Bearer YOUR_FIREBASE_TOKEN"
```

### Admin Endpoints (Authentication + Admin Role Required)

#### 10. Create Recipe (Admin Only)
**POST** `/recipes`

**Headers:**
```
Authorization: Bearer <firebase-admin-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "name": "Admin Created Recipe",
  "category": "Dessert",
  "description": "Full description...",
  "demo_description": "Short description",
  "prep_time": "45 mins",
  "servings": 6,
  "difficulty": "Medium",
  "image": "https://s3-url.com/image.jpg",
  "ingredients": ["ingredient1", "ingredient2"],
  "instructions": ["step1", "step2"],
  "nutrition_facts": {
    "calories": "200",
    "protein": "8g"
  },
  "pro_tips": ["tip1", "tip2"],
  "rating": 0,
  "reviews": 0
}
```

**Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/recipes" \
  -H "Authorization: Bearer ADMIN_FIREBASE_TOKEN" \
  -H "Content-Type: application/json" \
  -d @new_recipe.json
```

#### 11. Update Recipe (Admin Only)
**PUT** `/recipes/:id`

**Headers:**
```
Authorization: Bearer <firebase-admin-token>
Content-Type: application/json
```

**Request Body (partial update supported):**
```json
{
  "name": "Updated Recipe Name",
  "rating": 4.8,
  "reviews": 25
}
```

**Example:**
```bash
curl -X PUT "http://localhost:8080/api/v1/recipes/RECP00000001" \
  -H "Authorization: Bearer ADMIN_FIREBASE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Updated Name", "rating": 4.8}'
```

#### 12. Delete Recipe (Admin Only)
**DELETE** `/recipes/:id`

**Headers:**
```
Authorization: Bearer <firebase-admin-token>
```

**Example:**
```bash
curl -X DELETE "http://localhost:8080/api/v1/recipes/RECP00000001" \
  -H "Authorization: Bearer ADMIN_FIREBASE_TOKEN"
```

### Admin Recipe Review Endpoints

#### 13. Get Pending Recipe Reviews
**GET** `/admin/recipes/pending`

**Headers:**
```
Authorization: Bearer <firebase-admin-token>
```

**Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/admin/recipes/pending" \
  -H "Authorization: Bearer ADMIN_FIREBASE_TOKEN"
```

#### 14. Get Pending Review by ID
**GET** `/admin/recipes/pending/:id`

**Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/admin/recipes/pending/REV123" \
  -H "Authorization: Bearer ADMIN_FIREBASE_TOKEN"
```

#### 15. Approve Recipe Review
**POST** `/admin/recipes/pending/:id/approve`

**Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/admin/recipes/pending/REV123/approve" \
  -H "Authorization: Bearer ADMIN_FIREBASE_TOKEN"
```

#### 16. Reject Recipe Review
**POST** `/admin/recipes/pending/:id/reject`

**Request Body:**
```json
{
  "reason": "Recipe doesn't meet quality standards"
}
```

**Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/admin/recipes/pending/REV123/reject" \
  -H "Authorization: Bearer ADMIN_FIREBASE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"reason": "Not suitable"}'
```

## Testing Examples

### Test 1: Get All Recipes
```bash
curl -X GET "http://localhost:8080/api/v1/recipes"
```

### Test 2: Get Recipe by ID
```bash
curl -X GET "http://localhost:8080/api/v1/recipes/RECP00000001"
```

### Test 3: Filter by Category
```bash
curl -X GET "http://localhost:8080/api/v1/recipes?category=Breakfast"
```

### Test 4: Search Recipes
```bash
curl -X GET "http://localhost:8080/api/v1/recipes?search=dosa"
```

### Test 5: Filter by Ingredient
```bash
curl -X GET "http://localhost:8080/api/v1/recipes?ingredient=moringa"
```

### Test 6: Pagination
```bash
curl -X GET "http://localhost:8080/api/v1/recipes?page=1&page_size=2"
```

## Seeded Recipes

After running the migration, you'll have 4 recipes:

1. **RECP00000001** - Dosa Moringa (Breakfast)
2. **RECP00000002** - Moringa Dessert Cake (Dessert)
3. **RECP00000003** - Moringa Smoothie (Beverage)
4. **RECP00000004** - Moringa Tea (Beverage)

## Postman Collection

You can import these endpoints into Postman:

1. Create a new collection: "Recipes API"
2. Set base URL variable: `{{base_url}}` = `http://localhost:8080/api/v1`
3. Set auth token variable: `{{auth_token}}` = your Firebase token
4. Add all endpoints above as requests

## Error Responses

### 400 Bad Request
```json
{
  "error": "Recipe ID is required"
}
```

### 404 Not Found
```json
{
  "error": "Recipe not found",
  "details": "..."
}
```

### 401 Unauthorized
```json
{
  "error": "Authentication required"
}
```

### 403 Forbidden
```json
{
  "error": "Admin access required"
}
```

### 500 Internal Server Error
```json
{
  "error": "Failed to fetch recipes",
  "details": "..."
}
```

## Notes

- All timestamps are in UTC
- Image URLs should point to your S3 bucket or CDN
- Nutrition facts are stored as JSONB for flexibility
- Ingredients and instructions are stored as PostgreSQL arrays
- The `recipe_id` field is the same as `id` for backward compatibility
- Soft deletes are supported via `deleted_at` field

