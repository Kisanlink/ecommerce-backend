# Recipe Approval System

This document describes the implementation of the recipe approval system where users can submit recipes for admin review before they become visible to all users.

## Overview

The recipe approval system consists of:
1. **User Recipe Submission**: Users can submit recipes for review
2. **Admin Review Process**: Admins can approve or reject submitted recipes
3. **Recipe Publication**: Approved recipes are added to the main recipes table and become visible to all users

## Architecture

### Database Tables

#### 1. `pending_recipe_reviews` (DynamoDB)
- **Primary Key**: `review_id` (String)
- **Global Secondary Indexes**:
  - `user-id-index`: (user_id, submitted_at) - for querying user's submissions
  - `status-index`: (status, submitted_at) - for querying by status

#### 2. `recipes` (DynamoDB)
- Existing table where approved recipes are stored
- Primary Key: `recipee_id` (String)

### Models

#### PendingRecipeReview
```go
type PendingRecipeReview struct {
    ReviewID        string                    // Primary key
    UserID          string                    // User who submitted
    Status          PendingRecipeReviewStatus // pending, approved, rejected
    SubmittedAt     string                    // Submission timestamp
    ReviewedAt      string                    // Review timestamp
    ReviewedBy      string                    // Admin who reviewed
    RejectionReason string                    // Reason for rejection
    
    // Recipe data (same structure as Recipe model)
    Name            string
    Category        string
    Description     string
    // ... other recipe fields
}
```

## API Endpoints

### User Endpoints

#### Submit Recipe for Review
```
POST /api/v1/recipes/submit
```
- **Authentication**: Required (Firebase)
- **Body**: Recipe data
- **Response**: Pending review details

#### Get User's Submitted Recipes
```
GET /api/v1/recipes/my-submissions
```
- **Authentication**: Required (Firebase)
- **Response**: List of user's submitted recipes with their status

### Admin Endpoints

#### Get Pending Recipe Reviews
```
GET /api/v1/admin/recipes/pending
```
- **Authentication**: Required (Firebase + Admin role)
- **Query Parameters**:
  - `page`: Page number (default: 1)
  - `page_size`: Items per page (default: 10)
  - `status`: Filter by status (pending, approved, rejected)
- **Response**: Paginated list of pending reviews

#### Get Specific Pending Review
```
GET /api/v1/admin/recipes/pending/{id}
```
- **Authentication**: Required (Firebase + Admin role)
- **Response**: Specific pending review details

#### Approve Recipe
```
POST /api/v1/admin/recipes/pending/{id}/approve
```
- **Authentication**: Required (Firebase + Admin role)
- **Response**: Approved recipe and updated review status

#### Reject Recipe
```
POST /api/v1/admin/recipes/pending/{id}/reject
```
- **Authentication**: Required (Firebase + Admin role)
- **Body**: `{"reason": "Rejection reason"}`
- **Response**: Updated review status

#### Delete Pending Review
```
DELETE /api/v1/admin/recipes/pending/{id}
```
- **Authentication**: Required (Firebase + Admin role)
- **Response**: Confirmation of deletion

## Workflow

### 1. User Recipe Submission
1. User submits recipe via `POST /api/v1/recipes/submit`
2. Recipe is saved to `pending_recipe_reviews` table with status `pending`
3. User receives confirmation with review ID

### 2. Admin Review Process
1. Admin views pending recipes via `GET /api/v1/admin/recipes/pending`
2. Admin reviews specific recipe via `GET /api/v1/admin/recipes/pending/{id}`
3. Admin either:
   - **Approves**: Recipe is moved to main `recipes` table, review status updated to `approved`
   - **Rejects**: Review status updated to `rejected` with reason

### 3. Recipe Publication
- Approved recipes are immediately available via existing recipe endpoints
- Rejected recipes remain in pending table for reference but are not published

## Setup Instructions

### 1. Create DynamoDB Table
Run the table creation script:
```bash
go run create_pending_recipe_reviews_table.go
```

### 2. Environment Variables
Set the following environment variable:
```bash
DYNAMODB_PENDING_RECIPE_REVIEWS_TABLE=pending_recipe_reviews
```

### 3. Deploy Changes
The system is ready to use after deploying the code changes.

## Security Considerations

1. **Authentication**: All endpoints require Firebase authentication
2. **Authorization**: Admin endpoints require admin role verification
3. **Input Validation**: All recipe data is validated before storage
4. **Audit Trail**: All review actions are logged with admin user ID and timestamps

## Error Handling

- **400 Bad Request**: Invalid input data or missing required fields
- **401 Unauthorized**: Missing or invalid authentication
- **403 Forbidden**: User lacks admin privileges for admin endpoints
- **404 Not Found**: Recipe review not found
- **500 Internal Server Error**: Database or server errors

## Testing

### Test Scenarios
1. **User Submission**: Submit a recipe and verify it's saved with pending status
2. **Admin Approval**: Approve a recipe and verify it appears in main recipes table
3. **Admin Rejection**: Reject a recipe with reason and verify status update
4. **User View Submissions**: Verify users can see their submission status
5. **Admin View Pending**: Verify admins can see all pending reviews

### Sample API Calls

#### Submit Recipe
```bash
curl -X POST http://localhost:8080/api/v1/recipes/submit \
  -H "Authorization: Bearer <firebase-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Recipe",
    "category": "Dessert",
    "description": "A test recipe",
    "prep_time": "30 minutes",
    "servings": 4,
    "difficulty": "Medium",
    "ingredients": ["ingredient1", "ingredient2"],
    "instructions": ["step1", "step2"]
  }'
```

#### Approve Recipe
```bash
curl -X POST http://localhost:8080/api/v1/admin/recipes/pending/{review-id}/approve \
  -H "Authorization: Bearer <admin-firebase-token>"
```

#### Reject Recipe
```bash
curl -X POST http://localhost:8080/api/v1/admin/recipes/pending/{review-id}/reject \
  -H "Authorization: Bearer <admin-firebase-token>" \
  -H "Content-Type: application/json" \
  -d '{"reason": "Recipe does not meet quality standards"}'
```

## Future Enhancements

1. **Email Notifications**: Notify users when their recipes are approved/rejected
2. **Bulk Operations**: Allow admins to approve/reject multiple recipes at once
3. **Review Comments**: Allow admins to add detailed feedback
4. **Recipe Categories**: Auto-categorize recipes based on content
5. **Quality Scoring**: Implement automated quality checks before admin review
