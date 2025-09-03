# Delhivery B2C Shipping Integration Documentation

## 🚀 Quick Start Configuration

### API Credentials
```
DELHIVERY_API_TOKEN: <set via environment/secret store>
STAGING_BASE_URL: https://staging-express.delhivery.com
PRODUCTION_BASE_URL: https://track.delhivery.com
```

### Headers Template
```json
{
  "Authorization": "Token ${DELHIVERY_API_TOKEN}",
  "Content-Type": "application/json",
  "Accept": "application/json"
}
```

## 📋 Complete Integration Flow

### 1. Warehouse Setup (One-time)
**Purpose**: Register your pickup location
**Endpoint**: `POST /api/backend/clientwarehouse/create/`

```json
{
  "phone": "9999999999",
  "city": "Kota",
  "name": "warehouse_name",
  "pin": "110042",
  "address": "warehouse_address",
  "country": "India",
  "email": "warehouse@company.com",
  "registered_name": "Company Name",
  "return_address": "return_address",
  "return_pin": "110042",
  "return_city": "Kota",
  "return_state": "Delhi",
  "return_country": "India"
}
```

### 2. Check Pincode Serviceability
**Purpose**: Validate if delivery is available to destination
**Endpoint**: `GET /c/api/pin-codes/json/?filter_codes={pincode}`

**URLs**:
- Test: `https://staging-express.delhivery.com/c/api/pin-codes/json/?filter_codes=194103`
- Prod: `https://track.delhivery.com/c/api/pin-codes/json/?filter_codes=194103`

### 3. Calculate Shipping Cost
**Purpose**: Get shipping charges before order creation
**Endpoint**: `GET /api/kinko/v1/invoice/charges/.json`

**Parameters**:
- `md=S` (Mode: Surface; default in backend)
- `ss=Delivered` (Service Status; configurable)
- `d_pin=110053` (Destination PIN)
- `o_pin=110042` (Origin PIN)
- `cgm=10` (Charged Weight in grams; integer grams)
- `pt=Pre-paid` (Payment Type)

**URLs**:
- Test: `https://staging-express.delhivery.com/api/kinko/v1/invoice/charges/.json?md=S&ss=Delivered&d_pin=110053&o_pin=110042&cgm=10&pt=Pre-paid`
- Prod: `https://track.delhivery.com/api/kinko/v1/invoice/charges/.json?md=S&ss=Delivered&d_pin=110053&o_pin=110042&cgm=10&pt=Pre-paid`

### 4. Get Estimated TAT (Delivery Time)
**Purpose**: Get expected delivery timeline
**Endpoint**: `GET /api/dc/expected_tat`

**Parameters**:
- `origin_pin=122003`
- `destination_pin=136118`
- `mot=S` (Mode of Transport: Surface)
- `pdt=B2C` (Product Type)
- `expected_pickup_date=2024-05-31`

**URLs**:
- Test: `https://staging-express.delhivery.com/api/dc/expected_tat?origin_pin=122003&destination_pin=136118&mot=S&pdt=B2C&expected_pickup_date=2024-05-31`
- Prod: `https://track.delhivery.com/api/dc/expected_tat?origin_pin=122003&destination_pin=136118&mot=S&pdt=B2C&expected_pickup_date=2024-05-31`

### 5. Fetch Waybill Numbers (Optional)
**Purpose**: Pre-generate tracking numbers
**Endpoint**: `GET /waybill/api/bulk/json/?count={count}`

**URLs**:
- Test: `https://staging-express.delhivery.com/waybill/api/bulk/json/?count=5`
- Prod: `https://track.delhivery.com/waybill/api/bulk/json/?count=5`

### 6. Create Shipment
**Purpose**: Create actual shipment order
**Endpoint**: `POST /api/cmu/create.json`

**Payload Format**:
```
format=json&data={
  "shipments": [
    {
      "name": "Customer Name",
      "add": "Customer Address",
      "pin": "110042",
      "city": "City Name",
      "state": "State Name",
      "country": "India",
      "phone": "9999999999",
      "order": "ORDER_ID_123",
      "payment_mode": "Prepaid", // or "COD"
      "return_pin": "110042",
      "return_city": "Warehouse City",
      "return_phone": "9999999999",
      "return_add": "Warehouse Address",
      "return_state": "Warehouse State",
      "return_country": "India",
      "products_desc": "Product Description",
      "hsn_code": "12345678",
      "cod_amount": "0", // Amount for COD orders
      "order_date": null,
      "total_amount": "500",
      "seller_add": "Seller Address",
      "seller_name": "Seller Name",
      "seller_inv": "INVOICE_123",
      "quantity": "1",
      "waybill": "", // Leave empty for auto-generation
      "shipment_width": "100",
      "shipment_height": "100",
      "weight": "500", // in grams
      "shipping_mode": "Surface", // or "Express"
      "address_type": "home" // or "office"
    }
  ],
  "pickup_location": {
    "name": "warehouse_name"
  }
}
```

**URLs**:
- Test: `https://staging-express.delhivery.com/api/cmu/create.json`
- Prod: `https://track.delhivery.com/api/cmu/create.json`

### 7. Create Pickup Request
**Purpose**: Schedule pickup from warehouse
**Endpoint**: `POST /fm/request/new/`

**Payload**:
```json
{
  "pickup_time": "11:00:00",
  "pickup_date": "2023-12-29",
  "pickup_location": "warehouse_name",
  "expected_package_count": 1
}
```

**URLs**:
- Test: `https://staging-express.delhivery.com/fm/request/new/`
- Prod: `https://track.delhivery.com/fm/request/new/`

### 8. Generate Shipping Label
**Purpose**: Generate printable shipping label
**Endpoint**: `GET /api/p/packing_slip`

**Parameters**:
- `wbns=7035xxxxxxxxxxx` (Waybill Number)
- `pdf=true`
- `pdf_size=4R` (Label size)

**URLs**:
- Test: `https://staging-express.delhivery.com/api/p/packing_slip?wbns=7035xxxxxxxxxxx&pdf=true&pdf_size=4R`
- Prod: `https://track.delhivery.com/api/p/packing_slip?wbns=7035xxxxxxxxxxx&pdf=true&pdf_size=4R`

### 9. Track Shipment
**Purpose**: Get real-time shipment status
**Endpoint**: `GET /api/v1/packages/json/`

**Parameters**:
- `waybill=1122345678722` (Waybill Number)
- `ref_ids=` (Reference Order ID - optional)

**URLs**:
- Test: `https://staging-express.delhivery.com/api/v1/packages/json/?waybill=1122345678722&ref_ids=`
- Prod: `https://track.delhivery.com/api/v1/packages/json/?waybill=1122345678722&ref_ids=`

## 🔄 Shipment Status Types

### Forward Shipment Statuses
| Status Type | Status | Description |
|-------------|--------|-------------|
| UD | Manifested | Order created in Delhivery system |
| UD | Not Picked | Shipment not picked from warehouse |
| UD | In Transit | Shipment moving to destination |
| UD | Pending | Reached destination city, awaiting delivery |
| UD | Dispatched | Out for delivery to customer |
| DL | Delivered | Successfully delivered to customer |

### Return Shipment Statuses
Applied when shipment cannot be delivered and is being returned to origin.

## 🎯 Common E-commerce Integration Patterns

### Pattern 1: Checkout Flow Integration
1. **At Checkout**: Check pincode serviceability
2. **Show Shipping Options**: Calculate shipping cost + TAT
3. **Order Confirmation**: Create shipment
4. **Post-Order**: Generate label + Create pickup request

Backend delivery estimate endpoints
- POST /api/v1/cart/checkout/estimate-shipping (Protected)
- POST /api/v1/delivery/estimate (Protected)

### Pattern 2: Bulk Order Processing
1. **Batch Processing**: Fetch waybills in bulk
2. **Create Multiple Shipments**: Use bulk shipment creation
3. **Schedule Pickup**: Single pickup request for multiple orders

### Pattern 3: Real-time Tracking
1. **Webhook Setup**: Configure status push webhooks
2. **Database Updates**: Update order status automatically
3. **Customer Notifications**: Send tracking updates to customers

## 🛠️ Code Generation Helpers

### Environment Configuration
```typescript
interface DelhiveryConfig {
  baseUrl: string;
  token: string;
  isProduction: boolean;
}

const config: DelhiveryConfig = {
  baseUrl: process.env.NODE_ENV === 'production' 
    ? 'https://track.delhivery.com'
    : 'https://staging-express.delhivery.com',
  token: process.env.DELHIVERY_API_TOKEN || '68b8f90b85d277fc158946b905bf58cb43726d3e',
  isProduction: process.env.NODE_ENV === 'production'
};
```

### Common Data Structures
```typescript
interface ShipmentData {
  name: string;
  add: string;
  pin: string;
  city: string;
  state: string;
  country: string;
  phone: string;
  order: string;
  payment_mode: 'Prepaid' | 'COD';
  weight: string; // in grams
  shipping_mode: 'Surface' | 'Express';
  total_amount?: string;
  cod_amount?: string;
}

interface PincodeCheck {
  delivery_codes: Array<{
    postal_code: {
      pin: string;
      district_name: string;
      state_name: string;
    };
    cash: boolean; // COD availability
    pre_paid: boolean; // Prepaid availability
  }>;
}

interface ShippingCost {
  total_amount: number;
  zone: string;
  weight_slab: string;
}
```

## 📝 Webhook Configuration

### Status Push Webhook
**Purpose**: Real-time shipment status updates
**Requirements**: 
- Endpoint URL with HTTPS
- Authentication method (Bearer token, Basic auth, etc.)
- Separate webhook for scan push and document push

### Document Push Webhook Types
1. **POD (Proof of Delivery)**: Delivery confirmation documents
2. **Sorter Image**: Package sorting verification
3. **QC Image**: Quality control images

### Webhook Payload Structure
```json
{
  "waybill": "7035xxxxxxxxxxxxx",
  "order_id": "ORDER_123",
  "status": "Delivered",
  "status_type": "DL",
  "timestamp": "2023-12-29T10:30:00Z",
  "location": "Delivery Location",
  "remarks": "Delivered to customer"
}
```

## 🔧 Error Handling

### Common API Errors
- **401 Unauthorized**: Invalid API token
- **400 Bad Request**: Invalid request parameters
- **404 Not Found**: Resource not found
- **500 Internal Server Error**: Delhivery system error

### Retry Strategy
- Implement exponential backoff for failed requests
- Maximum 3 retry attempts for transient errors
- Log all API responses for debugging

## 📞 Support & Documentation
- Integration Support: lastmile-integration@delhivery.com
- Business POC: Contact your assigned Delhivery representative
- Webhook Setup: Fill webhook requirement document and email to integration team

---

*This documentation is optimized for GitHub Copilot code generation. Use specific function names and variable names mentioned here for best results.*