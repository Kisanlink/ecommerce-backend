package responses

// This file defines small, reusable response envelopes aligned with our API style.
// It mirrors the light-weight patterns used in AAA service, but keeps our existing
// handler logic intact. Handlers can opt in to these structs where convenient.

// APIResponse is a generic success envelope.
// Example usage:
//  ctx.JSON(http.StatusOK, responses.APIResponse{Success: true, Data: payload})
// Without changing existing response shapes, this provides a common contract.
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse is a generic error envelope.
// Example usage:
//  ctx.JSON(http.StatusBadRequest, responses.ErrorResponse{Error: "invalid input"})
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
}

// PageInfo contains pagination metadata.
type PageInfo struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// PaginatedResponse wraps list payloads with PageInfo.
// Data should typically be a slice.
type PaginatedResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Page    PageInfo    `json:"page"`
}
