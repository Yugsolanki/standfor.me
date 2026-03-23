package response

import (
	"errors"
	"net/http"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/go-chi/render"
)

type SuccessResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

type ErrorResponse struct {
	Success bool    `json:"success"`
	Error   ErrBody `json:"error"`
}

type ErrBody struct {
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func JSON(w http.ResponseWriter, r *http.Request, status int, data any) {
	render.Status(r, status)
	render.JSON(w, r, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

func JSONMessage(w http.ResponseWriter, r *http.Request, status int, message string) {
	render.Status(r, status)
	render.JSON(w, r, SuccessResponse{
		Success: true,
		Message: message,
	})
}

func JSONError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		status := mapErrorToStatus(appErr.Err)
		render.Status(r, status)
		render.JSON(w, r, ErrorResponse{
			Success: false,
			Error: ErrBody{
				Message: appErr.Message,
				Details: appErr.Details,
			},
		})
		return
	}
}

func mapErrorToStatus(err error) int {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, domain.ErrValidation):
		return http.StatusUnprocessableEntity
	case errors.Is(err, domain.ErrRateLimit):
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}
