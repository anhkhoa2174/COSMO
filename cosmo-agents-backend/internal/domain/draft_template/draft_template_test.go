package draft_template

import (
	"testing"

	"github.com/google/uuid"
)

func TestDraftTemplateFields(t *testing.T) {
	campaignID := uuid.New()
	templateID := uuid.New()
	draft := DraftTemplate{
		Intent:     "Interested",
		CampaignID: campaignID,
		TemplateID: templateID,
	}

	if (DraftTemplate{}).TableName() != "draft_templates" {
		t.Fatalf("unexpected table name")
	}

	if draft.Intent == "" {
		t.Fatal("intent should be required")
	}
	if draft.CampaignID != campaignID {
		t.Fatalf("expected campaign id %s, got %s", campaignID, draft.CampaignID)
	}
	if draft.TemplateID != templateID {
		t.Fatalf("expected template id %s, got %s", templateID, draft.TemplateID)
	}
}
