package template

import (
	"testing"

	"github.com/google/uuid"
)

func TestTemplateBeforeCreateDefaults(t *testing.T) {
	template := Template{}

	if err := template.BeforeCreate(nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if template.ID == uuid.Nil {
		t.Fatal("expected template ID to be set")
	}

	if template.Position != defaultTemplatePosition {
		t.Fatalf("expected default position %.1f, got %.1f", defaultTemplatePosition, template.Position)
	}

	if string(template.CMetadata) != "{}" {
		t.Fatalf("expected default cmetadata '{}', got %s", string(template.CMetadata))
	}
}

func TestEmailTypeForIndex(t *testing.T) {
	tests := []struct {
		index int
		want  string
	}{
		{index: 0, want: "First Email"},
		{index: 1, want: "Follow-up Email 1"},
		{index: 4, want: "Follow-up Email 4"},
		{index: -1, want: "First Email"},
	}

	for _, tc := range tests {
		if got := EmailTypeForIndex(tc.index); got != tc.want {
			t.Fatalf("EmailTypeForIndex(%d) = %s, want %s", tc.index, got, tc.want)
		}
	}
}

func TestCalculatePositionBetween(t *testing.T) {
	pos, err := CalculatePositionBetween(1000, 2000)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pos != 1500 {
		t.Fatalf("expected midpoint 1500, got %.1f", pos)
	}

	if _, err := CalculatePositionBetween(0, 0); err == nil {
		t.Fatal("expected error when both positions are zero")
	}
}

func TestNormalizePosition(t *testing.T) {
	if got := NormalizePosition(0); got != 1000 {
		t.Fatalf("expected normalize(0) = 1000, got %.1f", got)
	}
	if got := NormalizePosition(2); got != 3000 {
		t.Fatalf("expected normalize(2) = 3000, got %.1f", got)
	}
}
