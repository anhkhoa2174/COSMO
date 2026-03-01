package schema

import "errors"

// APIResponse is the standard API response format
// Matches Python FastAPI response structure
type APIResponse[T any] struct {
	Status string `json:"status"`
	Data   T      `json:"data"`
	Error  *Error `json:"error,omitempty"`
}

// Error represents an API error
// Matches Python FastAPI error response structure
type Error struct {
	ErrorCode int         `json:"error_code"`
	Message   string      `json:"message"`
	Detail    interface{} `json:"detail"`
}

// PaginatedResponse represents a paginated list response
type PaginatedResponse[T any] struct {
	List   []T   `json:"list"`
	Total  int64 `json:"total"`
	Offset int   `json:"offset"`
	Limit  int   `json:"limit"`
}

// SuccessResponse creates a success response
func SuccessResponse[T any](data T) APIResponse[T] {
	return APIResponse[T]{
		Status: "success",
		Data:   data,
	}
}

// ErrorResponse creates an error response
// detail can be nil, string, or any other type
func ErrorResponse(code int, message string, detail interface{}) APIResponse[interface{}] {
	// Convert empty string detail to nil to match Python API
	if detail == "" {
		detail = nil
	}
	return APIResponse[interface{}]{
		Status: "error",
		Error: &Error{
			ErrorCode: code,
			Message:   message,
			Detail:    detail,
		},
	}
}

var ErrForbidden = errors.New("forbidden")
