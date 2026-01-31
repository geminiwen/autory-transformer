package errors

import "fmt"

// APIError represents an Anthropic-style API error
type APIError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Error types
const (
	InvalidRequestError    = "invalid_request_error"
	AuthenticationError    = "authentication_error"
	RateLimitError         = "rate_limit_error"
	APIErrorType           = "api_error"
	OverloadedError        = "overloaded_error"
	NotFoundError          = "not_found_error"
)

// NewAPIError creates a new API error
func NewAPIError(errorType, message string) *APIError {
	return &APIError{
		Type:    errorType,
		Message: message,
	}
}

// NewInvalidRequestError creates an invalid request error
func NewInvalidRequestError(message string) *APIError {
	return NewAPIError(InvalidRequestError, message)
}

// NewAuthenticationError creates an authentication error
func NewAuthenticationError(message string) *APIError {
	return NewAPIError(AuthenticationError, message)
}

// MapHTTPStatusToErrorType maps HTTP status codes to Anthropic error types
func MapHTTPStatusToErrorType(statusCode int) string {
	switch statusCode {
	case 400:
		return InvalidRequestError
	case 401:
		return AuthenticationError
	case 404:
		return NotFoundError
	case 429:
		return RateLimitError
	case 500:
		return APIErrorType
	case 503:
		return OverloadedError
	default:
		return APIErrorType
	}
}
