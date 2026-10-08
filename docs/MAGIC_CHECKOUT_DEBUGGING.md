# Magic Checkout Debugging Guide

## 🚨 **CRITICAL ISSUE IDENTIFIED**

The error you're seeing indicates that Razorpay Magic Checkout is trying to call **Razorpay's API server** instead of **your backend server**. This is a **configuration issue**.

### **Error Analysis:**
```
XHR failed loading: PATCH "https://api.razorpay.com/v1/magic/customer/order_Rc2odoNwUIwxkN?key_id=rzp_test_RbZdqWTb0nVG4u"
```

**Problem**: The URL `https://api.razorpay.com/v1/magic/customer/...` should be pointing to **your backend**, not Razorpay's API.

## 🛠️ **SOLUTION IMPLEMENTED**

### **1. Enhanced Customer Endpoint**
- **Endpoint**: `PATCH /api/v1/magic-checkout/customer/:order_id`
- **Purpose**: Handle the failing PATCH request from Razorpay
- **Features**:
  - ✅ Comprehensive logging of all request data
  - ✅ Form data parsing and processing
  - ✅ Automatic address creation and storage
  - ✅ Razorpay order notes updating

### **2. Comprehensive Request Logging**
The enhanced endpoint now logs:
- ✅ **Request Method**: GET, POST, PATCH, etc.
- ✅ **Headers**: All HTTP headers including key_id
- ✅ **Query Parameters**: Including the key_id parameter
- ✅ **Form Data**: Complete customer and address data
- ✅ **JSON Data**: Any JSON payload sent
- ✅ **Client Information**: IP, User-Agent, Referer

### **3. Automatic Address Processing**
When form data is received, the system:
- ✅ **Extracts Customer Data**: Contact, email, name
- ✅ **Extracts Address Data**: Line1, line2, city, state, zipcode, country
- ✅ **Creates OrderAddress**: Stores in orders_address table
- ✅ **Updates Razorpay Notes**: Links address to order

## 🔧 **CONFIGURATION FIX NEEDED**

### **Root Cause:**
Your Magic Checkout is configured to call Razorpay's API instead of your backend.

### **Required Configuration:**
You need to configure Magic Checkout to use **your backend URLs**:

```javascript
// WRONG (current configuration)
const magicCheckoutConfig = {
  // This is pointing to Razorpay's API - INCORRECT
  customer_endpoint: "https://api.razorpay.com/v1/magic/customer/"
}

// CORRECT (required configuration)
const magicCheckoutConfig = {
  // Should point to YOUR backend
  customer_endpoint: "https://your-domain.com/api/v1/magic-checkout/customer/"
}
```

### **Your Backend Endpoints (Ready to Use):**
```
✅ GET    /api/v1/magic-checkout/customer/:order_id
✅ POST   /api/v1/magic-checkout/customer/:order_id  
✅ PATCH  /api/v1/magic-checkout/customer/:order_id
✅ POST   /api/v1/magic-checkout/get-promotions
✅ POST   /api/v1/magic-checkout/apply-promotion
✅ POST   /api/v1/magic-checkout/shipping-info
✅ POST   /api/v1/magic-checkout/get-saved-addresses
✅ POST   /api/v1/magic-checkout/complete
✅ POST   /api/v1/magic-checkout/webhook
```

## 🧪 **TESTING THE FIX**

### **1. Check Current Logs**
After deploying, look for these log messages:
```
=== MAGIC CHECKOUT CUSTOMER REQUEST (CRITICAL) ===
=== HANDLING PATCH REQUEST FOR MAGIC CHECKOUT ===
=== EXTRACTED CUSTOMER DATA ===
=== ORDER ADDRESS CREATED FROM FORM DATA ===
```

### **2. Test the Endpoint Directly**
```bash
# Test PATCH endpoint
curl -X PATCH "http://your-domain/api/v1/magic-checkout/customer/order_123?key_id=rzp_test_xxx" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "customer_details[contact]=+919876543210&customer_details[email]=test@example.com"

# Test GET endpoint  
curl -X GET "http://your-domain/api/v1/magic-checkout/customer/order_123?key_id=rzp_test_xxx"
```

### **3. Monitor Database**
Check the `orders_address` table for new records after Magic Checkout requests.

## 🎯 **EXPECTED RESULTS**

### **Before Fix:**
- ❌ PATCH request fails with XHR error
- ❌ No customer data captured
- ❌ No address stored
- ❌ Shipment creation fails

### **After Fix:**
- ✅ PATCH request succeeds with comprehensive logging
- ✅ Customer data extracted and logged
- ✅ Address automatically created in orders_address table
- ✅ Razorpay order notes updated with address ID
- ✅ Shipment creation will work with valid address

## 🚀 **NEXT STEPS**

### **1. Deploy and Test**
1. Deploy the enhanced Magic Checkout handler
2. Test a Magic Checkout transaction
3. Monitor logs for the new debug messages

### **2. Fix Configuration**
1. Update your Magic Checkout configuration to use your backend URLs
2. Ensure all endpoints point to your domain, not Razorpay's API

### **3. Verify Address Storage**
1. Check that OrderAddress records are being created
2. Verify that customer data is being captured
3. Confirm that shipment creation works with the stored addresses

## 📋 **LOG MONITORING CHECKLIST**

Look for these specific log messages to track the fix:

- [ ] `MAGIC CHECKOUT CUSTOMER REQUEST (CRITICAL)` - Request received
- [ ] `HANDLING PATCH REQUEST FOR MAGIC CHECKOUT` - PATCH method handled
- [ ] `Magic Checkout PATCH - Form Data` - Form data parsed
- [ ] `EXTRACTED CUSTOMER DATA` - Customer info extracted
- [ ] `ORDER ADDRESS CREATED FROM FORM DATA` - Address stored successfully
- [ ] `Successfully updated Razorpay notes` - Order notes updated

## 🔍 **DEBUGGING COMMANDS**

### **Check Recent Logs:**
```bash
# Look for Magic Checkout logs
grep "MAGIC CHECKOUT" /path/to/your/logs

# Check for PATCH requests
grep "PATCH.*customer" /path/to/your/logs

# Monitor address creation
grep "ORDER ADDRESS CREATED" /path/to/your/logs
```

### **Database Queries:**
```sql
-- Check recent OrderAddress records
SELECT * FROM order_addresses ORDER BY created_at DESC LIMIT 10;

-- Check Razorpay orders with notes
SELECT id, razorpay_order_id, razorpay_notes_json FROM orders 
WHERE razorpay_order_id LIKE 'order_%' ORDER BY created_at DESC LIMIT 10;
```

The enhanced logging will help you identify exactly where the Magic Checkout process is failing and what data Razorpay is sending! 🔍