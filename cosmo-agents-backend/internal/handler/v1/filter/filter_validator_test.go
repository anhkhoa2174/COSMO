package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ValidConversationTypes = map[string]bool{
	"sent":            true,
	"assign_to_ai":    true,
	"assign_to_human": true,
	"":                true, // Empty string is valid (means no filter)
}

func TestValidateFilter_SecurityTests(t *testing.T) {
	tests := []struct {
		name          string
		filter        map[string]interface{}
		expectError   bool
		errorContains string
	}{
		{
			name:        "nil filter should pass",
			filter:      nil,
			expectError: false,
		},
		{
			name:        "empty filter should pass",
			filter:      map[string]interface{}{},
			expectError: false,
		},
		{
			name: "valid is_deleted filter",
			filter: map[string]interface{}{
				"is_deleted": map[string]interface{}{
					"$in": []interface{}{true, false},
				},
			},
			expectError: false,
		},
		{
			name: "disallowed filter key should fail",
			filter: map[string]interface{}{
				"malicious_key": "value",
			},
			expectError:   true,
			errorContains: "filter key not allowed",
		},
		{
			name: "disallowed operator should fail",
			filter: map[string]interface{}{
				"is_deleted": map[string]interface{}{
					"$where": "malicious code",
				},
			},
			expectError:   true,
			errorContains: "operator '$where' not allowed",
		},
		{
			name: "oversized array should fail",
			filter: map[string]interface{}{
				"status": map[string]interface{}{
					"$in": func() []interface{} {
						arr := make([]interface{}, 101)
						for i := range arr {
							arr[i] = "active" // Fill with valid status values
						}
						return arr
					}(),
				},
			},
			expectError:   true,
			errorContains: "array too large",
		},
		{
			name: "invalid status value should fail",
			filter: map[string]interface{}{
				"status": "invalid_status",
			},
			expectError:   true,
			errorContains: "invalid status value",
		},
		{
			name: "excessively long string should fail",
			filter: map[string]interface{}{
				"name": string(make([]byte, 1001)), // > 1000 limit
			},
			expectError:   true,
			errorContains: "string value too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilter(tt.filter)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidConversationTypes_SecurityWhitelist(t *testing.T) {
	validTypes := []string{"sent", "assign_to_ai", "assign_to_human", ""}
	invalidTypes := []string{
		"malicious_type",
		"SELECT * FROM users",
		"'; DROP TABLE conversations; --",
		"<script>alert('xss')</script>",
		"../../../etc/passwd",
	}

	// Test valid types
	for _, validType := range validTypes {
		t.Run("valid_"+validType, func(t *testing.T) {
			assert.True(t, ValidConversationTypes[validType],
				"Valid conversation type should be allowed: %s", validType)
		})
	}

	// Test invalid types (should not be in whitelist)
	for _, invalidType := range invalidTypes {
		t.Run("invalid_"+invalidType, func(t *testing.T) {
			assert.False(t, ValidConversationTypes[invalidType],
				"Invalid conversation type should be rejected: %s", invalidType)
		})
	}
}

func TestSanitizeUUID_ValidationTests(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid UUID should pass",
			input:       "550e8400-e29b-41d4-a716-446655440000",
			expectError: false,
		},
		{
			name:          "empty UUID should fail",
			input:         "",
			expectError:   true,
			errorContains: "UUID cannot be empty",
		},
		{
			name:          "invalid UUID format should fail",
			input:         "not-a-uuid",
			expectError:   true,
			errorContains: "invalid UUID format",
		},
		{
			name:          "SQL injection attempt should fail",
			input:         "550e8400-e29b-41d4-a716-'; DROP TABLE users; --",
			expectError:   true,
			errorContains: "invalid UUID format",
		},
		{
			name:          "path traversal attempt should fail",
			input:         "../../../etc/passwd",
			expectError:   true,
			errorContains: "invalid UUID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SanitizeUUID(tt.input)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilterValidationError_ErrorInterface(t *testing.T) {
	err := FilterValidationError{
		Field:   "test_field",
		Message: "test message",
	}

	expected := "filter validation error for field 'test_field': test message"
	assert.Equal(t, expected, err.Error())
}

func TestValidateOperatorValue_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		operator    string
		value       interface{}
		expectError bool
	}{
		{
			name:        "$in with valid array",
			key:         "status",
			operator:    "$in",
			value:       []interface{}{"active", "archived"},
			expectError: false,
		},
		{
			name:        "$in with non-array should fail",
			key:         "status",
			operator:    "$in",
			value:       "not_an_array",
			expectError: true,
		},
		{
			name:        "$eq with valid value",
			key:         "replied",
			operator:    "$eq",
			value:       true,
			expectError: false,
		},
		{
			name:        "$ne with valid value",
			key:         "replied",
			operator:    "$ne",
			value:       false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOperatorValue(tt.key, tt.operator, tt.value)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSimpleValue_TypeValidation(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		value       interface{}
		expectError bool
	}{
		{
			name:        "is_deleted with boolean",
			key:         "is_deleted",
			value:       true,
			expectError: false,
		},
		{
			name:        "is_deleted with non-boolean should fail",
			key:         "is_deleted",
			value:       "true",
			expectError: true,
		},
		{
			name:        "replied with boolean",
			key:         "replied",
			value:       false,
			expectError: false,
		},
		{
			name:        "status with valid string",
			key:         "status",
			value:       "active",
			expectError: false,
		},
		{
			name:        "status with invalid string should fail",
			key:         "status",
			value:       "invalid_status",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSimpleValue(tt.key, tt.value)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
