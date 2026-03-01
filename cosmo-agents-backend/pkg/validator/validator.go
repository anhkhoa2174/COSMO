package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator wraps go-playground/validator
type Validator struct {
	validate *validator.Validate
}

// New creates a new validator instance
func New() *Validator {
	return &Validator{
		validate: validator.New(),
	}
}

// Validate validates a struct
func (v *Validator) Validate(data interface{}) error {
	if err := v.validate.Struct(data); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return formatValidationErrors(validationErrors)
		}
		return err
	}
	return nil
}

// ValidationError represents a validation error with field details
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors is a collection of validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// Error implements the error interface
func (ve ValidationErrors) Error() string {
	var messages []string
	for _, err := range ve.Errors {
		messages = append(messages, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return strings.Join(messages, "; ")
}

// formatValidationErrors converts validator errors to a custom format
func formatValidationErrors(errs validator.ValidationErrors) ValidationErrors {
	var validationErrs []ValidationError

	for _, err := range errs {
		validationErrs = append(validationErrs, ValidationError{
			Field:   err.Field(),
			Message: formatFieldError(err),
		})
	}

	return ValidationErrors{Errors: validationErrs}
}

// formatFieldError creates a user-friendly error message
func formatFieldError(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "min":
		return fmt.Sprintf("Minimum value is %s", err.Param())
	case "max":
		return fmt.Sprintf("Maximum value is %s", err.Param())
	case "uuid":
		return "Invalid UUID format"
	case "url":
		return "Invalid URL format"
	case "oneof":
		return fmt.Sprintf("Must be one of: %s", err.Param())
	case "gt":
		return fmt.Sprintf("Must be greater than %s", err.Param())
	case "gte":
		return fmt.Sprintf("Must be greater than or equal to %s", err.Param())
	case "lt":
		return fmt.Sprintf("Must be less than %s", err.Param())
	case "lte":
		return fmt.Sprintf("Must be less than or equal to %s", err.Param())
	default:
		return fmt.Sprintf("Validation failed on '%s' tag", err.Tag())
	}
}
