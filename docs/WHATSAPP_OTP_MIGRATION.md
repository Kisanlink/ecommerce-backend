# WhatsApp OTP Authentication Migration & Architecture

## Overview

This document describes the architectural changes and implementation details for migrating phone authentication from **Firebase SMS** to **WhatsApp OTP via Meta Business API**.

---

## 1. Why the Migration Was Done

1. **Firebase SMS Billing Limitations**: Firebase SMS requires Google Cloud pay-as-you-go billing with SMS fees, which resulted in `BILLING_NOT_ENABLED` errors.
2. **Meta Cloud API Cost & Deliverability**: WhatsApp has higher deliverability and lower cost compared to SMS in key operating regions (e.g., India).
3. **Better User Experience**: No client-side reCAPTCHA challenges or invisible reCAPTCHA verifiers needed.

---

## 2. Architecture & Data Flow

```
+--------------------------------------------------------------------------------------------------+
|                                        LOGIN / SIGNUP FLOW                                       |
+--------------------------------------------------------------------------------------------------+

 1. Send Code
    [User] ---> (Enters Phone: +919876543210)
                 |
                 v
            [Frontend: Login.tsx] ---> POST /api/v1/auth/phone/send-code { phone_number }
                                         |
                                         v
                                  [Backend: Gin Router]
                                         |
                                         v
                                  [FirebaseAuthHandler.SendPhoneVerificationCode]
                                         |
                                         +---> [OTPService.GenerateAndSaveOTP]
                                         |     - Check rate limit (1 OTP / 60s cooldown)
                                         |     - Crypto random 6-digit code
                                         |     - bcrypt hash generated
                                         |     - Store in `phone_verifications` table (NeonDB)
                                         |     - 5-minute expiry
                                         |
                                         +---> [WhatsAppService.SendOTPMessage]
                                         |     - Meta Cloud API call
                                         |     - Uses Auth Template or Plain Text fallback
                                         |     - Recipient gets WhatsApp message with 6-digit OTP
                                         |     - If send fails → OTPService.DeleteOTP() rolls back DB record
                                         |
            [Frontend] <-----------------+ Returns { message, session_info }


 2. Verify Code
    [User] ---> (Enters 6-digit code)
                 |
                 v
            [Frontend: Login.tsx] ---> POST /api/v1/auth/phone/verify-code { phone_number, code, session_info }
                                         |
                                         v
                                  [FirebaseAuthHandler.VerifyPhoneCode]
                                         |
                                         +---> [OTPService.VerifyOTP]
                                         |     - Match session_info & phone_number
                                         |     - Check expiry & attempt count (max 5)
                                         |     - bcrypt verify code
                                         |     - Mark record `is_used = true`
                                         |
                                         +---> [NeonDB: User & Profile Management]
                                         |     - Lookup user by phone
                                         |     - If not found -> CreatePhoneUser()
                                         |     - Ensure Profile exists via CreateAutoProfile()
                                         |
                                         +---> [Firebase Auth Sync & Minting]
                                         |     - Get or create Firebase user by phone
                                         |     - Link Firebase UID to NeonDB user
                                         |     - Mint Firebase Custom Token
                                         |     - Exchange for real `id_token` & `refresh_token`
                                         |
            [Frontend] <-----------------+ Returns { id_token, refresh_token, expires_in, user }
                 |
                 v
      Stored in localStorage (identical to previous flow)
      User redirect to Checkout / Account
```

---

## 3. Detailed File & Code Changes

### 3.1 Backend Changes

| Component | File | Description |
|---|---|---|
| **Model** | [`entities/models/phone_verification.go`](file:///c:/Users/Karthikeya%20Akhandam/Documents/Kisanlink/amriti-backend/ecommerce-backend/entities/models/phone_verification.go) | New `PhoneVerification` model storing `phone_number`, `code_hash`, `session_info`, `expires_at`, `attempts`, `is_used`. |
| **Migration** | [`internal/database/migration.go`](file:///c:/Users/Karthikeya%20Akhandam/Documents/Kisanlink/amriti-backend/ecommerce-backend/internal/database/migration.go) | Added `&models.PhoneVerification{}` to `modelsToMigrate` list for AutoMigrate. |
| **Service** | [`internal/services/otp_service.go`](file:///c:/Users/Karthikeya%20Akhandam/Documents/Kisanlink/amriti-backend/ecommerce-backend/internal/services/otp_service.go) | New `OTPService` managing generation, bcrypt hashing, cooldown checks, and verification. |
| **Service** | [`internal/services/whatsapp_service.go`](file:///c:/Users/Karthikeya%20Akhandam/Documents/Kisanlink/amriti-backend/ecommerce-backend/internal/services/whatsapp_service.go) | Added `SendOTPMessage`, `sendTemplateOTP`, `sendTextOTP`, and `executeWhatsAppRequest`. |
| **Config** | [`internal/config/config.go`](file:///c:/Users/Karthikeya%20Akhandam/Documents/Kisanlink/amriti-backend/ecommerce-backend/internal/config/config.go) | Added `OTPTemplate` field loaded from `WHATSAPP_OTP_TEMPLATE_NAME`. |
| **Service Manager** | [`internal/services/service_manager.go`](file:///c:/Users/Karthikeya%20Akhandam/Documents/Kisanlink/amriti-backend/ecommerce-backend/internal/services/service_manager.go) | Added `OTPService` initialization and exposure. |
| **Routes** | [`internal/routes/auth/auth_routes.go`](file:///c:/Users/Karthikeya%20Akhandam/Documents/Kisanlink/amriti-backend/ecommerce-backend/internal/routes/auth/auth_routes.go) | Injected `WhatsAppService` and `OTPService` into `FirebaseAuthHandler`. |
| **Requests** | [`entities/requests/auth_requests.go`](file:///c:/Users/Karthikeya%20Akhandam/Documents/Kisanlink/amriti-backend/ecommerce-backend/entities/requests/auth_requests.go) | Removed dead `recaptcha_token` field from `SendPhoneCodeRequest`; added optional `name` field to `VerifyPhoneCodeRequest` (used as display name for new users). |
| **Handler** | [`internal/handlers/firebase_auth_handler.go`](file:///c:/Users/Karthikeya%20Akhandam/Documents/Kisanlink/amriti-backend/ecommerce-backend/internal/handlers/firebase_auth_handler.go) | Re-implemented `SendPhoneVerificationCode` and `VerifyPhoneCode` to use WhatsApp OTP + Firebase token minting. |

### 3.2 Frontend Changes

| Component | File | Description |
|---|---|---|
| **Service** | [`src/services/authService.ts`](file:///c:/Users/Karthikeya%20Akhandam/Documents/Kisanlink/amrti/Amrti/src/services/authService.ts) | Removed reCAPTCHA initialization, container dependencies, and token acquisition. |
| **UI / Component** | [`src/pages/Login.tsx`](file:///c:/Users/Karthikeya%20Akhandam/Documents/Kisanlink/amrti/Amrti/src/pages/Login.tsx) | Removed reCAPTCHA DOM containers, removed clearRecaptcha imports, updated helper labels to WhatsApp, updated send and resend handlers. |

---

## 4. API Endpoints Reference

### 4.1 Send WhatsApp Verification Code
- **Endpoint**: `POST /api/v1/auth/phone/send-code`
- **Request Body**:
  ```json
  {
    "phone_number": "+917386727007"
  }
  ```
- **Response (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Verification code sent via WhatsApp successfully",
    "data": {
      "session_info": "c7a82c69-2fb0-4382-95f7-f9d2208033c5"
    }
  }
  ```
- **Error Responses**:
  - `429 Too Many Requests`: `"Please wait 60 seconds before requesting another code"`
  - `400 Bad Request`: `"Phone number is required"`
  - `500 Internal Server Error`: `"Failed to send WhatsApp message. Please check your phone number and try again."`

### 4.2 Verify WhatsApp Code & Authenticate
- **Endpoint**: `POST /api/v1/auth/phone/verify-code`
- **Request Body**:
  ```json
  {
    "phone_number": "+917386727007",
    "code": "849201",
    "session_info": "c7a82c69-2fb0-4382-95f7-f9d2208033c5",
    "name": "Ravi Kumar"
  }
  ```
  (`name` is optional — used as display name only when creating a new user account)
- **Response (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Phone authentication successful",
    "data": {
      "id_token": "eyJhbGciOi...",
      "refresh_token": "AEu4IL...",
      "expires_in": "3600",
      "user": {
        "id": "USER0182",
        "uid": "fb_uid_12345",
        "phone_number": "+917386727007",
        "name": "User",
        "is_new_user": false,
        "is_verified": true,
        "is_active": true,
        "created_at": "2026-10-15T12:00:00Z"
      }
    }
  }
  ```

---

## 5. Security Highlights

1. **Rate Limiting**: Cooldown of 60 seconds enforced at the database level per phone number.
2. **One-Way Hashing**: Plaintext OTP is never stored in the database; only bcrypt hashes with cost factor 10 are retained.
3. **Atomic Attempt Counter**: The attempt increment uses `UPDATE ... WHERE attempts < 5` + `RowsAffected == 0` check, preventing concurrent bypass via race conditions.
4. **Expiration**: Every OTP expires automatically after 5 minutes.
5. **Session Isolation**: Each request generates a unique UUID session token, preventing code replay or session hijacking.
6. **Send Rollback**: If the WhatsApp API call fails after the OTP is saved, the DB record is deleted (`DeleteOTP`) so the user is not stuck in a 60s cooldown with an undelivered code.

---

## 6. Render Deployment Notes

Render runs the compiled binary directly — `make migrate` is **not** available as a separate step.

The server auto-migrates all required tables on startup in `cmd/server/main.go` via `AutoMigrateModels()`. This includes `phone_verifications` (the `PhoneVerification` model), so no manual migration step is needed on first deploy.

`AutoMigrateModels` is idempotent — it only creates or alters tables, never drops data. Re-deploying the same binary is safe.
