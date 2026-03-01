package field_validation

import (
	"testing"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldValidationService_ValidateField(t *testing.T) {
	service := NewFieldValidationService()

	tests := []struct {
		name      string
		value     interface{}
		fieldDef  map[string]interface{}
		wantErr   bool
		errSubstr string
	}{
		{
			name:  "valid email",
			value: "test@example.com",
			fieldDef: map[string]interface{}{
				"name":      "Email",
				"data_type": string(domain.CustomFieldDataTypeEmail),
			},
			wantErr: false,
		},
		{
			name:  "invalid email",
			value: "not-an-email",
			fieldDef: map[string]interface{}{
				"name":      "Email",
				"data_type": string(domain.CustomFieldDataTypeEmail),
			},
			wantErr:   true,
			errSubstr: "Invalid email format",
		},
		{
			name:  "valid number int",
			value: 42,
			fieldDef: map[string]interface{}{
				"name":      "Age",
				"data_type": string(domain.CustomFieldDataTypeNumber),
			},
			wantErr: false,
		},
		{
			name:  "valid number string",
			value: "42.5",
			fieldDef: map[string]interface{}{
				"name":      "Price",
				"data_type": string(domain.CustomFieldDataTypeNumber),
			},
			wantErr: false,
		},
		{
			name:  "invalid number",
			value: "not-a-number",
			fieldDef: map[string]interface{}{
				"name":      "Age",
				"data_type": string(domain.CustomFieldDataTypeNumber),
			},
			wantErr:   true,
			errSubstr: "Not a valid number",
		},
		{
			name:  "valid URL",
			value: "https://example.com",
			fieldDef: map[string]interface{}{
				"name":      "Website",
				"data_type": string(domain.CustomFieldDataTypeURL),
			},
			wantErr: false,
		},
		{
			name:  "valid text",
			value: "Short text",
			fieldDef: map[string]interface{}{
				"name":      "Description",
				"data_type": string(domain.CustomFieldDataTypeText),
			},
			wantErr: false,
		},
		{
			name:  "text too long",
			value: string(make([]byte, 300)),
			fieldDef: map[string]interface{}{
				"name":      "Description",
				"data_type": string(domain.CustomFieldDataTypeText),
			},
			wantErr:   true,
			errSubstr: "Text too long",
		},
		{
			name:  "valid select",
			value: "active",
			fieldDef: map[string]interface{}{
				"name":      "Status",
				"data_type": string(domain.CustomFieldDataTypeSelect),
				"options":   []interface{}{"active", "inactive", "pending"},
			},
			wantErr: false,
		},
		{
			name:  "invalid select",
			value: "unknown",
			fieldDef: map[string]interface{}{
				"name":      "Status",
				"data_type": string(domain.CustomFieldDataTypeSelect),
				"options":   []interface{}{"active", "inactive"},
			},
			wantErr:   true,
			errSubstr: "Value not in options",
		},
		{
			name:  "valid phone",
			value: "+1-234-567-8900",
			fieldDef: map[string]interface{}{
				"name":      "Phone",
				"data_type": "phone_number",
			},
			wantErr: false,
		},
		{
			name:  "empty value skips validation",
			value: "",
			fieldDef: map[string]interface{}{
				"name":      "Email",
				"data_type": string(domain.CustomFieldDataTypeEmail),
			},
			wantErr: false,
		},
		{
			name:  "nil value skips validation",
			value: nil,
			fieldDef: map[string]interface{}{
				"name":      "Email",
				"data_type": string(domain.CustomFieldDataTypeEmail),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ValidateField(tt.value, tt.fieldDef)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				assert.NoError(t, err)
				// Result can be nil for nil input
			}
		})
	}
}

func TestFieldValidationService_ValidateForm(t *testing.T) {
	service := NewFieldValidationService()

	tests := []struct {
		name        string
		fields      []map[string]interface{}
		formData    map[string]interface{}
		expectValid bool
		expectData  map[string]interface{}
		errorFields []string
	}{
		{
			name: "all required fields present and valid",
			fields: []map[string]interface{}{
				{
					"name":            "Email",
					"normalized_name": "email",
					"data_type":       string(domain.CustomFieldDataTypeEmail),
					"is_required":     true,
				},
				{
					"name":            "Age",
					"normalized_name": "age",
					"data_type":       string(domain.CustomFieldDataTypeNumber),
					"is_required":     false,
				},
			},
			formData: map[string]interface{}{
				"email": "test@example.com",
				"age":   "25",
			},
			expectValid: true,
			expectData: map[string]interface{}{
				"email": "test@example.com",
				"age":   "25",
			},
		},
		{
			name: "required field missing",
			fields: []map[string]interface{}{
				{
					"name":            "Email",
					"normalized_name": "email",
					"data_type":       string(domain.CustomFieldDataTypeEmail),
					"is_required":     true,
				},
			},
			formData:    map[string]interface{}{},
			expectValid: false,
			errorFields: []string{"email"},
		},
		{
			name: "invalid field value",
			fields: []map[string]interface{}{
				{
					"name":            "Email",
					"normalized_name": "email",
					"data_type":       string(domain.CustomFieldDataTypeEmail),
					"is_required":     true,
				},
			},
			formData: map[string]interface{}{
				"email": "invalid-email",
			},
			expectValid: false,
			errorFields: []string{"email"},
		},
		{
			name: "fallback value applied",
			fields: []map[string]interface{}{
				{
					"name":            "Country",
					"normalized_name": "country",
					"data_type":       string(domain.CustomFieldDataTypeText),
					"is_required":     false,
					"fallback_value":  "USA",
				},
			},
			formData:    map[string]interface{}{},
			expectValid: true,
			expectData: map[string]interface{}{
				"country": "USA",
			},
		},
		{
			name: "multiple validation errors",
			fields: []map[string]interface{}{
				{
					"name":            "Email",
					"normalized_name": "email",
					"data_type":       string(domain.CustomFieldDataTypeEmail),
					"is_required":     true,
				},
				{
					"name":            "Age",
					"normalized_name": "age",
					"data_type":       string(domain.CustomFieldDataTypeNumber),
					"is_required":     true,
				},
			},
			formData:    map[string]interface{}{},
			expectValid: false,
			errorFields: []string{"email", "age"},
		},
		{
			name: "optional fields with validation",
			fields: []map[string]interface{}{
				{
					"name":            "Email",
					"normalized_name": "email",
					"data_type":       string(domain.CustomFieldDataTypeEmail),
					"is_required":     false,
				},
			},
			formData: map[string]interface{}{
				"email": "test@example.com",
			},
			expectValid: true,
			expectData: map[string]interface{}{
				"email": "test@example.com",
			},
		},
		{
			name: "empty optional field skips validation",
			fields: []map[string]interface{}{
				{
					"name":            "Email",
					"normalized_name": "email",
					"data_type":       string(domain.CustomFieldDataTypeEmail),
					"is_required":     false,
				},
			},
			formData:    map[string]interface{}{},
			expectValid: true,
			expectData:  map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.ValidateForm(tt.fields, tt.formData)

			assert.Equal(t, tt.expectValid, result.Valid)

			if tt.expectValid {
				if tt.expectData != nil {
					for key, expectedValue := range tt.expectData {
						assert.Equal(t, expectedValue, result.Data[key], "mismatch for key: %s", key)
					}
				}
			} else {
				assert.Greater(t, len(result.Errors), 0, "should have validation errors")
				for _, field := range tt.errorFields {
					assert.Contains(t, result.Errors, field, "should have error for field: %s", field)
				}
			}
		})
	}
}

func TestFieldValidationService_RegisterValidator(t *testing.T) {
	service := NewFieldValidationService()

	// Register custom validator
	customValidator := func(value interface{}, fieldDef map[string]interface{}) *string {
		str, ok := value.(string)
		if !ok || str != "custom" {
			msg := "Must be 'custom'"
			return &msg
		}
		return nil
	}

	service.RegisterValidator("custom_type", customValidator)

	// Test custom validator
	fieldDef := map[string]interface{}{
		"name":      "Custom Field",
		"data_type": "custom_type",
	}

	// Valid value
	_, err := service.ValidateField("custom", fieldDef)
	assert.NoError(t, err)

	// Invalid value
	_, err = service.ValidateField("invalid", fieldDef)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Must be 'custom'")
}

func TestFieldValidationService_isEmpty(t *testing.T) {
	service := NewFieldValidationService()

	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"nil value", nil, true},
		{"empty string", "", true},
		{"whitespace string", "   ", false}, // Note: trimming not implemented
		{"non-empty string", "test", false},
		{"zero number", 0, false},
		{"positive number", 42, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isEmpty(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		FieldName: "email",
		Message:   "Invalid format",
	}

	expectedMsg := "email: Invalid format"
	assert.Equal(t, expectedMsg, err.Error())
}

func TestValidationResult(t *testing.T) {
	t.Run("successful validation", func(t *testing.T) {
		result := ValidationResult{
			Valid:  true,
			Errors: map[string]string{},
			Data: map[string]interface{}{
				"email": "test@example.com",
			},
		}

		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
		require.NotNil(t, result.Data)
		assert.Equal(t, "test@example.com", result.Data["email"])
	})

	t.Run("failed validation", func(t *testing.T) {
		result := ValidationResult{
			Valid: false,
			Errors: map[string]string{
				"email": "Invalid email format",
				"age":   "Not a valid number",
			},
			Data: map[string]interface{}{},
		}

		assert.False(t, result.Valid)
		assert.Len(t, result.Errors, 2)
		assert.Contains(t, result.Errors, "email")
		assert.Contains(t, result.Errors, "age")
	})
}

func TestFieldValidationService_AdditionalBranches(t *testing.T) {
	service := NewFieldValidationService()

	_, err := service.ValidateField(123, map[string]interface{}{
		"name":      "URL",
		"data_type": string(domain.CustomFieldDataTypeURL),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Must be a string")

	_, err = service.ValidateField("not-a-date", map[string]interface{}{
		"name":      "Date",
		"data_type": string(domain.CustomFieldDataTypeDate),
	})
	assert.Error(t, err)

	_, err = service.ValidateField("choice", map[string]interface{}{
		"name":      "Select",
		"data_type": string(domain.CustomFieldDataTypeSelect),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "No options defined")

	// Unknown validator should pass through without error
	val, err := service.ValidateField(123, map[string]interface{}{
		"name":      "Unknown",
		"data_type": "unregistered",
	})
	assert.NoError(t, err)
	assert.Equal(t, 123, val)

	_, err = service.ValidateField("bad-phone", map[string]interface{}{
		"name":      "Phone",
		"data_type": "phone_number",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid phone number format")
}
