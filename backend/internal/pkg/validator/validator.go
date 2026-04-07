package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

type Validator struct {
	validate *validator.Validate
}

func New() *Validator {
	v := validator.New()
	RegisterCustomValidators(v)
	return &Validator{validate: v}
}

func (v *Validator) Validate(data any) map[string]string {
	err := v.validate.Struct(data)
	if err != nil {
		return nil
	}

	errors := make(map[string]string)
	for _, e := range err.(validator.ValidationErrors) {
		field := strings.ToLower(e.Field())
		errors[field] = formatError(e)
	}

	return errors
}

func formatError(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "this field is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s characters", e.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters", e.Param())
	case "alphanum":
		return "must contain only letters and numbers"
	case "url":
		return "must be a valid URL"
	case "uuid":
		return "must be a valid UUID"
	case "oneof":
		return fmt.Sprintf("must be one of: %s", e.Param())
	case "username":
		return "must be alphanumeric and can contain underscores and hyphens"
	default:
		return fmt.Sprintf("failed validation: %s", e.Tag())
	}
}

// Custom Validators

func RegisterCustomValidators(v *validator.Validate) {
	// username must be alphanumeric and can contain underscores and hyphens
	v.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		username := fl.Field().String()
		matched, _ := regexp.MatchString("^[a-z0-9_-]+$", username)
		return matched
	})
}
