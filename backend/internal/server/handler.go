package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/Yugsolanki/standfor-me/internal/pkg/response"
	"github.com/Yugsolanki/standfor-me/internal/validator"
)

// --- Handler Function Types ---

type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

func (s *Server) handle(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			response.JSONError(w, r, err)
		}
	}
}

// --- Request Decoding ---

func Decode[T any](r *http.Request) (T, error) {
	var dest T

	// Check context before doing any work
	if err := r.Context().Err(); err != nil {
		return dest, contextToAppError("Decode", err)
	}

	if r.Body == nil || r.ContentLength == 0 {
		return dest, &domain.AppError{
			Err:     domain.ErrBadRequest,
			Op:      "Decode",
			Message: "Request body must not be empty",
		}
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(r.Body)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Reject extra fields - prevent typos

	if err := decoder.Decode(&dest); err != nil {
		// check if decoding failed because the context was canceled
		// (e.g., client disconnected mid-upload)
		if ctxErr := r.Context().Err(); ctxErr != nil {
			return dest, contextToAppError("Decode", ctxErr)
		}
		return dest, &domain.AppError{
			Err:     domain.ErrBadRequest,
			Op:      "Decode",
			Message: fmt.Sprintf("Invalid JSON: %s", humanizeJSONError(err)),
			Cause:   err,
		}
	}

	// Validate the struct using your existing validator package.
	if validationError := validator.Validate(dest); len(validationError) > 0 {
		return dest, &domain.AppError{
			Err:     domain.ErrValidation,
			Op:      "Decode",
			Message: "Validation failed. Please check the errors and try again.",
			Details: validationError,
		}
	}

	return dest, nil
}

// --- Context Helpers ---

// contextToAppError converts a context error to the appropriate domain.AppError.
func contextToAppError(op string, err error) *domain.AppError {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return domain.NewTimeoutError(op)
	case errors.Is(err, context.Canceled):
		return domain.NewCanceledError(op)
	default:
		return domain.NewInternalError(op, err)
	}
}

// ContextAlive checks if the request context is still valid.
// Use this before expensive operations (DB queries, external API calls, etc.)
// to avoid wasted work if the client has already disconnected or the
// timeout has fired.
//
// Usage:
//
//	func (s *CauseService) HeavyOperation(ctx context.Context) error {
//	    if err := server.ContextAlive(ctx); err != nil {
//	        return err
//	    }
//	    // ... proceed with expensive work
//	}
func ContextAlive(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return contextToAppError("ContextAlive", ctx.Err())
	default:
		return nil
	}
}

// WithOperationTimeout creates a child context with a tighter deadline
// for a specific operation (e.g., a single DB query or external API call).
//
// This is useful when the overall request timeout is 30s, but you want
// individual operations to fail faster.
//
// Usage:
//
//	func (r *CauseRepo) FindByID(ctx context.Context, id string) (*Cause, error) {
//	    ctx, cancel := server.WithOperationTimeout(ctx, 5*time.Second)
//	    defer cancel()
//	    return r.db.QueryRow(ctx, "SELECT ...")
//	}
func WithOperationTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// --- JSON Error Humanization ---

// humanizeJSONError converts Go's technical JSON errors into user-friendly messages.
func humanizeJSONError(err error) string {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError

	switch {
	case errors.As(err, &syntaxErr):
		return fmt.Sprintf("malformed JSON at position %d", syntaxErr.Offset)
	case errors.As(err, &typeErr):
		return fmt.Sprintf(
			"field %q expects type %s but got %s",
			typeErr.Field, typeErr.Type.String(), typeErr.Value,
		)
	default:
		errMsg := err.Error()
		if len(errMsg) > 0 {
			return errMsg
		}
		return "invalid JSON format"
	}
}
