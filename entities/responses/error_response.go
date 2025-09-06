package responses

// Domain-level error codes where useful. Handlers can populate Code when
// returning typed errors. Kept minimal and optional.
const (
	ErrBadRequest    = "bad_request"
	ErrUnauthorized  = "unauthorized"
	ErrForbidden     = "forbidden"
	ErrNotFound      = "not_found"
	ErrConflict      = "conflict"
	ErrUnprocessable = "unprocessable_entity"
	ErrRateLimited   = "rate_limited"
	ErrInternal      = "internal_error"
)
