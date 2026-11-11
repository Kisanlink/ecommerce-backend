package services

import (
	"encoding/json"
	"testing"
)

// Reproduces the production 503: Razorpay Magic Checkout returns tax_details as an ARRAY.
// Before the fix (map[string]interface{}) this unmarshal returned
//   "cannot unmarshal array into Go struct field RazorpayOrder.tax_details..."
// and GetOrder failed -> /verify 503 -> paid-but-no-order.
func TestRazorpayOrder_TaxDetailsArray_Decodes(t *testing.T) {
	// Minimal shape mirroring the real Magic Checkout order response.
	body := []byte(`{
		"id": "order_TxDntiJaq9wpCS",
		"entity": "order",
		"amount": 18000,
		"amount_paid": 18000,
		"status": "paid",
		"tax_details": [],
		"customer_details": {
			"shipping_address": {"name":"Test","line1":"1 St","city":"Hyderabad","state":"Telangana","country":"india","zipcode":"500081"}
		}
	}`)

	var o RazorpayOrder
	if err := json.Unmarshal(body, &o); err != nil {
		t.Fatalf("decode failed (the bug): %v", err)
	}
	if o.ID != "order_TxDntiJaq9wpCS" || o.AmountPaid != 18000 {
		t.Fatalf("wrong decode: id=%q amount_paid=%d", o.ID, o.AmountPaid)
	}
	if o.CustomerDetails == nil || o.CustomerDetails.ShippingAddress == nil ||
		o.CustomerDetails.ShippingAddress.Zipcode != "500081" {
		t.Fatalf("shipping address not decoded: %+v", o.CustomerDetails)
	}

	// Also prove the other shapes Razorpay might send never break the decode.
	for _, shape := range []string{`[]`, `[{"name":"GST","rate":18}]`, `{"gst":18}`, `null`} {
		b := []byte(`{"id":"order_x","tax_details":` + shape + `}`)
		var o2 RazorpayOrder
		if err := json.Unmarshal(b, &o2); err != nil {
			t.Fatalf("tax_details=%s failed: %v", shape, err)
		}
	}
}
