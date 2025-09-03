# Delhivery API Integration Guide

## Overview
This document provides complete integration details for Delhivery's shipping API, including order creation and shipment tracking functionality.

## API Configuration

### Base URLs
- **Test Environment**: `https://staging-express.delhivery.com`
- **Production Environment**: `https://track.delhivery.com`

### Authentication
All API requests require Bearer Token authentication:
```
Authorization: Token YOUR_API_TOKEN_HERE
```

## 1. Shipment Creation API

### Endpoint
- **Test**: `POST https://staging-express.delhivery.com/api/cmu/create.json`
- **Production**: `POST https://track.delhivery.com/api/cmu/create.json`

### Headers
```
Content-Type: application/json
Authorization: Token YOUR_API_TOKEN_HERE
Accept: application/json
```

### Payment Modes
- **Forward Flow**: `COD` or `Prepaid`
- **Reverse Flow**: `Pickup`
- **Replacement**: `REPL`

### Request Format
```json
{
  "format": "json",
  "data": {
    "shipments": [
      {
        // Shipment details here
      }
    ],
    "pickup_location": {
      "name": "warehouse_name"
    }
  }
}
```

### Single Package Shipment (SPS) Parameters

| Parameter | Mandatory | Type | Description |
|-----------|-----------|------|-------------|
| `name` | Yes | string | Name of the consignee |
| `order` | Yes | string | Unique Order ID |
| `phone` | Yes | string | Consignee phone number |
| `add` | Yes | string | Address of the consignee |
| `pin` | Yes | integer | Pincode of the consignee |
| `pickup_location` | Yes | object | Warehouse name (case sensitive) |
| `payment_mode` | Yes | string | COD/Prepaid/Pickup/REPL |
| `address_type` | No | string | home/office |
| `city` | No | string | City of the consignee |
| `state` | No | string | State of the consignee |
| `country` | No | string | Country (mandatory for Bangladesh: 'BD') |
| `weight` | No | float | Weight in grams |
| `cod_amount` | No | float | COD amount (if applicable) |
| `total_amount` | No | float | Total order amount |
| `products_desc` | No | string | Product description |
| `quantity` | No | string | Item quantity |
| `shipping_mode` | No | string | Surface/Express |
| `shipment_height` | No | float | Height in cm |
| `shipment_width` | No | float | Width in cm |
| `shipment_length` | No | float | Length in cm |
| `fragile_shipment` | No | boolean | true/false |
| `dangerous_good` | No | boolean | true/false |
| `plastic_packaging` | No | boolean | true/false |
| `return_name` | No | string | Return contact name |
| `return_address` | No | string | Return address |
| `return_city` | No | string | Return city |
| `return_state` | No | string | Return state |
| `return_country` | No | string | Return country |
| `return_pin` | No | integer | Return pincode |
| `return_phone` | No | string | Return phone |
| `seller_name` | No | string | Seller name |
| `seller_add` | No | string | Seller address |
| `seller_inv` | No | string | Seller invoice |
| `ewbn` | No | string | E-waybill number (for packages ≥ 50k) |
| `hsn_code` | No | string | HSN Code for e-waybill |
| `waybill` | No | string | Custom waybill number |

### Sample SPS Request
```json
{
  "format": "json",
  "data": {
    "shipments": [
      {
        "name": "John Doe",
        "add": "123 Main Street, Sector 15",
        "pin": "122001",
        "city": "Gurugram",
        "state": "Haryana",
        "country": "India",
        "phone": "9999999999",
        "order": "ORD001234",
        "payment_mode": "COD",
        "cod_amount": "1500",
        "total_amount": "1500",
        "products_desc": "Electronics - Mobile Phone",
        "quantity": "1",
        "weight": "200",
        "shipping_mode": "Surface",
        "shipment_width": "15",
        "shipment_height": "20",
        "shipment_length": "10",
        "return_name": "Your Store Name",
        "return_add": "456 Business District",
        "return_city": "Delhi",
        "return_state": "Delhi",
        "return_pin": "110001",
        "return_phone": "9876543210",
        "seller_name": "Your Company",
        "fragile_shipment": false,
        "dangerous_good": false
      }
    ],
    "pickup_location": {
      "name": "Main_Warehouse_Delhi"
    }
  }
}
```

### Multi Package Shipment (MPS) Parameters

Additional parameters for MPS:

| Parameter | Mandatory | Type | Description |
|-----------|-----------|------|-------------|
| `mps_amount` | Yes | integer | Sum of all package amounts for COD |
| `mps_children` | Yes | integer | Total number of packages (master + child) |
| `master_id` | Yes | string | Master waybill number |
| `shipment_type` | Yes | string | Must be "MPS" |
| `waybill` | Yes | string | Unique waybill for each package |

### Sample MPS Request
```json
{
  "pickup_location": {
    "name": "Main_Warehouse_Delhi"
  },
  "shipments": [
    {
      "order": "MPS001234",
      "name": "John Doe",
      "phone": "9999999999",
      "add": "123 Main Street",
      "pin": "600063",
      "city": "Chennai",
      "state": "Tamil Nadu",
      "country": "India",
      "payment_mode": "COD",
      "weight": "500",
      "mps_amount": "2500",
      "mps_children": "2",
      "products_desc": "Package 1 - Books",
      "shipment_type": "MPS",
      "master_id": "DHL12345678901",
      "waybill": "DHL12345678901",
      "total_amount": "2500"
    },
    {
      "order": "MPS001234",
      "name": "John Doe",
      "phone": "9999999999",
      "add": "123 Main Street",
      "pin": "600063",
      "city": "Chennai",
      "state": "Tamil Nadu",
      "country": "India",
      "payment_mode": "COD",
      "weight": "300",
      "mps_amount": "2500",
      "mps_children": "2",
      "products_desc": "Package 2 - Stationery",
      "shipment_type": "MPS",
      "master_id": "DHL12345678901",
      "waybill": "DHL12345678902",
      "total_amount": "2500"
    }
  ]
}
```

### Reverse Pickup Configuration
For return/reverse shipments, set:
- `payment_mode`: "Pickup"
- Customer info becomes pickup location
- `return_add` and related fields define delivery address

### Replacement Shipment Configuration
For replacement shipments, set:
- `payment_mode`: "REPL"
- Pickup location = pickup address
- Customer address = exchange location
- Return address = final delivery address

## 2. Shipment Tracking API

### Endpoint
- **Test**: `GET https://staging-express.delhivery.com/api/v1/packages/json/`
- **Production**: `GET https://track.delhivery.com/api/v1/packages/json/`

### Headers
```
Authorization: Token YOUR_API_TOKEN_HERE
Content-Type: application/json
```

### Parameters
| Parameter | Mandatory | Type | Description |
|-----------|-----------|------|-------------|
| `waybill` | Yes | string | Waybill number(s) - comma-separated for multiple |
| `ref_ids` | No | string | Order ID(s) - comma-separated for multiple |

### Sample Tracking Request
```
GET https://staging-express.delhivery.com/api/v1/packages/json/?waybill=DHL12345678901&ref_ids=ORD001234
```

### Multiple Waybill Tracking
```
GET https://staging-express.delhivery.com/api/v1/packages/json/?waybill=DHL12345678901,DHL12345678902,DHL12345678903
```

## Implementation Examples

### JavaScript/Node.js Integration

#### 1. Create Shipment Function
```javascript
async function createShipment(shipmentData) {
    const url = process.env.NODE_ENV === 'production' 
        ? 'https://track.delhivery.com/api/cmu/create.json'
        : 'https://staging-express.delhivery.com/api/cmu/create.json';
    
    const payload = {
        format: 'json',
        data: shipmentData
    };
    
    try {
        const response = await fetch(url, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Token ${process.env.DELHIVERY_API_TOKEN}`,
                'Accept': 'application/json'
            },
            body: JSON.stringify(payload)
        });
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const result = await response.json();
        return result;
    } catch (error) {
        console.error('Error creating shipment:', error);
        throw error;
    }
}
```

#### 2. Track Shipment Function
```javascript
async function trackShipment(waybill, orderId = '') {
    const baseUrl = process.env.NODE_ENV === 'production' 
        ? 'https://track.delhivery.com/api/v1/packages/json/'
        : 'https://staging-express.delhivery.com/api/v1/packages/json/';
    
    const params = new URLSearchParams({
        waybill: waybill,
        ref_ids: orderId
    });
    
    const url = `${baseUrl}?${params}`;
    
    try {
        const response = await fetch(url, {
            method: 'GET',
            headers: {
                'Authorization': `Token ${process.env.DELHIVERY_API_TOKEN}`,
                'Content-Type': 'application/json'
            }
        });
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const result = await response.json();
        return result;
    } catch (error) {
        console.error('Error tracking shipment:', error);
        throw error;
    }
}
```

### Python Integration

#### 1. Create Shipment Function
```python
import requests
import json
import os

def create_shipment(shipment_data):
    base_url = "https://track.delhivery.com" if os.getenv("ENVIRONMENT") == "production" else "https://staging-express.delhivery.com"
    url = f"{base_url}/api/cmu/create.json"
    
    payload = {
        "format": "json",
        "data": shipment_data
    }
    
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Token {os.getenv('DELHIVERY_API_TOKEN')}",
        "Accept": "application/json"
    }
    
    try:
        response = requests.post(url, json=payload, headers=headers)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        print(f"Error creating shipment: {e}")
        raise
```

#### 2. Track Shipment Function
```python
def track_shipment(waybill, order_id=""):
    base_url = "https://track.delhivery.com" if os.getenv("ENVIRONMENT") == "production" else "https://staging-express.delhivery.com"
    url = f"{base_url}/api/v1/packages/json/"
    
    params = {
        "waybill": waybill,
        "ref_ids": order_id
    }
    
    headers = {
        "Authorization": f"Token {os.getenv('DELHIVERY_API_TOKEN')}",
        "Content-Type": "application/json"
    }
    
    try:
        response = requests.get(url, params=params, headers=headers)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        print(f"Error tracking shipment: {e}")
        raise
```

## Rate Limits and Performance

### Shipment Creation API
- **Average Latency**: 283.57ms
- **P99 Latency**: 1.59s
- **Rate Limit**: Contact Delhivery for specific limits

### Shipment Tracking API
- **Average Latency**: 130.31ms
- **P99 Latency**: 529.15ms
- **Rate Limit**: 750 requests per 5 minutes per IP
- **Batch Tracking**: Up to 50 waybills per request

## Error Handling

### Common Error Responses
```json
{
    "status": "fail",
    "error": "Error description here"
}
```

### Best Practices
1. Always validate required fields before API calls
2. Implement retry logic with exponential backoff
3. Log API responses for debugging
4. Handle rate limiting gracefully
5. Validate pincode and address data
6. Use staging environment for testing

## Important Notes

1. **Order ID Uniqueness**: Each order ID must be unique across all shipments
2. **Pickup Location**: Must exactly match registered warehouse name (case-sensitive)
3. **Special Characters**: Avoid `&`, `#`, `%`, `;`, `\` in JSON body
4. **Waybill Management**: For MPS, waybills must be pre-fetched
5. **Address Validation**: Ensure complete and accurate address information
6. **Phone Numbers**: Use valid 10-digit Indian mobile numbers
7. **Pincode Verification**: Validate serviceability before creating shipments

## Environment Variables Setup

Create a `.env` file with:
```
DELHIVERY_API_TOKEN=your_api_token_here
ENVIRONMENT=development  # or production
```

no need of return order
i just need the ship ment to be placed and order id to et generated and the url tracking should be working fine 

