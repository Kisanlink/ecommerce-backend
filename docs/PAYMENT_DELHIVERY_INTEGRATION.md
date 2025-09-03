# Payment Flow with Delhivery Integration

This document explains the complete payment verification flow that now includes automatic Delhivery shipment creation and AWB tracking.

## Flow Overview

1. **Payment Creation** → 2. **Payment Verification** → 3. **Order Creation** → 4. **Delhivery Shipment** → 5. **AWB Tracking**

## Detailed Flow

### 1. Create Payment Order
**Endpoint:** `POST /api/v1/payments/create-order`
- Creates Razorpay order with cart details
- Stores shipping address and charges in order notes
- Returns order_id for payment processing

### 2. Payment Processing
- User completes payment via Razorpay Checkout
- Razorpay provides payment_id and signature

### 3. Payment Verification & Order Creation
**Endpoint:** `POST /api/v1/payments/verify`

#### Request:
```json
{
    "razorpay_order_id": "order_123456",
    "razorpay_payment_id": "pay_123456", 
    "razorpay_signature": "signature_hash"
}
```

#### Process:
1. **Signature Verification**: Validates payment authenticity
2. **Order Creation**: Creates order from user's cart
3. **Payment Status Update**: Marks order as paid
4. **Order Metadata**: Stores Razorpay details and shipping info
5. **🆕 Delhivery Shipment**: Creates shipment automatically
6. **🆕 AWB Tracking**: Updates order with waybill number
7. **Cart Cleanup**: Clears user's cart
8. **Invoice Generation**: Creates order invoice

#### Enhanced Response:
```json
{
    "success": true,
    "message": "Payment verified and order created successfully",
    "payment_id": "pay_123456",
    "order_id": "order_uuid",
    "awb_number": "123456789012",
    "tracking_url": "https://track.delhivery.com/p/123456789012"
}
```

### 4. Delhivery Integration Details

#### Shipment Creation Process:
1. **Get Order Items**: Retrieves order with all items
2. **Build Shipment Request**: Creates Delhivery API payload
3. **API Call**: POST to Delhivery shipment creation endpoint
4. **Parse Response**: Extracts waybill number from response
5. **Update Order**: Stores AWB, status, and tracking URL

#### Delhivery API Request Structure:
```json
{
    "shipments": [{
        "name": "Customer Name",
        "add": "Shipping Address",
        "phone": "9876543210",
        "pin": "110001",
        "cod": 0,
        "payment_mode": "Prepaid",
        "return_add": "Return Address",
        "return_city": "Return City",
        "return_name": "Business Name",
        "return_phone": "1234567890",
        "return_pin": "110002",
        "products_desc": "Product 1 x2, Product 2 x1",
        "order": "order_uuid",
        "shipment_width": 20,
        "shipment_height": 10,
        "weight": 1.5,
        "seller_add": "Pickup Address"
    }]
}
```

#### Response Handling:
- Extracts waybill number from `packages[0].waybill`
- Updates order with tracking information
- Logs success/failure for monitoring

## Database Changes

### Order Model Enhancements:
```go
type Order struct {
    // ... existing fields ...
    
    // Delhivery tracking fields
    DelhiveryWaybill string `gorm:"column:delhivery_waybill"`
    DelhiveryStatus  string `gorm:"column:delhivery_status"`
    TrackingURL      string `gorm:"column:tracking_url"`
}
```

## Error Handling

### Payment Verification Errors:
- **Invalid Signature**: Returns failure response
- **Order Creation Failed**: Returns 500 error
- **Payment Status Update Failed**: Logs warning, continues

### Delhivery Integration Errors:
- **Shipment Creation Failed**: Logs error, payment still succeeds
- **Order Update Failed**: Logs error, AWB not stored but shipment exists
- **API Timeout**: Retries not implemented, logs error

## Configuration Required

### Environment Variables:
```env
DELHIVERY_API_KEY=your_delhivery_api_key
DELHIVERY_BASE_URL=https://track.delhivery.com/api/cmu/create.json
BUSINESS_NAME=Your Business Name
RETURN_ADDRESS=Your Return Address
RETURN_CITY=Your Return City
RETURN_PHONE=Your Return Phone
RETURN_PINCODE=Your Return PIN
```

## Frontend Integration

The enhanced response now includes AWB tracking information:

```javascript
// After successful payment verification
const response = await fetch('/api/v1/payments/verify', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify(verificationData)
});

const result = await response.json();

if (result.success) {
    // Show success message
    console.log('Order created:', result.order_id);
    
    // Display tracking information if available
    if (result.awb_number) {
        console.log('AWB Number:', result.awb_number);
        console.log('Track at:', result.tracking_url);
        
        // Show tracking info to user
        showTrackingInfo(result.awb_number, result.tracking_url);
    }
}
```

## Monitoring & Logging

### Key Log Events:
1. **Payment verification success/failure**
2. **Order creation success/failure**
3. **Delhivery shipment creation attempts**
4. **AWB number extraction and storage**
5. **API call failures and retries**

### Metrics to Monitor:
- Payment verification success rate
- Delhivery shipment creation success rate
- Time taken for complete flow
- AWB number availability rate

## Testing

Use the provided test script `test_payment_with_delhivery.ps1` to verify the complete flow:

```powershell
# Run the test script
./test_payment_with_delhivery.ps1
```

Expected successful flow:
1. Payment verification succeeds
2. Order is created
3. Delhivery shipment is created
4. AWB number is returned
5. Tracking URL is provided

## Troubleshooting

### Common Issues:

1. **No AWB Number Returned**:
   - Check Delhivery API credentials
   - Verify shipment request format
   - Check API response logs

2. **Shipment Creation Failed**:
   - Verify pickup address configuration
   - Check pincode serviceability
   - Validate customer address format

3. **Order Not Updated with AWB**:
   - Database connection issues
   - Order model field mapping
   - Transaction rollback scenarios

### Debug Endpoints:
- Check order details: `GET /api/v1/orders/{order_id}`
- Verify Delhivery config: Check environment variables
- Monitor logs: Application logs for Delhivery API calls

## Future Enhancements

1. **Webhook Integration**: Handle Delhivery status updates
2. **Retry Logic**: Implement retry for failed shipment creation
3. **Bulk Shipment**: Support multiple orders in single API call
4. **Status Tracking**: Periodic status updates from Delhivery
5. **Return Management**: Handle return shipment creation
