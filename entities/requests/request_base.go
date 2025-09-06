package requests

// This file provides lightweight base interfaces and helpers for request objects,
// inspired by the shared patterns in the AAA service. It does not change any
// existing behaviors; it only adds common contracts that current request types
// in this module can (and already do) satisfy.

// Validatable is implemented by request types that expose field-level validation.
// Example: func (r *AddToCartRequest) IsValid() error { ... }
type Validatable interface {
	IsValid() error
}

// Sanitizable is implemented by request types that sanitize incoming string fields.
// Example: func (r *AddToCartRequest) Sanitize() { ... }
type Sanitizable interface {
	Sanitize()
}

// SanitizeIf calls Sanitize when a request implements Sanitizable.
func SanitizeIf(v any) {
	if s, ok := v.(Sanitizable); ok {
		s.Sanitize()
	}
}

// ValidateIf calls IsValid when a request implements Validatable and returns any error found.
func ValidateIf(v any) error {
	if val, ok := v.(Validatable); ok {
		return val.IsValid()
	}
	return nil
}
