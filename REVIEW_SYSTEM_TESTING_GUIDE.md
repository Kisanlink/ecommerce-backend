# Product Review System Testing Guide

This document provides comprehensive examples for testing the product review system using DynamoDB for storage.

## API Endpoints Overview

### 1. Create Review
- **POST** `/api/v1/products/{id}/reviews/`
- **Authentication**: Required (Firebase Auth)

### 2. Get Product Reviews
- **GET** `/api/v1/products/{id}/reviews`
- **Authentication**: Optional (affects response data)

### 3. Get User Reviews
- **GET** `/api/v1/users/{id}/reviews`
- **Authentication**: Optional

### 4. Update Review
- **PUT** `/api/v1/products/{id}/reviews/{reviewId}`
- **Authentication**: Required (owner only)

### 5. Delete Review
- **DELETE** `/api/v1/products/{id}/reviews/{reviewId}`
- **Authentication**: Required (owner or admin)

### 6. Like/Unlike Review
- **POST** `/api/v1/products/{productId}/reviews/{reviewId}/like`
- **Authentication**: Required

### 7. Get Review Statistics
- **GET** `/api/v1/products/{productId}/reviews/stats`
- **Authentication**: Optional

---

## Sample Test Requests

### Environment Setup
```bash
# Base URL
BASE_URL="http://localhost:8080/api/v1"

# Replace with your actual Firebase auth token
AUTH_TOKEN="your_firebase_auth_token_here"

# Sample Product ID (use an existing product)
PRODUCT_ID="PROD123456"

# Sample User ID (use your Firebase user ID)
USER_ID="firebase_user_id_here"
```

### 1. Create a Review

```bash
curl -X POST "${BASE_URL}/products/${PRODUCT_ID}/reviews" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AUTH_TOKEN}" \
  -d '{
    "rating": 5,
    "title": "Excellent Product!",
    "comment": "This product exceeded my expectations. Great quality and fast shipping. Would definitely recommend to others!"
  }'
```

**Response:**
```json
{
  "message": "Review created successfully",
  "data": {
    "id": "REV12345678",
    "product_id": "PROD123456",
    "user_id": "firebase_user_id",
    "user_name": "John Doe",
    "user_email": "john.doe@example.com",
    "rating": 5,
    "title": "Excellent Product!",
    "comment": "This product exceeded my expectations...",
    "likes_count": 0,
    "is_user_liked": false,
    "is_verified": false,
    "is_owner": true,
    "is_active": true,
    "review_date": "2025-01-08T10:30:00Z",
    "updated_date": "2025-01-08T10:30:00Z",
    "time_ago": "Just now",
    "can_edit": true,
    "can_delete": true
  }
}
```

### 2. Get Product Reviews with Filters

```bash
# Get all reviews for a product (first page, 10 items)
curl -X GET "${BASE_URL}/products/${PRODUCT_ID}/reviews?page=1&limit=10&sort_by=date&order=desc" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AUTH_TOKEN}"

# Get reviews with rating filter
curl -X GET "${BASE_URL}/products/${PRODUCT_ID}/reviews?min_rating=4&max_rating=5&sort_by=rating&order=desc" \
  -H "Content-Type: application/json"

# Get reviews sorted by likes
curl -X GET "${BASE_URL}/products/${PRODUCT_ID}/reviews?sort_by=likes&order=desc&limit=5" \
  -H "Content-Type: application/json"
```

**Response:**
```json
{
  "message": "Reviews retrieved successfully",
  "data": {
    "reviews": [
      {
        "id": "REV12345678",
        "product_id": "PROD123456",
        "user_name": "John Doe",
        "rating": 5,
        "title": "Excellent Product!",
        "comment": "This product exceeded my expectations...",
        "likes_count": 3,
        "is_user_liked": false,
        "is_verified": false,
        "time_ago": "2 hours ago",
        "can_edit": false,
        "can_delete": false
      }
    ],
    "total_count": 15,
    "page": 1,
    "limit": 10,
    "total_pages": 2,
    "has_next": true,
    "has_previous": false
  }
}
```

### 3. Get User Reviews

```bash
curl -X GET "${BASE_URL}/users/${USER_ID}/reviews?page=1&limit=5" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AUTH_TOKEN}"
```

### 4. Update a Review

```bash
REVIEW_ID="REV12345678"

curl -X PUT "${BASE_URL}/products/${PRODUCT_ID}/reviews/${REVIEW_ID}" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AUTH_TOKEN}" \
  -d '{
    "rating": 4,
    "title": "Good Product with Minor Issues",
    "comment": "Updated my review after using it for a month. Overall good but had some minor quality issues."
  }'
```

### 5. Like/Unlike a Review

```bash
REVIEW_ID="REV12345678"

# Like a review
curl -X POST "${BASE_URL}/products/${PRODUCT_ID}/reviews/${REVIEW_ID}/like" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AUTH_TOKEN}" \
  -d '{
    "action": "like"
  }'

# Unlike a review
curl -X POST "${BASE_URL}/products/${PRODUCT_ID}/reviews/${REVIEW_ID}/like" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AUTH_TOKEN}" \
  -d '{
    "action": "unlike"
  }'
```

**Response:**
```json
{
  "success": true,
  "message": "Review liked successfully",
  "review_id": "REV12345678",
  "likes_count": 4,
  "is_user_liked": true
}
```

### 6. Get Review Statistics

```bash
curl -X GET "${BASE_URL}/products/${PRODUCT_ID}/reviews/stats" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AUTH_TOKEN}"
```

**Response:**
```json
{
  "message": "Review statistics retrieved successfully",
  "data": {
    "product_id": "PROD123456",
    "total_reviews": 25,
    "average_rating": 4.2,
    "rating_breakdown": {
      "1": 1,
      "2": 2,
      "3": 4,
      "4": 8,
      "5": 10
    },
    "verified_reviews": 15,
    "unverified_reviews": 10,
    "recent_reviews": [
      {
        "id": "REV12345678",
        "user_name": "John Doe",
        "rating": 5,
        "title": "Excellent Product!",
        "time_ago": "2 hours ago"
      }
    ],
    "top_reviews": [
      {
        "id": "REV87654321",
        "user_name": "Jane Smith",
        "rating": 5,
        "title": "Amazing quality!",
        "likes_count": 15,
        "time_ago": "1 week ago"
      }
    ]
  }
}
```

### 7. Delete a Review

```bash
REVIEW_ID="REV12345678"

curl -X DELETE "${BASE_URL}/products/${PRODUCT_ID}/reviews/${REVIEW_ID}" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AUTH_TOKEN}"
```

**Response:**
```json
{
  "message": "Review deleted successfully"
}
```

---

## Advanced Testing Scenarios

### 1. Test Pagination

```bash
# Get page 1
curl -X GET "${BASE_URL}/products/${PRODUCT_ID}/reviews?page=1&limit=3" -H "Authorization: Bearer ${AUTH_TOKEN}"

# Get page 2
curl -X GET "${BASE_URL}/products/${PRODUCT_ID}/reviews?page=2&limit=3" -H "Authorization: Bearer ${AUTH_TOKEN}"
```

### 2. Test Sorting Options

```bash
# Sort by date (newest first)
curl -X GET "${BASE_URL}/products/${PRODUCT_ID}/reviews?sort_by=date&order=desc" -H "Authorization: Bearer ${AUTH_TOKEN}"

# Sort by rating (highest first)
curl -X GET "${BASE_URL}/products/${PRODUCT_ID}/reviews?sort_by=rating&order=desc" -H "Authorization: Bearer ${AUTH_TOKEN}"

# Sort by likes (most liked first)
curl -X GET "${BASE_URL}/products/${PRODUCT_ID}/reviews?sort_by=likes&order=desc" -H "Authorization: Bearer ${AUTH_TOKEN}"
```

### 3. Test Rating Filters

```bash
# Only 5-star reviews
curl -X GET "${BASE_URL}/products/${PRODUCT_ID}/reviews?min_rating=5&max_rating=5" -H "Authorization: Bearer ${AUTH_TOKEN}"

# 4-5 star reviews
curl -X GET "${BASE_URL}/products/${PRODUCT_ID}/reviews?min_rating=4" -H "Authorization: Bearer ${AUTH_TOKEN}"

# 1-2 star reviews (negative reviews)
curl -X GET "${BASE_URL}/products/${PRODUCT_ID}/reviews?max_rating=2" -H "Authorization: Bearer ${AUTH_TOKEN}"
```

### 4. Test Error Scenarios

```bash
# Try to create review without authentication
curl -X POST "${BASE_URL}/products/${PRODUCT_ID}/reviews" \
  -H "Content-Type: application/json" \
  -d '{"rating": 5, "title": "Test", "comment": "Test comment"}'

# Try to create duplicate review
curl -X POST "${BASE_URL}/products/${PRODUCT_ID}/reviews" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AUTH_TOKEN}" \
  -d '{"rating": 5, "title": "Test", "comment": "Test comment"}'

# Try to update someone else's review
SOMEONE_ELSES_REVIEW_ID="REV87654321"
curl -X PUT "${BASE_URL}/products/${PRODUCT_ID}/reviews/${SOMEONE_ELSES_REVIEW_ID}" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AUTH_TOKEN}" \
  -d '{"rating": 1, "title": "Bad", "comment": "Trying to update someone else review"}'
```

---

## Database Verification

### Check DynamoDB Table
```bash
# List tables
aws dynamodb list-tables --region us-east-1

# Describe the reviews table
aws dynamodb describe-table --table-name product_reviews --region us-east-1

# Scan all reviews (for testing only)
aws dynamodb scan --table-name product_reviews --region us-east-1 --max-items 10
```

### Query Examples
```bash
# Get all reviews for a specific product
aws dynamodb query \
  --table-name product_reviews \
  --key-condition-expression "product_id = :pid" \
  --expression-attribute-values '{":pid":{"S":"PROD123456"}}' \
  --region us-east-1

# Get a specific review by product and user
aws dynamodb get-item \
  --table-name product_reviews \
  --key '{"product_id":{"S":"PROD123456"},"user_id":{"S":"firebase_user_id"}}' \
  --region us-east-1
```

---

## Load Testing

### Create Multiple Reviews
```bash
#!/bin/bash
PRODUCT_ID="PROD123456"
AUTH_TOKEN="your_token"
BASE_URL="http://localhost:8080/api/v1"

for i in {1..10}; do
  curl -X POST "${BASE_URL}/products/${PRODUCT_ID}/reviews" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${AUTH_TOKEN}" \
    -d "{
      \"rating\": $((1 + RANDOM % 5)),
      \"title\": \"Test Review $i\",
      \"comment\": \"This is a test review number $i with sufficient length to meet validation requirements.\"
    }"
  sleep 1
done
```

### Like Multiple Reviews
```bash
#!/bin/bash
PRODUCT_ID="PROD123456"
AUTH_TOKEN="your_token"
BASE_URL="http://localhost:8080/api/v1"

# Get list of review IDs first
REVIEW_IDS=("REV12345678" "REV87654321" "REV11111111")

for review_id in "${REVIEW_IDS[@]}"; do
  curl -X POST "${BASE_URL}/products/${PRODUCT_ID}/reviews/${review_id}/like" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${AUTH_TOKEN}" \
    -d '{"action": "like"}'
  sleep 1
done
```

---

## Expected System Behavior

### Review Creation
1. ✅ Users can only create one review per product
2. ✅ Rating must be between 1-5 stars
3. ✅ Title must be 5-100 characters
4. ✅ Comment must be 10-1000 characters
5. ✅ User information is cached in review document
6. ✅ Authentication is required

### Review Retrieval
1. ✅ Public access for reading reviews
2. ✅ Pagination support (max 100 items per page)
3. ✅ Multiple sorting options (date, rating, likes)
4. ✅ Rating filters support
5. ✅ Authenticated users see additional fields (like status, edit permissions)

### Review Updates
1. ✅ Only review owner can update
2. ✅ All fields can be updated except IDs and timestamps
3. ✅ Validation applied on updates

### Review Deletion
1. ✅ Only owner or admin can delete
2. ✅ Soft delete (sets is_active = false)
3. ✅ Deleted reviews don't appear in listings

### Like System
1. ✅ Users can like/unlike any review
2. ✅ Can't like own review multiple times
3. ✅ Like counts update in real-time
4. ✅ Authentication required

### Statistics
1. ✅ Real-time calculation of average rating
2. ✅ Rating breakdown by star count
3. ✅ Recent and top reviews included
4. ✅ Verified vs unverified review counts

This comprehensive testing guide should help you verify all aspects of the review system functionality.
