# Coupon Table Normalization Summary

## Overview
This document summarizes the normalization of the coupons table structure, moving from a denormalized JSON-based rules system to a properly normalized relational structure.

## Changes Made

### 1. Database Schema Changes

#### Before (Denormalized)
```sql
coupons (
    id UUID PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    rules JSONB NOT NULL,  -- Contains all rule data in JSON
    count INTEGER DEFAULT 0,
    used_count INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_by VARCHAR(255),
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

#### After (Normalized)
```sql
-- Main coupons table (simplified)
coupons (
    id VARCHAR(255) PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    count INTEGER DEFAULT 0 CHECK (count >= 0),
    used_count INTEGER DEFAULT 0 CHECK (used_count >= 0 AND used_count <= count),
    is_active BOOLEAN DEFAULT true,
    created_by VARCHAR(255),
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- New normalized rules table
coupon_rules (
    id VARCHAR(255) PRIMARY KEY,
    coupon_id VARCHAR(255) NOT NULL,
    rule_type VARCHAR(20) NOT NULL CHECK (rule_type IN ('percentage', 'fixed', 'free_shipping')),
    discount_value DECIMAL(10,2) NOT NULL CHECK (discount_value >= 0),
    min_amount DECIMAL(10,2) DEFAULT 0 CHECK (min_amount >= 0),
    max_discount DECIMAL(10,2),
    valid_from TIMESTAMP,
    valid_until TIMESTAMP,
    user_limit INTEGER,
    first_time_only BOOLEAN DEFAULT false,
    categories JSONB,
    products JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_coupon_rules_coupon_id FOREIGN KEY (coupon_id) REFERENCES coupons(id) ON DELETE CASCADE
);
```

### 2. Model Changes

#### New CouponRule Model
- **File**: `entities/models/coupon_rule.go`
- **Purpose**: Represents individual coupon rules in normalized form
- **Features**:
  - Proper validation methods
  - Discount calculation logic
  - Category/product filtering
  - Validity checking

#### Updated Coupon Model
- **File**: `entities/models/coupon.go`
- **Changes**:
  - Removed JSON `Rules` field
  - Added `Rules []CouponRule` relationship
  - Updated methods to work with multiple rules
  - Added `GetBestRule()` method for optimal discount selection

### 3. Repository Layer Changes

#### New CouponRuleRepository
- **File**: `internal/repositories/coupon_rule_repository.go`
- **Features**:
  - CRUD operations for coupon rules
  - Query by coupon ID, type, category, product
  - Validation queries for cart compatibility

#### Updated CouponRepository
- **File**: `internal/repositories/coupon_repository.go`
- **New Methods**:
  - `GetWithRules()` - Get coupon with all rules
  - `GetByCodeWithRules()` - Get coupon by code with rules
  - `GetAllWithRules()` - Get all coupons with rules
  - `GetActiveWithRules()` - Get active coupons with rules

### 4. Service Layer Changes

#### Updated CouponService
- **File**: `internal/services/coupon_service.go`
- **Changes**:
  - Updated to work with normalized structure
  - Added coupon rule management methods
  - Enhanced validation logic
  - Better error handling and rollback support

#### New Methods Added:
- `CreateCouponRule()`
- `UpdateCouponRule()`
- `DeleteCouponRule()`

### 5. API Layer Changes

#### Updated Request Models
- **File**: `entities/requests/coupon_requests.go`
- **New Models**:
  - `CreateCouponRuleRequest`
  - `UpdateCouponRuleRequest`

#### Updated Handlers
- **File**: `internal/handlers/coupon_handler.go`
- **New Endpoints**:
  - `POST /api/v1/coupons/rules` - Create coupon rule
  - `PUT /api/v1/coupons/rules` - Update coupon rule
  - `DELETE /api/v1/coupons/rules/{id}` - Delete coupon rule

#### Updated Routes
- **File**: `internal/routes/coupon_routes.go`
- **Changes**:
  - Added coupon rule management routes
  - Updated service initialization to include CouponRuleRepository

### 6. Migration Scripts

#### Forward Migration
- **File**: `migrations/normalize_coupons_table.sql`
- **Purpose**: Migrate from JSON rules to normalized structure
- **Features**:
  - Creates new `coupon_rules` table
  - Migrates existing JSON data to normalized format
  - Adds proper constraints and indexes
  - Includes verification queries

#### Rollback Migration
- **File**: `migrations/rollback_normalize_coupons_table.sql`
- **Purpose**: Revert to original JSON-based structure
- **Features**:
  - Recreates JSON rules column
  - Migrates data back to JSON format
  - Drops normalized tables
  - Removes added constraints

## Benefits of Normalization

### 1. Database Benefits
- **Proper Normalization**: Eliminates 1NF violations
- **Better Querying**: Can query specific rule conditions efficiently
- **Improved Indexing**: Can create indexes on rule fields
- **Data Integrity**: Foreign key constraints ensure referential integrity

### 2. Performance Benefits
- **Faster Queries**: Indexed rule fields improve query performance
- **Better Caching**: Individual rules can be cached separately
- **Reduced Storage**: Eliminates JSON overhead for simple queries

### 3. Development Benefits
- **Type Safety**: Strong typing for rule fields
- **Better Validation**: Database-level constraints
- **Easier Testing**: Individual rules can be tested in isolation
- **Maintainability**: Clear separation of concerns

### 4. Business Benefits
- **Multiple Rules**: Support for multiple rules per coupon
- **Flexible Discounts**: Easy to add new discount types
- **Better Analytics**: Can analyze rule usage patterns
- **Audit Trail**: Individual rule changes can be tracked

## API Usage Examples

### Creating a Coupon with Rules
```bash
# 1. Create the coupon
POST /api/v1/coupons
{
  "code": "SAVE20",
  "description": "20% off for new users",
  "count": 100,
  "created_by": "admin"
}

# 2. Add a rule to the coupon
POST /api/v1/coupons/rules
{
  "coupon_id": "COUP_123",
  "rule_type": "percentage",
  "discount_value": 20,
  "min_amount": 100,
  "max_discount": 50,
  "first_time_only": true
}
```

### Getting Coupon with Rules
```bash
GET /api/v1/coupons/SAVE20
# Returns coupon with all associated rules
```

### Validating a Coupon
```bash
POST /api/v1/coupons/validate
{
  "code": "SAVE20",
  "cart_total": 150,
  "user_id": "USER_456"
}
# Returns best applicable rule and discount amount
```

## Migration Strategy

### 1. Pre-Migration
- Backup existing data
- Test migration on staging environment
- Verify all existing coupons have rules

### 2. Migration Execution
- Run `normalize_coupons_table.sql`
- Verify data integrity
- Test API endpoints

### 3. Post-Migration
- Update application code
- Deploy new version
- Monitor for issues

### 4. Rollback Plan
- If issues arise, run `rollback_normalize_coupons_table.sql`
- Revert to previous application version
- Restore from backup if necessary

## Testing Checklist

- [ ] Migration script runs successfully
- [ ] All existing coupons have corresponding rules
- [ ] API endpoints work with new structure
- [ ] Coupon validation works correctly
- [ ] Discount calculations are accurate
- [ ] Rollback migration works
- [ ] Performance is acceptable
- [ ] No data loss during migration

## Future Enhancements

1. **Rule Templates**: Predefined rule configurations
2. **Rule Inheritance**: Rules that apply to multiple coupons
3. **Rule Scheduling**: Time-based rule activation
4. **Rule Analytics**: Usage statistics per rule
5. **Rule Versioning**: Track rule changes over time

## Conclusion

The normalization of the coupons table provides a solid foundation for a more flexible and maintainable coupon system. The new structure supports multiple rules per coupon, better querying capabilities, and improved data integrity while maintaining backward compatibility through the API layer.

