# Quick Postman Test URLs - Product API

## ✅ **Working URLs (Copy & Paste into Postman)**

### **GET Requests (No Auth Required):**

```
GET http://localhost:8080/api/v1/products
GET http://localhost:8080/api/v1/products/PROD00000001  
GET http://localhost:8080/api/v1/products/PROD00000002
GET http://localhost:8080/api/v1/products/PROD00000003
GET http://localhost:8080/api/v1/products/search?q=powder
GET http://localhost:8080/api/v1/products/search?q=turmeric
GET http://localhost:8080/api/v1/products/category/Spices
GET http://localhost:8080/api/v1/products/category/Herbs
GET http://localhost:8080/api/v1/products/category/Superfoods
GET http://localhost:8080/api/v1/products?min_price=100&max_price=200
GET http://localhost:8080/api/v1/products?in_stock=true
GET http://localhost:8080/api/v1/products?page=1&per_page=5
```

### **POST Request (Admin Auth Required):**
```
POST http://localhost:8080/api/v1/products
Content-Type: application/json
Authorization: Bearer YOUR_TOKEN

{
    "name": "Test Product",
    "description": "A premium test product",
    "category": "Health",
    "price": 149.99,
    "actual_price": 199.99,
    "discount_percent": 25,
    "image_url": "https://example.com/test.jpg",
    "stock": 100,
    "review_count": 0,
    "rating": 0.0
}
```

### **PUT Request (Admin Auth Required):**
```
PUT http://localhost:8080/api/v1/products/PROD00000001
Content-Type: application/json
Authorization: Bearer YOUR_TOKEN

{
    "name": "Updated Product Name",
    "description": "Updated description",
    "category": "Superfoods",
    "price": 179.99,
    "actual_price": 229.99,
    "discount_percent": 22,
    "image_url": "https://example.com/updated.jpg",
    "stock": 85,
    "review_count": 15,
    "rating": 4.3
}
```

### **DELETE Request (Admin Auth Required):**
```
DELETE http://localhost:8080/api/v1/products/PROD00000001
Authorization: Bearer YOUR_TOKEN
```

## 📊 **Categories with Product Counts:**
- `Spices` - 1 product (Ginger Powder)  
- `Herbs` - 4 products (Ashwagandha, Neem, Tulsi, Brahmi)
- `Superfoods` - 4 products (Moringa, Amla, Spirulina, Wheatgrass)

## 🔍 **Valid Product IDs:**
- PROD00000001 (Turmeric Powder - but might be updated)
- PROD00000002 (Ginger Powder)
- PROD00000003 (Ashwagandha Powder)
- PROD00000004 (Moringa Powder)
- PROD00000005 (Amla Powder)
- PROD00000006 (Spirulina Powder)
- PROD00000007 (Wheatgrass Powder)
- PROD00000008 (Neem Powder)
- PROD00000009 (Tulsi Powder)
- PROD00000010 (Brahmi Powder)

## ⚠️ **Important Notes:**
- **Case Sensitive:** Use exact category names: `Superfoods` NOT `SuperFoods`
- **Server Running:** Make sure server is running on port 8080
- **Admin Auth:** POST/PUT/DELETE require Firebase token + admin role
- **Valid JSON:** Ensure proper JSON format for POST/PUT requests
