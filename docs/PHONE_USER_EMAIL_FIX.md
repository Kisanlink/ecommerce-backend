# Phone User Email Fix

## Problem
Phone-only users were having their phone numbers stored in the `email` field of the profiles table, or fake emails were being generated. Phone users should not need an email - their identity is their mobile number.

## Solution
Made email optional in profiles table and updated the flow so:
1. Phone users can have profiles without email
2. Email is left empty for phone users during profile creation
3. Users can add email later through profile update if they want

## Changes Made

### 1. Profile Model (`entities/models/profile.go`)
- **Before**: `Email string` with `gorm:"size:255;not null"` (required)
- **After**: `Email string` with `gorm:"size:255"` (optional, nullable)

### 2. Profile Service (`internal/services/profile_service.go`)
- **Before**: Generated fake emails like `user_{userID}@kisanlink.local` for phone users
- **After**: Leaves email empty (`""`) for phone users
- Only uses email if:
  - User has email in database, OR
  - Firebase provides a valid email (contains `@`)

### 3. Profile Update Request (`entities/requests/profile_request.go`)
- **Added**: `Email *string` field to allow users to add/update email later
- Users can now add email through profile update endpoint

### 4. Database Migration (`migrations/make_profile_email_nullable.sql`)
- Makes `email` column nullable in `profiles` table
- Removes NOT NULL constraint
- Creates partial unique index for email (only when not NULL)
- Updates existing empty string emails to NULL

## Migration Steps

### Run the migration:
```bash
# Using the provided script
bash migrations/run_make_profile_email_nullable.sh

# Or directly with psql
psql "your_connection_string" -f migrations/make_profile_email_nullable.sql
```

### Verify:
```sql
-- Check profiles with NULL email (phone users)
SELECT COUNT(*) FROM profiles WHERE email IS NULL;

-- Check profiles with email
SELECT COUNT(*) FROM profiles WHERE email IS NOT NULL AND email != '';
```

## New Flow

### Phone User Sign-up/Sign-in:
1. User signs up/signs in with phone number
2. Profile is created with:
   - `first_name`: "User" (default)
   - `last_name`: "" (empty)
   - `email`: "" (empty - no fake email)
   - Other fields: defaults

### Adding Email Later:
Users can add email through profile update:
```json
PUT /api/v1/profiles
{
  "email": "user@example.com"
}
```

## Benefits
1. ✅ Phone users don't need email - mobile number is their identity
2. ✅ No fake/generated emails cluttering the database
3. ✅ Users can optionally add email later if they want
4. ✅ Cleaner data model - email only when provided
5. ✅ Better user experience - no forced email requirement

## Testing

### Test Phone Sign-up:
1. Sign up with phone number
2. Check profile - email should be empty/NULL
3. Update profile with email
4. Check profile - email should be updated

### Test Email Sign-up:
1. Sign up with email
2. Check profile - email should be set
3. Profile should work normally

## Notes
- Email is still stored in `users` table for authentication
- Email in `profiles` table is optional and for display/preferences only
- Phone number is the primary identifier for phone users
- Email can be added/updated anytime through profile update

