u are using a b2c account 

Common Issues in B2C Integration
Issue	Cause	Fix
Waybill missing	Pickup location name doesn’t match saved pickup address	Use the exact pickup location name from dashboard (case-sensitive)
Tracking URL empty	Waybill not returned → URL can’t be built	Ensure valid pickup location + proper request
Shipment fails silently	Missing mandatory fields like pin, phone, order	Always send full shipment object
Auth error	Wrong or expired token	Use Authorization: Token {YOUR_API_KEY}
Response not parsed	Struct mismatch in Go/Python	Always unmarshal into the right response struct (packages[].waybill)
3. B2C Shipment Creation API

Endpoint:

POST https://track.delhivery.com/api/cmu/create.json


Headers:

Content-Type: application/json
Authorization: Token <YOUR_API_TOKEN>


Payload Format (B2C)

{
  "pickup_location": "MYSHOP01",   // must match pickup address name in dashboard
  "shipments": [
    {
      "name": "Customer Name",
      "add": "123 Street, Area",
      "pin": "110001",
      "phone": "9876543210",
      "order": "ORDER12345",
      "cod_amount": 500,           // 0 if prepaid
      "payment_mode": "COD"        // or "Prepaid"
    }
  ]
}

4. Example Successful Response
{
  "packages": [
    {
      "waybill": "DEL123456789",
      "refnum": "ORDER12345",
      "status": "Success"
    }
  ],
  "success": true
}

5. Tracking URL

Delhivery does not return tracking URL.

You must build it manually:

https://track.delhivery.com/p/{WAYBILL}


Example:

https://track.delhivery.com/p/DEL123456789

6. How to Handle Responses in Go
type DelhiveryShipmentResponse struct {
    Packages []struct {
        Waybill string `json:"waybill"`
        RefNum  string `json:"refnum"`
        Status  string `json:"status"`
    } `json:"packages"`
    Success bool `json:"success"`
}

if resp.Success && len(resp.Packages) > 0 {
    waybill := resp.Packages[0].Waybill
    trackingURL := fmt.Sprintf("https://track.delhivery.com/p/%s", waybill)
}

7. Potential Error Cases

Invalid Pickup Location → Response will be empty or fail without waybill.

Invalid Pincode → API rejects with serviceability error.

COD amount mismatch → If payment_mode = "Prepaid" but cod_amount > 0.

Auth Token wrong → 401 Unauthorized.

8. Checklist for Stable Integration

✅ Add at least one pickup location in Dashboard → Settings → Pickup Addresses.
✅ Use that exact name as pickup_location.
✅ Always send order, pin, phone, payment_mode.
✅ Build tracking URL manually with returned waybill.
✅ Log full API request/response for debugging.