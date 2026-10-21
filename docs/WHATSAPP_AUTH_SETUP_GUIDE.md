# WhatsApp Business API (Meta) Authentication Setup Guide

This step-by-step guide is designed for developers who are new to Meta Business Manager and WhatsApp Cloud API. It walks through creating the required WhatsApp Authentication message template, configuring tokens, and setting environment variables.

---

## 1. Prerequisites Checklist

Ensure you have:
1. A **Meta for Developers** account ([developers.facebook.com](https://developers.facebook.com/)).
2. A **Meta Business Portfolio / Business Manager** ([business.facebook.com](https://business.facebook.com/)).
3. Access to your active **WhatsApp Business Account (WABA)**.
4. Your current credentials (already used in production for order confirmation messages):
   - **Phone Number ID**: `824983564042346`
   - **Business Account ID (WABA ID)**: `1344632900291362`
   - **Permanent System User Access Token**

---

## 2. Understanding WhatsApp Message Types for OTP

WhatsApp Meta API supports two ways to send OTPs:

| Type | How It Works | Best For | Requirement |
|---|---|---|---|
| **Authentication Template** (Recommended) | Official pre-approved template with a 1-tap "Copy Code" button directly inside WhatsApp. | Production & Live Customers | Template approval by Meta (usually approved within minutes). |
| **Plain Text Message** (Fallback) | Standard text message sent using WhatsApp Cloud API endpoint. | Development / Testing numbers | Allowed during 24h customer support window or test numbers. |

The backend is configured to **automatically use the template if configured**, and gracefully fall back to text message if the template variable is left blank.

---

## 3. Step-by-Step: Creating the Authentication Template in Meta

Follow these steps in your browser:

### Step 3.1: Navigate to Message Templates
1. Go to the [WhatsApp Manager Templates Page](https://business.facebook.com/wa/manage/message-templates/).
2. If prompted, select your Business Portfolio (e.g., *Amriti Foods* / *Kisanlink*).
3. In the top right corner, click the blue **Create template** button.

---

### Step 3.2: Select Category & Name
1. **Category**: Choose **Authentication**.
2. **Name**: Enter `amrti_auth_otp` (lowercase letters and underscores only).
3. **Language**: Choose **English** (or `en_US` / `en`).
4. Click **Continue**.

---

### Step 3.3: Configure Template Content
1. **Code Delivery Method**:
   - Choose **Copy code** (this automatically adds a convenient "Copy Code" button on the recipient's phone).
2. **Message Content**:
   - Meta automatically provides the standard authentication text:
     ```
     {{1}} is your verification code. For your security, do not share this code.
     ```
   - Parameter `{{1}}` represents the 6-digit OTP code sent by our backend.
3. **Button Configuration**:
   - Button type: **Copy code**
   - Button text: `Copy code` (Meta default).
4. **Expiration (Optional)**:
   - You can set code expiration to 5 minutes (matching our backend expiration).

---

### Step 3.4: Submit for Review
1. Click **Submit** in the bottom right corner.
2. Meta reviews Authentication templates using automated checks. They are usually approved **within 2 to 15 minutes**.
3. Once approved, the template status will show a green dot with **Approved**.

---

## 4. Environment Variables Setup

Configure these variables in your deployment environment (e.g., Render, AWS, Docker `.env`):

| Variable Name | Required | Default / Example | Description |
|---|---|---|---|
| `WHATSAPP_ENABLED` | **Yes** | `true` | Enables WhatsApp service in backend |
| `WHATSAPP_ACCESS_TOKEN` | **Yes** | `EAA...` | System User Permanent Token with `whatsapp_business_messaging` permission |
| `WHATSAPP_PHONE_NUMBER_ID` | **Yes** | `824983564042346` | ID of the WhatsApp sender phone number |
| `WHATSAPP_BUSINESS_ACCOUNT_ID` | **Yes** | `1344632900291362` | Your WhatsApp Business Account ID |
| `WHATSAPP_API_BASE_URL` | No | `https://graph.facebook.com/v22.0` | Meta Graph API endpoint URL |
| `WHATSAPP_OTP_TEMPLATE_NAME` | **Recommended** | `amrti_auth_otp` | The name of the approved template created in Step 3. If empty, falls back to plain text. |
| `FIREBASE_WEB_API_KEY` | **Yes** | `AIzaSy...` | Firebase Web API key used to exchange custom tokens for ID tokens |
| `ADMIN_SECRET_KEY` | **Yes** | *(secret string)* | Secret key required to call the admin setup endpoint (`POST /api/v1/admin/setup`). If unset, the endpoint returns 500. Must be set in production. |

---

## 5. Finding Your IDs in Meta Business Suite (If Needed)

If you ever need to locate your IDs again:
1. Open [WhatsApp Manager](https://business.facebook.com/wa/manage/home/).
2. In the left navigation, click **Account Tools** -> **Phone numbers**.
3. You will see your registered phone number, its display name, and its **Phone number ID**.
4. In the top URL bar or under **Settings** -> **Basic information**, you will find your **WhatsApp Business Account ID**.

---

## 6. How System User Access Tokens Work

WhatsApp Cloud API uses Bearer tokens to authenticate API requests:
- **Never use temporary tokens (24 hours)** for production.
- Always use a **Permanent System User Token**:
  1. Go to [Meta Business Settings -> System Users](https://business.facebook.com/settings/system-users).
  2. Select your Admin System User.
  3. Ensure it has access to the WhatsApp account with `whatsapp_business_messaging` and `whatsapp_business_management` permissions.
  4. Click **Generate new token**, check the permissions, select **Never expire**, and save the token securely into `WHATSAPP_ACCESS_TOKEN`.

---

## 7. Testing Your Setup

### Quick Test via cURL
Run this command to test sending an OTP verification code:

```bash
curl -X POST https://your-backend-domain.com/api/v1/auth/phone/send-code \
  -H "Content-Type: application/json" \
  -d '{"phone_number": "+91XXXXXXXXXX"}'
```

Expected Response:
```json
{
  "success": true,
  "message": "Verification code sent via WhatsApp successfully",
  "data": {
    "session_info": "c7a82c69-2fb0-4382-95f7-f9d2208033c5"
  }
}
```

Check your WhatsApp on `+91XXXXXXXXXX`. You will receive a message with the 6-digit code.

---

## 8. Common Troubleshooting

| Issue | Cause | Solution |
|---|---|---|
| `WhatsApp service is disabled` (log warning only) | `WHATSAPP_ENABLED` is `false` or missing | The send is skipped silently (returns success, no message sent). Useful for local dev. Set `WHATSAPP_ENABLED=true` for production. |
| `WhatsApp API error (status 401)` | Expired or invalid `WHATSAPP_ACCESS_TOKEN` | Re-generate permanent token in Meta System Users and update env. |
| `WhatsApp API error (status 400): Template not found` | `WHATSAPP_OTP_TEMPLATE_NAME` doesn't match approved template | Check the spelling or case in WhatsApp Manager. Must match exactly (e.g., `amrti_auth_otp`). |
| `Please wait 60 seconds before requesting another code` | Rate limiting triggered | Wait 60 seconds before requesting a new OTP for that phone number. |
| `Too many failed attempts` | More than 5 invalid codes entered | Request a new code to reset the session. |
