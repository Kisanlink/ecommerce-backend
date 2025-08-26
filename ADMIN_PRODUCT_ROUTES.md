# Admin Product API Routes - Complete Guide

## 🔐 Authentication Required
All admin routes require:
- **Firebase Authentication Token**
- **User must have `role = 'admin'` in the database**

## Admin Product Routes

### 1. CREATE Product (Admin Only)
```http
POST http://localhost:8080/api/v1/products
```

**Headers:**
```
Content-Type: application/json
Authorization: Bearer YOUR_FIREBASE_TOKEN
```

**JSON Body:**
```json
{
    "name": "Premium Organic Turmeric",
    "description": "High-quality organic turmeric powder with 95% curcumin content. Perfect for cooking, teas, and health supplements.",
    "category": "Spices",
    "price": 299.99,
    "actual_price": 399.99,
    "discount_percent": 25,
    "image_url": "https://example.com/images/premium-turmeric.jpg",
    "stock": 150,
    "review_count": 0,
    "rating": 0.0
}
```

**Success Response (201):**
```json
{
    "success": true,
    "message": "Product created successfully",
    "data": {
        "id": "PROD00000011",
        "created_at": "2025-08-26T23:45:12.123456+05:30",
        "updated_at": "2025-08-26T23:45:12.123456+05:30",
        "created_by": "admin_user_id",
        "updated_by": "",
        "name": "Premium Organic Turmeric",
        "description": "High-quality organic turmeric powder...",
        "category": "Spices",
        "price": 299.99,
        "actual_price": 399.99,
        "discount_percent": 25,
        "image_url": "https://example.com/images/premium-turmeric.jpg",
        "stock": 150,
        "stock_status": "In Stock",
        "review_count": 0,
        "rating": 0.0,
        "is_active": true
    },
    "timestamp": "2025-08-26T23:45:12.456789+05:30"
}
```

### 2. UPDATE Product (Admin Only)
```http
PUT http://localhost:8080/api/v1/products/PROD00000001
```

**Headers:**
```
Content-Type: application/json
Authorization: Bearer YOUR_FIREBASE_TOKEN
```

**JSON Body:**
```json
{
    "name": "Updated Premium Turmeric Powder",
    "description": "Enhanced organic turmeric powder with 98% curcumin content. Laboratory tested for purity and potency.",
    "category": "Superfoods",
    "price": 349.99,
    "actual_price": 449.99,
    "discount_percent": 22,
    "image_url": "https://example.com/images/updated-turmeric.jpg",
    "stock": 125,
    "review_count": 45,
    "rating": 4.8
}
```

**Success Response (200):**
```json
{
    "success": true,
    "message": "Product updated successfully",
    "data": {
        "id": "PROD00000001",
        "created_at": "2025-08-26T19:58:45.501428+05:30",
        "updated_at": "2025-08-26T23:47:30.789012+05:30",
        "created_by": "system",
        "updated_by": "admin_user_id",
        "name": "Updated Premium Turmeric Powder",
        "description": "Enhanced organic turmeric powder...",
        "category": "Superfoods",
        "price": 349.99,
        "actual_price": 449.99,
        "discount_percent": 22,
        "image_url": "https://example.com/images/updated-turmeric.jpg",
        "stock": 125,
        "stock_status": "In Stock",
        "review_count": 45,
        "rating": 4.8,
        "is_active": true
    },
    "timestamp": "2025-08-26T23:47:30.999999+05:30"
}
```

### 3. DELETE Product (Soft Delete - Admin Only)
```http
DELETE http://localhost:8080/api/v1/products/PROD00000001
```

**Headers:**
```
Authorization: Bearer YOUR_FIREBASE_TOKEN
```

**No Body Required**

**Success Response (200):**
```json
{
    "success": true,
    "message": "Product deleted successfully",
    "data": null,
    "timestamp": "2025-08-26T23:50:15.123456+05:30"
}
```

## 📝 JSON Field Validation Rules

### Required Fields:
- `name` (string, required)
- `description` (string, required)  
- `category` (string, required)
- `price` (float, required, min: 0)
- `actual_price` (float, required, min: 0)
- `stock` (int, required, min: 0)

### Optional Fields:
- `discount_percent` (int, 0-100, default: 0)
- `image_url` (string, optional)
- `review_count` (int, min: 0, default: 0)
- `rating` (float, 0-5, default: 0.0)

### Valid Categories:
- `"Spices"`
- `"Herbs"`  
- `"Superfoods"`
- `"Health"`
- `"Tea"`
- Any custom category name

## 🚨 Error Responses

### 401 Unauthorized (No Token):
```json
{
    "success": false,
    "message": "Missing or invalid Authorization header",
    "timestamp": "2025-08-26T23:52:10.123456+05:30"
}
```

### 403 Forbidden (Not Admin):
```json
{
    "error": "Admin access required"
}
```

### 400 Bad Request (Invalid JSON):
```json
{
    "success": false,
    "message": "Invalid request format",
    "error": {
        "code": "VALIDATION_ERROR",
        "message": "JSON parsing failed",
        "details": "invalid character '}' looking for beginning of object key string"
    },
    "timestamp": "2025-08-26T23:53:00.123456+05:30"
}
```

### 404 Not Found (Product doesn't exist):
```json
{
    "success": false,
    "message": "Product not found",
    "error": {
        "code": "NOT_FOUND",
        "message": "Product",
        "details": "PROD00000999"
    },
    "timestamp": "2025-08-26T23:54:00.123456+05:30"
}
```

## 🔧 Testing with Different JSON Bodies

### Minimal Valid Product:
```json
{
    "name": "Basic Product",
    "description": "A basic product description",
    "category": "Health",
    "price": 99.99,
    "actual_price": 129.99,
    "stock": 50
}
```

### Product with Discount:
```json
{
    "name": "Discounted Superfood",
    "description": "Premium superfood powder on sale",
    "category": "Superfoods",
    "price": 199.99,
    "actual_price": 299.99,
    "discount_percent": 33,
    "stock": 75,
    "image_url": "https://example.com/superfood.jpg"
}
```

### High-Rated Product:
```json
{
    "name": "Top Rated Herb",
    "description": "Customer favorite herb powder with excellent reviews",
    "category": "Herbs",
    "price": 149.99,
    "actual_price": 179.99,
    "discount_percent": 17,
    "stock": 200,
    "review_count": 150,
    "rating": 4.9,
    "image_url": "https://example.com/top-herb.jpg"
}
```

### Out of Stock Product:
```json
{
    "name": "Rare Spice",
    "description": "Rare imported spice (currently restocking)",
    "category": "Spices",
    "price": 599.99,
    "actual_price": 699.99,
    "discount_percent": 14,
    "stock": 0,
    "review_count": 25,
    "rating": 4.6
}
```

## 🎯 Postman Test Collection Structure

Create these requests in Postman:

**📁 Admin Product Routes**
- POST Create Product
- PUT Update Product  
- DELETE Remove Product

**📁 Test Data Sets**
- Minimal Product
- Discounted Product
- High-Rated Product
- Out of Stock Product

## 🔑 Authentication Setup Steps

1. **Get Firebase Token:**
   ```http
   POST http://localhost:8080/api/v1/auth/signup
   ```
   or
   ```http
   POST http://localhost:8080/api/v1/auth/signin
   ```

2. **Set Admin Role:**
   ```sql
   UPDATE users SET role = 'admin' WHERE firebase_uid = 'YOUR_FIREBASE_UID';
   ```

3. **Use in Headers:**
   ```
   Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
   ```

## ⚠️ Important Notes

- **Stock Status:** Automatically set to "In Stock" if stock > 0, "Out of Stock" if stock = 0
- **Soft Delete:** DELETE doesn't permanently remove, just marks as deleted
- **Audit Trail:** All changes track created_by/updated_by fields
- **Validation:** Server validates all fields before saving
- **Case Sensitive:** Category names are case-sensitive
