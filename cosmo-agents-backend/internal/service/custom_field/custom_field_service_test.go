package custom_field

import (
	"strings"
	"testing"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomFieldService_ValidateDataType(t *testing.T) {
	service := NewCustomFieldService()

	tests := []struct {
		name      string
		field     map[string]interface{}
		value     string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "valid email",
			field: map[string]interface{}{
				"name":      "Email Field",
				"data_type": string(domain.CustomFieldDataTypeEmail),
			},
			value:   "test@example.com",
			wantErr: false,
		},
		{
			name: "invalid email",
			field: map[string]interface{}{
				"name":      "Email Field",
				"data_type": string(domain.CustomFieldDataTypeEmail),
			},
			value:     "invalid-email",
			wantErr:   true,
			errSubstr: "not a valid email",
		},
		{
			name: "valid number string",
			field: map[string]interface{}{
				"name":      "Age",
				"data_type": string(domain.CustomFieldDataTypeNumber),
			},
			value:   "25",
			wantErr: false,
		},
		{
			name: "invalid number",
			field: map[string]interface{}{
				"name":      "Age",
				"data_type": string(domain.CustomFieldDataTypeNumber),
			},
			value:     "not-a-number",
			wantErr:   true,
			errSubstr: "not a valid number",
		},
		{
			name: "valid URL",
			field: map[string]interface{}{
				"name":      "Website",
				"data_type": string(domain.CustomFieldDataTypeURL),
			},
			value:   "https://example.com",
			wantErr: false,
		},
		{
			name: "valid text within limit",
			field: map[string]interface{}{
				"name":      "Description",
				"data_type": string(domain.CustomFieldDataTypeText),
			},
			value:   "This is a valid description",
			wantErr: false,
		},
		{
			name: "text too long",
			field: map[string]interface{}{
				"name":      "Description",
				"data_type": string(domain.CustomFieldDataTypeText),
			},
			value:     string(make([]byte, 256)), // 256 characters
			wantErr:   true,
			errSubstr: "exceeds maximum length",
		},
		{
			name: "valid select option",
			field: map[string]interface{}{
				"name":      "Status",
				"data_type": string(domain.CustomFieldDataTypeSelect),
				"options":   []interface{}{"active", "inactive", "pending"},
			},
			value:   "active",
			wantErr: false,
		},
		{
			name: "invalid select option",
			field: map[string]interface{}{
				"name":      "Status",
				"data_type": string(domain.CustomFieldDataTypeSelect),
				"options":   []interface{}{"active", "inactive"},
			},
			value:     "invalid-status",
			wantErr:   true,
			errSubstr: "not in options",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ValidateDataType(tt.field, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.value, result)
			}
		})
	}
}

func TestCustomFieldService_ValidateCustomFieldsModel(t *testing.T) {
	service := NewCustomFieldService()

	tests := []struct {
		name         string
		customFields []map[string]interface{}
		wantErr      bool
		errSubstr    string
	}{
		{
			name: "valid select field with options",
			customFields: []map[string]interface{}{
				{
					"name":      "Status",
					"data_type": string(domain.CustomFieldDataTypeSelect),
					"options":   []interface{}{"active", "inactive"},
				},
			},
			wantErr: false,
		},
		{
			name: "select field without options",
			customFields: []map[string]interface{}{
				{
					"name":      "Status",
					"data_type": string(domain.CustomFieldDataTypeSelect),
				},
			},
			wantErr:   true,
			errSubstr: "options is required",
		},
		{
			name: "non-select field with options",
			customFields: []map[string]interface{}{
				{
					"name":      "Email",
					"data_type": string(domain.CustomFieldDataTypeEmail),
					"options":   []interface{}{"test@example.com"},
				},
			},
			wantErr:   true,
			errSubstr: "only allowed for select",
		},
		{
			name: "field with valid sample data",
			customFields: []map[string]interface{}{
				{
					"name":        "Email",
					"data_type":   string(domain.CustomFieldDataTypeEmail),
					"sample_data": "test@example.com",
				},
			},
			wantErr: false,
		},
		{
			name: "field with invalid sample data",
			customFields: []map[string]interface{}{
				{
					"name":        "Email",
					"data_type":   string(domain.CustomFieldDataTypeEmail),
					"sample_data": "invalid-email",
				},
			},
			wantErr:   true,
			errSubstr: "not a valid email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ValidateCustomFieldsModel(tt.customFields)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.customFields, result)
			}
		})
	}
}

func TestCustomFieldService_ValidateCustomField(t *testing.T) {
	service := NewCustomFieldService()

	tests := []struct {
		name              string
		customFields      []map[string]interface{}
		customFieldEntity map[string]interface{}
		wantErr           bool
		errSubstr         string
		expectedEntity    map[string]interface{}
	}{
		{
			name: "valid required field present",
			customFields: []map[string]interface{}{
				{
					"name":            "Email",
					"normalized_name": "email",
					"data_type":       string(domain.CustomFieldDataTypeEmail),
					"is_required":     true,
				},
			},
			customFieldEntity: map[string]interface{}{
				"email": "test@example.com",
			},
			wantErr: false,
			expectedEntity: map[string]interface{}{
				"email": "test@example.com",
			},
		},
		{
			name: "required field missing",
			customFields: []map[string]interface{}{
				{
					"name":            "Email",
					"normalized_name": "email",
					"data_type":       string(domain.CustomFieldDataTypeEmail),
					"is_required":     true,
				},
			},
			customFieldEntity: map[string]interface{}{},
			wantErr:           true,
			errSubstr:         "is a required custom field",
		},
		{
			name: "fallback value applied",
			customFields: []map[string]interface{}{
				{
					"name":            "Country",
					"normalized_name": "country",
					"data_type":       string(domain.CustomFieldDataTypeText),
					"is_required":     false,
					"fallback_value":  "USA",
				},
			},
			customFieldEntity: map[string]interface{}{
				"email": "test@example.com", // System field required
			},
			wantErr: false,
			expectedEntity: map[string]interface{}{
				"email":   "test@example.com",
				"country": "USA",
			},
		},
		{
			name:         "system field email required",
			customFields: []map[string]interface{}{},
			customFieldEntity: map[string]interface{}{
				"phone": "1234567890",
			},
			wantErr:   true,
			errSubstr: "email is a required field",
		},
		{
			name:         "system field email valid",
			customFields: []map[string]interface{}{},
			customFieldEntity: map[string]interface{}{
				"email": "test@example.com",
			},
			wantErr: false,
			expectedEntity: map[string]interface{}{
				"email": "test@example.com",
			},
		},
		{
			name:         "system field email invalid",
			customFields: []map[string]interface{}{},
			customFieldEntity: map[string]interface{}{
				"email": "invalid-email",
			},
			wantErr:   true,
			errSubstr: "not a valid email",
		},
		{
			name:         "system field phone valid",
			customFields: []map[string]interface{}{},
			customFieldEntity: map[string]interface{}{
				"email": "test@example.com",
				"phone": "+1-234-567-8900",
			},
			wantErr: false,
			expectedEntity: map[string]interface{}{
				"email": "test@example.com",
				"phone": "+1-234-567-8900",
			},
		},
		{
			name:         "system field phone invalid",
			customFields: []map[string]interface{}{},
			customFieldEntity: map[string]interface{}{
				"email": "test@example.com",
				"phone": "invalid-phone",
			},
			wantErr:   true,
			errSubstr: "not a valid phone number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ValidateCustomField(tt.customFields, tt.customFieldEntity)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)
				if tt.expectedEntity != nil {
					assert.Equal(t, tt.expectedEntity, result)
				}
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("isValidEmail", func(t *testing.T) {
		tests := []struct {
			email string
			valid bool
		}{
			{"test@example.com", true},
			{"user.name@example.co.uk", true},
			{"invalid-email", false},
			{"@example.com", false},
			{"test@", false},
		}

		for _, tt := range tests {
			result := isValidEmail(tt.email)
			assert.Equal(t, tt.valid, result, "email: %s", tt.email)
		}
	})

	t.Run("isValidURL", func(t *testing.T) {
		tests := []struct {
			url   string
			valid bool
		}{
			{"https://example.com", true},
			{"http://example.com/path", true},
			{"ftp://files.example.com", true},
			{"not-a-url", false},
			{"//example.com", false},
		}

		for _, tt := range tests {
			result := isValidURL(tt.url)
			assert.Equal(t, tt.valid, result, "url: %s", tt.url)
		}
	})

	t.Run("parseDateTime", func(t *testing.T) {
		tests := []struct {
			dateStr string
			valid   bool
		}{
			{"2024-01-15", true},
			{"2024-01-15T10:30:00Z", true},
			{"01/15/2024", true},
			{"invalid-date", false},
		}

		for _, tt := range tests {
			_, err := ParseDateTime(tt.dateStr)
			if tt.valid {
				assert.NoError(t, err, "date: %s", tt.dateStr)
			} else {
				assert.Error(t, err, "date: %s", tt.dateStr)
			}
		}
	})
}

func TestCustomFieldService_ValidateDataType_Edges(t *testing.T) {
	service := NewCustomFieldService()
	number := map[string]interface{}{"name": "Budget", "data_type": string(domain.CustomFieldDataTypeNumber)}
	text := map[string]interface{}{"name": "Note", "data_type": string(domain.CustomFieldDataTypeText)}

	for _, v := range []string{"NaN", "Inf", "-Infinity"} {
		_, err := service.ValidateDataType(number, v)
		assert.Error(t, err, v)
	}
	_, err := service.ValidateDataType(number, "-12.5")
	assert.NoError(t, err)

	// 255 characters of Vietnamese text is ~500 bytes but within the limit.
	vi := strings.Repeat("ạ", 255)
	_, err = service.ValidateDataType(text, vi)
	assert.NoError(t, err)
	_, err = service.ValidateDataType(text, vi+"ạ")
	assert.Error(t, err)
}
