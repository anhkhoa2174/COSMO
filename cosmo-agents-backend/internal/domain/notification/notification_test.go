package notification

import (
	"github.com/google/uuid"
	"testing"
)

func TestNotificationTableName(t *testing.T) {
	if (Notification{}).TableName() != "notifications" {
		t.Fatalf("unexpected table name")
	}
}

func TestNotificationFields(t *testing.T) {
	userID := uuid.New()
	campaignID := uuid.New()
	n := Notification{UserID: userID, CampaignID: campaignID}

	if n.UserID != userID {
		t.Fatalf("expected user id to match")
	}
	if n.CampaignID != campaignID {
		t.Fatalf("expected campaign id to match")
	}
}
