# Database Structure Changes - Phone Authentication Implementation

## Overview
This document outlines the database structure changes made to support phone number authentication alongside existing email/password and Google Sign-In methods.

## Changes Made

### 1. Users Table Structure Changes

#### Before (Email-Only Authentication):
```sql
CREATE TABLE users (
    id VARCHAR(255) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    firebase_uid VARCHAR(255) UNIQUE,
    phone_number VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    is_verified BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    role VARCHAR(20) DEFAULT 'user',
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    -- ... other base model fields
);
```

#### After (Multi-Authentication Support):
```sql
CREATE TABLE users (
    id VARCHAR(255) PRIMARY KEY,
    email VARCHAR(255) UNIQUE,                    -- Made optional
    password VARCHAR(255),                        -- Made optional
    firebase_uid VARCHAR(255),                    -- Made nullable with proper constraint
    phone_number VARCHAR(255) UNIQUE NOT NULL,    -- Made required and unique
    name VARCHAR(255) NOT NULL,
    is_verified BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    role VARCHAR(20) DEFAULT 'user',
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    -- ... other base model fields
);
```

### 2. Key Changes Explained

#### Email Field Changes:
- **Before**: `email VARCHAR(255) UNIQUE NOT NULL`
- **After**: `email VARCHAR(255) UNIQUE` (nullable)
- **Reason**: Phone-only users don't need email addresses

#### Password Field Changes:
- **Before**: `password VARCHAR(255) NOT NULL`
- **After**: `password VARCHAR(255)` (nullable)
- **Reason**: Phone-only users don't have passwords

#### Phone Number Field Changes:
- **Before**: `phone_number VARCHAR(255)` (optional)
- **After**: `phone_number VARCHAR(255) UNIQUE NOT NULL` (required)
- **Reason**: Phone number is now the primary identifier for phone authentication

#### Firebase UID Field Changes:
- **Before**: `firebase_uid VARCHAR(255) UNIQUE` (but didn't handle NULL properly)
- **After**: `firebase_uid VARCHAR(255)` with proper NULL handling
- **Reason**: Multiple users can have NULL firebase_uid, but non-NULL values must be unique

### 3. Constraints and Indexes

#### New Constraints:
```sql
-- Phone number must be unique and not null
ALTER TABLE users ADD CONSTRAINT users_phone_number_unique UNIQUE (phone_number);
ALTER TABLE users ALTER COLUMN phone_number SET NOT NULL;

-- Firebase UID unique when not null (allows multiple NULL values)
CREATE UNIQUE INDEX uni_users_firebase_uid ON users (firebase_uid) WHERE firebase_uid IS NOT NULL;

-- Email unique when not null (allows multiple NULL values)
CREATE UNIQUE INDEX uni_users_email ON users (email) WHERE email IS NOT NULL;

-- Either email or phone_number must be provided
ALTER TABLE users ADD CONSTRAINT users_email_or_phone_check 
CHECK (email IS NOT NULL OR phone_number IS NOT NULL);
```

#### Performance Indexes:
```sql
CREATE INDEX idx_users_phone_number ON users(phone_number);
CREATE INDEX idx_users_firebase_uid ON users(firebase_uid) WHERE firebase_uid IS NOT NULL;
CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
```

### 4. Data Migration Process

#### Step 1: Handle Existing Data
- Updated all users with NULL/empty phone numbers to have unique phone numbers
- Fixed duplicate phone numbers by appending row numbers
- Converted empty firebase_uid values to NULL

#### Step 2: Schema Changes
- Made email and password nullable
- Made phone_number required and unique
- Fixed firebase_uid constraint to properly handle NULL values

#### Step 3: Add Constraints
- Added unique constraints with proper NULL handling
- Added check constraints to ensure data integrity
- Added performance indexes

### 5. Authentication Methods Supported

#### 1. Email/Password Authentication:
- **Fields Used**: `email`, `password`
- **User Type**: Traditional users with email and password
- **Example**: `email: "user@example.com", password: "hashed_password"`

#### 2. Phone Number Authentication:
- **Fields Used**: `phone_number`
- **User Type**: Phone-only users
- **Example**: `phone_number: "+1234567890"`

#### 3. Google Sign-In:
- **Fields Used**: `email`, `firebase_uid`
- **User Type**: Users who sign in with Google
- **Example**: `email: "user@gmail.com", firebase_uid: "google_firebase_uid"`

### 6. User Creation Patterns

#### Email User:
```go
user := &User{
    Email:    "user@example.com",
    Password: "hashed_password",
    Name:     "John Doe",
    // phone_number will be set to unique value
}
```

#### Phone User:
```go
user := &User{
    PhoneNumber: "+1234567890",
    Name:        "John Doe",
    IsVerified:  true, // Phone verification means user is verified
    // email and password will be NULL
}
```

#### Google User:
```go
user := &User{
    Email:       "user@gmail.com",
    FirebaseUID: "google_firebase_uid",
    Name:        "John Doe",
    // phone_number will be set to unique value
}
```

### 7. API Response Changes

#### Before (Email Authentication):
```json
{
  "user": {
    "id": "USER123",
    "email": "user@example.com",
    "name": "John Doe",
    "firebase_uid": "firebase_uid",
    "is_verified": false,
    "is_active": true
  }
}
```

#### After (Phone Authentication):
```json
{
  "user": {
    "id": "USER123",
    "phone_number": "+1234567890",
    "name": "John Doe",
    "firebase_uid": "firebase_uid",
    "is_verified": true,
    "is_active": true
  }
}
```

### 8. Migration Files

#### Main Migration:
- `migrations/cleanup_user_table_structure.sql` - Complete database restructure

#### Previous Migrations:
- `migrations/update_user_phone_auth.sql` - Initial phone auth setup
- `migrations/add_multiple_images_columns.sql` - Image support
- `migrations/add_delhivery_columns.sql` - Delivery integration

### 9. Auto-Migration Updates

#### GORM Model Changes:
```go
type User struct {
    *base.BaseModel `gorm:"embedded"`
    Email           string  `json:"email" gorm:"unique"`           // Made optional
    Name            string  `json:"name" gorm:"not null"`
    Password        string  `json:"-"`                             // Made optional
    FirebaseUID     *string `json:"firebase_uid,omitempty"`        // Made pointer for NULL
    PhoneNumber     string  `json:"phone_number" gorm:"unique"`    // Made required
    IsVerified      bool    `json:"is_verified" gorm:"default:false"`
    IsActive        bool    `json:"is_active" gorm:"default:true"`
    Role            string  `json:"role" gorm:"type:varchar(20);default:user;index"`
}
```

### 10. Testing Considerations

#### Test Data Setup:
```sql
-- Email user
INSERT INTO users (id, email, password, name, phone_number) 
VALUES ('USER001', 'test@example.com', 'hashed_password', 'Test User', '+1111111111');

-- Phone user
INSERT INTO users (id, phone_number, name, is_verified) 
VALUES ('USER002', '+2222222222', 'Phone User', true);

-- Google user
INSERT INTO users (id, email, firebase_uid, name, phone_number) 
VALUES ('USER003', 'google@gmail.com', 'google_uid', 'Google User', '+3333333333');
```

### 11. Rollback Plan

If rollback is needed:
```sql
-- Remove new constraints
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_phone_number_unique;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_or_phone_check;
DROP INDEX IF EXISTS uni_users_firebase_uid;
DROP INDEX IF EXISTS uni_users_email;

-- Restore original structure
ALTER TABLE users ALTER COLUMN email SET NOT NULL;
ALTER TABLE users ALTER COLUMN password SET NOT NULL;
ALTER TABLE users ALTER COLUMN phone_number DROP NOT NULL;

-- Remove unique constraint from phone_number
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_phone_number_unique;
```

### 12. Performance Impact

#### Positive Impacts:
- Faster phone number lookups with dedicated index
- Better query performance for authentication
- Proper NULL handling reduces constraint violations

#### Considerations:
- Additional storage for phone numbers
- More complex queries for multi-authentication support
- Index maintenance overhead

### 13. Security Considerations

#### Phone Number Privacy:
- Phone numbers are stored in plain text (consider encryption for production)
- Unique constraint prevents duplicate accounts
- Proper validation on phone number format

#### Firebase Integration:
- Firebase UID properly linked to local user records
- NULL values handled correctly for users without Firebase accounts
- Token validation through Firebase Admin SDK

---

## Summary

The database structure has been successfully updated to support multiple authentication methods while maintaining backward compatibility. The changes enable:

1. **Email/Password Authentication** (existing)
2. **Phone Number Authentication** (new)
3. **Google Sign-In** (existing)
4. **Proper NULL handling** for optional fields
5. **Data integrity** through appropriate constraints
6. **Performance optimization** through strategic indexing

All changes are documented and can be rolled back if needed. The system now supports elder users who prefer phone number authentication over email-based systems.






