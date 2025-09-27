# Admin API Organization

This document outlines the organized structure of admin-only API endpoints in the Swagger documentation.

## Overview

Admin endpoints have been separated into dedicated sections with clear "Admin -" prefixes to provide better clarity and organization. This makes it easy for administrators to identify which functions they have access to and what operations they can perform.

## Admin API Sections

### 🔧 **Admin - Coupons**
**Purpose**: Manage discount coupons and promotional codes

**Available Operations:**
- `POST /api/v1/coupons` - Create a new coupon
- `GET /api/v1/coupons` - Get all coupons (with pagination)
- `PUT /api/v1/coupons/{id}` - Update an existing coupon
- `DELETE /api/v1/coupons/{id}` - Delete a coupon

**Admin Capabilities:**
- Create discount coupons with various rule types
- Set minimum order amounts and maximum discounts
- Configure usage limits and expiration dates
- Manage coupon categories and product restrictions
- View all coupons with usage statistics

### 🎯 **Admin - Coupon Rules**
**Purpose**: Manage complex coupon rules and conditions

**Available Operations:**
- `POST /api/v1/coupon-rules/{couponId}` - Create a new coupon rule
- `PUT /api/v1/coupon-rules/{couponId}/{ruleId}` - Update an existing coupon rule
- `DELETE /api/v1/coupon-rules/{couponId}/{ruleId}` - Delete a coupon rule

**Admin Capabilities:**
- Define complex discount rules
- Set category-specific discounts
- Configure product-specific promotions
- Manage user eligibility rules
- Set time-based restrictions

### 📦 **Admin - Products**
**Purpose**: Manage product catalog and inventory

**Available Operations:**
- `POST /api/v1/products` - Create a new product

**Admin Capabilities:**
- Add new products to the catalog
- Set product details, pricing, and descriptions
- Configure product categories and tags
- Manage product images and specifications

### 📊 **Admin - Inventory**
**Purpose**: Manage stock levels and inventory operations

**Available Operations:**
- `POST /api/v1/inventory` - Create a new inventory record
- `PUT /api/v1/inventory/product/{product_id}` - Update inventory information
- `DELETE /api/v1/inventory/product/{product_id}` - Delete inventory record
- `PUT /api/v1/inventory/stock/{product_id}` - Update stock quantity

**Admin Capabilities:**
- Create and manage inventory records
- Update stock quantities
- Set reorder points and stock levels
- Monitor low stock items
- Manage inventory across multiple locations

### 🛒 **Admin - Cart**
**Purpose**: Manage cart operations and cleanup

**Available Operations:**
- `POST /api/v1/cart/admin/cleanup-duplicates` - Clean up duplicate carts

**Admin Capabilities:**
- Clean up duplicate or orphaned carts
- Monitor cart health and performance
- Manage cart-related data integrity

## User vs Admin Endpoints

### **Regular User Endpoints**
These endpoints are available to all authenticated users:

**Coupons (User):**
- `GET /api/v1/coupons/{code}` - Get coupon by code
- `GET /api/v1/coupons/active` - Get active coupons
- `POST /api/v1/coupons/validate` - Validate coupon
- `POST /api/v1/coupons/apply` - Apply coupon to user
- `GET /api/v1/coupons/{code}/stats` - Get coupon usage statistics

**Products (User):**
- `GET /api/v1/products` - Get all products
- `GET /api/v1/products/{id}` - Get product by ID
- `GET /api/v1/products/search` - Search products

**Cart (User):**
- `GET /api/v1/cart` - Get user's cart
- `POST /api/v1/cart/items` - Add item to cart
- `PUT /api/v1/cart/items/{product_id}` - Update cart item
- `DELETE /api/v1/cart/items/{product_id}` - Remove from cart

### **Admin-Only Endpoints**
These endpoints require admin authentication and are clearly marked with "Admin -" tags:

- All endpoints in the "Admin -" sections listed above
- Require Firebase authentication with admin privileges
- Protected by admin middleware guards

## Authentication Requirements

### **Regular Users**
- Firebase authentication required
- Standard user permissions
- Can access public and user-specific endpoints

### **Admin Users**
- Firebase authentication required
- Admin role verification
- Access to both user and admin endpoints
- Additional security checks for sensitive operations

## Benefits of This Organization

1. **Clear Separation**: Easy to distinguish between user and admin capabilities
2. **Better Navigation**: Admins can quickly find relevant endpoints
3. **Security Clarity**: Clear indication of privileged operations
4. **Documentation**: Self-documenting API structure
5. **Maintenance**: Easier to manage and update admin-specific features

## Accessing Admin Documentation

1. **Swagger UI**: Visit `http://localhost:8080/docs`
2. **Look for "Admin -" sections** in the API documentation
3. **Use Firebase admin authentication** when testing endpoints
4. **Check security requirements** for each admin endpoint

## Security Notes

- All admin endpoints require Firebase authentication
- Admin role verification is enforced at the middleware level
- Sensitive operations are logged for audit purposes
- Admin endpoints should only be accessible in secure environments
- Consider disabling admin endpoints in production if not needed

This organization provides a clear, maintainable structure for API documentation that makes it easy for administrators to understand their capabilities and for developers to manage the codebase effectively.
