package custom_field

import (
	"testing"

	"github.com/google/uuid"
)

func TestCustomFieldNormalizeAndValidateDefaults(t *testing.T) {
	cf := &CustomField{
		UserID:     uuid.New(),
		Name:       "  Lead Score ",
		DataType:   CustomFieldDataTypeNumber,
		EntityType: CustomFieldEntityContact,
	}

	if err := cf.BeforeCreate(nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cf.NormalizedName != "lead_score" {
		t.Fatalf("expected normalized name lead_score, got %s", cf.NormalizedName)
	}
	if cf.Options == nil || len(cf.Options) != 0 {
		t.Fatalf("expected empty options, got %v", cf.Options)
	}
}

func TestCustomFieldValidateInvalidDataType(t *testing.T) {
	cf := &CustomField{
		UserID:     uuid.New(),
		Name:       "Invalid",
		DataType:   CustomFieldDataType("unknown"),
		EntityType: CustomFieldEntityContact,
	}

	if err := cf.BeforeCreate(nil); err == nil {
		t.Fatal("expected error for invalid data type")
	}
}

func TestCustomFieldValidateInvalidEntity(t *testing.T) {
	cf := &CustomField{
		UserID:     uuid.New(),
		Name:       "Invalid",
		DataType:   CustomFieldDataTypeText,
		EntityType: CustomFieldEntity("invalid"),
	}

	if err := cf.BeforeCreate(nil); err == nil {
		t.Fatal("expected error for invalid entity type")
	}
}

func TestNormalizeCustomFieldName(t *testing.T) {
	if got := normalizeCustomFieldName("  First Name "); got != "first_name" {
		t.Fatalf("expected first_name, got %s", got)
	}
}
