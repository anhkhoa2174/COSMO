package lead_form_integration

import (
	"testing"

	"github.com/google/uuid"
)

func TestLeadFormIntegrationTable(t *testing.T) {
	if (LeadFormIntegration{}).TableName() != "lead_form_integrations" {
		t.Fatalf("unexpected table")
	}
}

func TestLeadFieldMappingTable(t *testing.T) {
	mapping := LeadFieldMapping{
		CampaignID:        uuid.New(),
		FormIntegrationID: uuid.New(),
		ExternalFieldName: "email",
		MappingType:       MappingTypeContactField,
	}

	if mapping.TableName() != "lead_field_mappings" {
		t.Fatalf("unexpected table")
	}
}
