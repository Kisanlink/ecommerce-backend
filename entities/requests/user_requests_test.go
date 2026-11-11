package requests

import (
	"encoding/json"
	"testing"
)

// AUTH-01 regression: a user submitting privileged fields to the self-service
// profile update must NOT be able to set them. The sanitized DTO has no role/
// is_verified/is_active fields, so they are dropped at bind time and never reach
// the service's update map.
func TestUpdateUserProfileRequest_DropsPrivilegedFields(t *testing.T) {
	body := []byte(`{
		"name": "Mallory",
		"phone_number": "9998887777",
		"role": "admin",
		"is_verified": true,
		"is_active": false
	}`)

	var req UpdateUserProfileRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	m := req.SanitizedUpdateMap()

	for _, forbidden := range []string{"role", "is_verified", "is_active"} {
		if _, present := m[forbidden]; present {
			t.Fatalf("privileged field %q leaked into update map: %+v", forbidden, m)
		}
	}
	if m["name"] != "Mallory" {
		t.Fatalf("expected name to pass through, got %+v", m)
	}
	if m["phone_number"] != "9998887777" {
		t.Fatalf("expected phone_number to pass through, got %+v", m)
	}
}

// Omitted fields must not appear in the map (so UpdateUser skips them rather than
// clearing them), while an explicitly empty phone clears it.
func TestUpdateUserProfileRequest_OmittedVsEmpty(t *testing.T) {
	var omitted UpdateUserProfileRequest
	if err := json.Unmarshal([]byte(`{"name":"Only Name"}`), &omitted); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	m := omitted.SanitizedUpdateMap()
	if _, ok := m["phone_number"]; ok {
		t.Fatalf("omitted phone_number should not be in map: %+v", m)
	}

	var emptyPhone UpdateUserProfileRequest
	if err := json.Unmarshal([]byte(`{"phone_number":""}`), &emptyPhone); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	m2 := emptyPhone.SanitizedUpdateMap()
	if v, ok := m2["phone_number"]; !ok || v != "" {
		t.Fatalf("explicit empty phone_number should pass through to clear it: %+v", m2)
	}
}
