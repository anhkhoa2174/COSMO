package location

import (
	"testing"
)

func TestNormalizeLocation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *LocationMapping
	}{
		{
			name:  "HCM abbreviation",
			input: "HCM",
			expected: &LocationMapping{
				City:    "Ho Chi Minh City",
				State:   "",
				Country: "Vietnam",
			},
		},
		{
			name:  "hcm lowercase",
			input: "hcm",
			expected: &LocationMapping{
				City:    "Ho Chi Minh City",
				State:   "",
				Country: "Vietnam",
			},
		},
		{
			name:  "Saigon",
			input: "Saigon",
			expected: &LocationMapping{
				City:    "Ho Chi Minh City",
				State:   "",
				Country: "Vietnam",
			},
		},
		{
			name:  "NYC abbreviation",
			input: "NYC",
			expected: &LocationMapping{
				City:    "New York",
				State:   "New York",
				Country: "United States",
			},
		},
		{
			name:  "Full city name - Hanoi",
			input: "Hanoi",
			expected: &LocationMapping{
				City:    "Hanoi",
				State:   "",
				Country: "Vietnam",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeLocation(tt.input)
			if result == nil {
				t.Fatalf("expected result, got nil")
			}
			if result.City != tt.expected.City {
				t.Errorf("City: expected %s, got %s", tt.expected.City, result.City)
			}
			if result.State != tt.expected.State {
				t.Errorf("State: expected %s, got %s", tt.expected.State, result.State)
			}
			if result.Country != tt.expected.Country {
				t.Errorf("Country: expected %s, got %s", tt.expected.Country, result.Country)
			}
		})
	}
}

func TestInferCountryFromCity(t *testing.T) {
	tests := []struct {
		name     string
		city     string
		expected string
	}{
		{
			name:     "Ho Chi Minh City",
			city:     "Ho Chi Minh City",
			expected: "Vietnam",
		},
		{
			name:     "HCM abbreviation",
			city:     "HCM",
			expected: "Vietnam",
		},
		{
			name:     "New York",
			city:     "New York",
			expected: "United States",
		},
		{
			name:     "London",
			city:     "London",
			expected: "United Kingdom",
		},
		{
			name:     "Unknown city",
			city:     "Unknown City",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InferCountryFromCity(tt.city)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestParseLocationString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *LocationMapping
	}{
		{
			name:  "Just abbreviation",
			input: "HCM",
			expected: &LocationMapping{
				City:    "Ho Chi Minh City",
				State:   "",
				Country: "Vietnam",
			},
		},
		{
			name:  "City, Country format",
			input: "HCM, Vietnam",
			expected: &LocationMapping{
				City:    "Ho Chi Minh City",
				State:   "",
				Country: "Vietnam",
			},
		},
		{
			name:  "City, State, Country format",
			input: "San Francisco, CA, USA",
			expected: &LocationMapping{
				City:    "San Francisco",
				State:   "CA",
				Country: "Usa",
			},
		},
		{
			name:  "Full city name only",
			input: "Singapore",
			expected: &LocationMapping{
				City:    "Singapore",
				State:   "",
				Country: "Singapore",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseLocationString(tt.input)
			if result == nil {
				t.Fatalf("expected result, got nil")
			}
			if result.City != tt.expected.City {
				t.Errorf("City: expected %s, got %s", tt.expected.City, result.City)
			}
			if result.State != tt.expected.State {
				t.Errorf("State: expected %s, got %s", tt.expected.State, result.State)
			}
			if result.Country != tt.expected.Country {
				t.Errorf("Country: expected %s, got %s", tt.expected.Country, result.Country)
			}
		})
	}
}
