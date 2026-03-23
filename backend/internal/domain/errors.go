package domain

import "errors"

var (
	// Generic
	ErrNotFound     = errors.New("resources not found")
	ErrConflict     = errors.New("resources already exists")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrInvalidInput = errors.New("invalid input")
	ErrInternal     = errors.New("internal server error")
	ErrValidation   = errors.New("validation failed")
	ErrRateLimit    = errors.New("rate limit exceeded")

	// User-specific
	ErrUserNotFoud     = errors.New("user not found")
	ErrEmailTaken      = errors.New("email already registered")
	ErrUsernameTaken   = errors.New("username already taken")
	ErrInvalidPassword = errors.New("invalid password")
)

type AppError struct {
	Err     error
	Message string
	Details map[string]string
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Err.Error()
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(err error, message string) *AppError {
	return &AppError{
		Err:     err,
		Message: message,
	}
}

func NewValidationError(details map[string]string) *AppError {
	return &AppError{
		Err:     ErrValidation,
		Message: "validation failed",
		Details: details,
	}
}
