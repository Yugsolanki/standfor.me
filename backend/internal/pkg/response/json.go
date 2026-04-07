package response

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/Yugsolanki/standfor-me/internal/pkg/logger"
	"github.com/Yugsolanki/standfor-me/internal/pkg/requestid"
)

type SuccessResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

type ErrorResponse struct {
	Success   bool    `json:"success"`
	Error     ErrBody `json:"error"`
	RequestID string  `json:"request_id,omitempty"`
}

type ErrBody struct {
	Message string            `json:"message"`
	Code    string            `json:"code,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// --- Success Response Functions ---

// JSON writes a successful JSON response with the given status code and data.
// The data is wrapped in a standard SuccessResponse envelope.
//
// Usage:
//
//	response.JSON(w, r, http.StatusOK, profile)
func JSON(w http.ResponseWriter, r *http.Request, status int, data any) {
	writeJSON(w, r, status, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// JSONMessage writes a successful JSON response with just a message (no data payload).
// Useful for operations that don't return a resource (e.g., "cause added successfully").
//
// Usage:
//
//	response.JSONMessage(w, r, http.StatusOK, "Profile updated successfully")
func JSONMessage(w http.ResponseWriter, r *http.Request, status int, message string) {
	writeJSON(w, r, status, SuccessResponse{
		Success: true,
		Message: message,
	})
}

// ---- Error Response Functions ----

// JSONError processes an error and writes the appropriate HTTP error response.
//
// It handles:
//  1. Context cancellation (client disconnected) → 499 or skip
//  2. Context deadline exceeded (timeout) → 504 Gateway Timeout
//  3. domain.AppError → mapped HTTP status with structured error body
//  4. Unknown errors → 500 Internal Server Error (details hidden from client)
//
// This is the ONLY function handlers should use for error responses.
// It automatically enriches the canonical log line with error details.
//
// Usage:
//
//	profile, err := h.service.GetProfile(r.Context(), username)
//	if err != nil {
//	    response.JSONError(w, r, err)
//	    return
//	}
func JSONError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()
	requestID := requestid.GetRequestID(ctx)

	// 1. Check for context errors first
	if ctxErr := ctx.Err(); ctxErr != nil {
		handleContextError(w, r, ctxErr, err, requestID)
		return
	}

	// 2. Check for domain.AppError
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		handleAppError(w, r, appErr, requestID)
		return
	}

	// 3. Check for context errors in the error chain
	// Sometimes the context error is wrapped inside the original error
	if errors.Is(err, context.DeadlineExceeded) {
		handleContextError(w, r, context.DeadlineExceeded, err, requestID)
		return
	}
	if errors.Is(err, context.Canceled) {
		handleContextError(w, r, context.Canceled, err, requestID)
		return
	}

	// 4. Unknown/Unexpected error
	handleUnknownError(w, r, err, requestID)
}

// JSONValidationError writes a 422 Unprocessable Entity response for validation failures.
// It accepts the field-level errors map returned by validator.Validate() and wraps them
// into the standard ErrorResponse envelope.
//
// Usage:
//
//	if errs := s.validator.Validate(body); errs != nil {
//	    response.JSONValidationError(w, r, errs)
//	    return
//	}
func JSONValidationError(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	ctx := r.Context()
	requestID := requestid.GetRequestID(ctx)

	logger.AddFields(ctx, map[string]any{
		"error":         "validation failed",
		"error_type":    "VALIDATION_ERROR",
		"error_status":  http.StatusUnprocessableEntity,
		"error_details": errors,
	})

	writeJSON(w, r, http.StatusUnprocessableEntity, ErrorResponse{
		Success:   false,
		RequestID: requestID,
		Error: ErrBody{
			Message: "Validation failed. Please check the errors and try again.",
			Code:    "VALIDATION_ERROR",
			Details: errors,
		},
	})
}

// handleContextError processes context-related errors (timeout, cancellation)
func handleContextError(w http.ResponseWriter, r *http.Request, ctxErr, originalErr error, requestID string) {
	ctx := r.Context()

	switch {
	case errors.Is(ctxErr, context.DeadlineExceeded):
		// The request exceeded its timeout (set by the Timeout middleware
		// or a per-operation context).
		logger.AddFields(ctx, map[string]any{
			"error":       "context_deadline_exceeded",
			"error_cause": originalErr.Error(),
		})

		writeJSON(w, r, http.StatusGatewayTimeout, ErrorResponse{
			Success:   false,
			RequestID: requestID,
			Error: ErrBody{
				Message: "The request took too long to process. Please try again.",
				Code:    "TIMEOUT",
			},
		})

	case errors.Is(ctxErr, context.Canceled):
		// The client disconnected before the server finished processing.
		// This is common and usually not an error worth alerting on.
		//
		// 499 is a non-standard status (used by nginx) to indicate
		// "Client Closed Request." We log it but use 499 for observability.
		logger.AddFields(ctx, map[string]any{
			"error":            "context_cancelled",
			"client_cancelled": true,
		})

		// Most clients won't see this response since they already disconnected,
		// but we write it for consistency and logging.
		writeJSON(w, r, 499, ErrorResponse{
			Success:   false,
			RequestID: requestID,
			Error: ErrBody{
				Message: "The request was cancelled",
				Code:    "CANCELLED",
			},
		})
	}
}

// handleAppError processes structured domain errors
func handleAppError(w http.ResponseWriter, r *http.Request, appErr *domain.AppError, requestID string) {
	ctx := r.Context()
	status := mapErrorToStatus(appErr)

	// Enrich the canonical log line with error context
	logFields := map[string]any{
		"error":        appErr.Message,
		"error_type":   mapErrorToCode(appErr),
		"error_status": status,
	}
	if appErr.Op != "" {
		logFields["error_op"] = appErr.Op
	}
	if appErr.Cause != nil {
		logFields["error_cause"] = appErr.Cause.Error()
	}
	if len(appErr.Details) > 0 {
		logFields["error_details"] = appErr.Details
	}
	logger.AddFields(ctx, logFields)

	writeJSON(w, r, status, ErrorResponse{
		Success:   false,
		RequestID: requestID,
		Error: ErrBody{
			Message: appErr.Message,
			Code:    mapErrorToCode(appErr),
			Details: appErr.Details,
		},
	})
}

// handleUnknownError processes unexpected errors that don't match any known type.
// The actual error is logged but NEVER exposed to the client.
func handleUnknownError(w http.ResponseWriter, r *http.Request, err error, requestID string) {
	ctx := r.Context()

	logger.AddFields(ctx, map[string]any{
		"error":       err.Error(),
		"error_type":  "INTERNAL",
		"error_class": "unhandled",
	})

	writeJSON(w, r, http.StatusInternalServerError, ErrorResponse{
		Success:   false,
		RequestID: requestID,
		Error: ErrBody{
			Message: "An unexpected error occurred. Please try again.",
			Code:    "INTERNAL",
		},
	})
}

// --- Mapping Functions ---

// mapErrorToStatus maps a domain.AppError to an HTTP status code.
func mapErrorToStatus(err error) int {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound // 404
	case errors.Is(err, domain.ErrValidation):
		return http.StatusUnprocessableEntity // 422
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict // 409
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized // 401
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden // 403
	case errors.Is(err, domain.ErrRateLimit):
		return http.StatusTooManyRequests // 429
	case errors.Is(err, domain.ErrTimeout):
		return http.StatusGatewayTimeout // 504
	case errors.Is(err, domain.ErrCanceled):
		return 499 // client closed request
	case errors.Is(err, domain.ErrPayloadTooLarge):
		return http.StatusRequestEntityTooLarge // 413
	case errors.Is(err, domain.ErrBadRequest):
		return http.StatusBadRequest // 400
	case errors.Is(err, domain.ErrServiceUnavailable):
		return http.StatusServiceUnavailable // 503
	case errors.Is(err, domain.ErrExternalServiceError):
		return http.StatusBadGateway // 502
	case errors.Is(err, domain.ErrInvalidCredentials):
		return http.StatusUnauthorized // 401
	default:
		return http.StatusInternalServerError // 500
	}
}

// mapErrorToCode returns a machine-readable error code string.
// These are stable identifiers that frontend clients can switch on.
func mapErrorToCode(err *domain.AppError) string {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, domain.ErrValidation):
		return "VALIDATION_ERROR"
	case errors.Is(err, domain.ErrConflict):
		return "CONFLICT"
	case errors.Is(err, domain.ErrUnauthorized):
		return "UNAUTHORIZED"
	case errors.Is(err, domain.ErrForbidden):
		return "FORBIDDEN"
	case errors.Is(err, domain.ErrRateLimit):
		return "RATE_LIMIT_EXCEEDED"
	case errors.Is(err, domain.ErrTimeout):
		return "TIMEOUT"
	case errors.Is(err, domain.ErrCanceled):
		return "CANCELLED"
	case errors.Is(err, domain.ErrPayloadTooLarge):
		return "PAYLOAD_TOO_LARGE"
	case errors.Is(err, domain.ErrBadRequest):
		return "BAD_REQUEST"
	case errors.Is(err, domain.ErrServiceUnavailable):
		return "SERVICE_UNAVAILABLE"
	case errors.Is(err, domain.ErrExternalServiceError):
		return "EXTERNAL_SERVICE_ERROR"
	case errors.Is(err, domain.ErrInvalidCredentials):
		return "INVALID_CREDENTIALS"
	default:
		return "INTERNAL"
	}
}

// ---- Internal Helpers ----

// writeJSON marshals data to JSON and writes it to the response.
// If marshaling fails, it falls back to a plain text error.
func writeJSON(w http.ResponseWriter, r *http.Request, status int, data any) {
	// Check if client is still connected before doing work.
	if err := r.Context().Err(); err != nil {
		// Client is gone. Don't bother writing.
		// The canonical logger will still capture the request.
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	body, err := json.Marshal(data)
	if err != nil {
		// JSON marshaling failed — this is a programming error.
		// Fall back to a safe response.
		slog.Error("failed to marshal JSON response",
			"error", err,
			"request_id", requestid.GetRequestID(r.Context()),
		)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"success":false,"error":{"message":"Internal Server Error","code":"INTERNAL"}}`))
		return
	}

	w.WriteHeader(status)
	_, _ = w.Write(body)
}
