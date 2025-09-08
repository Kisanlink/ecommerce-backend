# Postman Review System Testing Guide

## Base URL
```
http://localhost:8080/api/v1
```

## Authentication Required
For authenticated endpoints, add Firebase JWT token in Headers:
```
Authorization: Bearer YOUR_FIREBASE_JWT_TOKEN
Content-Type: application/json
```

---

## 1. POST - Create a Product Review
**Endpoint:** `POST /products/{product_id}/reviews`  
**Authentication:** Required  
**URL Example:** `http://localhost:8080/api/v1/products/test-product-123/reviews`

### Request Body:
```json
{
  "product_id": "test-product-123",
  "rating": 5,
  "title": "Excellent product quality!",
  "comment": "I absolutely love this product. The quality is outstanding and it exceeded my expectations. Would definitely recommend to others!"
}
```

### Expected Response (201 Created):
```json
{
  "success": true,
  "message": "Review created successfully",
  "data": {
    "id": "review_abc123def456",
    "product_id": "test-product-123",
    "user_id": "user_firebase_uid",
    "user_name": "John Doe",
    "user_email": "john@example.com",
    "rating": 5,
    "title": "Excellent product quality!",
    "comment": "I absolutely love this product. The quality is outstanding and it exceeded my expectations. Would definitely recommend to others!",
    "likes_count": 0,
    "is_user_liked": false,
    "is_verified": false,
    "is_owner": true,
    "is_active": true,
    "review_date": "2025-09-08T10:30:00Z",
    "updated_date": "2025-09-08T10:30:00Z",
    "time_ago": "Just now",
    "can_edit": true,
    "can_delete": true
  }
}
```

---

## 2. GET - Get All Reviews for a Product
**Endpoint:** `GET /products/{product_id}/reviews`  
**Authentication:** Not Required  
**URL Example:** `http://localhost:8080/api/v1/products/test-product-123/reviews`

### Query Parameters (Optional):
```
?page=1&limit=10&sort_by=date&order=desc&min_rating=1&max_rating=5
```

### Full URL with Parameters:
```
http://localhost:8080/api/v1/products/test-product-123/reviews?page=1&limit=10&sort_by=date&order=desc&min_rating=1&max_rating=5
```

### Expected Response (200 OK):
```json
{
  "success": true,
  "message": "Reviews retrieved successfully",
  "data": {
    "reviews": [
      {
        "id": "review_abc123def456",
        "product_id": "test-product-123",
        "user_id": "user_firebase_uid",
        "user_name": "John Doe",
        "rating": 5,
        "title": "Excellent product quality!",
        "comment": "I absolutely love this product. The quality is outstanding and it exceeded my expectations.",
        "likes_count": 3,
        "is_user_liked": false,
        "is_verified": false,
        "is_owner": false,
        "is_active": true,
        "review_date": "2025-09-08T10:30:00Z",
        "updated_date": "2025-09-08T10:30:00Z",
        "time_ago": "2 hours ago",
        "can_edit": false,
        "can_delete": false
      },
      {
        "id": "review_def456ghi789",
        "product_id": "test-product-123",
        "user_id": "another_user_uid",
        "user_name": "Jane Smith",
        "rating": 4,
        "title": "Good value for money",
        "comment": "The product is quite good for the price. Delivery was fast and packaging was excellent.",
        "likes_count": 1,
        "is_user_liked": false,
        "is_verified": true,
        "is_owner": false,
        "is_active": true,
        "review_date": "2025-09-07T15:20:00Z",
        "updated_date": "2025-09-07T15:20:00Z",
        "time_ago": "1 day ago",
        "can_edit": false,
        "can_delete": false
      }
    ],
    "total_count": 25,
    "page": 1,
    "limit": 10,
    "total_pages": 3,
    "has_next": true,
    "has_previous": false
  }
}
```

---

## 3. PUT - Update a Review
**Endpoint:** `PUT /products/{product_id}/reviews/{review_id}`  
**Authentication:** Required (Owner only)  
**URL Example:** `http://localhost:8080/api/v1/products/test-product-123/reviews/review_abc123def456`

### Request Body:
```json
{
  "rating": 4,
  "title": "Updated: Good product with minor issues",
  "comment": "After using for a week, I found some minor issues but overall it's a good product. The quality is decent and customer service is helpful."
}
```

### Expected Response (200 OK):
```json
{
  "success": true,
  "message": "Review updated successfully",
  "data": {
    "id": "review_abc123def456",
    "product_id": "test-product-123",
    "user_id": "user_firebase_uid",
    "user_name": "John Doe",
    "rating": 4,
    "title": "Updated: Good product with minor issues",
    "comment": "After using for a week, I found some minor issues but overall it's a good product. The quality is decent and customer service is helpful.",
    "likes_count": 3,
    "is_user_liked": false,
    "is_verified": false,
    "is_owner": true,
    "is_active": true,
    "review_date": "2025-09-08T10:30:00Z",
    "updated_date": "2025-09-08T12:45:00Z",
    "time_ago": "2 hours ago",
    "can_edit": true,
    "can_delete": true
  }
}
```

---

## 4. DELETE - Delete a Review
**Endpoint:** `DELETE /products/{product_id}/reviews/{review_id}`  
**Authentication:** Required (Owner or Admin)  
**URL Example:** `http://localhost:8080/api/v1/products/test-product-123/reviews/review_abc123def456`

### Request Body: None

### Expected Response (200 OK):
```json
{
  "success": true,
  "message": "Review deleted successfully",
  "data": {
    "review_id": "review_abc123def456",
    "deleted_at": "2025-09-08T13:00:00Z"
  }
}
```

---

## 5. POST - Like/Unlike a Review
**Endpoint:** `POST /products/{product_id}/reviews/{review_id}/like`  
**Authentication:** Required  
**URL Example:** `http://localhost:8080/api/v1/products/test-product-123/reviews/review_def456ghi789/like`

### Request Body (Like):
```json
{
  "review_id": "review_def456ghi789",
  "product_id": "test-product-123",
  "action": "like"
}
```

### Request Body (Unlike):
```json
{
  "review_id": "review_def456ghi789",
  "product_id": "test-product-123",
  "action": "unlike"
}
```

### Expected Response (200 OK):
```json
{
  "success": true,
  "message": "Review liked successfully",
  "review_id": "review_def456ghi789",
  "likes_count": 4,
  "is_user_liked": true
}
```

---

## 6. GET - Get Review Statistics for a Product
**Endpoint:** `GET /products/{product_id}/reviews/stats`  
**Authentication:** Not Required  
**URL Example:** `http://localhost:8080/api/v1/products/test-product-123/reviews/stats`

### Expected Response (200 OK):
```json
{
  "success": true,
  "message": "Review statistics retrieved successfully",
  "data": {
    "product_id": "test-product-123",
    "total_reviews": 25,
    "average_rating": 4.2,
    "rating_breakdown": {
      "1": 1,
      "2": 2,
      "3": 5,
      "4": 8,
      "5": 9
    },
    "recent_reviews": [
      {
        "id": "review_latest123",
        "user_name": "Alice Johnson",
        "rating": 5,
        "title": "Amazing product!",
        "comment": "Exceeded my expectations completely...",
        "likes_count": 2,
        "time_ago": "1 hour ago"
      }
    ],
    "top_reviews": [
      {
        "id": "review_popular456",
        "user_name": "Bob Wilson",
        "rating": 4,
        "title": "Very satisfied with purchase",
        "comment": "Great quality and fast delivery...",
        "likes_count": 15,
        "time_ago": "3 days ago"
      }
    ],
    "verified_reviews": 18,
    "unverified_reviews": 7
  }
}
```

---

## 7. GET - Get All Reviews by a User
**Endpoint:** `GET /users/{user_id}/reviews`  
**Authentication:** Not Required  
**URL Example:** `http://localhost:8080/api/v1/users/user_firebase_uid/reviews`

### Query Parameters (Optional):
```
?page=1&limit=10&sort_by=date&order=desc
```

### Expected Response (200 OK):
```json
{
  "success": true,
  "message": "User reviews retrieved successfully",
  "data": {
    "reviews": [
      {
        "id": "review_user123",
        "product_id": "product-456",
        "user_id": "user_firebase_uid",
        "user_name": "John Doe",
        "rating": 5,
        "title": "Great purchase!",
        "comment": "Really happy with this product...",
        "likes_count": 5,
        "is_user_liked": false,
        "is_verified": true,
        "is_owner": true,
        "review_date": "2025-09-08T09:15:00Z",
        "time_ago": "5 hours ago"
      }
    ],
    "total_count": 12,
    "page": 1,
    "limit": 10,
    "total_pages": 2,
    "has_next": true,
    "has_previous": false
  }
}
```

---

## Error Responses

### 400 Bad Request:
```json
{
  "success": false,
  "error": "Validation failed",
  "message": "Rating must be between 1 and 5",
  "details": {
    "field": "rating",
    "code": "INVALID_RANGE"
  }
}
```

### 401 Unauthorized:
```json
{
  "success": false,
  "error": "Unauthorized",
  "message": "Authentication token is required"
}
```

### 403 Forbidden:
```json
{
  "success": false,
  "error": "Forbidden",
  "message": "You can only edit your own reviews"
}
```

### 404 Not Found:
```json
{
  "success": false,
  "error": "Not Found",
  "message": "Review not found"
}
```

### 409 Conflict:
```json
{
  "success": false,
  "error": "Conflict",
  "message": "You have already reviewed this product"
}
```

---

## Testing Notes

1. **Authentication**: Make sure to get a valid Firebase JWT token and include it in the Authorization header for protected endpoints.

2. **Product ID**: Use valid product IDs that exist in your system. For testing, you can use `test-product-123`.

3. **User ID**: The user ID should be the Firebase UID of the authenticated user.

4. **Review ID**: Use valid review IDs returned from create/get operations.

5. **Validation**: 
   - Rating: Must be 1-5
   - Title: 5-100 characters
   - Comment: 10-1000 characters

6. **Pagination**: Default page=1, limit=10. Maximum limit is 100.

7. **Sorting Options**: 
   - sort_by: "date", "rating", "likes"
   - order: "asc", "desc"

8. **Rate Limiting**: Be mindful of rate limits when testing multiple requests.

## Sample Test Sequence

1. **POST** Create a review
2. **GET** Fetch all reviews for the product
3. **GET** Get review statistics
4. **POST** Like the review
5. **PUT** Update your review
6. **GET** Get your reviews
7. **DELETE** Delete your review

This covers all the main functionality of the review system!
