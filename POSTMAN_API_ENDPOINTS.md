# Product API Endpoints for Postman Testing

Base URL: `http://localhost:8080`

## Public Endpoints (No Authentication Required)

### 1. Get All Products
- **Method:** GET
- **URL:** `http://localhost:8080/api/v1/products`
- **Query Parameters (Optional):**
  - `page=1` (default: 1)
  - `per_page=20` (default: 20)
  - `category=Spices` (filter by category)
  - `min_price=50` (minimum price filter)
  - `max_price=200` (maximum price filter)
  - `in_stock=true` (only products in stock)

### 2. Get Product by ID
- **Method:** GET
- **URL:** `http://localhost:8080/api/v1/products/PROD00000001`
- **Replace `PROD00000001` with any valid product ID**

### 3. Get Products by Category
- **Method:** GET
- **URL:** `http://localhost:8080/api/v1/products/category/Superfoods`
- **Available Categories:** 
  - `Spices` (1 product: Ginger Powder)
  - `Herbs` (4 products: Ashwagandha, Neem, Tulsi, Brahmi Powders)
  - `Superfoods` (4 products: Moringa, Amla, Spirulina, Wheatgrass Powders)
- **Example URLs:**
  - `http://localhost:8080/api/v1/products/category/Spices`
  - `http://localhost:8080/api/v1/products/category/Herbs`
  - `http://localhost:8080/api/v1/products/category/Superfoods`
- **Query Parameters (Optional):**
  - `page=1`
  - `per_page=20`

### 4. Search Products
- **Method:** GET
- **URL:** `http://localhost:8080/api/v1/products/search?q=powder`
- **Query Parameters:**
  - `q=powder` (search term - searches in name, description, category)
  - `page=1` (optional)
  - `per_page=20` (optional)

## Admin-Only Endpoints (Requires Authentication + Admin Role)

### 5. Create New Product
- **Method:** POST
- **URL:** `http://localhost:8080/api/v1/products`
- **Headers Required:**
  - `Content-Type: application/json`
  - `Authorization: Bearer YOUR_FIREBASE_TOKEN`
- **Body (JSON):**
```json
{
    "name": "New Test Product",
    "description": "A premium test product for health and wellness",
    "category": "Health",
    "price": 149.99,
    "actual_price": 199.99,
    "discount_percent": 25,
    "image_url": "https://example.com/images/test-product.jpg",
    "stock": 100,
    "review_count": 0,
    "rating": 0.0
}
```

### 6. Update Product
- **Method:** PUT
- **URL:** `http://localhost:8080/api/v1/products/PROD00000001`
- **Headers Required:**
  - `Content-Type: application/json`
  - `Authorization: Bearer YOUR_FIREBASE_TOKEN`
- **Body (JSON):**
```json
{
    "name": "Updated Product Name",
    "description": "Updated description with new features",
    "category": "Superfoods",
    "price": 179.99,
    "actual_price": 229.99,
    "discount_percent": 22,
    "image_url": "https://example.com/images/updated-product.jpg",
    "stock": 85,
    "review_count": 15,
    "rating": 4.3
}
```

### 7. Delete Product (Soft Delete)
- **Method:** DELETE
- **URL:** `http://localhost:8080/api/v1/products/PROD00000001`
- **Headers Required:**
  - `Authorization: Bearer YOUR_FIREBASE_TOKEN`

## ✅ **Working Test URLs for Postman:**

### **Public Endpoints (Working Now):**
1. **Get All Products:** `http://localhost:8080/api/v1/products`
2. **Get Product by ID:** `http://localhost:8080/api/v1/products/PROD00000001`
3. **Search Products:** `http://localhost:8080/api/v1/products/search?q=powder`
4. **Get Spices Category:** `http://localhost:8080/api/v1/products/category/Spices`
5. **Get Herbs Category:** `http://localhost:8080/api/v1/products/category/Herbs`  
6. **Get Superfoods Category:** `http://localhost:8080/api/v1/products/category/Superfoods`
7. **Filter by Price:** `http://localhost:8080/api/v1/products?min_price=100&max_price=200`
8. **Get In-Stock Products:** `http://localhost:8080/api/v1/products?in_stock=true`

### **Admin Endpoints (Require Authentication):**
9. **Create Product:** `POST http://localhost:8080/api/v1/products` (with JSON body)
10. **Update Product:** `PUT http://localhost:8080/api/v1/products/PROD00000001` (with JSON body)
11. **Delete Product:** `DELETE http://localhost:8080/api/v1/products/PROD00000001`

## Sample Product IDs for Testing

Based on the seeded data, you can use these product IDs:
- `PROD00000001` - Turmeric Powder
- `PROD00000002` - Ginger Powder  
- `PROD00000003` - Ashwagandha Powder
- `PROD00000004` - Moringa Powder
- `PROD00000005` - Amla Powder
- `PROD00000006` - Neem Powder
- `PROD00000007` - Brahmi Powder
- `PROD00000008` - Triphala Powder
- `PROD00000009` - Tulsi Powder
- `PROD00000010` - Arjuna Powder

## Authentication Setup

To test admin-only endpoints:

1. **Get Firebase Token:**
   - Use the auth endpoints to sign up/login
   - POST `http://localhost:8080/api/v1/auth/signup` or `/signin`
   - Get the token from the response

2. **Set Admin Role:**
   - Ensure your user has `role = 'admin'` in the users table
   - You can check with: `SELECT * FROM users WHERE firebase_uid = 'YOUR_UID';`
   - Update with: `UPDATE users SET role = 'admin' WHERE firebase_uid = 'YOUR_UID';`

3. **Add Authorization Header:**
   - In Postman, go to Headers tab
   - Add key: `Authorization`
   - Add value: `Bearer YOUR_TOKEN_HERE`

## Expected Responses

### Success Response (GET/POST/PUT):
```json
{
    "success": true,
    "message": "Operation completed successfully",
    "data": { /* product data */ },
    "timestamp": "2025-08-26T20:39:03.452876+05:30"
}
```

### Paginated Response (List endpoints):
```json
{
    "success": true,
    "message": "Products retrieved successfully",
    "data": [ /* array of products */ ],
    "timestamp": "2025-08-26T20:39:03.452876+05:30",
    "pagination": {
        "page": 1,
        "per_page": 20,
        "total": 10,
        "total_pages": 1,
        "has_next": false,
        "has_prev": false
    }
}
```

### Error Response (Unauthorized):
```json
{
    "success": false,
    "message": "Missing or invalid Authorization header",
    "timestamp": "2025-08-26T20:39:20.7289681+05:30"
}
```

### Error Response (Forbidden - Non-admin):
```json
{
    "error": "Admin access required"
}
```

## Testing Tips

1. **Start with Public Endpoints:** Test GET operations first to verify server is running
2. **Check Data:** Use GET all products to see available product IDs
3. **Test Authentication:** Try admin endpoints without auth to see proper error responses
4. **Use Valid Data:** Follow the JSON schema for create/update operations
5. **Check Validation:** Try invalid data to test validation errors

## Postman Collection Setup

You can organize these in Postman folders:
- **📂 Public Endpoints**
  - Get All Products
  - Get Product by ID
  - Search Products
  - Get by Category
- **📂 Admin Endpoints**
  - Create Product
  - Update Product
  - Delete Product

Set environment variables in Postman:
- `base_url`: `http://localhost:8080`
- `auth_token`: `YOUR_FIREBASE_TOKEN`
