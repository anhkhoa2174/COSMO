package contact

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	customFieldRepo "github.com/rockship/cosmo-agents-go/internal/repository/custom_field"
)

// ContactFieldValidator validates contact fields against standard and custom fields
type ContactFieldValidator struct {
	customFieldRepo *customFieldRepo.CustomFieldRepository
	standardFields  map[string]bool
}

// NewContactFieldValidator creates a new ContactFieldValidator with standard contact fields
func NewContactFieldValidator(customFieldRepo *customFieldRepo.CustomFieldRepository) *ContactFieldValidator {
	return &ContactFieldValidator{
		customFieldRepo: customFieldRepo,
		standardFields:  buildStandardContactFields(),
	}
}

// buildStandardContactFields builds a map of standard contact fields using reflection
// This eliminates the need to hardcode field names
func buildStandardContactFields() map[string]bool {
	fields := make(map[string]bool)

	// Use reflection on domain.Contact to get all JSON tags
	t := reflect.TypeOf(domain.Contact{})
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			// Handle json tags like "field_name,omitempty"
			fieldName := strings.Split(jsonTag, ",")[0]
			fields[fieldName] = true
		}
	}

	// Also include commonly used field names that might not be in struct
	additionalFields := []string{
		"first_name", "last_name", "email", "phone", "company",
		"job_title", "address", "city", "state", "country",
		"postal_code", "website", "linkedin_url", "twitter_handle",
		"lead_score", "lead_status", "lead_source", "tags",
		"notes", "created_at", "updated_at", "organization_id",
		"user_id", "attributes", "profile", "custom_fields",
		"confirmed_facts", "ai_insights", "insight_validation", "scores",
	}

	for _, field := range additionalFields {
		fields[field] = true
	}

	return fields
}

// ValidateContactFields validates that all fields in the attributes map are valid
// Returns an error if any field is not found in standard or custom fields
func (v *ContactFieldValidator) ValidateContactFields(ctx context.Context, attributes map[string]interface{}, organizationID uuid.UUID) error {
	if len(attributes) == 0 {
		return nil
	}

	// Get custom fields for organization
	customFields, err := v.customFieldRepo.FindByOrganizationID(ctx, organizationID)
	if err != nil {
		return fmt.Errorf("failed to fetch custom fields: %w", err)
	}

	// Build custom field map
	customFieldMap := make(map[string]bool)
	for _, cf := range customFields {
		customFieldMap[cf.NormalizedName] = true
	}

	// Validate each field
	var invalidFields []string
	for key := range attributes {
		normalizedKey := normalizeFieldName(key)
		if !v.standardFields[normalizedKey] && !customFieldMap[normalizedKey] {
			invalidFields = append(invalidFields, key)
		}
	}

	if len(invalidFields) > 0 {
		return fmt.Errorf("invalid fields: %s", strings.Join(invalidFields, ", "))
	}

	return nil
}

// IsStandardField checks if a field is a standard contact field
func (v *ContactFieldValidator) IsStandardField(fieldName string) bool {
	return v.standardFields[normalizeFieldName(fieldName)]
}

// GetStandardFields returns all standard contact field names
func (v *ContactFieldValidator) GetStandardFields() []string {
	fields := make([]string, 0, len(v.standardFields))
	for field := range v.standardFields {
		fields = append(fields, field)
	}
	return fields
}

// GetCustomFields retrieves custom fields for an organization
func (v *ContactFieldValidator) GetCustomFields(ctx context.Context, organizationID uuid.UUID) ([]domain.CustomField, error) {
	return v.customFieldRepo.FindByOrganizationID(ctx, organizationID)
}

// normalizeFieldName converts field names to lowercase and replaces spaces with underscores
func normalizeFieldName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "_"))
}
