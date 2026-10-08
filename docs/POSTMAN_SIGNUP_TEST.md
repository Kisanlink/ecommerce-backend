# Postman Test Guide - Signup with Automatic Profile Creation

## ✅ Profile Creation Status

**YES!** Profiles are now **automatically created** during signup. You don't need a separate endpoint.

### What Happens During Signup:

1. ✅ User created in NeonDB (`users` table)
2. ✅ User created in Firebase Auth
3. ✅ Firebase UID linked to user
4. ✅ **Profile automatically created in NeonDB (`profiles` table)** ← NEW!
5. ✅ User signed in automatically (returns tokens)

---

## 📍 Signup Endpoint

### Endpoint Details

**Method**: `POST`  
**URL**: `http://localhost:8080/api/v1/auth/signup`  
**Content-Type**: `application/json`

### Request Body

```json
{
  "email": "testuser@example.com",
  "password": "password123",
  "name": "Test User"
}
```

### Request Body Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `email` | string | ✅ Yes | Valid email address |
| `password` | string | ✅ Yes | Minimum 6 characters |
| `name` | string | ✅ Yes | User's full name |

---

## 🧪 Postman Setup

### Step 1: Create New Request

1. Open Postman
2. Click **New** → **HTTP Request**
3. Set method to **POST**
4. Enter URL: `http://localhost:8080/api/v1/auth/signup`

### Step 2: Configure Headers

Go to **Headers** tab:
- Key: `Content-Type`
- Value: `application/json`

### Step 3: Add Request Body

Go to **Body** tab:
1. Select **raw**
2. Select **JSON** from dropdown
3. Paste this JSON:

```json
{
  "email": "newuser@example.com",
  "password": "password123",
  "name": "New User"
}
```

### Step 4: Send Request

Click **Send** button.

---

## ✅ Expected Response (Success - 201)

```json
{
  "message": "User created and signed in successfully",
  "user": {
    "id": "USER123456",
    "uid": "firebase_uid_here",
    "email": "newuser@example.com",
    "name": "New User",
    "is_verified": false,
    "is_active": true,
    "created_at": "2025-11-29T12:00:00Z"
  },
  "tokens": {
    "id_token": "eyJhbGciOiJSUzI1NiIs...",
    "refresh_token": "AEu4IL0...",
    "expires_in": "3600"
  }
}
```

---

## 🔍 Verify Profile Creation

### Check Server Logs

After signup, you should see in server logs:
```
✅ User successfully saved to NeonDB: ID=USER123456, Email=newuser@example.com
✅ Profile successfully created in NeonDB during signup: ProfileID=PROF789, UserID=USER123456
```

### Check NeonDB Database

1. Go to https://console.neon.tech
2. Select your project
3. Open SQL Editor
4. Run this query:

```sql
-- Check if user was created
SELECT id, email, name, created_at 
FROM users 
WHERE email = 'newuser@example.com';

-- Check if profile was automatically created
SELECT p.id, p.user_id, p.first_name, p.last_name, p.created_at, u.email
FROM profiles p
JOIN users u ON p.user_id = u.id
WHERE u.email = 'newuser@example.com';
```

You should see:
- ✅ 1 row in `users` table
- ✅ 1 row in `profiles` table (automatically created!)

---

## ❌ Error Responses

### 400 - User Already Exists

```json
{
  "success": false,
  "message": "User already exists",
  "error": "VALIDATION_ERROR: email (User with this email already exists)"
}
```

### 400 - Invalid Request

```json
{
  "success": false,
  "message": "Invalid request payload",
  "error": "Key: 'SignupRequest.Email' Error:Field validation for 'Email' failed on the 'email' tag"
}
```

### 500 - Server Error

```json
{
  "success": false,
  "message": "Failed to create user in database",
  "error": "..."
}
```

---

## 📋 Complete Postman Collection

### Collection JSON (Import to Postman)

```json
{
  "info": {
    "name": "Amrti Ecommerce API",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Signup",
      "request": {
        "method": "POST",
        "header": [
          {
            "key": "Content-Type",
            "value": "application/json"
          }
        ],
        "body": {
          "mode": "raw",
          "raw": "{\n  \"email\": \"testuser@example.com\",\n  \"password\": \"password123\",\n  \"name\": \"Test User\"\n}"
        },
        "url": {
          "raw": "http://localhost:8080/api/v1/auth/signup",
          "protocol": "http",
          "host": ["localhost"],
          "port": "8080",
          "path": ["api", "v1", "auth", "signup"]
        }
      }
    }
  ]
}
```

---

## 🎯 Quick Test Steps

1. **Start your server**: `go run cmd/server/main.go`
2. **Open Postman**
3. **Create POST request**: `http://localhost:8080/api/v1/auth/signup`
4. **Add JSON body**:
   ```json
   {
     "email": "test@example.com",
     "password": "password123",
     "name": "Test User"
   }
   ```
5. **Send request**
6. **Check response** - should return user + tokens
7. **Check server logs** - should show profile creation
8. **Verify in NeonDB** - check `users` and `profiles` tables

---

## ✨ What Gets Created

### In `users` table:
- ✅ User record with email, name, hashed password
- ✅ Firebase UID linked
- ✅ Timestamps and audit fields

### In `profiles` table (AUTOMATIC):
- ✅ Profile record with user_id
- ✅ First name and last name extracted from user name
- ✅ Default values (country: India, language: en, etc.)
- ✅ Linked to user via foreign key

### In Firebase:
- ✅ Firebase Auth user account
- ✅ Email/password authentication enabled

---

## 🔗 Related Endpoints

- **Login**: `POST /api/v1/auth/login`
- **Get Profile**: `GET /api/v1/auth/profile` (requires authentication)
- **Update Profile**: `PUT /api/v1/auth/profile` (requires authentication)

---

**Note**: Profile creation happens automatically - no need to call `/profiles/auto` endpoint anymore! 🎉



