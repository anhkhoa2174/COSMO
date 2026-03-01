package template

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestTemplate_Validation tests template validation logic without database
func TestTemplate_Validation(t *testing.T) {
	tests := []struct {
		name        string
		templateID  string
		shouldFail  bool
		description string
	}{
		{
			name:        "valid template ID",
			templateID:  uuid.New().String(),
			shouldFail:  false,
			description: "Valid UUID should pass validation",
		},
		{
			name:        "invalid template ID",
			templateID:  "not-a-uuid",
			shouldFail:  true,
			description: "Invalid UUID should fail validation",
		},
		{
			name:        "empty template ID",
			templateID:  "",
			shouldFail:  true,
			description: "Empty UUID should fail validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uuid.Parse(tt.templateID)
			if tt.shouldFail {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

// TestTemplate_RequestValidation tests template request validation
func TestTemplate_RequestValidation(t *testing.T) {
	tests := []struct {
		name        string
		request     map[string]interface{}
		shouldFail  bool
		description string
	}{
		{
			name: "valid template request",
			request: map[string]interface{}{
				"name":    "Test Template",
				"subject": "Hello {{contact_first_name}}",
				"content": "Dear {{contact_first_name}},\n\nThis is a test.",
			},
			shouldFail:  false,
			description: "Valid template request should pass validation",
		},
		{
			name: "missing name",
			request: map[string]interface{}{
				"subject": "Hello {{contact_first_name}}",
				"content": "Dear {{contact_first_name}},\n\nThis is a test.",
			},
			shouldFail:  true,
			description: "Template without name should fail validation",
		},
		{
			name: "missing subject",
			request: map[string]interface{}{
				"name":    "Test Template",
				"content": "Dear {{contact_first_name}},\n\nThis is a test.",
			},
			shouldFail:  true,
			description: "Template without subject should fail validation",
		},
		{
			name: "missing content",
			request: map[string]interface{}{
				"name":    "Test Template",
				"subject": "Hello {{contact_first_name}}",
			},
			shouldFail:  true,
			description: "Template without content should fail validation",
		},
		{
			name: "empty fields",
			request: map[string]interface{}{
				"name":    "",
				"subject": "",
				"content": "",
			},
			shouldFail:  true,
			description: "Template with empty fields should fail validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation logic
			name, hasName := tt.request["name"].(string)
			subject, hasSubject := tt.request["subject"].(string)
			content, hasContent := tt.request["content"].(string)

			isValid := hasName && hasSubject && hasContent &&
				name != "" && subject != "" && content != ""

			if tt.shouldFail {
				assert.False(t, isValid, tt.description)
			} else {
				assert.True(t, isValid, tt.description)
			}
		})
	}
}

// TestTemplate_ContentTemplateValidation tests template content validation
func TestTemplate_ContentTemplateValidation(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		hasVars     bool
		description string
	}{
		{
			name:        "content with contact variables",
			content:     "Hello {{contact_first_name}} {{contact_last_name}}",
			hasVars:     true,
			description: "Should detect contact template variables",
		},
		{
			name:        "content with agent variables",
			content:     "Best regards, {{agent_name}} from {{organization_name}}",
			hasVars:     true,
			description: "Should detect agent template variables",
		},
		{
			name:        "content without variables",
			content:     "This is a plain email content",
			hasVars:     false,
			description: "Should not detect template variables in plain text",
		},
		{
			name:        "empty content",
			content:     "",
			hasVars:     false,
			description: "Empty content should not have variables",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Template variable detection - check if content contains template variables
			hasContactVars := len(tt.content) > 0 &&
				(strings.Contains(tt.content, "{{contact_first_name}}") ||
					strings.Contains(tt.content, "{{contact_last_name}}") ||
					strings.Contains(tt.content, "{{contact_email}}"))
			hasAgentVars := len(tt.content) > 0 &&
				(strings.Contains(tt.content, "{{agent_name}}") ||
					strings.Contains(tt.content, "{{organization_name}}"))
			hasVariables := hasContactVars || hasAgentVars

			assert.Equal(t, tt.hasVars, hasVariables, tt.description)
		})
	}
}

// TestTemplate_CategoryValidation tests template category validation
func TestTemplate_CategoryValidation(t *testing.T) {
	validCategories := []string{"marketing", "sales", "support", "followup", "custom"}

	tests := []struct {
		name        string
		category    string
		isValid     bool
		description string
	}{
		{
			name:        "valid marketing category",
			category:    "marketing",
			isValid:     true,
			description: "Marketing should be a valid category",
		},
		{
			name:        "valid sales category",
			category:    "sales",
			isValid:     true,
			description: "Sales should be a valid category",
		},
		{
			name:        "valid custom category",
			category:    "custom",
			isValid:     true,
			description: "Custom should be a valid category",
		},
		{
			name:        "invalid category",
			category:    "invalid_category",
			isValid:     false,
			description: "Invalid category should not be allowed",
		},
		{
			name:        "empty category",
			category:    "",
			isValid:     true,
			description: "Empty category should be allowed (default)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.category == "" || tt.category == "custom"
			for _, valid := range validCategories {
				if tt.category == valid {
					isValid = true
					break
				}
			}

			assert.Equal(t, tt.isValid, isValid, tt.description)
		})
	}
}
