package domain

import (
	"errors"
	"fmt"
)

var (
	// Generic
	ErrNotFound             = errors.New("resources not found")
	ErrConflict             = errors.New("resources already exists")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrForbidden            = errors.New("forbidden")
	ErrInternal             = errors.New("internal server error")
	ErrValidation           = errors.New("validation failed")
	ErrRateLimit            = errors.New("rate limit exceeded")
	ErrTimeout              = errors.New("request timeout")
	ErrCanceled             = errors.New("request cancelled")
	ErrServiceUnavailable   = errors.New("service unavailable")
	ErrExternalServiceError = errors.New("external service error")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrPayloadTooLarge      = errors.New("payload too large")
	ErrBadRequest           = errors.New("bad request")
)

type AppError struct {
	Err     error
	Message string
	Details map[string]string
	Op      string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %s", e.Op, e.Message, e.Cause.Error())
	}
	return fmt.Sprintf("%s: %s", e.Op, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(op string, err error, message string) *AppError {
	return &AppError{
		Op:      op,
		Err:     err,
		Message: message,
	}
}

// --- Constructor Helpers ---
func NewNotFoundError(op, message string) *AppError {
	return &AppError{Err: ErrNotFound, Op: op, Message: message}
}

func NewValidationError(op, message string, details map[string]string) *AppError {
	return &AppError{
		Err:     ErrValidation,
		Op:      op,
		Message: message,
		Details: details,
	}
}

func NewConflictError(op, message string) *AppError {
	return &AppError{Err: ErrConflict, Op: op, Message: message}
}

func NewUnauthorizedError(op, message string) *AppError {
	return &AppError{Err: ErrUnauthorized, Op: op, Message: message}
}

func NewForbiddenError(op, message string) *AppError {
	return &AppError{Err: ErrForbidden, Op: op, Message: message}
}

func NewInternalError(op string, cause error) *AppError {
	return &AppError{Err: ErrInternal, Op: op, Message: "An unexpected error occurred. Please try again.", Cause: cause}
}

func NewRateLimitError(op string) *AppError {
	return &AppError{Err: ErrRateLimit, Op: op, Message: "You are sending too many requests. Please try again later."}
}

func NewTimeoutError(op string) *AppError {
	return &AppError{Err: ErrTimeout, Op: op, Message: "The request took too long to process. Please try again."}
}

func NewCanceledError(op string) *AppError {
	return &AppError{Err: ErrCanceled, Op: op, Message: "The request was cancelled."}
}

func NewServiceUnavailableError(op string) *AppError {
	return &AppError{Err: ErrServiceUnavailable, Op: op, Message: "The service is currently unavailable. Please try again later."}
}

func NewExternalServiceError(op string) *AppError {
	return &AppError{Err: ErrExternalServiceError, Op: op, Message: "The external service is currently unavailable. Please try again later."}
}

func NewInvalidCredentialsError(op string) *AppError {
	return &AppError{Err: ErrInvalidCredentials, Op: op, Message: "Invalid credentials. Please try again."}
}

func NewPayloadTooLargeError(op string) *AppError {
	return &AppError{Err: ErrPayloadTooLarge, Op: op, Message: "Payload too large. Please try again."}
}

func NewBadRequestError(op, message string) *AppError {
	return &AppError{Err: ErrBadRequest, Op: op, Message: message}
}
