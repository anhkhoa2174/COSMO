package custom_field

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// TestCustomFieldConstants tests all constants and their values
func TestCustomFieldConstants(t *testing.T) {
	t.Run("CustomFieldEntity constants", func(t *testing.T) {
		assert.Equal(t, CustomFieldEntity("contact"), CustomFieldEntityContact)
		assert.Equal(t, CustomFieldEntity("company"), CustomFieldEntityCompany)

		// Test string conversion
		assert.Equal(t, "contact", string(CustomFieldEntityContact))
		assert.Equal(t, "company", string(CustomFieldEntityCompany))
	})

	t.Run("CustomFieldDataType constants", func(t *testing.T) {
		assert.Equal(t, CustomFieldDataType("text"), CustomFieldDataTypeText)
		assert.Equal(t, CustomFieldDataType("number"), CustomFieldDataTypeNumber)
		assert.Equal(t, CustomFieldDataType("email"), CustomFieldDataTypeEmail)
		assert.Equal(t, CustomFieldDataType("select"), CustomFieldDataTypeSelect)
		assert.Equal(t, CustomFieldDataType("date"), CustomFieldDataTypeDate)
		assert.Equal(t, CustomFieldDataType("url"), CustomFieldDataTypeURL)

		// Test string conversion
		dataTypes := []CustomFieldDataType{
			CustomFieldDataTypeText,
			CustomFieldDataTypeNumber,
			CustomFieldDataTypeEmail,
			CustomFieldDataTypeSelect,
			CustomFieldDataTypeDate,
			CustomFieldDataTypeURL,
		}

		for _, dt := range dataTypes {
			assert.NotEmpty(t, string(dt))
		}
	})
}

// TestCustomFieldStructure tests the CustomField struct
func TestCustomFieldStructure(t *testing.T) {
	t.Run("CustomField initialization", func(t *testing.T) {
		userID := uuid.New()
		orgID := uuid.New()
		fieldID := uuid.New()

		options := pq.StringArray{"option1", "option2", "option3"}
		sampleData := "Sample value"
		fallbackValue := "Default value"

		cf := &CustomField{
			Base:           base.Base{ID: fieldID},
			TimestampMixin: base.TimestampMixin{},
			UserID:         userID,
			OrganizationID: &orgID,
			Name:           "Custom Field Name",
			NormalizedName: "custom_field_name",
			DataType:       CustomFieldDataTypeSelect,
			EntityType:     CustomFieldEntityContact,
			IsRequired:     true,
			Options:        options,
			SampleData:     &sampleData,
			FallbackValue:  &fallbackValue,
		}

		// Verify all fields
		assert.Equal(t, fieldID, cf.ID)
		assert.Equal(t, userID, cf.UserID)
		assert.Equal(t, &orgID, cf.OrganizationID)
		assert.Equal(t, "Custom Field Name", cf.Name)
		assert.Equal(t, "custom_field_name", cf.NormalizedName)
		assert.Equal(t, CustomFieldDataTypeSelect, cf.DataType)
		assert.Equal(t, CustomFieldEntityContact, cf.EntityType)
		assert.True(t, cf.IsRequired)
		assert.Equal(t, options, cf.Options)
		assert.Equal(t, &sampleData, cf.SampleData)
		assert.Equal(t, &fallbackValue, cf.FallbackValue)
	})

	t.Run("CustomField with minimal data", func(t *testing.T) {
		userID := uuid.New()
		cf := &CustomField{
			UserID:     userID,
			Name:       "Test Field",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityCompany,
		}

		assert.Equal(t, userID, cf.UserID)
		assert.Equal(t, "Test Field", cf.Name)
		assert.Equal(t, CustomFieldDataTypeText, cf.DataType)
		assert.Equal(t, CustomFieldEntityCompany, cf.EntityType)
		assert.False(t, cf.IsRequired)
		assert.Nil(t, cf.OrganizationID)
		assert.Nil(t, cf.SampleData)
		assert.Nil(t, cf.FallbackValue)
	})

	t.Run("CustomField with zero values", func(t *testing.T) {
		cf := &CustomField{}

		// Verify zero values
		assert.Equal(t, uuid.Nil, cf.ID)
		assert.Equal(t, uuid.Nil, cf.UserID)
		assert.Empty(t, cf.Name)
		assert.Empty(t, cf.NormalizedName)
		assert.Empty(t, string(cf.DataType))
		assert.Empty(t, string(cf.EntityType))
		assert.False(t, cf.IsRequired)
		assert.Nil(t, cf.OrganizationID)
		assert.Nil(t, cf.SampleData)
		assert.Nil(t, cf.FallbackValue)
		assert.Nil(t, cf.Options)
	})
}

// TestCustomFieldTableName tests the TableName method
func TestCustomFieldTableName(t *testing.T) {
	t.Run("TableName returns correct value", func(t *testing.T) {
		cf := &CustomField{}
		assert.Equal(t, "custom_fields", cf.TableName())
	})
}

// TestCustomFieldBeforeCreate tests the BeforeCreate hook
func TestCustomFieldBeforeCreate(t *testing.T) {
	t.Run("BeforeCreate with valid data", func(t *testing.T) {
		userID := uuid.New()
		cf := &CustomField{
			UserID:     userID,
			Name:       "  Test Field  ",
			DataType:   CustomFieldDataTypeNumber,
			EntityType: CustomFieldEntityContact,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, cf.ID)
		assert.Equal(t, "Test Field", cf.Name) // Should be trimmed
		assert.Equal(t, "test_field", cf.NormalizedName)
		assert.NotNil(t, cf.Options) // Should initialize empty options
		assert.Equal(t, 0, len(cf.Options))
	})

	t.Run("BeforeCreate with existing ID preserves it", func(t *testing.T) {
		userID := uuid.New()
		existingID := uuid.New()
		cf := &CustomField{
			Base:       base.Base{ID: existingID},
			UserID:     userID,
			Name:       "Test Field",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityContact,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, existingID, cf.ID)
	})

	t.Run("BeforeCreate with all data types", func(t *testing.T) {
		userID := uuid.New()
		dataTypes := []CustomFieldDataType{
			CustomFieldDataTypeText,
			CustomFieldDataTypeNumber,
			CustomFieldDataTypeEmail,
			CustomFieldDataTypeSelect,
			CustomFieldDataTypeDate,
			CustomFieldDataTypeURL,
		}

		for _, dataType := range dataTypes {
			cf := &CustomField{
				UserID:     userID,
				Name:       "Test " + string(dataType),
				DataType:   dataType,
				EntityType: CustomFieldEntityContact,
			}

			err := cf.BeforeCreate(nil)
			assert.NoError(t, err, "Should succeed for data type: %s", dataType)
			assert.Equal(t, "test_"+string(dataType), cf.NormalizedName)
		}
	})

	t.Run("BeforeCreate with company entity", func(t *testing.T) {
		userID := uuid.New()
		cf := &CustomField{
			UserID:     userID,
			Name:       "Company Field",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityCompany,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, "company_field", cf.NormalizedName)
	})
}

// TestCustomFieldBeforeUpdate tests the BeforeUpdate hook
func TestCustomFieldBeforeUpdate(t *testing.T) {
	t.Run("BeforeUpdate normalizes name", func(t *testing.T) {
		cf := &CustomField{
			UserID:     uuid.New(),
			Name:       "  Updated Field  ",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityContact,
		}

		err := cf.BeforeUpdate(nil)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Field", cf.Name)
		assert.Equal(t, "updated_field", cf.NormalizedName)
	})

	t.Run("BeforeUpdate validates data", func(t *testing.T) {
		cf := &CustomField{
			UserID:     uuid.New(),
			Name:       "Test",
			DataType:   CustomFieldDataType("invalid"),
			EntityType: CustomFieldEntityContact,
		}

		err := cf.BeforeUpdate(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid data_type")
	})
}

// TestNormalizeAndValidate tests the normalizeAndValidate method
func TestNormalizeAndValidate(t *testing.T) {
	t.Run("Valid field normalization", func(t *testing.T) {
		cf := &CustomField{
			Name:       "  Test Field Name  ",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityContact,
			IsRequired: true,
		}

		err := cf.normalizeAndValidate()
		assert.NoError(t, err)
		assert.Equal(t, "Test Field Name", cf.Name)
		assert.Equal(t, "test_field_name", cf.NormalizedName)
	})

	t.Run("Empty name validation", func(t *testing.T) {
		cf := &CustomField{
			Name:       "",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityContact,
		}

		err := cf.normalizeAndValidate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("Whitespace only name validation", func(t *testing.T) {
		cf := &CustomField{
			Name:       "   ",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityContact,
		}

		err := cf.normalizeAndValidate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("Nil options initialization", func(t *testing.T) {
		cf := &CustomField{
			Name:       "Test",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityContact,
		}

		err := cf.normalizeAndValidate()
		assert.NoError(t, err)
		assert.NotNil(t, cf.Options)
		assert.Equal(t, 0, len(cf.Options))
	})

	t.Run("Existing options preserved", func(t *testing.T) {
		options := pq.StringArray{"opt1", "opt2"}
		cf := &CustomField{
			Name:       "Test",
			DataType:   CustomFieldDataTypeSelect,
			EntityType: CustomFieldEntityContact,
			Options:    options,
		}

		err := cf.normalizeAndValidate()
		assert.NoError(t, err)
		assert.Equal(t, options, cf.Options)
	})
}

// TestCustomFieldNormalizeName tests the name normalization function
func TestCustomFieldNormalizeName(t *testing.T) {
	t.Run("Basic name normalization", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"  First Name  ", "first_name"},
			{"Last Name", "last_name"},
			{"email address", "email_address"},
			{"PHONE_NUMBER", "phone_number"},
			{"Company-Name", "company-name"},
			{"Multiple   spaces   between", "multiple___spaces___between"},
			{"", ""},
			{"  ", ""},
			{"Single", "single"},
			{"UPPERCASE", "uppercase"},
			{"lowercase", "lowercase"},
			{"Mixed Case With Spaces", "mixed_case_with_spaces"},
			{"123 Number Start", "123_number_start"},
			{"Special!@#$%Characters", "special!@#$%characters"},
		}

		for _, tc := range testCases {
			result := normalizeCustomFieldName(tc.input)
			assert.Equal(t, tc.expected, result, "Input: %q", tc.input)
		}
	})

	t.Run("Complex name scenarios", func(t *testing.T) {
		// Test leading/trailing/multiple spaces
		assert.Equal(t, "lead_score", normalizeCustomFieldName("  Lead Score  "))
		assert.Equal(t, "first_name", normalizeCustomFieldName("First Name"))
		assert.Equal(t, "last_name", normalizeCustomFieldName("last name"))
		assert.Equal(t, "email", normalizeCustomFieldName("email"))
		assert.Equal(t, "contact_source", normalizeCustomFieldName("Contact Source"))
		assert.Equal(t, "priority_level", normalizeCustomFieldName("Priority Level"))
	})
}

// TestCustomFieldValidationScenarios tests various validation scenarios
func TestCustomFieldValidationScenarios(t *testing.T) {
	t.Run("All valid data types", func(t *testing.T) {
		validDataTypes := []CustomFieldDataType{
			CustomFieldDataTypeText,
			CustomFieldDataTypeNumber,
			CustomFieldDataTypeEmail,
			CustomFieldDataTypeSelect,
			CustomFieldDataTypeDate,
			CustomFieldDataTypeURL,
		}

		for _, dataType := range validDataTypes {
			cf := &CustomField{
				UserID:     uuid.New(),
				Name:       "Test Field",
				DataType:   dataType,
				EntityType: CustomFieldEntityContact,
			}

			err := cf.BeforeCreate(nil)
			assert.NoError(t, err, "Data type %s should be valid", dataType)
		}
	})

	t.Run("Invalid data types", func(t *testing.T) {
		invalidDataTypes := []string{
			"invalid",
			"boolean",
			"datetime",
			"array",
			"object",
			"",
		}

		for _, invalidType := range invalidDataTypes {
			cf := &CustomField{
				UserID:     uuid.New(),
				Name:       "Test Field",
				DataType:   CustomFieldDataType(invalidType),
				EntityType: CustomFieldEntityContact,
			}

			err := cf.BeforeCreate(nil)
			assert.Error(t, err, "Data type %s should be invalid", invalidType)
			assert.Contains(t, err.Error(), "invalid data_type")
		}
	})

	t.Run("All valid entities", func(t *testing.T) {
		validEntities := []CustomFieldEntity{
			CustomFieldEntityContact,
			CustomFieldEntityCompany,
		}

		for _, entity := range validEntities {
			cf := &CustomField{
				UserID:     uuid.New(),
				Name:       "Test Field",
				DataType:   CustomFieldDataTypeText,
				EntityType: entity,
			}

			err := cf.BeforeCreate(nil)
			assert.NoError(t, err, "Entity %s should be valid", entity)
		}
	})

	t.Run("Invalid entities", func(t *testing.T) {
		invalidEntities := []string{
			"invalid",
			"user",
			"campaign",
			"deal",
			"task",
			"",
		}

		for _, invalidEntity := range invalidEntities {
			cf := &CustomField{
				UserID:     uuid.New(),
				Name:       "Test Field",
				DataType:   CustomFieldDataTypeText,
				EntityType: CustomFieldEntity(invalidEntity),
			}

			err := cf.BeforeCreate(nil)
			assert.Error(t, err, "Entity %s should be invalid", invalidEntity)
			assert.Contains(t, err.Error(), "invalid entity_type")
		}
	})
}

// TestCustomFieldOptions tests options handling
func TestCustomFieldOptions(t *testing.T) {
	t.Run("Options for select type", func(t *testing.T) {
		options := pq.StringArray{"Option 1", "Option 2", "Option 3"}
		cf := &CustomField{
			UserID:     uuid.New(),
			Name:       "Select Field",
			DataType:   CustomFieldDataTypeSelect,
			EntityType: CustomFieldEntityContact,
			Options:    options,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, options, cf.Options)
		assert.Equal(t, 3, len(cf.Options))
		assert.Equal(t, "Option 1", cf.Options[0])
		assert.Equal(t, "Option 2", cf.Options[1])
		assert.Equal(t, "Option 3", cf.Options[2])
	})

	t.Run("Empty options for non-select types", func(t *testing.T) {
		cf := &CustomField{
			UserID:     uuid.New(),
			Name:       "Text Field",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityContact,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotNil(t, cf.Options)
		assert.Equal(t, 0, len(cf.Options))
	})

	t.Run("Options with empty array", func(t *testing.T) {
		options := pq.StringArray{}
		cf := &CustomField{
			UserID:     uuid.New(),
			Name:       "Field with no options",
			DataType:   CustomFieldDataTypeSelect,
			EntityType: CustomFieldEntityContact,
			Options:    options,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, options, cf.Options)
		assert.Equal(t, 0, len(cf.Options))
	})

	t.Run("Options with special characters", func(t *testing.T) {
		options := pq.StringArray{"Option \"A\"", "Option 'B'", "Option (C)", "Option [D]"}
		cf := &CustomField{
			UserID:     uuid.New(),
			Name:       "Special Options",
			DataType:   CustomFieldDataTypeSelect,
			EntityType: CustomFieldEntityContact,
			Options:    options,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, 4, len(cf.Options))
	})
}

// TestCustomFieldSampleDataAndFallback tests optional string fields
func TestCustomFieldSampleDataAndFallback(t *testing.T) {
	t.Run("With sample data", func(t *testing.T) {
		sample := "Sample value"
		cf := &CustomField{
			UserID:     uuid.New(),
			Name:       "Test Field",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityContact,
			SampleData: &sample,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, &sample, cf.SampleData)
		assert.Equal(t, "Sample value", *cf.SampleData)
	})

	t.Run("Without sample data", func(t *testing.T) {
		cf := &CustomField{
			UserID:     uuid.New(),
			Name:       "Test Field",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityContact,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Nil(t, cf.SampleData)
	})

	t.Run("With fallback value", func(t *testing.T) {
		fallback := "Default value"
		cf := &CustomField{
			UserID:        uuid.New(),
			Name:          "Test Field",
			DataType:      CustomFieldDataTypeText,
			EntityType:    CustomFieldEntityContact,
			FallbackValue: &fallback,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, &fallback, cf.FallbackValue)
		assert.Equal(t, "Default value", *cf.FallbackValue)
	})

	t.Run("Without fallback value", func(t *testing.T) {
		cf := &CustomField{
			UserID:     uuid.New(),
			Name:       "Test Field",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityContact,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Nil(t, cf.FallbackValue)
	})

	t.Run("Empty string values", func(t *testing.T) {
		empty := ""
		cf := &CustomField{
			UserID:        uuid.New(),
			Name:          "Test Field",
			DataType:      CustomFieldDataTypeText,
			EntityType:    CustomFieldEntityContact,
			SampleData:    &empty,
			FallbackValue: &empty,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, &empty, cf.SampleData)
		assert.Equal(t, "", *cf.SampleData)
		assert.Equal(t, &empty, cf.FallbackValue)
		assert.Equal(t, "", *cf.FallbackValue)
	})
}

// TestCustomFieldOrganizationID tests organization handling
func TestCustomFieldOrganizationID(t *testing.T) {
	t.Run("With organization ID", func(t *testing.T) {
		orgID := uuid.New()
		cf := &CustomField{
			UserID:         uuid.New(),
			Name:           "Org Field",
			DataType:       CustomFieldDataTypeText,
			EntityType:     CustomFieldEntityContact,
			OrganizationID: &orgID,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, &orgID, cf.OrganizationID)
	})

	t.Run("Without organization ID", func(t *testing.T) {
		cf := &CustomField{
			UserID:     uuid.New(),
			Name:       "No Org Field",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityContact,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Nil(t, cf.OrganizationID)
	})

	t.Run("Update organization ID", func(t *testing.T) {
		cf := &CustomField{
			UserID:     uuid.New(),
			Name:       "Test",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityContact,
		}

		err := cf.BeforeCreate(nil)
		require.NoError(t, err)

		orgID := uuid.New()
		cf.OrganizationID = &orgID
		assert.Equal(t, &orgID, cf.OrganizationID)

		cf.OrganizationID = nil
		assert.Nil(t, cf.OrganizationID)
	})
}

// TestCustomFieldIsRequired tests the IsRequired field
func TestCustomFieldIsRequired(t *testing.T) {
	t.Run("Required field", func(t *testing.T) {
		cf := &CustomField{
			UserID:     uuid.New(),
			Name:       "Required Field",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityContact,
			IsRequired: true,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.True(t, cf.IsRequired)
	})

	t.Run("Optional field", func(t *testing.T) {
		cf := &CustomField{
			UserID:     uuid.New(),
			Name:       "Optional Field",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityContact,
			IsRequired: false,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.False(t, cf.IsRequired)
	})

	t.Run("Default is required false", func(t *testing.T) {
		cf := &CustomField{
			UserID:     uuid.New(),
			Name:       "Default Field",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityContact,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.False(t, cf.IsRequired)
	})
}

// TestCustomFieldEdgeCases tests edge cases and error conditions
func TestCustomFieldEdgeCases(t *testing.T) {
	t.Run("Multiple validation errors", func(t *testing.T) {
		cf := &CustomField{
			UserID:     uuid.New(),
			Name:       "",
			DataType:   CustomFieldDataType("invalid"),
			EntityType: CustomFieldEntity("invalid"),
		}

		// Should catch the first error (empty name)
		err := cf.normalizeAndValidate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("Very long name", func(t *testing.T) {
		longName := string(make([]byte, 1000))
		cf := &CustomField{
			UserID:     uuid.New(),
			Name:       longName,
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityContact,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, len(longName), len(cf.Name))
	})

	t.Run("Name with special characters", func(t *testing.T) {
		cf := &CustomField{
			UserID:     uuid.New(),
			Name:       "Field @#$%^&*()_+-={}[]|\\:;\"'<>,.?/~`",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityContact,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Contains(t, cf.Name, "@#$%^&*()")
	})

	t.Run("Base BeforeCreate error propagation", func(t *testing.T) {
		// This tests error propagation from Base.BeforeCreate
		cf := &CustomField{
			UserID:     uuid.Nil, // Invalid UserID for testing
			Name:       "Test Field",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityContact,
		}

		// Base.BeforeCreate should still work even with nil UserID
		// The validation happens in normalizeAndValidate
		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, cf.ID)
	})
}

// TestCustomFieldRealWorldScenarios tests real-world usage scenarios
func TestCustomFieldRealWorldScenarios(t *testing.T) {
	t.Run("Contact custom fields", func(t *testing.T) {
		userID := uuid.New()

		// Lead Score field
		leadScore := &CustomField{
			UserID:     userID,
			Name:       "Lead Score",
			DataType:   CustomFieldDataTypeNumber,
			EntityType: CustomFieldEntityContact,
			IsRequired: false,
		}

		err := leadScore.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, "lead_score", leadScore.NormalizedName)

		// Preferred Contact Method
		contactMethod := &CustomField{
			UserID:     userID,
			Name:       "Preferred Contact Method",
			DataType:   CustomFieldDataTypeSelect,
			EntityType: CustomFieldEntityContact,
			IsRequired: true,
			Options:    pq.StringArray{"Email", "Phone", "SMS"},
		}

		err = contactMethod.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, "preferred_contact_method", contactMethod.NormalizedName)
		assert.Equal(t, 3, len(contactMethod.Options))

		// Next Follow-up Date
		followUpDate := &CustomField{
			UserID:     userID,
			Name:       "Next Follow-up Date",
			DataType:   CustomFieldDataTypeDate,
			EntityType: CustomFieldEntityContact,
			SampleData: stringPtr("2024-12-18"),
		}

		err = followUpDate.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, "next_follow-up_date", followUpDate.NormalizedName)
	})

	t.Run("Company custom fields", func(t *testing.T) {
		userID := uuid.New()
		orgID := uuid.New()

		// Industry
		industry := &CustomField{
			UserID:         userID,
			OrganizationID: &orgID,
			Name:           "Industry",
			DataType:       CustomFieldDataTypeSelect,
			EntityType:     CustomFieldEntityCompany,
			IsRequired:     true,
			Options:        pq.StringArray{"Technology", "Healthcare", "Finance", "Manufacturing"},
		}

		err := industry.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, "industry", industry.NormalizedName)

		// Website
		website := &CustomField{
			UserID:         userID,
			OrganizationID: &orgID,
			Name:           "Company Website",
			DataType:       CustomFieldDataTypeURL,
			EntityType:     CustomFieldEntityCompany,
			FallbackValue:  stringPtr("https://example.com"),
		}

		err = website.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, "company_website", website.NormalizedName)
	})

	t.Run("Email validation field", func(t *testing.T) {
		userID := uuid.New()

		emailField := &CustomField{
			UserID:     userID,
			Name:       "Secondary Email",
			DataType:   CustomFieldDataTypeEmail,
			EntityType: CustomFieldEntityContact,
			IsRequired: false,
			SampleData: stringPtr("secondary@example.com"),
		}

		err := emailField.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, "secondary_email", emailField.NormalizedName)
		assert.Equal(t, "secondary@example.com", *emailField.SampleData)
	})
}

// TestCustomFieldCombinations tests various field combinations
func TestCustomFieldCombinations(t *testing.T) {
	t.Run("Text field with all optional fields", func(t *testing.T) {
		userID := uuid.New()
		orgID := uuid.New()
		sample := "Sample text"
		fallback := "Default text"

		cf := &CustomField{
			UserID:         userID,
			OrganizationID: &orgID,
			Name:           "Complete Text Field",
			DataType:       CustomFieldDataTypeText,
			EntityType:     CustomFieldEntityContact,
			IsRequired:     true,
			Options:        pq.StringArray{"opt1"},
			SampleData:     &sample,
			FallbackValue:  &fallback,
		}

		err := cf.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, "complete_text_field", cf.NormalizedName)
		assert.Equal(t, &sample, cf.SampleData)
		assert.Equal(t, &fallback, cf.FallbackValue)
	})

	t.Run("Minimal field for each entity", func(t *testing.T) {
		userID := uuid.New()

		// Contact field
		contactField := &CustomField{
			UserID:     userID,
			Name:       "Contact Field",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityContact,
		}

		err := contactField.BeforeCreate(nil)
		assert.NoError(t, err)

		// Company field
		companyField := &CustomField{
			UserID:     userID,
			Name:       "Company Field",
			DataType:   CustomFieldDataTypeText,
			EntityType: CustomFieldEntityCompany,
		}

		err = companyField.BeforeCreate(nil)
		assert.NoError(t, err)
	})
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

// TestCustomFieldJSONSerialization tests JSON serialization
func TestCustomFieldJSONSerialization(t *testing.T) {
	t.Run("JSON serialization with all fields", func(t *testing.T) {
		userID := uuid.New()
		orgID := uuid.New()
		sample := "Sample value"
		fallback := "Default value"
		options := pq.StringArray{"Option1", "Option2"}

		cf := &CustomField{
			Base:           base.Base{ID: uuid.New()},
			UserID:         userID,
			OrganizationID: &orgID,
			Name:           "Test Field",
			NormalizedName: "test_field",
			DataType:       CustomFieldDataTypeSelect,
			EntityType:     CustomFieldEntityContact,
			IsRequired:     true,
			Options:        options,
			SampleData:     &sample,
			FallbackValue:  &fallback,
		}

		// Serialize to JSON (testing that it can be marshaled)
		data, err := json.Marshal(cf)
		require.NoError(t, err)
		assert.Contains(t, string(data), "test_field")
		assert.Contains(t, string(data), "Test Field")

		// Deserialize back
		var cf2 CustomField
		err = json.Unmarshal(data, &cf2)
		require.NoError(t, err)
		assert.Equal(t, cf.Name, cf2.Name)
		assert.Equal(t, cf.DataType, cf2.DataType)
		assert.Equal(t, cf.EntityType, cf2.EntityType)
	})
}

// TestCustomFieldPerformance tests performance with many fields
func TestCustomFieldPerformance(t *testing.T) {
	t.Run("Create many custom fields", func(t *testing.T) {
		userID := uuid.New()

		for i := 0; i < 100; i++ {
			cf := &CustomField{
				UserID:     userID,
				Name:       "Field " + string(rune(i)),
				DataType:   CustomFieldDataTypeText,
				EntityType: CustomFieldEntityContact,
			}

			err := cf.BeforeCreate(nil)
			if err != nil {
				t.Fatalf("Failed to create field %d: %v", i, err)
			}
		}
	})
}

// TestCustomFieldAllowedDataTypes tests the allowed data types map
func TestCustomFieldAllowedDataTypes(t *testing.T) {
	t.Run("Check all allowed data types", func(t *testing.T) {
		allowedTypes := []CustomFieldDataType{
			CustomFieldDataTypeText,
			CustomFieldDataTypeNumber,
			CustomFieldDataTypeEmail,
			CustomFieldDataTypeSelect,
			CustomFieldDataTypeDate,
			CustomFieldDataTypeURL,
		}

		for _, dataType := range allowedTypes {
			_, exists := allowedDataTypes[dataType]
			assert.True(t, exists, "Data type %s should be in allowed map", dataType)
		}
	})
}
