package integration

import (
	"testing"

	"github.com/google/uuid"
)

func TestIntegrationTableName(t *testing.T) {
	if (Integration{}).TableName() != "integrations" {
		t.Fatalf("unexpected table")
	}
}

func TestIntegrationFields(t *testing.T) {
	integration := Integration{
		ID:       "integration-1",
		UserID:   uuid.New(),
		Source:   SourceIntegrationHubspot,
		SourceID: "hubspot-123",
	}

	if integration.Source != SourceIntegrationHubspot {
		t.Fatalf("source mismatch")
	}
}
