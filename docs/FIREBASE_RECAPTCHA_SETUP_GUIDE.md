# Firebase reCAPTCHA Configuration Guide

## 🔧 **Required Firebase Console Configuration**

### **Step 1: Enable Phone Authentication**

1. **Go to Firebase Console**
   - Navigate to: https://console.firebase.google.com/
   - Select your project: `amrti-m1355`

2. **Enable Phone Authentication**
   - Go to **Authentication** → **Sign-in method**
   - Find **Phone** in the providers list
   - Click on **Phone** to configure it
   - Toggle **Enable** to **ON**
   - Click **Save**

### **Step 2: Configure Authorized Domains** ⚠️ **CRITICAL**

This is the most important step for reCAPTCHA to work:

1. **Navigate to Authorized Domains**
   - Go to **Authentication** → **Settings** → **Authorized domains**

2. **Add Required Domains**
   ```
   localhost
   127.0.0.1
   amrti-m1355.firebaseapp.com
   ```

3. **For Production** (add when deploying):
   ```
   yourdomain.com
   www.yourdomain.com
   ```

### **Step 3: Configure reCAPTCHA Settings**

1. **Go to reCAPTCHA Settings**
   - In **Authentication** → **Settings** → **reCAPTCHA**

2. **Enable reCAPTCHA**
   - Make sure reCAPTCHA is **enabled**
   - Choose **reCAPTCHA v3** (recommended for invisible)
   - Or **reCAPTCHA v2** (for visible widget)

### **Step 4: Get Firebase Web API Key**

1. **Go to Project Settings**
   - Click the gear icon ⚙️ next to "Project Overview"
   - Select **Project settings**

2. **Find Your Web App**
   - Scroll down to **Your apps** section
   - Find your web app or create one if needed

3. **Copy the Web API Key**
   - Copy the **Web API Key**: `AIzaSyCPzNrFV3Q6a4rcAQF-60G3h3K1WBQqQFI`
   - This should match what's in your HTML file

### **Step 5: Set Backend Environment Variable**

Your backend needs the Firebase Web API Key to call Firebase REST API:

1. **Set Environment Variable**
   ```bash
   export FIREBASE_WEB_API_KEY="AIzaSyCPzNrFV3Q6a4rcAQF-60G3h3K1WBQqQFI"
   ```

2. **Or add to your .env file**:
   ```
   FIREBASE_WEB_API_KEY=AIzaSyCPzNrFV3Q6a4rcAQF-60G3h3K1WBQqQFI
   ```

3. **Or set in your deployment environment**

## 🧪 **Testing Configuration**

### **Test 1: Check Authorized Domains**
- Open your app in browser
- Check browser console for any domain-related errors
- Should see: "reCAPTCHA verified successfully"

### **Test 2: Test Phone Authentication**
1. Enter a valid phone number: `+1234567890`
2. Complete reCAPTCHA challenge
3. Should receive SMS (or see test mode message)

### **Test 3: Check Backend Connection**
- Your HTML page will automatically test backend connectivity
- Should see: "Backend is running and accessible"

## 🚨 **Common Issues & Solutions**

### **Issue 1: "reCAPTCHA verification failed"**
**Solution**: Add `localhost` to authorized domains

### **Issue 2: "Invalid reCAPTCHA token"**
**Solution**: Ensure reCAPTCHA is enabled in Firebase Console

### **Issue 3: "Firebase Web API Key not configured"**
**Solution**: Set `FIREBASE_WEB_API_KEY` environment variable

### **Issue 4: "Domain not authorized"**
**Solution**: Add your domain to authorized domains list

## 📱 **Development vs Production**

### **Development Setup**
```
Authorized Domains:
- localhost
- 127.0.0.1
- amrti-m1355.firebaseapp.com
```

### **Production Setup**
```
Authorized Domains:
- yourdomain.com
- www.yourdomain.com
- yourdomain.firebaseapp.com
```

## 🔍 **Verification Checklist**

- [ ] Phone authentication enabled in Firebase Console
- [ ] `localhost` added to authorized domains
- [ ] reCAPTCHA enabled in Firebase Console
- [ ] Firebase Web API Key set in backend environment
- [ ] Backend server running and accessible
- [ ] Frontend can connect to backend API

## 🎯 **Expected Behavior After Setup**

1. **Page Load**: "Backend is running and accessible"
2. **Phone Input**: Auto-formats with country code
3. **reCAPTCHA**: Loads and displays properly
4. **Send Code**: "Verification code sent! Check your phone"
5. **OTP Input**: Auto-verifies when 6 digits entered
6. **Success**: "Phone authentication successful! Welcome!"

## 📞 **Test Phone Numbers (Development)**

For testing without real SMS, add these in Firebase Console:
- Phone: `+1234567890`, Code: `123456`
- Phone: `+1987654321`, Code: `654321`

## 🔗 **Useful Links**

- [Firebase Console](https://console.firebase.google.com/)
- [Firebase Phone Auth Docs](https://firebase.google.com/docs/auth/web/phone-auth)
- [reCAPTCHA Configuration](https://firebase.google.com/docs/auth/web/phone-auth#reCAPTCHA)
