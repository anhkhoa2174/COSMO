package googleads

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// validateStruct validates a struct using validator tags
func validateStruct(s interface{}) error {
	if err := validate.Struct(s); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var errors []string
			for _, validationErr := range validationErrors {
				errors = append(errors, formatValidationError(validationErr))
			}
			return fmt.Errorf("%s", strings.Join(errors, "; "))
		}
		return err
	}
	return nil
}

// formatValidationError formats a validation error into a readable message
func formatValidationError(err validator.FieldError) string {
	field := err.Field()

	// Get JSON tag name if available
	if err.StructField() != "" {
		if tag := getJSONTag(err.StructNamespace()); tag != "" {
			field = tag
		}
	}

	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s", field, err.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", field, err.Param())
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", field)
	default:
		return fmt.Sprintf("%s failed validation for %s", field, err.Tag())
	}
}

// getJSONTag extracts the JSON tag name from a struct field
func getJSONTag(namespace string) string {
	parts := strings.Split(namespace, ".")
	if len(parts) < 2 {
		return ""
	}

	fieldName := parts[len(parts)-1]

	// Convert PascalCase to snake_case
	var result strings.Builder
	for i, r := range fieldName {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}

	return strings.ToLower(result.String())
}

// Helper to get struct field by name using reflection
func getStructField(s interface{}, fieldName string) (reflect.StructField, bool) {
	t := reflect.TypeOf(s)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.FieldByName(fieldName)
}
