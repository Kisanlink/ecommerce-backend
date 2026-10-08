# Firebase Authentication Setup Guide

## Problem Solved: MISSING_CLIENT_IDENTIFIER Error

The "MISSING_CLIENT_IDENTIFIER" error was occurring because Firebase phone authentication requires a `recaptchaToken` for security purposes. This has been fixed in the backend.

## What Was Fixed

### 1. Backend Changes
- ✅ Updated `SendPhoneCodeRequest` to include `recaptchaToken` field
- ✅ Modified `FirebasePhoneRequest` to include `recaptchaToken`
- ✅ Enhanced error handling for phone authentication
- ✅ Added specific error messages for `MISSING_CLIENT_IDENTIFIER` and `INVALID_RECAPTCHA_TOKEN`

### 2. Frontend Test File
- ✅ Created comprehensive `index.html` with all authentication methods
- ✅ Implemented proper reCAPTCHA integration for phone authentication
- ✅ Added email, Google, and phone authentication buttons
- ✅ Included proper error handling and user feedback

## Setup Instructions

### Step 1: Configure Firebase Console

1. **Go to Firebase Console**: https://console.firebase.google.com/
2. **Select your project**: `amrti-m1355`
3. **Enable Authentication Methods**:
   - Go to **Authentication** → **Sign-in method**
   - Enable **Email/Password**
   - Enable **Google**
   - Enable **Phone**

### Step 2: Get Firebase Configuration

1. **Go to Project Settings** (gear icon)
2. **Scroll to "Your apps"** section
3. **Add a web app** if you don't have one:
   - Click **Add app** → Web icon (`</>`)
   - Register your app
4. **Copy the config object**:
   ```javascript
   const firebaseConfig = {
     apiKey: "your-api-key",
     authDomain: "amrti-m1355.firebaseapp.com",
     projectId: "amrti-m1355",
     storageBucket: "amrti-m1355.appspot.com",
     messagingSenderId: "your-sender-id",
     appId: "your-app-id"
   };
   ```

### Step 3: Update Environment Variables

Set these environment variables in your system or `.env` file:

```env
# Required for all authentication methods
FIREBASE_WEB_API_KEY=your_actual_firebase_web_api_key_here

# Optional: If using environment variables instead of service account file
FIREBASE_PROJECT_ID=amrti-m1355
FIREBASE_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n"
FIREBASE_CLIENT_EMAIL=firebase-adminsdk-fbsvc@amrti-m1355.iam.gserviceaccount.com
```

### Step 4: Configure the HTML Test File

1. **Open `index.html`** in your project root
2. **Replace the Firebase config** with your actual values:
   ```javascript
   const firebaseConfig = {
     apiKey: "YOUR_ACTUAL_API_KEY",
     authDomain: "amrti-m1355.firebaseapp.com",
     projectId: "amrti-m1355",
     storageBucket: "amrti-m1355.appspot.com",
     messagingSenderId: "YOUR_ACTUAL_SENDER_ID",
     appId: "YOUR_ACTUAL_APP_ID"
   };
   ```

### Step 5: Test Authentication Methods

1. **Start your backend server**:
   ```bash
   cd ecommerce-backend
   go run cmd/server/main.go
   ```

2. **Open `index.html`** in a web browser

3. **Test each authentication method**:
   - **Email**: Enter email and password, click "Sign In with Email"
   - **Google**: Click "Sign In with Google" and follow OAuth flow
   - **Phone**: Enter phone number, click "Send Verification Code", enter the code

## Phone Authentication Specific Setup

### For Testing (Development)
1. **Go to Firebase Console** → **Authentication** → **Sign-in method** → **Phone**
2. **Add test phone numbers**:
   - Phone number: `+1234567890`
   - Verification code: `123456`
3. **Use these test numbers** in your HTML file

### For Production
1. **Enable reCAPTCHA** in Firebase Console
2. **Add authorized domains** in Firebase Console → **Authentication** → **Settings**
3. **Configure reCAPTCHA** for your domain

## Troubleshooting

### Common Issues

1. **"MISSING_CLIENT_IDENTIFIER" Error**:
   - ✅ **Fixed**: Now requires `recaptchaToken` in requests
   - Ensure reCAPTCHA is properly configured in Firebase Console

2. **"Firebase Web API Key not configured"**:
   - Set `FIREBASE_WEB_API_KEY` environment variable
   - Restart your backend server

3. **"Phone authentication not enabled"**:
   - Enable Phone authentication in Firebase Console
   - Ensure you're on Blaze plan (pay-as-you-go)

4. **"Invalid phone number format"**:
   - Use international format: `+1234567890`
   - Include country code

### Testing Endpoints

You can test the backend endpoints directly:

```bash
# Send phone verification code
curl -X POST http://localhost:8080/api/v1/auth/phone/send-code \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "+1234567890",
    "recaptcha_token": "test-token"
  }'

# Check Firebase status
curl http://localhost:8080/debug/firebase
```

## Security Notes

1. **Never expose your Firebase config** in production
2. **Use environment variables** for sensitive data
3. **Enable App Check** for additional security
4. **Configure authorized domains** properly
5. **Use HTTPS** in production

## Next Steps

1. **Test all authentication methods** using the HTML file
2. **Integrate with your frontend application**
3. **Set up proper error handling** in your app
4. **Configure production settings** when ready to deploy

The "MISSING_CLIENT_IDENTIFIER" error should now be resolved with the proper reCAPTCHA token implementation!
