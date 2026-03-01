package contact

import (
	"reflect"
	"strings"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// FieldMapper handles field mapping for contact imports (HubSpot, CSV, etc.)
type FieldMapper struct {
	standardFields map[string]struct{}
}

// NewFieldMapper creates a new FieldMapper
func NewFieldMapper() *FieldMapper {
	return &FieldMapper{
		standardFields: buildStandardFieldSet(),
	}
}

// buildStandardFieldSet creates a set of standard contact fields
func buildStandardFieldSet() map[string]struct{} {
	fields := map[string]struct{}{
		"first_name":     {},
		"last_name":      {},
		"email":          {},
		"phone":          {},
		"company":        {},
		"job_title":      {},
		"address":        {},
		"city":           {},
		"country":        {},
		"state":          {},
		"zip":            {},
		"postal_code":    {},
		"website":        {},
		"linkedin_url":   {},
		"twitter_handle": {},
		"lead_score":     {},
		"lead_status":    {},
		"lead_source":    {},
		"source_id":      {},
		"hubspot_id":     {},
		"profile":        {},
		"tags":           {},
		"do_not_contact": {},
		"notes":          {},
	}

	return fields
}

// FlattenMapping converts a mapping struct to a flat map
// Generic function that works with any mapping structure using reflection
func (m *FieldMapper) FlattenMapping(mapping interface{}) map[string]string {
	result := make(map[string]string)

	val := reflect.ValueOf(mapping)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Get JSON tag name
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Handle json tags like "field_name,omitempty"
		fieldName := strings.Split(jsonTag, ",")[0]

		// Handle pointer to string fields
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			if field.Type().Elem().Kind() == reflect.String {
				value := field.Elem().String()
				if strings.TrimSpace(value) != "" {
					result[fieldName] = strings.TrimSpace(value)
				}
			}
		}

		// Handle map fields (for custom fields)
		if field.Kind() == reflect.Map {
			for _, key := range field.MapKeys() {
				mapValue := field.MapIndex(key)
				keyStr := key.String()
				valueStr := mapValue.String()

				if strings.TrimSpace(keyStr) != "" && strings.TrimSpace(valueStr) != "" {
					result[strings.TrimSpace(keyStr)] = strings.TrimSpace(valueStr)
				}
			}
		}
	}

	return result
}

// BuildAllowedFields creates a set of allowed fields including custom fields
func (m *FieldMapper) BuildAllowedFields(customFields []domain.CustomField) map[string]struct{} {
	allowed := make(map[string]struct{})

	// Add standard fields
	for field := range m.standardFields {
		allowed[field] = struct{}{}
	}

	// Add custom fields
	for _, cf := range customFields {
		if name := strings.TrimSpace(cf.NormalizedName); name != "" {
			allowed[name] = struct{}{}
		}
	}

	return allowed
}

// IsStandardField checks if a field is a standard contact field
func (m *FieldMapper) IsStandardField(fieldName string) bool {
	_, exists := m.standardFields[strings.ToLower(strings.TrimSpace(fieldName))]
	return exists
}

// GetStandardFields returns all standard field names
func (m *FieldMapper) GetStandardFields() []string {
	fields := make([]string, 0, len(m.standardFields))
	for field := range m.standardFields {
		fields = append(fields, field)
	}
	return fields
}

// HubSpotFieldMapping provides default HubSpot to internal field mapping
var HubSpotFieldMapping = map[string]string{
	"firstname":            "first_name",
	"lastname":             "last_name",
	"email":                "email",
	"phone":                "phone",
	"company":              "company",
	"jobtitle":             "job_title",
	"address":              "address",
	"city":                 "city",
	"country":              "country",
	"state":                "state",
	"zip":                  "zip",
	"website":              "website",
	"linkedin_url":         "linkedin_url",
	"twitterhandle":        "twitter_handle",
	"hs_lead_status":       "lead_status",
	"hs_analytics_source":  "lead_source",
	"notes_last_contacted": "notes",
}

// MapHubSpotField maps a HubSpot field name to internal field name
func (m *FieldMapper) MapHubSpotField(hubspotField string) string {
	if internalField, exists := HubSpotFieldMapping[strings.ToLower(hubspotField)]; exists {
		return internalField
	}
	return hubspotField
}
