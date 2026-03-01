package location

import (
	"strings"
)

// LocationMapping represents a complete location with normalized values
type LocationMapping struct {
	City    string
	State   string
	Country string
}

// CityAbbreviations maps common city abbreviations to their full names and locations
var cityAbbreviations = map[string]LocationMapping{
	// Vietnam
	"hcm":     {City: "Ho Chi Minh City", State: "", Country: "Vietnam"},
	"hcmc":    {City: "Ho Chi Minh City", State: "", Country: "Vietnam"},
	"saigon":  {City: "Ho Chi Minh City", State: "", Country: "Vietnam"},
	"hanoi":   {City: "Hanoi", State: "", Country: "Vietnam"},
	"hn":      {City: "Hanoi", State: "", Country: "Vietnam"},
	"da nang": {City: "Da Nang", State: "", Country: "Vietnam"},
	"danang":  {City: "Da Nang", State: "", Country: "Vietnam"},

	// United States
	"nyc":     {City: "New York", State: "New York", Country: "United States"},
	"ny":      {City: "New York", State: "New York", Country: "United States"},
	"la":      {City: "Los Angeles", State: "California", Country: "United States"},
	"sf":      {City: "San Francisco", State: "California", Country: "United States"},
	"sfo":     {City: "San Francisco", State: "California", Country: "United States"},
	"chi":     {City: "Chicago", State: "Illinois", Country: "United States"},
	"seattle": {City: "Seattle", State: "Washington", Country: "United States"},
	"boston":  {City: "Boston", State: "Massachusetts", Country: "United States"},
	"dc":      {City: "Washington", State: "District of Columbia", Country: "United States"},

	// United Kingdom
	"london": {City: "London", State: "", Country: "United Kingdom"},
	"ldn":    {City: "London", State: "", Country: "United Kingdom"},

	// Singapore
	"sg":        {City: "Singapore", State: "", Country: "Singapore"},
	"singapore": {City: "Singapore", State: "", Country: "Singapore"},
}

// CityToCountry maps full city names to their countries
var cityToCountry = map[string]string{
	"ho chi minh city": "Vietnam",
	"hanoi":            "Vietnam",
	"da nang":          "Vietnam",
	"new york":         "United States",
	"los angeles":      "United States",
	"san francisco":    "United States",
	"chicago":          "United States",
	"seattle":          "United States",
	"boston":           "United States",
	"washington":       "United States",
	"london":           "United Kingdom",
	"singapore":        "Singapore",
}

// NormalizeLocation takes a location string and returns normalized city, state, country
func NormalizeLocation(input string) *LocationMapping {
	if input == "" {
		return nil
	}

	// Normalize input: lowercase and trim
	normalized := strings.ToLower(strings.TrimSpace(input))

	// Check if it's a known abbreviation
	if mapping, ok := cityAbbreviations[normalized]; ok {
		return &mapping
	}

	// Check if it's a known city (for country inference)
	if country, ok := cityToCountry[normalized]; ok {
		return &LocationMapping{
			City:    capitalizeWords(input),
			State:   "",
			Country: country,
		}
	}

	// Return as-is if no mapping found
	return &LocationMapping{
		City:    capitalizeWords(input),
		State:   "",
		Country: "",
	}
}

// InferCountryFromCity attempts to infer country from a city name
func InferCountryFromCity(city string) string {
	if city == "" {
		return ""
	}

	normalized := strings.ToLower(strings.TrimSpace(city))

	// Check direct mapping
	if country, ok := cityToCountry[normalized]; ok {
		return country
	}

	// Check if city is actually an abbreviation
	if mapping, ok := cityAbbreviations[normalized]; ok {
		return mapping.Country
	}

	return ""
}

// capitalizeWords capitalizes the first letter of each word
func capitalizeWords(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// ParseLocationString attempts to parse a location string that might contain city, state, country
// Examples: "HCM, Vietnam", "San Francisco, CA, USA", "London"
func ParseLocationString(input string) *LocationMapping {
	if input == "" {
		return nil
	}

	parts := strings.Split(input, ",")
	mapping := &LocationMapping{}

	switch len(parts) {
	case 1:
		// Just city or abbreviation
		normalized := NormalizeLocation(strings.TrimSpace(parts[0]))
		if normalized != nil {
			return normalized
		}
	case 2:
		// City, Country or City, State
		city := strings.TrimSpace(parts[0])
		second := strings.TrimSpace(parts[1])

		// Normalize city first
		cityMapping := NormalizeLocation(city)
		if cityMapping != nil {
			mapping.City = cityMapping.City

			// If second part looks like a country, use it
			if len(second) > 2 {
				mapping.Country = capitalizeWords(second)
			} else {
				// Might be a state
				mapping.State = strings.ToUpper(second)
			}

			// If we inferred a country from city, keep it unless explicitly overridden
			if mapping.Country == "" && cityMapping.Country != "" {
				mapping.Country = cityMapping.Country
			}
		}
	case 3:
		// City, State, Country
		city := strings.TrimSpace(parts[0])
		state := strings.TrimSpace(parts[1])
		country := strings.TrimSpace(parts[2])

		cityMapping := NormalizeLocation(city)
		if cityMapping != nil {
			mapping.City = cityMapping.City
			mapping.State = state
			mapping.Country = capitalizeWords(country)
		}
	}

	return mapping
}
