package email

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

func TestEmailBeforeCreateDefaults(t *testing.T) {
	email := Email{}

	if err := email.BeforeCreate(nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if email.ID == uuid.Nil {
		t.Fatal("expected email ID to be set")
	}
	if email.Status != EmailStatusSending {
		t.Fatalf("expected default status %q, got %q", EmailStatusSending, email.Status)
	}
	if email.Attachments == nil || len(email.Attachments) != 0 {
		t.Fatalf("expected empty attachments slice, got %v", email.Attachments)
	}
	if email.Labels == nil || len(email.Labels) != 0 {
		t.Fatalf("expected empty labels slice, got %v", email.Labels)
	}
	if email.Intents == nil || len(email.Intents) != 0 {
		t.Fatalf("expected empty intents slice, got %v", email.Intents)
	}
}

func TestValidateStatus(t *testing.T) {
	valid := []EmailStatus{EmailStatusSending, EmailStatusSent, EmailStatusInbox}
	for _, status := range valid {
		if err := ValidateStatus(status); err != nil {
			t.Fatalf("expected status %q to be valid, got error %v", status, err)
		}
	}

	if err := ValidateStatus(EmailStatus("unknown")); err == nil {
		t.Fatal("expected invalid status error")
	}
}

func TestEmailBeforeCreatePreservesProvidedSlices(t *testing.T) {
	email := Email{
		Attachments: pq.StringArray{"id-1"},
		Labels:      pq.StringArray{"important"},
		Intents:     pq.StringArray{"Interested"},
		Status:      EmailStatusSent,
	}

	if err := email.BeforeCreate(nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(email.Attachments) != 1 || email.Attachments[0] != "id-1" {
		t.Fatalf("expected attachments to be preserved, got %v", email.Attachments)
	}
	if email.Status != EmailStatusSent {
		t.Fatalf("expected status to remain %q, got %q", EmailStatusSent, email.Status)
	}
}
