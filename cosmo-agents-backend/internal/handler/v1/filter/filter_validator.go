package filter

import (
	"fmt"
	"reflect"

	"github.com/google/uuid"
)

// AllowedFilterKeys defines which filter keys are safe to use
var AllowedFilterKeys = map[string]bool{
	"is_deleted": true,
	"status":     true,
	"replied":    true,
	"name":       true, // Generic string field for testing
	// Add more as needed, but validate each one carefully
}

// AllowedFilterOperators defines which MongoDB-style operators are safe
var AllowedFilterOperators = map[string]bool{
	"$in": true,
	"$eq": true,
	"$ne": true,
	// Do not add $where, $regex or other dangerous operators
}

// FilterValidationError represents a filter validation error
type FilterValidationError struct {
	Field   string
	Message string
}

func (e FilterValidationError) Error() string {
	return fmt.Sprintf("filter validation error for field '%s': %s", e.Field, e.Message)
}

// ValidateFilter safely validates user-provided filters to prevent injection attacks
func ValidateFilter(filter map[string]interface{}) error {
	if filter == nil {
		return nil // Empty filter is always safe
	}

	for key, value := range filter {
		// Check if the filter key is in our whitelist
		if !AllowedFilterKeys[key] {
			return FilterValidationError{
				Field:   key,
				Message: "filter key not allowed",
			}
		}

		// Validate the filter value structure
		if err := validateFilterValue(key, value); err != nil {
			return err
		}
	}

	return nil
}

// validateFilterValue validates the structure and content of filter values
func validateFilterValue(key string, value interface{}) error {
	switch v := value.(type) {
	case map[string]interface{}:
		// Handle MongoDB-style operators like {"$in": [...]}
		for operator, operatorValue := range v {
			if !AllowedFilterOperators[operator] {
				return FilterValidationError{
					Field:   key,
					Message: fmt.Sprintf("operator '%s' not allowed", operator),
				}
			}

			// Validate operator-specific values
			if err := validateOperatorValue(key, operator, operatorValue); err != nil {
				return err
			}
		}

	case string, bool, int, int64, float64:
		// Simple values are generally safe for whitelisted keys
		if err := validateSimpleValue(key, v); err != nil {
			return err
		}

	default:
		return FilterValidationError{
			Field:   key,
			Message: fmt.Sprintf("unsupported value type: %T", v),
		}
	}

	return nil
}

// validateOperatorValue validates values for specific operators
func validateOperatorValue(key, operator string, value interface{}) error {
	switch operator {
	case "$in":
		// $in should contain an array
		slice := reflect.ValueOf(value)
		if slice.Kind() != reflect.Slice {
			return FilterValidationError{
				Field:   key,
				Message: "$in operator requires an array value",
			}
		}

		// Validate each item in the array
		for i := 0; i < slice.Len(); i++ {
			item := slice.Index(i).Interface()
			if err := validateSimpleValue(key, item); err != nil {
				return FilterValidationError{
					Field:   key,
					Message: fmt.Sprintf("invalid item at index %d: %s", i, err.Error()),
				}
			}
		}

		// Limit array size to prevent DoS
		if slice.Len() > 100 {
			return FilterValidationError{
				Field:   key,
				Message: "array too large (max 100 items)",
			}
		}

	case "$eq", "$ne":
		// Simple equality/inequality operations
		if err := validateSimpleValue(key, value); err != nil {
			return err
		}
	}

	return nil
}

// validateSimpleValue validates simple scalar values
func validateSimpleValue(key string, value interface{}) error {
	switch key {
	case "is_deleted":
		// Should be boolean
		if _, ok := value.(bool); !ok {
			return FilterValidationError{
				Field:   key,
				Message: "must be boolean",
			}
		}

	case "replied":
		// Should be boolean
		if _, ok := value.(bool); !ok {
			return FilterValidationError{
				Field:   key,
				Message: "must be boolean",
			}
		}

	case "status":
		// Should be string and validate against known statuses
		str, ok := value.(string)
		if !ok {
			return FilterValidationError{
				Field:   key,
				Message: "must be string",
			}
		}

		// Validate against known conversation statuses
		validStatuses := map[string]bool{
			"active":   true,
			"archived": true,
			"deleted":  true,
			// Add other valid statuses as needed
		}

		if !validStatuses[str] {
			return FilterValidationError{
				Field:   key,
				Message: "invalid status value",
			}
		}
	}

	// Additional validation: prevent excessively long strings
	if str, ok := value.(string); ok && len(str) > 1000 {
		return FilterValidationError{
			Field:   key,
			Message: "string value too long (max 1000 characters)",
		}
	}

	return nil
}

// SanitizeUUID validates and sanitizes UUID strings to prevent injection
func SanitizeUUID(value string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, FilterValidationError{
			Field:   "uuid",
			Message: "UUID cannot be empty",
		}
	}

	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, FilterValidationError{
			Field:   "uuid",
			Message: "invalid UUID format",
		}
	}

	return parsed, nil
}
