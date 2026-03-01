package contact

import "testing"

func TestContactTypeConstants(t *testing.T) {
	if TypePullHubspotContacts == "" || TypeContactImportCSV == "" || TypeContactEnrich == "" || TypeContactImportHubspot == "" {
		t.Fatalf("expected contact task types to be non-empty")
	}
}

func TestParseFirstName(t *testing.T) {
	value := "Jane"
	input := map[string]any{
		"contact": map[string]any{
			"first_name": value,
		},
	}

	got := parseFirstName(input)
	if got == nil || *got != value {
		t.Fatalf("expected first name %s, got %v", value, got)
	}

	if parseFirstName(nil) != nil {
		t.Fatalf("expected nil when input missing")
	}
}

func TestParseInt(t *testing.T) {
	if parseInt("10") != 10 {
		t.Fatalf("expected 10")
	}
	if parseInt("bad") != 0 {
		t.Fatalf("expected 0 on invalid input")
	}
}

func TestGetPropertyOrNA(t *testing.T) {
	props := map[string]interface{}{"name": "Alice", "age": 30}
	if got := getPropertyOrNA(props, "name"); got != "Alice" {
		t.Fatalf("expected Alice, got %s", got)
	}
	if got := getPropertyOrNA(props, "missing"); got == "" {
		t.Fatalf("expected default value for missing key")
	}
	if got := getPropertyOrNA(props, "age"); got == "" {
		t.Fatalf("expected fallback value for non-string entry")
	}
}

func TestParseSchemaV2(t *testing.T) {
	input := map[string]any{
		"contact": map[string]any{
			"first_name": "Bob",
		},
	}
	var result struct {
		Contact struct {
			FirstName string `mapstructure:"first_name"`
		} `mapstructure:"contact"`
	}
	if err := parseSchemaV2(input, &result); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.Contact.FirstName != "Bob" {
		t.Fatalf("expected Bob, got %s", result.Contact.FirstName)
	}
}
