package requests

// UpdateUserProfileRequest is the sanitized payload a user may submit to update
// THEIR OWN account via the self-service endpoint (PUT /users/profile).
//
// It deliberately exposes ONLY non-privileged fields. Privileged columns
// (role, is_verified, is_active) are intentionally absent so they cannot be set
// through mass assignment — a normal user must never be able to self-promote to
// admin. Changes to those columns go exclusively through the admin-guarded
// user-management routes.
type UpdateUserProfileRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	PhoneNumber *string `json:"phone_number,omitempty" validate:"omitempty,max=20"`
}

// SanitizedUpdateMap returns a map containing only the fields the user actually
// provided, restricted to the self-service-allowed columns. It is the ONLY
// bridge from this request to UserService.UpdateUser, guaranteeing no privileged
// field (role/is_verified/is_active) can reach the service from the self path.
func (r *UpdateUserProfileRequest) SanitizedUpdateMap() map[string]interface{} {
	out := make(map[string]interface{}, 2)
	if r.Name != nil {
		out["name"] = *r.Name
	}
	if r.PhoneNumber != nil {
		out["phone_number"] = *r.PhoneNumber
	}
	return out
}
