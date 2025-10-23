# Phone Number Authentication Setup Guide

This guide explains how to set up phone number authentication with Firebase for your ecommerce backend.

## Backend Changes Completed ✅

The following backend changes have been implemented:

### 1. Database Schema Updates
- **User Model**: Made `email` optional and `phone_number` required
- **Migration**: Created `update_user_phone_auth.sql` to update existing database
- **New Constructor**: Added `NewPhoneUser()` for phone-only users

### 2. API Endpoints Added
- `POST /api/v1/auth/phone/send-code` - Send SMS verification code
- `POST /api/v1/auth/phone/verify-code` - Verify SMS code and authenticate
- `POST /api/v1/auth/phone/signup` - Alternative phone signup

### 3. Request Models
- `PhoneSignupRequest` - For phone-based signup
- `PhoneLoginRequest` - For phone-based login
- `SendPhoneCodeRequest` - For sending verification codes
- `VerifyPhoneCodeRequest` - For verifying SMS codes

### 4. Service Layer Updates
- `CreatePhoneUser()` - Creates users with phone numbers only
- `GetUserByPhoneNumber()` - Retrieves users by phone number
- Repository method `GetUserByPhoneNumber()` added

## Firebase Console Configuration Required 🔧

### Step 1: Enable Phone Authentication

1. **Go to Firebase Console**
   - Navigate to [Firebase Console](https://console.firebase.google.com/)
   - Select your project

2. **Enable Phone Authentication**
   - Go to **Authentication** → **Sign-in method**
   - Find **Phone** in the providers list
   - Click on **Phone** to configure it
   - Toggle **Enable** to ON
   - Click **Save**

### Step 2: Configure Phone Authentication Settings

1. **Test Phone Numbers (Development)**
   - In the Phone provider settings, scroll down to **Phone numbers for testing**
   - Add test phone numbers in format: `+1234567890`
   - Add corresponding verification codes (e.g., `123456`)
   - This allows testing without sending real SMS

2. **App Verification (Production)**
   - For production, you'll need to configure app verification
   - Add your app's SHA-1/SHA-256 fingerprints
   - Configure reCAPTCHA settings if needed

### Step 3: Configure SMS Provider (Optional)

1. **Default SMS Provider**
   - Firebase uses Google's SMS service by default
   - No additional configuration needed for basic usage

2. **Custom SMS Provider (Advanced)**
   - If you need custom SMS templates or providers
   - Go to **Authentication** → **Templates**
   - Customize SMS templates for different languages

### Step 4: Set Up Billing (Required for SMS)

1. **Enable Billing**
   - Phone authentication requires a paid Firebase plan
   - Go to **Project Settings** → **Usage and billing**
   - Enable billing for your project
   - SMS messages are charged per message sent

2. **Set Usage Limits**
   - Set daily/monthly limits to control costs
   - Monitor usage in the Firebase Console

## Environment Variables Required 🔑

Make sure these environment variables are set in your `.env` file:

```env
# Firebase Configuration
FIREBASE_WEB_API_KEY=your_firebase_web_api_key
FIREBASE_SERVICE_ACCOUNT_PATH=internal/config/firebase_service_account.json

# Database Configuration
DATABASE_URL=your_database_connection_string
```

## API Usage Examples 📱

### 1. Send Verification Code

```bash
curl -X POST http://localhost:8080/api/v1/auth/phone/send-code \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "+1234567890"
  }'
```

**Response:**
```json
{
  "success": true,
  "message": "Verification code sent successfully",
  "data": {
    "session_info": "session_info_string"
  }
}
```

### 2. Verify Code and Authenticate

```bash
curl -X POST http://localhost:8080/api/v1/auth/phone/verify-code \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "+1234567890",
    "code": "123456"
  }'
```

**Response:**
```json
{
  "success": true,
  "message": "Phone authentication successful",
  "data": {
    "id_token": "firebase_id_token",
    "refresh_token": "firebase_refresh_token",
    "expires_in": "3600",
    "user": {
      "uid": "firebase_uid",
      "phone_number": "+1234567890",
      "is_new_user": true,
      "id": "local_user_id",
      "name": "User",
      "is_verified": true,
      "is_active": true,
      "created_at": "2025-01-27T10:00:00Z"
    }
  }
}
```

### 3. Alternative Phone Signup

```bash
curl -X POST http://localhost:8080/api/v1/auth/phone/signup \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "+1234567890",
    "name": "John Doe"
  }'
```

## Database Migration 🗄️

Run the migration to update your database schema:

```sql
-- Run the migration file
\i migrations/update_user_phone_auth.sql
```

**Important Notes:**
- The migration makes `email` optional and `phone_number` required
- Existing users will need phone numbers added
- Test the migration on a development database first

## Testing Strategy 🧪

### 1. Development Testing
- Use Firebase test phone numbers
- No real SMS will be sent
- Use predefined verification codes

### 2. Production Testing
- Test with real phone numbers
- Monitor Firebase Console for SMS delivery
- Check billing usage

### 3. Error Handling
- Test invalid phone numbers
- Test expired verification codes
- Test rate limiting scenarios

## Security Considerations 🔒

1. **Rate Limiting**
   - Firebase has built-in rate limiting
   - Monitor for abuse in Firebase Console

2. **Phone Number Validation**
   - Validate phone number format on frontend
   - Use international format (+country_code_number)

3. **Session Management**
   - Store session info securely
   - Implement proper token refresh

## Troubleshooting 🔧

### Common Issues:

1. **"Phone authentication not enabled"**
   - Check Firebase Console → Authentication → Sign-in method
   - Ensure Phone provider is enabled

2. **"Invalid phone number"**
   - Use international format: +1234567890
   - Check phone number validation

3. **"SMS not received"**
   - Check Firebase Console for delivery status
   - Verify billing is enabled
   - Check phone number format

4. **"Verification code expired"**
   - Codes expire after 5 minutes
   - Request a new code

## Benefits for Elder Users 👴👵

✅ **No Email Required** - Direct phone number signup
✅ **Simple Process** - Just SMS verification
✅ **Familiar Interface** - Most people know their phone number
✅ **Google Sign-In Still Available** - Multiple options for users

## Next Steps 🚀

1. **Run Database Migration**
2. **Configure Firebase Console**
3. **Test with Development Phone Numbers**
4. **Update Frontend to Use New Endpoints**
5. **Deploy to Production**
6. **Monitor Usage and Billing**

## Support 📞

For issues or questions:
- Check Firebase Console logs
- Review backend logs for errors
- Test with Firebase test phone numbers first
- Monitor Firebase billing usage

