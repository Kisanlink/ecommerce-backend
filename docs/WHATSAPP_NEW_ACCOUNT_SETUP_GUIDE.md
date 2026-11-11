# Complete Beginner's Guide: Setting Up WhatsApp Business Cloud API From Scratch

This guide is specifically written for setting up a brand-new Meta Developer App, WhatsApp Business Account (WABA), and phone number from **zero to production**, with **ZERO code changes** required in the backend or frontend.

---

## ⚡ Quick Answers First

1. **Does creating a new account require code changes?**
   **No! Zero code changes.** The codebase is completely dynamic. You only update the **environment variables** in your hosting dashboard (e.g., Render).
2. **What environment variables will change?**
   Only **3 to 5 values**:
   - `WHATSAPP_ACCESS_TOKEN` (your new permanent system user token)
   - `WHATSAPP_PHONE_NUMBER_ID` (your new phone number ID)
   - `WHATSAPP_BUSINESS_ACCOUNT_ID` (your new WABA ID)
   - `WHATSAPP_OTP_TEMPLATE_NAME` (`amrti_auth_otp`)
   - `WHATSAPP_ORDER_TEMPLATE_NAME` (`order_confirmation` or `mani_test1`)
3. **How many templates does the entire codebase actually use?**
   The application uses **exactly 2 templates in total**:
   - **Template 1: Authentication OTP Template** (For user phone login & signup)
   - **Template 2: Order Confirmation Template** (For order placement, payment confirmation & tracking updates)

---

## 📋 What You Need Before Starting
1. A personal Facebook account (to log into Meta for Developers).
2. A phone number dedicated to WhatsApp Business that is:
   - Capable of receiving SMS or phone call for a one-time OTP verification.
   - **NOT currently active on WhatsApp on your personal phone** (if it is on regular WhatsApp or WhatsApp Business mobile app, delete that WhatsApp account inside the app: *Settings > Account > Delete Account*).
3. A business name (e.g. *Amriti* or *Amriti Foods*).

---

## 🧭 Step-by-Step Walkthrough

```
[Step 1: Meta Dev Account] ➡️ [Step 2: Create Meta App] ➡️ [Step 3: Add WhatsApp Product]
                                                                        ⬇️
[Step 7: Update Render Env] ⬅️ [Step 6: Create Templates] ⬅️ [Step 5: System User Token] ⬅️ [Step 4: Add Phone Number]
```

---

### Step 1: Register as a Meta Developer
1. Go to **[developers.facebook.com](https://developers.facebook.com/)**.
2. Click **Log In** in the top-right corner using your Facebook credentials.
3. Click **Get Started** (or **My Apps**) in the top right.
4. Accept the Terms and verify your account (via phone number/email) if prompted.

---

### Step 2: Create a New Meta App
1. Go to **[developers.facebook.com/apps](https://developers.facebook.com/apps/)**.
2. Click the green button: **Create App**.
3. Under *What do you want your app to do?*:
   - Select **Other** (or **Business** if shown directly) and click **Next**.
4. Under *Select an app type*:
   - Select **Business**.
   - Click **Next**.
5. Fill in App Details:
   - **App name**: e.g., `Amriti Notifications`
   - **App contact email**: your email
   - **Business Portfolio**: Select your Business portfolio if you already have one, or choose **No business portfolio yet** (Meta will create one automatically).
6. Click **Create app** (enter your Facebook password if prompted).

---

### Step 3: Add the WhatsApp Product to Your App
1. You will be on the **App Dashboard** with a list of products.
2. Find the **WhatsApp** tile and click **Set up**.
3. Meta will ask you to select or create a Meta Business Account. If asked, confirm your business details.
4. Once set up, on the left sidebar you will see a new menu item called **WhatsApp**.
5. Click **WhatsApp > API Setup** (or **Getting Started**).

---

### Step 4: Add Your Real Phone Number
*(On the API Setup page, Meta shows a test number. Do not use that for production; follow these steps to add your real phone number).*

1. On the **API Setup** page, scroll down to **Step 5: Add a phone number**.
2. Click the button: **Add phone number**.
3. Fill in the profile details:
   - **WhatsApp Business Profile Display Name**: `Amriti Foods` (or `Amriti`)
   - **Timezone**: `(GMT+05:30) India Standard Time`
   - **Category**: `Grocery` or `Food and Beverage`
   - **Business Description**: `Healthy traditional food products & wellness`
4. Click **Next**.
5. Enter your phone number:
   - Country code: `India (+91)`
   - Phone number: `XXXXXXXXXX`
   - Choose verification method: **Text Message (SMS)** or **Phone Call**.
6. Click **Next**. Enter the 6-digit verification code you receive on your phone.
7. Click **Finish**.

Now your phone number is registered on WhatsApp Cloud API!
On the **API Setup** page:
- Look at the dropdown **From**: select your newly added phone number.
- Right below the dropdown, you will see two crucial IDs:
  - 📌 **Phone number ID**: (a 15-digit number, e.g. `982347102938475`)
  - 📌 **WhatsApp Business Account ID**: (a 16-digit number, e.g. `234857192039485`)
- **Copy both numbers and save them in a notepad.**

---

### Step 5: Create a Permanent Access Token (System User)

> [!WARNING]
> The token shown on the *API Setup* page expires in **24 hours**. You must generate a **Permanent System User Token** so your production backend never gets disconnected.

1. Go to **[business.facebook.com/settings/system-users](https://business.facebook.com/settings/system-users)**.
2. Select your Business Account on the top-left dropdown if prompted.
3. Under **System Users**, click **Add** (or **Create System User**).
   - **System username**: `amriti-backend-service`
   - **System user role**: Select **Admin**.
   - Click **Create system user**.
4. Click on your newly created system user (`amriti-backend-service`), then click **Add Assets**:
   - In the modal, select **Apps** on the left.
   - Choose your app (`Amriti Notifications`).
   - On the right, toggle **Full control (Manage app)** to **ON**.
   - Click **Save Changes**.
5. Next, click **Generate new token** button next to the system user:
   - **Select app**: choose `Amriti Notifications`.
   - **Token expiration**: select **Never** (crucial!).
   - In the permissions list, check the following 2 permissions:
     - `whatsapp_business_messaging`
     - `whatsapp_business_management`
   - Scroll down and click **Generate token**.
6. A popup will show your new token (starts with `EAAG...`).
   - **Copy this token immediately** and save it in your notepad. (Meta will never show it again).

---

### Step 6: Create the 2 Required Message Templates

In your new WhatsApp Business Account, go to:
👉 **[business.facebook.com/wa/manage/message-templates/](https://business.facebook.com/wa/manage/message-templates/)**

---

#### 🟢 Template 1: Authentication OTP Template (For Login & Signup)

1. Click the blue button: **Create template**.
2. **Category**: Select **Authentication**.
3. **Name**: `amrti_auth_otp`
4. **Language**: `English` (or `English (US)`).
5. Click **Continue**.
6. **Code Delivery Method**: Select **Copy code**.
   - Meta automatically provides the standard text:
     ```text
     {{1}} is your verification code. For your security, do not share this code.
     ```
   - Button type: **Copy code**
   - Button label: `Copy code`
7. Click **Submit** in the bottom right corner.
8. Click **Confirm**. (Approved automatically within 2–5 minutes).

---

#### 🟢 Template 2: Order Confirmation & Tracking Template (For Orders)

In the codebase, order confirmation, payment confirmation, and tracking updates all share one template that expects **4 parameters**:
- `{{1}}` = Customer Name
- `{{2}}` = Order Number
- `{{3}}` = Tracking URL
- `{{4}}` = Ordered Items list

Here is how to create it:

1. In **Message Templates**, click **Create template**.
2. **Category**: Select **Utility** (or **Marketing**).
3. **Name**: `order_confirmation` *(or you can keep `mani_test1`)*.
4. **Language**: `English` (or `English (US)`).
5. Click **Continue**.
6. **Header** (Optional): None.
7. **Body**: Paste the following text into the Body box:
   ```text
   Hi {{1}}, thank you for your order!

   Your order #{{2}} has been confirmed.

   Order Details:
   {{4}}

   Track your order here:
   {{3}}

   Thank you for choosing Amrti!
   ```
8. **Add Sample Content** (Meta requires sample values for each variable before approval):
   - Click **Add sample content**:
     - `{{1}}`: `Karthikeya`
     - `{{2}}`: `ORD-12345`
     - `{{3}}`: `https://amrti.com/track-order`
     - `{{4}}`: `• Moringa Powder (200g) × 1`
   - Click **Done**.
9. **Buttons** (Optional): You can add a quick reply or URL button (e.g., "Visit Website" pointing to `https://amrti.com`).
10. Click **Submit** in the bottom right corner.

> [!NOTE]
> Utility templates are typically approved within **5 to 30 minutes**. Once approved, its status will turn green (`Approved`).

---

### Step 7: Update Your Environment Variables (Render Dashboard)

Open your hosting dashboard (e.g., [Render.com](https://dashboard.render.com/)) where your ecommerce backend is running:

1. Click on your backend service (`ecommerce-backend`).
2. Go to **Environment** (or **Environment Variables**).
3. Find and update the following variables with your newly copied values:

| Key | Value to Paste | Description |
|---|---|---|
| `WHATSAPP_ENABLED` | `true` | Enables WhatsApp service |
| `WHATSAPP_PHONE_NUMBER_ID` | `Your new 15-digit Phone Number ID` | From Step 4 (API Setup page) |
| `WHATSAPP_BUSINESS_ACCOUNT_ID` | `Your new 16-digit WABA ID` | From Step 4 (API Setup page) |
| `WHATSAPP_ACCESS_TOKEN` | `EAAG... (Permanent System User Token)` | From Step 5 (Generate Token) |
| `WHATSAPP_OTP_TEMPLATE_NAME` | `amrti_auth_otp` | Name of Template 1 from Step 6 |
| `WHATSAPP_ORDER_TEMPLATE_NAME` | `order_confirmation` (or `mani_test1`) | Name of Template 2 from Step 6 |
| `WHATSAPP_API_BASE_URL` | `https://graph.facebook.com/v22.0` | Default Meta API Base URL |

4. Click **Save Changes**. Render will automatically redeploy your backend in ~1 minute.

---

## 🧪 Step 8: How to Test

### 1. Test OTP Login:
1. Open your website `/login`.
2. Enter your phone number with country code (e.g. `+919876543210`).
3. Click **Send Verification Code**.
4. You will receive an instant WhatsApp message with the 6-digit code and a "Copy code" button!
5. Enter the code on the screen to verify and log in.

### 2. Test Order Confirmation:
1. Place a test order on the store.
2. As soon as the order or payment completes, you will receive the order confirmation WhatsApp message with the order number, item list, and tracking link.

---

## 💡 Troubleshooting Checklist

| Symptom | Cause | Fix |
|---|---|---|
| Status `401 Unauthorized` | Invalid access token | Ensure you generated a **Permanent System User** token with `whatsapp_business_messaging` permission (Step 5). |
| Status `400: (#100) Param template['name'] is not valid` | Template name mismatch | Verify in WhatsApp Manager that the template status is green `Approved` and the env var (`WHATSAPP_OTP_TEMPLATE_NAME` or `WHATSAPP_ORDER_TEMPLATE_NAME`) matches the name exactly. |
| Status `400: (#131030) Recipient phone number not in allowed list` | App in "Development" mode sending to unverified numbers | In Meta App Dashboard, top navigation toggle **App Mode** from **Development** to **Live**. Also ensure payment method is added in Meta Business settings if required. |
| Message not received on phone | Phone was not cleared from regular WhatsApp before setup | Delete regular WhatsApp account on that SIM before registering on Cloud API. |
