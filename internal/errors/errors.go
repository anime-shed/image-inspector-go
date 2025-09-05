package errors

import (
	"fmt"
	"net/http"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "validation"
	ErrorTypeNetwork      ErrorType = "network"
	ErrorTypeProcessing   ErrorType = "processing"
	ErrorTypeTimeout      ErrorType = "timeout"
	ErrorTypeUnauthorized ErrorType = "unauthorized"
	ErrorTypeNotFound     ErrorType = "not_found"
	ErrorTypeInternal     ErrorType = "internal"
)

// AppError represents a structured application error
type AppError struct {
	Type       ErrorType `json:"type"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	StatusCode int       `json:"status_code"`
	Cause      error     `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewValidationError creates a new validation error
func NewValidationError(message string, cause error) *AppError {
	return &AppError{
		Type:       ErrorTypeValidation,
		Message:    message,
		StatusCode: http.StatusBadRequest,
		Cause:      cause,
	}
}

// NewNetworkError creates a new network error
func NewNetworkError(message string, cause error) *AppError {
	return &AppError{
		Type:       ErrorTypeNetwork,
		Message:    message,
		StatusCode: http.StatusBadGateway,
		Cause:      cause,
	}
}

// NewProcessingError creates a new processing error
func NewProcessingError(message string, cause error) *AppError {
	return &AppError{
		Type:       ErrorTypeProcessing,
		Message:    message,
		StatusCode: http.StatusUnprocessableEntity,
		Cause:      cause,
	}
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(message string, cause error) *AppError {
	return &AppError{
		Type:       ErrorTypeTimeout,
		Message:    message,
		StatusCode: http.StatusGatewayTimeout,
		Cause:      cause,
	}
}

// NewInternalError creates a new internal error
func NewInternalError(message string, cause error) *AppError {
	return &AppError{
		Type:       ErrorTypeInternal,
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Cause:      cause,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(message string, cause error) *AppError {
	return &AppError{
		Type:       ErrorTypeNotFound,
		Message:    message,
		StatusCode: http.StatusNotFound,
		Cause:      cause,
	}
}

// IsType checks if the error is of a specific type
func IsType(err error, errorType ErrorType) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == errorType
	}
	return false
}

// GetStatusCode extracts the HTTP status code from an error
func GetStatusCode(err error) int {
	if appErr, ok := err.(*AppError); ok {
		return appErr.StatusCode
	}
	return http.StatusInternalServerError
}
