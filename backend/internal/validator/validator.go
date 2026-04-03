package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())
}

// Validate checks a struct against its `validate` tags.
// Returns nil if valid; otherwise a slice of FieldError.
func Validate(data interface{}) map[string]string {
	err := validate.Struct(data)
	if err == nil {
		return nil
	}

	// Convert validator errors into a user-friendly map
	errors := make(map[string]string)
	for _, e := range err.(validator.ValidationErrors) {
		// e.Field() returns the struct field name
		// We convert it to lowercase for JSON conventions
		field := strings.ToLower(e.Field())
		errors[field] = formatMessage(e)
	}

	return errors
}

func formatMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", e.Field())
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s characters", e.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters", e.Param())
	case "url":
		return "must be a valid URL"
	case "alphanum":
		return "must contain only letters and numbers"
	default:
		return fmt.Sprintf("failed on '%s' validation", e.Tag())
	}
}
