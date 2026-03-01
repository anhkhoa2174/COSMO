package field_validation

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	customFieldService "github.com/rockship/cosmo-agents-go/internal/service/custom_field"
)

// ValidationResult represents the result of form validation
type ValidationResult struct {
	Valid  bool                   `json:"valid"`
	Errors map[string]string      `json:"errors"`
	Data   map[string]interface{} `json:"data"`
}

// ValidationError represents a field validation error
type ValidationError struct {
	FieldName string
	Message   string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.FieldName, e.Message)
}

// FieldValidator is a function type for validating field values
type FieldValidator func(value interface{}, fieldDef map[string]interface{}) *string

// FieldValidationService provides strong typing for form field validation
type FieldValidationService struct {
	validators map[string]FieldValidator
}

// NewFieldValidationService creates a new field validation service
func NewFieldValidationService() *FieldValidationService {
	service := &FieldValidationService{
		validators: make(map[string]FieldValidator),
	}

	// Register default validators
	service.registerDefaultValidators()

	return service
}

// RegisterValidator registers a custom validator for a data type
func (s *FieldValidationService) RegisterValidator(dataType string, validator FieldValidator) {
	s.validators[dataType] = validator
}

// registerDefaultValidators sets up default field validators
func (s *FieldValidationService) registerDefaultValidators() {
	s.validators[string(domain.CustomFieldDataTypeEmail)] = s.validateEmail
	s.validators[string(domain.CustomFieldDataTypeNumber)] = s.validateNumber
	s.validators[string(domain.CustomFieldDataTypeURL)] = s.validateURL
	s.validators[string(domain.CustomFieldDataTypeDate)] = s.validateDate
	s.validators[string(domain.CustomFieldDataTypeText)] = s.validateText
	s.validators[string(domain.CustomFieldDataTypeSelect)] = s.validateSelect
	s.validators["phone_number"] = s.validatePhone
}

// validateEmail validates email format
func (s *FieldValidationService) validateEmail(value interface{}, fieldDef map[string]interface{}) *string {
	str, ok := value.(string)
	if !ok {
		msg := "Must be a string"
		return &msg
	}

	if _, err := mail.ParseAddress(str); err != nil {
		msg := "Invalid email format"
		return &msg
	}

	return nil
}

// validateNumber validates number format
func (s *FieldValidationService) validateNumber(value interface{}, fieldDef map[string]interface{}) *string {
	switch v := value.(type) {
	case int, int32, int64, float32, float64:
		return nil
	case string:
		if _, err := strconv.ParseFloat(v, 64); err != nil {
			msg := "Not a valid number"
			return &msg
		}
		return nil
	default:
		msg := "Not a valid number"
		return &msg
	}
}

// validateURL validates URL format
func (s *FieldValidationService) validateURL(value interface{}, fieldDef map[string]interface{}) *string {
	str, ok := value.(string)
	if !ok {
		msg := "Must be a string"
		return &msg
	}

	if _, err := url.Parse(str); err != nil {
		msg := "Invalid URL format"
		return &msg
	}

	return nil
}

// validateDate validates date format
func (s *FieldValidationService) validateDate(value interface{}, fieldDef map[string]interface{}) *string {
	str, ok := value.(string)
	if !ok {
		msg := "Must be a string"
		return &msg
	}

	if _, err := customFieldService.ParseDateTime(str); err != nil {
		msg := "Invalid date format"
		return &msg
	}

	return nil
}

// validateText validates text length
func (s *FieldValidationService) validateText(value interface{}, fieldDef map[string]interface{}) *string {
	str, ok := value.(string)
	if !ok {
		msg := "Must be a string"
		return &msg
	}

	if len(str) > 255 {
		msg := "Text too long (max 255 characters)"
		return &msg
	}

	return nil
}

// validateSelect validates select option
func (s *FieldValidationService) validateSelect(value interface{}, fieldDef map[string]interface{}) *string {
	str, ok := value.(string)
	if !ok {
		msg := "Must be a string"
		return &msg
	}

	optionsRaw, hasOptions := fieldDef["options"]
	if !hasOptions {
		msg := "No options defined for select field"
		return &msg
	}

	options, ok := optionsRaw.([]interface{})
	if !ok {
		msg := "Invalid options format"
		return &msg
	}

	for _, opt := range options {
		if optStr, ok := opt.(string); ok && optStr == str {
			return nil
		}
	}

	msg := "Value not in options"
	return &msg
}

// validatePhone validates phone number format
func (s *FieldValidationService) validatePhone(value interface{}, fieldDef map[string]interface{}) *string {
	str, ok := value.(string)
	if !ok {
		msg := "Must be a string"
		return &msg
	}

	phoneRegex := regexp.MustCompile(`^\+?\d{0,3}[-.\s]?(\(?\d{1,4}\)?)?[-.\s]?\d{1,4}[-.\s]?\d{1,4}[-.\s]?\d{1,9}(x\d{1,6})?$`)
	if !phoneRegex.MatchString(str) {
		msg := "Invalid phone number format"
		return &msg
	}

	return nil
}

// isEmpty checks if a value is empty
func (s *FieldValidationService) isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}

	if str, ok := value.(string); ok {
		return len(str) == 0
	}

	return false
}

// ValidateField validates a single field value
func (s *FieldValidationService) ValidateField(value interface{}, fieldDef map[string]interface{}) (interface{}, error) {
	// Skip validation for empty values (handled by required field check)
	if s.isEmpty(value) {
		return value, nil
	}

	fieldName, _ := fieldDef["name"].(string)
	if fieldName == "" {
		fieldName = "Unknown"
	}

	dataType, _ := fieldDef["data_type"].(string)
	if dataType == "" {
		dataType = "text"
	}

	// Get validator for this type
	validator, exists := s.validators[dataType]
	if !exists {
		return value, nil // No validator for this type
	}

	// Run validation
	if errMsg := validator(value, fieldDef); errMsg != nil {
		return nil, &ValidationError{
			FieldName: fieldName,
			Message:   *errMsg,
		}
	}

	return value, nil
}

// ValidateForm validates form data against field definitions
func (s *FieldValidationService) ValidateForm(
	fields []map[string]interface{},
	formData map[string]interface{},
) ValidationResult {
	// Create a copy to avoid modifying the original
	result := make(map[string]interface{})
	for k, v := range formData {
		result[k] = v
	}

	errors := make(map[string]string)

	// Create field map for quick lookup
	fieldMap := make(map[string]map[string]interface{})
	for _, field := range fields {
		if normalizedName, ok := field["normalized_name"].(string); ok {
			fieldMap[normalizedName] = field
		}
	}

	// Check required fields
	for name, field := range fieldMap {
		isRequired, _ := field["is_required"].(bool)
		if isRequired {
			value, exists := result[name]
			if !exists || s.isEmpty(value) {
				fieldName, _ := field["name"].(string)
				if fieldName == "" {
					fieldName = name
				}
				errors[name] = fmt.Sprintf("%s is required", fieldName)
			}
		}
	}

	// If we already have errors, return early
	if len(errors) > 0 {
		return ValidationResult{
			Valid:  false,
			Errors: errors,
			Data:   result,
		}
	}

	// Validate each field value
	for name, value := range result {
		if field, exists := fieldMap[name]; exists {
			validatedValue, err := s.ValidateField(value, field)
			if err != nil {
				if valErr, ok := err.(*ValidationError); ok {
					errors[name] = valErr.Message
				} else {
					errors[name] = err.Error()
				}
			} else {
				result[name] = validatedValue
			}
		}
	}

	// Apply fallback values for missing fields
	for name, field := range fieldMap {
		value, exists := result[name]
		if !exists || s.isEmpty(value) {
			if fallback, hasFallback := field["fallback_value"]; hasFallback {
				result[name] = fallback
			}
		}
	}

	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
		Data:   result,
	}
}
