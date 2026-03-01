package sale_rep

import (
	"testing"

	"github.com/google/uuid"
)

func TestSaleRepAllowedMergeTags(t *testing.T) {
	tags := SaleRepAllowedMergeTags(false)
	if len(tags) != 4 {
		t.Fatalf("expected 4 tags, got %d", len(tags))
	}
	withPrefix := SaleRepAllowedMergeTags(true)
	if withPrefix[0] != "sale_rep_first_name" {
		t.Fatalf("expected prefixed tag, got %s", withPrefix[0])
	}
}

func TestSaleRepToEmailTemplateContext(t *testing.T) {
	rep := &SaleRep{
		FirstName:      "Alice",
		LastName:       "Smith",
		Email:          "alice@example.com",
		CalendarLink:   "https://cal.alice",
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
	}

	ctx := rep.ToEmailTemplateContext()
	expected := map[string]string{
		"sale_rep_first_name":    "Alice",
		"sale_rep_last_name":     "Smith",
		"sale_rep_email":         "alice@example.com",
		"sale_rep_calendar_link": "https://cal.alice",
	}

	for key, want := range expected {
		if got := ctx[key]; got != want {
			t.Fatalf("expected %s to be %s, got %s", key, want, got)
		}
	}

	if len(ctx) != len(expected) {
		t.Fatalf("expected %d keys, got %d", len(expected), len(ctx))
	}
}
