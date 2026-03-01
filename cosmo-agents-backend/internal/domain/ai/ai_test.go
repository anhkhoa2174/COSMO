package ai

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompanyInfoSerialization(t *testing.T) {
	tests := []struct {
		name         string
		companyInfo  CompanyInfo
		expectedJSON string
	}{
		{
			name: "Complete company info",
			companyInfo: CompanyInfo{
				CompanyDescription: "Digital document signing platform",
				CompanyTargetingPersona: []string{
					"Legal Operations Manager",
					"Contract Manager",
				},
				ValueOffering: "Reduce contract signing time from days to minutes",
			},
			expectedJSON: `{
				"company_description": "Digital document signing platform",
				"company_targeting_persona": [
					"Legal Operations Manager",
					"Contract Manager"
				],
				"value_offering": "Reduce contract signing time from days to minutes"
			}`,
		},
		{
			name: "Empty company info",
			companyInfo: CompanyInfo{
				CompanyDescription:      "",
				CompanyTargetingPersona: []string{},
				ValueOffering:           "",
			},
			expectedJSON: `{
				"company_description": "",
				"company_targeting_persona": [],
				"value_offering": ""
			}`,
		},
		{
			name: "Partial company info",
			companyInfo: CompanyInfo{
				CompanyDescription: "Customer support platform",
				CompanyTargetingPersona: []string{
					"Customer Service Manager",
					"Support Team Lead",
				},
				ValueOffering: "",
			},
			expectedJSON: `{
				"company_description": "Customer support platform",
				"company_targeting_persona": [
					"Customer Service Manager",
					"Support Team Lead"
				],
				"value_offering": ""
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			jsonBytes, err := json.Marshal(tt.companyInfo)
			require.NoError(t, err)

			// Parse expected JSON to normalize formatting
			var expectedJSONNormalized map[string]interface{}
			err = json.Unmarshal([]byte(tt.expectedJSON), &expectedJSONNormalized)
			require.NoError(t, err)

			// Parse actual JSON
			var actualJSON map[string]interface{}
			err = json.Unmarshal(jsonBytes, &actualJSON)
			require.NoError(t, err)

			// Compare JSON objects
			assert.Equal(t, expectedJSONNormalized, actualJSON)

			// Test JSON unmarshaling
			var unmarshaled CompanyInfo
			err = json.Unmarshal(jsonBytes, &unmarshaled)
			require.NoError(t, err)

			// Verify unmarshaled data matches original
			assert.Equal(t, tt.companyInfo, unmarshaled)
		})
	}
}

func TestCompanyInfoUnmarshalingWithInvalidData(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
	}{
		{
			name:        "Invalid JSON",
			jsonData:    `{"company_description": "test",}`,
			expectError: true,
		},
		{
			name:        "Empty JSON object",
			jsonData:    `{}`,
			expectError: false, // Should not error - all fields are optional
		},
		{
			name:        "Null values",
			jsonData:    `{"company_description": null}`,
			expectError: false, // JSON null becomes empty string for string fields
		},
		{
			name:        "Invalid persona array",
			jsonData:    `{"company_targeting_persona": "not an array"}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var companyInfo CompanyInfo
			err := json.Unmarshal([]byte(tt.jsonData), &companyInfo)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCompanyInfoWithSpecialCharacters(t *testing.T) {
	companyInfo := CompanyInfo{
		CompanyDescription: "AI-powered platform with \"smart\" features & integrations",
		CompanyTargetingPersona: []string{
			"CEO/Founder",
			"C-Suite Executive",
			"VP of Operations",
		},
		ValueOffering: "Save 20+ hours/week with 99.9% accuracy",
	}

	// Test that special characters are properly handled
	jsonBytes, err := json.Marshal(companyInfo)
	require.NoError(t, err)

	var unmarshaled CompanyInfo
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, companyInfo, unmarshaled)
}

func TestCompanyInfoPersonaHandling(t *testing.T) {
	tests := []struct {
		name     string
		personas []string
	}{
		{
			name:     "Single persona",
			personas: []string{"Project Manager"},
		},
		{
			name:     "Multiple personas",
			personas: []string{"Project Manager", "Team Lead", "Developer"},
		},
		{
			name:     "Empty personas",
			personas: []string{},
		},
		{
			name:     "Personas with empty strings",
			personas: []string{"Manager", "", "Developer", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			companyInfo := CompanyInfo{
				CompanyDescription:      "Test company",
				CompanyTargetingPersona: tt.personas,
				ValueOffering:           "Test value",
			}

			jsonBytes, err := json.Marshal(companyInfo)
			require.NoError(t, err)

			var unmarshaled CompanyInfo
			err = json.Unmarshal(jsonBytes, &unmarshaled)
			require.NoError(t, err)

			// Verify personas are preserved exactly as-is (including empty strings if present)
			assert.Equal(t, len(tt.personas), len(unmarshaled.CompanyTargetingPersona))
			for i, persona := range tt.personas {
				if i < len(unmarshaled.CompanyTargetingPersona) {
					assert.Equal(t, persona, unmarshaled.CompanyTargetingPersona[i])
				}
			}
		})
	}
}
