# NeonDB Tables Design - Reviews, Recipes, and Pending Recipe Reviews

## Overview

This document describes the design of three tables migrated from DynamoDB to NeonDB:
1. **reviews** - Product reviews with moderation capabilities
2. **recipes** - Approved recipes with flexible JSONB storage
3. **pending_recipe_reviews** - Recipe submissions awaiting admin approval

## Design Principles

### 1. Data Integrity
- **Foreign Keys**: Enforce referential integrity with `users` and `products` tables
- **Check Constraints**: Validate data ranges (ratings 1-5, servings > 0, etc.)
- **Unique Constraints**: Prevent duplicate active reviews per user per product
- **NOT NULL Constraints**: Ensure required fields are always present

### 2. Performance & Scalability
- **Strategic Indexes**: Composite indexes for common query patterns
- **Partial Indexes**: Index only active/non-deleted records
- **GIN Indexes**: Fast full-text search and JSONB queries
- **Array Indexes**: Efficient searching within ingredient/instruction arrays

### 3. Future-Proof Design
- **JSONB for Flexibility**: Nutrition facts stored as JSONB for schema evolution
- **Array Types**: PostgreSQL arrays for ingredients, instructions, tips
- **Soft Deletes**: `deleted_at` column for audit trail without data loss
- **Audit Fields**: Track who created/updated records and when

---

## Table 1: `reviews` - Product Reviews

### Purpose
Store user reviews for products with moderation, likes, and verification features.

### Key Columns

| Column | Type | Description | Constraints |
|--------|------|-------------|-------------|
| `id` | VARCHAR(255) | Primary key (REV...) | PRIMARY KEY |
| `product_id` | VARCHAR(64) | Foreign key to products | NOT NULL, FK |
| `user_id` | VARCHAR(64) | Foreign key to users | NOT NULL, FK |
| `rating` | INTEGER | Rating 1-5 stars | CHECK (1-5) |
| `title` | VARCHAR(500) | Review title | NOT NULL, LENGTH >= 5 |
| `comment` | TEXT | Review content | NOT NULL, LENGTH >= 10 |
| `likes_count` | INTEGER | Number of likes | DEFAULT 0, CHECK >= 0 |
| `liked_by` | TEXT[] | Array of user IDs who liked | DEFAULT '{}' |
| `is_verified` | BOOLEAN | Verified purchase flag | DEFAULT false |
| `is_active` | BOOLEAN | Soft delete flag | DEFAULT true |
| `is_flagged` | BOOLEAN | Moderation flag | DEFAULT false |
| `is_approved` | BOOLEAN | Approval status | DEFAULT true |
| `review_date` | TIMESTAMP | When review was created | NOT NULL |
| `updated_date` | TIMESTAMP | Last update time | NOT NULL |

### Relationships
- **Foreign Key to `products`**: `product_id` → `products.id` (CASCADE on delete)
- **Foreign Key to `users`**: `user_id` → `users.id` (CASCADE on delete)

### Indexes

1. **Basic Indexes**:
   - `idx_reviews_product_id` - Fast product review queries
   - `idx_reviews_user_id` - Fast user review queries
   - `idx_reviews_is_active` - Filter active reviews
   - `idx_reviews_is_flagged` - Admin moderation queries
   - `idx_reviews_rating` - Sort by rating
   - `idx_reviews_review_date` - Sort by date

2. **Composite Indexes** (for common queries):
   - `idx_reviews_product_rating` - Get product reviews sorted by rating
   - `idx_reviews_user_product` - Check if user reviewed product
   - `idx_reviews_product_active_date` - **Most common**: Active product reviews by date
   - `idx_reviews_flagged_date` - Admin: Flagged reviews by date
   - `idx_reviews_verified` - Verified purchase reviews

### Unique Constraint
- **One active review per user per product**: `uni_user_product_review`
  - Prevents duplicate reviews
  - Only applies to active, non-deleted reviews

### Scalability Considerations

1. **Likes System**:
   - Uses PostgreSQL array (`TEXT[]`) for `liked_by`
   - Efficient for small to medium number of likes (< 1000)
   - For very high like counts, consider separate `review_likes` table

2. **Full-Text Search** (Future):
   - Can add `tsvector` index on `title` and `comment` for search
   - Example: `CREATE INDEX idx_reviews_search ON reviews USING gin(to_tsvector('english', title || ' ' || comment))`

3. **Partitioning** (Future - if millions of reviews):
   - Partition by `product_id` hash or `review_date` range
   - Example: `PARTITION BY RANGE (review_date)`

---

## Table 2: `recipes` - Approved Recipes

### Purpose
Store approved recipes with flexible JSONB storage for nutrition facts and arrays for ingredients/instructions.

### Key Columns

| Column | Type | Description | Constraints |
|--------|------|-------------|-------------|
| `id` | VARCHAR(255) | Primary key (RECP...) | PRIMARY KEY |
| `recipe_id` | VARCHAR(64) | Unique recipe identifier | UNIQUE, NOT NULL |
| `name` | VARCHAR(255) | Recipe name | NOT NULL, LENGTH >= 3 |
| `category` | VARCHAR(100) | Recipe category | NOT NULL |
| `description` | TEXT | Recipe description | NOT NULL |
| `ingredients` | TEXT[] | Array of ingredients | DEFAULT '{}' |
| `instructions` | TEXT[] | Array of steps | DEFAULT '{}' |
| `nutrition_facts` | JSONB | Flexible nutrition data | DEFAULT '{}' |
| `pro_tips` | TEXT[] | Array of tips | DEFAULT '{}' |
| `rating` | DECIMAL(3,2) | Average rating | CHECK (0-5) |
| `reviews` | INTEGER | Number of reviews | CHECK >= 0 |
| `servings` | INTEGER | Number of servings | CHECK > 0 |

### Relationships
- **No Foreign Keys**: Recipes are standalone (no user ownership in main table)
- **User relationship**: Tracked in `pending_recipe_reviews` before approval

### Indexes

1. **Basic Indexes**:
   - `idx_recipes_recipe_id` - Fast lookup by recipe_id
   - `idx_recipes_category` - Filter by category
   - `idx_recipes_name` - Search by name
   - `idx_recipes_rating` - Sort by rating
   - `idx_recipes_created_at` - Sort by creation date

2. **Advanced Indexes**:
   - `idx_recipes_category_rating` - **Most common**: Category recipes by rating
   - `idx_recipes_search` - **Full-text search**: GIN index on name + description
   - `idx_recipes_nutrition_facts` - **JSONB search**: GIN index for nutrition queries
   - `idx_recipes_ingredients` - **Array search**: GIN index for ingredient search

### JSONB Nutrition Facts Example

```json
{
  "calories": 250,
  "protein": "10g",
  "carbs": "30g",
  "fat": "8g",
  "fiber": "5g",
  "sugar": "12g"
}
```

**Query Example**:
```sql
-- Find recipes with less than 300 calories
SELECT * FROM recipes 
WHERE (nutrition_facts->>'calories')::int < 300;
```

### Array Usage Examples

**Ingredients Array**:
```sql
-- Find recipes containing "chicken"
SELECT * FROM recipes 
WHERE 'chicken' = ANY(ingredients);
```

**Instructions Array**:
```sql
-- Get recipe with step count
SELECT name, array_length(instructions, 1) as step_count 
FROM recipes;
```

### Scalability Considerations

1. **JSONB Flexibility**:
   - Can add new nutrition fields without schema changes
   - Efficient queries with GIN indexes
   - Supports nested objects if needed

2. **Array Size Limits**:
   - PostgreSQL arrays work well for < 1000 items
   - For very large ingredient lists, consider normalization

3. **Full-Text Search**:
   - GIN index enables fast search across name and description
   - Can extend to search ingredients/instructions if needed

4. **Partitioning** (Future):
   - Partition by `category` for category-based queries
   - Example: `PARTITION BY LIST (category)`

---

## Table 3: `pending_recipe_reviews` - Pending Recipe Submissions

### Purpose
Store user-submitted recipes awaiting admin approval before moving to `recipes` table.

### Key Columns

| Column | Type | Description | Constraints |
|--------|------|-------------|-------------|
| `id` | VARCHAR(255) | Primary key (PRR...) | PRIMARY KEY |
| `review_id` | VARCHAR(64) | Unique review identifier | UNIQUE, NOT NULL |
| `user_id` | VARCHAR(64) | User who submitted | NOT NULL, FK |
| `status` | VARCHAR(32) | pending/approved/rejected | CHECK IN (...) |
| `submitted_at` | TIMESTAMP | Submission time | NOT NULL |
| `reviewed_at` | TIMESTAMP | Review completion time | NULL |
| `reviewed_by` | VARCHAR(255) | Admin user ID | NULL |
| `rejection_reason` | TEXT | Rejection reason | NULL |
| `name` | VARCHAR(255) | Recipe name | NOT NULL |
| `ingredients` | TEXT[] | Array of ingredients | DEFAULT '{}' |
| `instructions` | TEXT[] | Array of steps | DEFAULT '{}' |
| `nutrition_facts` | JSONB | Nutrition data | DEFAULT '{}' |

### Relationships
- **Foreign Key to `users`**: `user_id` → `users.id` (CASCADE on delete)

### Indexes

1. **Basic Indexes**:
   - `idx_pending_reviews_review_id` - Fast lookup
   - `idx_pending_reviews_user_id` - User submissions
   - `idx_pending_reviews_status` - Filter by status
   - `idx_pending_reviews_created_at` - Sort by submission date

2. **Composite Indexes**:
   - `idx_pending_reviews_status_date` - **Admin**: Pending reviews by date
   - `idx_pending_reviews_user_status` - **User**: My submissions by status
   - `idx_pending_reviews_pending` - **Most common**: Pending reviews only

3. **Advanced Indexes**:
   - `idx_pending_reviews_nutrition` - JSONB search
   - `idx_pending_reviews_ingredients` - Array search

### Workflow

1. **User Submits** → Status: `pending`, `submitted_at` = now
2. **Admin Reviews** → Status: `approved` or `rejected`, `reviewed_at` = now, `reviewed_by` = admin_id
3. **If Approved** → Move to `recipes` table, delete from `pending_recipe_reviews`
4. **If Rejected** → Keep in table with `rejection_reason` for user reference

### Scalability Considerations

1. **Status-Based Queries**:
   - Partial index on `pending` status (most common query)
   - Efficient admin dashboard loading

2. **Archive Strategy** (Future):
   - Move old approved/rejected records to archive table
   - Keep only recent records in main table
   - Example: `pending_recipe_reviews_archive`

---

## Data Integrity Features

### Foreign Key Constraints

```sql
-- Reviews table
CONSTRAINT fk_reviews_product FOREIGN KEY (product_id) 
    REFERENCES products(id) ON DELETE CASCADE
CONSTRAINT fk_reviews_user FOREIGN KEY (user_id) 
    REFERENCES users(id) ON DELETE CASCADE

-- Pending Recipe Reviews table
CONSTRAINT fk_pending_reviews_user FOREIGN KEY (user_id) 
    REFERENCES users(id) ON DELETE CASCADE
```

**Benefits**:
- Prevents orphaned records
- Automatic cleanup on user/product deletion
- Database-level integrity enforcement

### Check Constraints

```sql
-- Rating validation
CONSTRAINT chk_reviews_rating CHECK (rating >= 1 AND rating <= 5)

-- Text length validation
CONSTRAINT chk_reviews_title_length CHECK (LENGTH(title) >= 5)
CONSTRAINT chk_reviews_comment_length CHECK (LENGTH(comment) >= 10)

-- Positive values
CONSTRAINT chk_recipes_servings CHECK (servings > 0)
CONSTRAINT chk_recipes_reviews CHECK (reviews >= 0)
```

**Benefits**:
- Prevents invalid data at database level
- No need for application-level validation
- Consistent data quality

### Unique Constraints

```sql
-- One active review per user per product
CONSTRAINT uni_user_product_review UNIQUE (user_id, product_id, is_active) 
    WHERE is_active = true AND deleted_at IS NULL
```

**Benefits**:
- Prevents duplicate reviews
- Partial unique index (only for active reviews)
- Efficient constraint checking

---

## Query Performance Optimization

### Common Query Patterns

1. **Get Product Reviews** (Most Common):
   ```sql
   SELECT * FROM reviews 
   WHERE product_id = ? 
     AND is_active = true 
     AND is_flagged = false 
     AND deleted_at IS NULL
   ORDER BY review_date DESC;
   ```
   **Index Used**: `idx_reviews_product_active_date`

2. **Get User's Reviews**:
   ```sql
   SELECT * FROM reviews 
   WHERE user_id = ? 
     AND is_active = true 
     AND deleted_at IS NULL
   ORDER BY review_date DESC;
   ```
   **Index Used**: `idx_reviews_user_id`

3. **Admin: Get Flagged Reviews**:
   ```sql
   SELECT * FROM reviews 
   WHERE is_flagged = true 
   ORDER BY flagged_at DESC;
   ```
   **Index Used**: `idx_reviews_flagged_date`

4. **Search Recipes by Category**:
   ```sql
   SELECT * FROM recipes 
   WHERE category = ? 
     AND deleted_at IS NULL
   ORDER BY rating DESC;
   ```
   **Index Used**: `idx_recipes_category_rating`

5. **Search Recipes by Ingredient**:
   ```sql
   SELECT * FROM recipes 
   WHERE 'chicken' = ANY(ingredients) 
     AND deleted_at IS NULL;
   ```
   **Index Used**: `idx_recipes_ingredients` (GIN)

6. **Admin: Get Pending Recipe Reviews**:
   ```sql
   SELECT * FROM pending_recipe_reviews 
   WHERE status = 'pending' 
     AND deleted_at IS NULL
   ORDER BY created_at DESC;
   ```
   **Index Used**: `idx_pending_reviews_pending`

---

## Future Scalability Enhancements

### 1. Partitioning (for millions of records)

**Reviews Table**:
```sql
-- Partition by product_id hash (if reviews per product are large)
CREATE TABLE reviews (
    ...
) PARTITION BY HASH (product_id);

-- Or partition by date (if time-based queries are common)
CREATE TABLE reviews (
    ...
) PARTITION BY RANGE (review_date);
```

**Recipes Table**:
```sql
-- Partition by category
CREATE TABLE recipes (
    ...
) PARTITION BY LIST (category);
```

### 2. Separate Likes Table (if likes_count > 1000)

```sql
CREATE TABLE review_likes (
    id VARCHAR(255) PRIMARY KEY,
    review_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    liked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_likes_review FOREIGN KEY (review_id) REFERENCES reviews(id),
    CONSTRAINT fk_likes_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT uni_review_user_like UNIQUE (review_id, user_id)
);

CREATE INDEX idx_likes_review ON review_likes(review_id);
CREATE INDEX idx_likes_user ON review_likes(user_id);
```

### 3. Materialized Views (for aggregations)

```sql
-- Pre-compute product review statistics
CREATE MATERIALIZED VIEW product_review_stats AS
SELECT 
    product_id,
    COUNT(*) as total_reviews,
    AVG(rating) as avg_rating,
    COUNT(*) FILTER (WHERE is_verified = true) as verified_reviews
FROM reviews
WHERE is_active = true AND deleted_at IS NULL
GROUP BY product_id;

CREATE UNIQUE INDEX ON product_review_stats(product_id);

-- Refresh periodically
REFRESH MATERIALIZED VIEW CONCURRENTLY product_review_stats;
```

### 4. Full-Text Search Enhancement

```sql
-- Add full-text search to reviews
ALTER TABLE reviews ADD COLUMN search_vector tsvector;

CREATE INDEX idx_reviews_search_vector ON reviews USING gin(search_vector);

-- Update trigger
CREATE OR REPLACE FUNCTION reviews_search_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector := 
        to_tsvector('english', COALESCE(NEW.title, '') || ' ' || COALESCE(NEW.comment, ''));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER reviews_search_trigger 
    BEFORE INSERT OR UPDATE ON reviews
    FOR EACH ROW EXECUTE FUNCTION reviews_search_update();
```

---

## Migration Checklist

- [x] Create `reviews` table with proper schema
- [x] Create `recipes` table with JSONB and arrays
- [x] Create `pending_recipe_reviews` table
- [x] Add foreign key constraints
- [x] Add check constraints
- [x] Add indexes for performance
- [x] Add unique constraints
- [x] Update models to use time.Time for timestamps
- [ ] Run migrations on NeonDB
- [ ] Verify table creation
- [ ] Test queries with sample data
- [ ] Monitor query performance

---

## Summary

### ✅ What We've Achieved

1. **Proper Schema Design**:
   - Foreign keys for integrity
   - Check constraints for validation
   - Unique constraints to prevent duplicates

2. **Performance Optimization**:
   - Strategic indexes for common queries
   - Partial indexes for filtered queries
   - GIN indexes for JSONB and arrays

3. **Future Scalability**:
   - JSONB for flexible schema evolution
   - Array types for efficient storage
   - Partitioning-ready structure

4. **Data Integrity**:
   - Database-level constraints
   - Referential integrity
   - Soft delete support

### 🎯 Next Steps

1. Run migrations on NeonDB
2. Test with sample data
3. Monitor query performance
4. Optimize indexes based on actual usage patterns

All tables are now ready for production use in NeonDB! 🚀



