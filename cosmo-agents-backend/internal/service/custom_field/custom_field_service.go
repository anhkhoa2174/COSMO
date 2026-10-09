package custom_field

import (
	"fmt"
	"math"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

var (
	// System fields that are always required/validated
	SystemFields = map[string]SystemField{
		"email": {
			DataType:   domain.CustomFieldDataTypeEmail,
			IsRequired: true,
		},
		"phone": {
			DataType:   "phone_number",
			IsRequired: false,
		},
	}

	// Phone number regex pattern
	phoneRegex = regexp.MustCompile(`^\+?\d{0,3}[-.\s]?(\(?\d{1,4}\)?)?[-.\s]?\d{1,4}[-.\s]?\d{1,4}[-.\s]?\d{1,9}(x\d{1,6})?$`)
)

// SystemField represents a system-defined field
type SystemField struct {
	DataType   domain.CustomFieldDataType
	IsRequired bool
}

// CustomFieldService handles custom field validation
type CustomFieldService struct{}

// NewCustomFieldService creates a new custom field service
func NewCustomFieldService() *CustomFieldService {
	return &CustomFieldService{}
}

// ValidateDataType validates a value against a custom field's data type
func (s *CustomFieldService) ValidateDataType(field map[string]interface{}, value string) (string, error) {
	dataType, _ := field["data_type"].(string)
	fieldName, _ := field["name"].(string)

	switch domain.CustomFieldDataType(dataType) {
	case domain.CustomFieldDataTypeSelect:
		options, ok := field["options"].([]interface{})
		if !ok {
			return "", fmt.Errorf("invalid options format for field %s", fieldName)
		}

		found := false
		for _, opt := range options {
			if optStr, ok := opt.(string); ok && optStr == value {
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("custom field %s with value %s is not in options", fieldName, value)
		}

	case domain.CustomFieldDataTypeNumber:
		// ParseFloat also accepts "NaN" and "Inf", which are not usable numbers.
		if n, err := strconv.ParseFloat(value, 64); err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
			return "", fmt.Errorf("custom field %s with value %s is not a valid number", fieldName, value)
		}

	case domain.CustomFieldDataTypeEmail:
		if !isValidEmail(value) {
			return "", fmt.Errorf("custom field %s with value %s is not a valid email", fieldName, value)
		}

	case domain.CustomFieldDataTypeURL:
		if !isValidURL(value) {
			return "", fmt.Errorf("custom field %s with value %s is not a valid url", fieldName, value)
		}

	case domain.CustomFieldDataTypeDate:
		if _, err := ParseDateTime(value); err != nil {
			return "", fmt.Errorf("custom field %s with value %s is not a valid date", fieldName, value)
		}

	case domain.CustomFieldDataTypeText:
		// Limit is in characters, not bytes: accented (e.g. Vietnamese) text
		// takes 2-3 bytes per character.
		if utf8.RuneCountInString(value) > 255 {
			return "", fmt.Errorf("custom field %s with value %s exceeds maximum length (255)", fieldName, value)
		}
	}

	return value, nil
}

// ValidateCustomFieldsModel validates custom field model for create and update
func (s *CustomFieldService) ValidateCustomFieldsModel(customFields []map[string]interface{}) ([]map[string]interface{}, error) {
	for _, field := range customFields {
		dataType, _ := field["data_type"].(string)
		options, hasOptions := field["options"]
		sampleData, hasSampleData := field["sample_data"].(string)
		fallbackValue, hasFallbackValue := field["fallback_value"].(string)

		// Validate SELECT type requires options
		if dataType == string(domain.CustomFieldDataTypeSelect) {
			if !hasOptions || options == nil {
				return nil, fmt.Errorf("options is required for select data type")
			}
		} else {
			// Options only allowed for SELECT type
			if hasOptions && options != nil {
				return nil, fmt.Errorf("options is only allowed for select data type")
			}
		}

		// Validate sample data if present
		if hasSampleData {
			if _, err := s.ValidateDataType(field, sampleData); err != nil {
				return nil, err
			}
		}

		// Validate fallback value if present
		if hasFallbackValue {
			if _, err := s.ValidateDataType(field, fallbackValue); err != nil {
				return nil, err
			}
		}
	}

	return customFields, nil
}

// ValidateCustomField validates custom field entity and applies fallback values
func (s *CustomFieldService) ValidateCustomField(
	customFields []map[string]interface{},
	customFieldEntity map[string]interface{},
) (map[string]interface{}, error) {
	// Validate custom fields
	for _, field := range customFields {
		isRequired, _ := field["is_required"].(bool)
		normalizedName, _ := field["normalized_name"].(string)
		fieldName, _ := field["name"].(string)
		fallbackValue, hasFallback := field["fallback_value"].(string)

		entityValue, hasValue := customFieldEntity[normalizedName]

		// Check required fields
		if isRequired && !hasValue {
			return nil, fmt.Errorf("%s is a required custom field. Please fill in the value", fieldName)
		}

		// Validate value if present
		if hasValue {
			valueStr, ok := entityValue.(string)
			if !ok {
				valueStr = fmt.Sprintf("%v", entityValue)
			}

			validatedValue, err := s.ValidateDataType(field, valueStr)
			if err != nil {
				return nil, err
			}
			customFieldEntity[normalizedName] = validatedValue
		} else if !hasValue && hasFallback {
			// Apply fallback value
			customFieldEntity[normalizedName] = fallbackValue
		}
	}

	// Validate system fields
	for key, field := range SystemFields {
		value, hasValue := customFieldEntity[key]

		if field.IsRequired && !hasValue {
			return nil, fmt.Errorf("%s is a required field. Please fill in the value", key)
		}

		if hasValue {
			valueStr, ok := value.(string)
			if !ok {
				valueStr = fmt.Sprintf("%v", value)
			}

			if key == "phone" && valueStr != "" {
				if !phoneRegex.MatchString(valueStr) {
					return nil, fmt.Errorf("%s is not a valid phone number", valueStr)
				}
			} else if key == "email" && valueStr != "" {
				if !isValidEmail(valueStr) {
					return nil, fmt.Errorf("%s is not a valid email", valueStr)
				}
			}
		}
	}

	return customFieldEntity, nil
}

// Helper functions

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func isValidURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func ParseDateTime(value string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04:05",
		"02/01/2006",
		"01/02/2006",
		"2006-01-02T15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date/time: %s", value)
}
