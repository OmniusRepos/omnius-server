package models

// ApiResponse is the standard YTS-compatible API response wrapper
type ApiResponse[T any] struct {
	Status        string `json:"status"`
	StatusMessage string `json:"status_message"`
	Data          T      `json:"data"`
}

// NewSuccessResponse creates a successful API response
func NewSuccessResponse[T any](data T) ApiResponse[T] {
	return ApiResponse[T]{
		Status:        "ok",
		StatusMessage: "Query was successful",
		Data:          data,
	}
}

// NewErrorResponse creates an error API response
func NewErrorResponse(message string) ApiResponse[any] {
	return ApiResponse[any]{
		Status:        "error",
		StatusMessage: message,
		Data:          nil,
	}
}
