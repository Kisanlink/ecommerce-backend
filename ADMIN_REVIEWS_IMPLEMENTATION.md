# Admin Review Functionality Implementation

## Overview
This document describes the implementation of admin review functionality for the ecommerce backend. The implementation includes admin routes, handlers, services, and request structures for managing product reviews.

## Features Implemented

### 1. Admin Request Structures
- **AdminReviewRequest**: For managing individual reviews (flag, unflag, approve, reject, force_delete)
- **AdminListReviewsRequest**: For listing all reviews with advanced filtering

### 2. Admin Service Methods
- **AdminListReviews**: Retrieve all reviews with admin-level filtering
- **AdminManageReview**: Perform admin actions on reviews
- **AdminGetFlaggedReviews**: Get all flagged reviews for review

### 3. Admin Handler Methods
- **AdminListReviews**: Handle GET /admin/reviews
- **AdminManageReview**: Handle PUT /admin/reviews/:reviewId
- **AdminGetFlaggedReviews**: Handle GET /admin/reviews/flagged
- **AdminBulkDeleteFlaggedReviews**: Handle DELETE /admin/reviews/flagged

### 4. Admin Routes
- **GET /api/v1/admin/reviews**: List all reviews with filtering
- **GET /api/v1/admin/reviews/flagged**: Get flagged reviews
- **PUT /api/v1/admin/reviews/:reviewId**: Manage review actions
- **DELETE /api/v1/admin/reviews/flagged**: Bulk delete all flagged reviews

### 5. Enhanced Review Model
Added admin-related fields to the Review model:
- `IsFlagged`: Whether the review has been flagged
- `IsApproved`: Whether the review has been approved
- `FlagReason`: Reason for flagging
- `FlaggedBy`: Admin who flagged the review
- `FlaggedAt`: When the review was flagged
- `ApprovedBy`: Admin who approved the review
- `ApprovedAt`: When the review was approved
- `RejectedBy`: Admin who rejected the review
- `RejectedAt`: When the review was rejected
- `RejectionReason`: Reason for rejection
- `DeletedBy`: Admin who deleted the review
- `DeletedAt`: When the review was deleted
- `DeletionReason`: Reason for deletion

## API Endpoints

### 1. List All Reviews (Admin)
```
GET /api/v1/admin/reviews
```

**Query Parameters:**
- `page` (int): Page number (default: 1)
- `limit` (int): Items per page (default: 20, max: 100)
- `sort_by` (string): Sort by field (review_date, updated_date, rating, likes_count, user_id, product_id)
- `order` (string): Sort order (asc, desc)
- `product_id` (string): Filter by product ID
- `user_id` (string): Filter by user ID
- `min_rating` (int): Minimum rating filter (1-5)
- `max_rating` (int): Maximum rating filter (1-5)
- `is_flagged` (bool): Filter by flagged status
- `is_active` (bool): Filter by active status
- `start_date` (string): Start date filter (RFC3339)
- `end_date` (string): End date filter (RFC3339)

**Response:**
```json
{
  "message": "Admin reviews retrieved successfully",
  "data": {
    "reviews": [...],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 100,
      "total_pages": 5
    }
  }
}
```

### 2. Get Flagged Reviews (Admin)
```
GET /api/v1/admin/reviews/flagged
```

**Query Parameters:**
- `page` (int): Page number (default: 1)
- `limit` (int): Items per page (default: 20, max: 100)

**Response:**
```json
{
  "message": "Flagged reviews retrieved successfully",
  "data": {
    "reviews": [...],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 5,
      "total_pages": 1
    }
  }
}
```

### 3. Manage Review (Admin)
```
PUT /api/v1/admin/reviews/:reviewId
```

**Request Body:**
```json
{
  "action": "flag|unflag|approve|reject|force_delete|hard_delete",
  "reason": "Optional reason for the action"
}
```

**Response:**
```json
{
  "message": "Review managed successfully",
  "data": {
    "id": "review_id",
    "product_id": "product_id",
    "user_id": "user_id",
    "user_name": "User Name",
    "rating": 5,
    "title": "Review Title",
    "comment": "Review comment",
    "is_flagged": true,
    "is_approved": false,
    "flag_reason": "Inappropriate content",
    "flagged_by": "admin_user_id",
    "flagged_at": "2024-01-01T00:00:00Z",
    "review_date": "2024-01-01T00:00:00Z",
    "updated_date": "2024-01-01T00:00:00Z"
  }
}
```

### 4. Bulk Delete All Flagged Reviews (Admin)
```
DELETE /api/v1/admin/reviews/flagged
```

**Request Body:**
```json
{
  "action": "hard_delete|force_delete",
  "reason": "Optional reason for the bulk deletion"
}
```

**Response:**
```json
{
  "message": "Bulk delete operation completed",
  "action": "hard_delete",
  "total_found": 15,
  "total_deleted": 14,
  "failed_deletes": 1,
  "deleted_review_ids": ["review_1", "review_2", "review_3"],
  "failed_review_ids": ["review_4"],
  "reason": "Bulk cleanup of spam reviews",
  "processed_at": "2024-01-15T10:30:00Z"
}
```

## Admin Actions

### 1. Flag Review
- Sets `is_flagged` to `true`
- Records `flag_reason`, `flagged_by`, and `flagged_at`
- Used to mark reviews for admin review

### 2. Unflag Review
- Sets `is_flagged` to `false`
- Clears flag-related fields
- Used to remove flag from a review

### 3. Approve Review
- Sets `is_approved` to `true`
- Records `approved_by` and `approved_at`
- Used to approve reviews for public display

### 4. Reject Review
- Sets `is_approved` to `false`
- Records `rejected_by`, `rejected_at`, and `rejection_reason`
- Used to reject reviews from public display

### 5. Force Delete Review
- Sets `is_active` to `false`
- Records `deleted_by`, `deleted_at`, and `deletion_reason`
- Used to soft-delete reviews (marks as inactive but keeps in database)

### 6. Hard Delete Review
- Permanently removes the review from the database
- Updates product review statistics
- Used to completely remove flagged or inappropriate reviews
- **Warning**: This action cannot be undone

## Security

All admin endpoints require:
1. **Firebase Authentication**: User must be authenticated
2. **Admin Role**: User must have admin role in the database
3. **AdminGuard Middleware**: Validates admin permissions

## Usage Examples

### Flag a Review
```bash
curl -X PUT "https://api.example.com/api/v1/admin/reviews/review_123" \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "flag",
    "reason": "Inappropriate language"
  }'
```

### List All Reviews with Filters
```bash
curl -X GET "https://api.example.com/api/v1/admin/reviews?is_flagged=true&min_rating=1&max_rating=3&page=1&limit=10" \
  -H "Authorization: Bearer <admin_token>"
```

### Get Flagged Reviews
```bash
curl -X GET "https://api.example.com/api/v1/admin/reviews/flagged?page=1&limit=20" \
  -H "Authorization: Bearer <admin_token>"
```

### Hard Delete a Review
```bash
curl -X PUT "https://api.example.com/api/v1/admin/reviews/review_123" \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "hard_delete",
    "reason": "Spam content - permanently removing"
  }'
```

### Bulk Delete All Flagged Reviews
```bash
curl -X DELETE "https://api.example.com/api/v1/admin/reviews/flagged" \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "hard_delete",
    "reason": "Bulk cleanup of spam reviews"
  }'
```

## Database Schema

The review model now includes the following admin-related fields in DynamoDB:

```json
{
  "is_flagged": "BOOL",
  "is_approved": "BOOL", 
  "flag_reason": "S",
  "flagged_by": "S",
  "flagged_at": "S",
  "approved_by": "S",
  "approved_at": "S",
  "rejected_by": "S",
  "rejected_at": "S",
  "rejection_reason": "S",
  "deleted_by": "S",
  "deleted_at": "S",
  "deletion_reason": "S"
}
```

## Implementation Notes

1. **Backward Compatibility**: All new fields are optional and won't break existing reviews
2. **Default Values**: New reviews are created with `is_approved: true` and `is_flagged: false`
3. **Audit Trail**: All admin actions are logged with timestamps and admin user IDs
4. **Soft Delete**: Reviews are soft-deleted (is_active: false) rather than hard-deleted
5. **Product Stats**: Product review statistics are updated when reviews are deleted

## Testing

The implementation includes comprehensive error handling and validation:
- Request validation for all admin actions
- Proper error responses for invalid actions
- Logging of all admin operations
- Graceful handling of missing reviews

## Future Enhancements

Potential future improvements:
1. Bulk review management operations
2. Review moderation queue
3. Automated flagging based on content analysis
4. Review analytics and reporting
5. Email notifications for flagged reviews
