package responses

// Convenience helpers for common responses. These are optional utilities that
// keep handlers concise. We don't change existing response shapes; these are
// additive and non-breaking.

// OK builds a simple success response with data.
func OK(data interface{}) APIResponse {
	return APIResponse{Success: true, Data: data}
}

// Msg builds a success response with message only.
func Msg(message string) APIResponse {
	return APIResponse{Success: true, Message: message}
}

// Fail builds an error response.
func Fail(err string) ErrorResponse {
	return ErrorResponse{Success: false, Error: err}
}
