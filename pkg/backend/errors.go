package backend

import (
	"fmt"
)

// Error codes
const (
	ErrCodeUnsupportedBackend   = "UnsupportedBackend"
	ErrCodeInvalidConfiguration = "InvalidConfiguration"
	ErrCodeAuthentication       = "AuthenticationError"
	ErrCodeNetwork              = "NetworkError"
	ErrCodeRateLimited          = "RateLimitedError"
	ErrCodeServiceUnavailable   = "ServiceUnavailableError"
	ErrCodeInvalidRequest       = "InvalidRequestError"
	ErrCodeContextLengthExceeded = "ContextLengthExceededError"
	ErrCodeContentFiltered      = "ContentFilteredError"
	ErrCodeUnknown              = "UnknownError"
)

// BackendError represents an error from a chat backend
type BackendError struct {
	Code      string
	Message   string
	Cause     error
	Retryable bool
}

// Error implements the error interface
func (e *BackendError) Error() string {
	msg := fmt.Sprintf("%s: %s", e.Code, e.Message)
	if e.Cause != nil {
		msg += fmt.Sprintf(" (caused by: %v)", e.Cause)
	}
	return msg
}

// Unwrap returns the cause of the error
func (e *BackendError) Unwrap() error {
	return e.Cause
}

// NewBackendError creates a new BackendError
func NewBackendError(code string, message string, cause error) *BackendError {
	retryable := false
	switch code {
	case ErrCodeNetwork, ErrCodeRateLimited, ErrCodeServiceUnavailable:
		retryable = true
	}
	
	return &BackendError{
		Code:      code,
		Message:   message,
		Cause:     cause,
		Retryable: retryable,
	}
}