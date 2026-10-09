package scraper

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
)

// Service handles URL scraping and AI-powered data extraction
type Service struct {
	openaiClient *ai.OpenAIClient
	httpClient   *http.Client
}

// NewService creates a new scraper service
func NewService(openaiClient *ai.OpenAIClient) *Service {
	return &Service{
		openaiClient: openaiClient,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ExtractedData represents data extracted from a URL
type ExtractedData struct {
	RawHTML   string                 `json:"raw_html"`
	RawText   string                 `json:"raw_text"`
	Fields    map[string]interface{} `json:"fields"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
}

// ExtractFromURL fetches a URL and uses AI to extract structured contact data
func (s *Service) ExtractFromURL(ctx context.Context, url string) (*ExtractedData, error) {
	log.Info().Str("url", url).Msg("Starting URL extraction")

	// Step 1: Fetch URL content
	html, text, err := s.fetchURL(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}

	// Step 2: Allow empty/short content but keep guard rails in prompt.
	// If empty, we still proceed so AI returns null/empty fields (no hallucination).
	if strings.TrimSpace(text) == "" {
		text = "[CONTENT_UNAVAILABLE]"
	}

	// Step 3: Use OpenAI to extract structured fields (with strict “no guess” rules)
	fields, err := s.extractWithAI(ctx, url, text)
	if err != nil {
		return nil, fmt.Errorf("failed to extract with AI: %w", err)
	}

	return &ExtractedData{
		RawHTML:   html,
		RawText:   text,
		Fields:    fields,
		Source:    url,
		Timestamp: time.Now().UTC(),
	}, nil
}

// ExtractFromImage uses OpenAI vision to extract structured contact data from a screenshot.
func (s *Service) ExtractFromImage(ctx context.Context, imageBytes []byte, contentType string) (*ExtractedData, error) {
	if s.openaiClient == nil {
		return nil, fmt.Errorf("openai client not configured")
	}

	if len(imageBytes) == 0 {
		return nil, fmt.Errorf("image is empty")
	}

	if contentType == "" {
		contentType = http.DetectContentType(imageBytes)
	}

	// Build data URL for OpenAI vision
	encoded := base64.StdEncoding.EncodeToString(imageBytes)
	dataURL := fmt.Sprintf("data:%s;base64,%s", contentType, encoded)

	prompt := `Extract structured contact information from this screenshot.

Only use information visible in the image. If a field is not visible, set it to null or an empty array/object.
Do NOT guess or infer any missing details.

Extract the following fields if available:
- name (full name as a single field)
- job_title
- company
- location (city, state, country)
- bio / summary
- skills (array)
- education (array of objects with school, degree, field, years)
- experience (array of objects with company, title, duration, description)
- social_links (object with linkedin, twitter, github, website, etc.)
- interests (array)
- languages (array)
- certifications (array)
- any other relevant professional information

Return ONLY valid JSON. No markdown, no explanations.`

	parts := []openai.ChatMessagePart{
		{
			Type: openai.ChatMessagePartTypeText,
			Text: prompt,
		},
		{
			Type: openai.ChatMessagePartTypeImageURL,
			ImageURL: &openai.ChatMessageImageURL{
				URL:    dataURL,
				Detail: openai.ImageURLDetailHigh,
			},
		},
	}

	content, err := s.openaiClient.ChatCompletionWithParts(
		ctx,
		"You are a professional data extraction assistant. Extract structured contact information from images and return only valid JSON.",
		parts,
		0.1,
		2000,
	)
	if err != nil {
		return nil, fmt.Errorf("openai vision error: %w", err)
	}

	var fields map[string]interface{}
	if err := json.Unmarshal([]byte(content), &fields); err != nil {
		log.Error().
			Err(err).
			Str("response", content).
			Msg("Failed to parse vision extraction response as JSON")
		return nil, fmt.Errorf("failed to parse vision response: %w", err)
	}

	log.Info().
		Int("fields_extracted", len(fields)).
		Msg("Successfully extracted data from image")

	return &ExtractedData{
		RawHTML:   "",
		RawText:   "",
		Fields:    fields,
		Source:    "image",
		Timestamp: time.Now().UTC(),
	}, nil
}

// ExtractFromText uses OpenAI to extract structured contact data from raw text.
func (s *Service) ExtractFromText(ctx context.Context, text string) (*ExtractedData, error) {
	if s.openaiClient == nil {
		return nil, fmt.Errorf("openai client not configured")
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("text is empty")
	}

	fields, err := s.extractWithAI(ctx, "extension", text)
	if err != nil {
		return nil, err
	}

	return &ExtractedData{
		RawHTML:   "",
		RawText:   text,
		Fields:    fields,
		Source:    "extension",
		Timestamp: time.Now().UTC(),
	}, nil
}

// fetchURL retrieves HTML and text content from a URL via direct GET
func (s *Service) fetchURL(ctx context.Context, url string) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; CosmoBot/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	text := string(bodyBytes)
	log.Info().
		Str("url", url).
		Int("content_length", len(text)).
		Msg("Successfully fetched URL")

	return text, text, nil
}

// extractWithAI uses OpenAI to extract structured data from text
func (s *Service) extractWithAI(ctx context.Context, url string, text string) (map[string]interface{}, error) {
	// Truncate text if too long (GPT-4 has token limits)
	maxChars := 12000
	if len(text) > maxChars {
		text = text[:maxChars] + "... [truncated]"
	}

	prompt := fmt.Sprintf(`Extract structured contact information from the following profile page.

URL: %s

Page Content:
%s

Extract the following fields ONLY if they are explicitly present in the Page Content below (do not infer from the URL or general knowledge). If Page Content is "[CONTENT_UNAVAILABLE]" or empty, return null/empty fields.
- name (full name as a single field)
- job_title
- company
- location (city, state, country)
- bio / summary
- skills (array)
- education (array of objects with school, degree, field, years)
- experience (array of objects with company, title, duration, description)
- social_links (object with linkedin, twitter, github, website, etc.)
- interests (array)
- languages (array)
- certifications (array)
- any other relevant professional information

Rules:
- If a field is not explicitly present in Page Content, set it to null or an empty array/object.
- If Page Content is missing or very short, return all fields as null/empty (do NOT guess).
- Do NOT invent or infer companies, schools, titles, or dates. Copy exact strings from the page.
- Do NOT use any external knowledge beyond Page Content. If the page is sparse, leave fields empty/null.
- Return ONLY a valid JSON object with the extracted fields. No markdown, no explanations.

Example output:
{
  "name": "John Doe",
  "job_title": "Senior Software Engineer",
  "company": "TechCorp",
  "location": "San Francisco, CA, USA",
  "bio": "Passionate developer...",
  "skills": ["Python", "Go", "React"],
  "social_links": {
    "linkedin": "https://linkedin.com/in/johndoe",
    "github": "https://github.com/johndoe"
  }
}`, url, text)

	// Use the wrapper's ChatCompletion method
	content, err := s.openaiClient.ChatCompletion(ctx, ai.ChatCompletionRequest{
		SystemPrompt: "You are a professional data extraction assistant. Extract structured contact information from web pages and return only valid JSON.",
		Messages: []ai.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.1, // Low temperature for consistent extraction
		MaxTokens:   2000,
	})

	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	// Parse JSON response
	var fields map[string]interface{}
	if err := json.Unmarshal([]byte(content), &fields); err != nil {
		log.Error().
			Err(err).
			Str("response", content).
			Msg("Failed to parse AI extraction response as JSON")
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	log.Info().
		Int("fields_extracted", len(fields)).
		Msg("Successfully extracted data with AI")

	return fields, nil
}

// MergeIntoContactProfile merges extracted fields into contact's custom_fields
// and returns contact-level updates for standard fields.
func (s *Service) MergeIntoContactProfile(
	existingProfile map[string]interface{},
	extractedFields map[string]interface{},
	sourceURL string,
) (map[string]interface{}, map[string]interface{}, []string) {
	if existingProfile == nil {
		existingProfile = make(map[string]interface{})
	}

	// Ensure custom_fields exists
	customFields, ok := existingProfile["custom_fields"].(map[string]interface{})
	if !ok {
		customFields = make(map[string]interface{})
	}

	fieldsAdded := []string{}
	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Standard fields that should go to top level (not custom_fields)
	standardFields := map[string]bool{
		"name":      true,
		"job_title": true,
		"company":   true,
		"email":     true,
		"phone":     true,
	}
	systemFields := map[string]bool{
		"address":  true,
		"city":     true,
		"state":    true,
		"country":  true,
		"location": true,
	}
	aliasFields := map[string]bool{
		"full_name":            true,
		"first_name":           true, // Derived to name
		"last_name":            true, // Derived to name
		"title":                true,
		"headline":             true,
		"position":             true,
		"role":                 true,
		"current_company":      true,
		"current_company_name": true,
		"currentCompany":       true,
		"currentcompany":       true,
		"email_address":        true,
		"contact_email":        true,
		"phone_number":         true,
		"contact_phone":        true,
		"street_address":       true,
		"street":               true,
		"source_url":           true,
	}

	contactUpdates := make(map[string]interface{})

	// Derive system fields from known aliases without guessing.
	derived := deriveSystemFields(extractedFields)
	for k, v := range derived {
		if _, exists := extractedFields[k]; !exists {
			extractedFields[k] = v
		}
	}

	for fieldName, value := range extractedFields {
		if value == nil {
			continue // Skip null values
		}

		// Skip empty strings / empty collections to avoid blank custom fields
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) == "" {
				continue
			}
		case []interface{}:
			if len(v) == 0 {
				continue
			}
		case map[string]interface{}:
			if len(v) == 0 {
				continue
			}
		}

		// Handle name field - store directly in contact
		if fieldName == "name" {
			if strVal := strings.TrimSpace(fmt.Sprintf("%v", value)); strVal != "" {
				contactUpdates["name"] = strVal
				existingProfile["name"] = strVal
			}
			continue
		}

		if fieldName == "linkedin_url" {
			if strVal := strings.TrimSpace(fmt.Sprintf("%v", value)); strVal != "" {
				existingProfile["linkedin_url"] = strVal
			}
			continue
		}

		// Map structured location object to system fields
		if fieldName == "location" {
			if locMap, ok := value.(map[string]interface{}); ok {
				city := normalizeLocationValue(locMap["city"])
				state := normalizeLocationValue(locMap["state"])
				country := normalizeLocationValue(locMap["country"])

				city, state, country = normalizeLocation(city, state, country)

				if city != "" {
					contactUpdates["city"] = city
					existingProfile["city"] = city
				}
				if state != "" {
					contactUpdates["state"] = state
					existingProfile["state"] = state
				}
				if country != "" {
					contactUpdates["country"] = country
					existingProfile["country"] = country
				}
				// IMPORTANT: Don't save location object to custom_fields, we already split it
				continue
			}
			if locStr, ok := value.(string); ok {
				if applyLocationString(locStr, contactUpdates, existingProfile) {
					continue
				}
			}
		}

		// For standard fields, capture to contactUpdates (avoid overwriting with empty)
		if standardFields[fieldName] {
			if strVal := fmt.Sprintf("%v", value); strings.TrimSpace(strVal) != "" {
				contactUpdates[fieldName] = strVal
				existingProfile[fieldName] = strVal
			}
			continue
		}

		// Map other system fields
		if systemFields[fieldName] {
			strVal := strings.TrimSpace(fmt.Sprintf("%v", value))
			if strVal == "" {
				continue
			}
			switch fieldName {
			case "city":
				normalizedCity, _, normalizedCountry := normalizeLocation(strVal, "", "")
				if normalizedCity != "" {
					contactUpdates["city"] = normalizedCity
					existingProfile["city"] = normalizedCity
				}
				if normalizedCountry != "" && normalizeLocationValue(contactUpdates["country"]) == "" {
					contactUpdates["country"] = normalizedCountry
					existingProfile["country"] = normalizedCountry
				}
			case "country":
				contactUpdates["country"] = normalizeCountry(strVal)
				existingProfile["country"] = normalizeCountry(strVal)
			default:
				contactUpdates[fieldName] = strVal
				existingProfile[fieldName] = strVal
			}
			continue
		}
		if aliasFields[fieldName] {
			continue
		}

		// Add to custom_fields with metadata
		customFields[fieldName] = map[string]interface{}{
			"value":      value,
			"source":     sourceURL,
			"updated_at": timestamp,
			"extracted":  true, // Flag to indicate this was auto-extracted
		}

		fieldsAdded = append(fieldsAdded, fieldName)
	}

	existingProfile["custom_fields"] = customFields

	// Also track extraction history
	extractionHistory, ok := existingProfile["url_extraction_history"].([]interface{})
	if !ok {
		extractionHistory = []interface{}{}
	}

	extractionHistory = append(extractionHistory, map[string]interface{}{
		"url":           sourceURL,
		"extracted_at":  timestamp,
		"fields_added":  fieldsAdded,
		"extraction_id": uuid.New().String(),
	})

	existingProfile["url_extraction_history"] = extractionHistory

	// Fallback: derive company/job_title from experience[0] if not already set
	if _, hasCompany := contactUpdates["company"]; !hasCompany {
		if expRaw, ok := extractedFields["experience"]; ok {
			if expArr, ok := expRaw.([]interface{}); ok && len(expArr) > 0 {
				if firstExp, ok := expArr[0].(map[string]interface{}); ok {
					if companyVal, ok := firstExp["company"]; ok {
						if s := strings.TrimSpace(fmt.Sprintf("%v", companyVal)); s != "" {
							contactUpdates["company"] = s
							existingProfile["company"] = s
						}
					}
					if titleVal, ok := firstExp["title"]; ok {
						if s := strings.TrimSpace(fmt.Sprintf("%v", titleVal)); s != "" {
							if _, hasTitle := contactUpdates["job_title"]; !hasTitle {
								contactUpdates["job_title"] = s
								existingProfile["job_title"] = s
							}
						}
					}
				}
			} else if expMap, ok := expRaw.(map[string]interface{}); ok {
				if companyVal, ok := expMap["company"]; ok {
					if s := strings.TrimSpace(fmt.Sprintf("%v", companyVal)); s != "" {
						contactUpdates["company"] = s
						existingProfile["company"] = s
					}
				}
				if titleVal, ok := expMap["title"]; ok {
					if s := strings.TrimSpace(fmt.Sprintf("%v", titleVal)); s != "" {
						if _, hasTitle := contactUpdates["job_title"]; !hasTitle {
							contactUpdates["job_title"] = s
							existingProfile["job_title"] = s
						}
					}
				}
			}
		}
	}

	return existingProfile, contactUpdates, fieldsAdded
}

func normalizeLocationValue(value interface{}) string {
	if value == nil {
		return ""
	}
	val := strings.TrimSpace(fmt.Sprintf("%v", value))
	if val == "" || val == "<nil>" {
		return ""
	}
	return val
}

func applyLocationString(locStr string, contactUpdates map[string]interface{}, existingProfile map[string]interface{}) bool {
	locStr = strings.TrimSpace(locStr)
	if locStr == "" {
		return false
	}
	existingProfile["location"] = locStr
	parts := strings.Split(locStr, ",")
	if len(parts) >= 1 {
		city := strings.TrimSpace(parts[0])
		// Clean up common LinkedIn location suffixes
		city = strings.TrimSuffix(city, " Metropolitan Area")
		city = strings.TrimSuffix(city, " Metro Area")
		city = strings.TrimSuffix(city, " Area")
		city = strings.TrimSpace(city)

		if city != "" {
			normalizedCity, _, normalizedCountry := normalizeLocation(city, "", "")
			if normalizedCity != "" {
				contactUpdates["city"] = normalizedCity
				existingProfile["city"] = normalizedCity
			}
			if normalizedCountry != "" && normalizeLocationValue(contactUpdates["country"]) == "" {
				contactUpdates["country"] = normalizedCountry
				existingProfile["country"] = normalizedCountry
			}
		}
	}
	if len(parts) >= 2 {
		country := strings.TrimSpace(parts[len(parts)-1])
		if country != "" {
			_, _, normalizedCountry := normalizeLocation("", "", country)
			if normalizedCountry != "" {
				contactUpdates["country"] = normalizedCountry
				existingProfile["country"] = normalizedCountry
			}
		}
	}
	return normalizeLocationValue(contactUpdates["city"]) != "" || normalizeLocationValue(contactUpdates["country"]) != ""
}

func normalizeLocation(city, state, country string) (string, string, string) {
	city = strings.TrimSpace(city)
	state = strings.TrimSpace(state)
	country = strings.TrimSpace(country)

	if city != "" {
		if normalizedCity, inferredCountry := lookupCityAlias(city); normalizedCity != "" {
			city = normalizedCity
			if country == "" && inferredCountry != "" {
				country = inferredCountry
			}
		}
	}

	if country != "" {
		country = normalizeCountry(country)
	}

	return city, state, country
}

func normalizeCountry(country string) string {
	key := normalizeKey(country)
	switch key {
	case "vn", "vietnam":
		return "Vietnam"
	case "usa", "us", "unitedstates", "unitedstatesofamerica":
		return "United States"
	case "uk", "unitedkingdom", "greatbritain":
		return "United Kingdom"
	default:
		return country
	}
}

func lookupCityAlias(city string) (string, string) {
	key := normalizeKey(city)
	switch key {
	case "hcm", "hcmc", "hochiminh", "hochiminhcity", "tphcm", "tphpcm", "saigon", "saigoncity":
		return "Ho Chi Minh City", "Vietnam"
	case "hn", "hanoi", "hanoicity":
		return "Hanoi", "Vietnam"
	case "danang", "dadang":
		return "Da Nang", "Vietnam"
	case "nyc", "newyork", "newyorkcity":
		return "New York", "United States"
	case "sf", "sanfrancisco", "sanfranciscocity":
		return "San Francisco", "United States"
	case "la", "losangeles", "losangelescity":
		return "Los Angeles", "United States"
	default:
		return "", ""
	}
}

func normalizeKey(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func deriveSystemFields(extractedFields map[string]interface{}) map[string]interface{} {
	derived := map[string]interface{}{}

	// Derive name from aliases (full_name, first_name + last_name for legacy data)
	if extractedFields["name"] == nil {
		if name := getStringField(extractedFields, "full_name"); name != "" {
			derived["name"] = name
		} else {
			// Combine first_name and last_name if they exist (for legacy API responses)
			firstName := getStringField(extractedFields, "first_name")
			lastName := getStringField(extractedFields, "last_name")
			if firstName != "" || lastName != "" {
				derived["name"] = strings.TrimSpace(firstName + " " + lastName)
			}
		}
	}

	// Job title aliases
	if extractedFields["job_title"] == nil {
		if title := getStringField(extractedFields, "job_title", "title", "headline", "position", "role"); title != "" {
			derived["job_title"] = title
		}
	}

	// Company aliases
	if extractedFields["company"] == nil {
		if company := getStringField(extractedFields, "company", "current_company", "current_company_name", "currentCompany"); company != "" {
			derived["company"] = company
		}
		if companyMap := getMapField(extractedFields, "current_company", "currentCompany"); companyMap != nil {
			if name := getStringFromValue(companyMap["name"]); name != "" {
				derived["company"] = name
			}
		}
	}

	// Email/phone aliases
	if extractedFields["email"] == nil {
		if email := getStringField(extractedFields, "email", "email_address", "contact_email"); email != "" {
			derived["email"] = email
		}
	}
	if extractedFields["phone"] == nil {
		if phone := getStringField(extractedFields, "phone", "phone_number", "contact_phone"); phone != "" {
			derived["phone"] = phone
		}
	}

	// Address aliases
	if extractedFields["address"] == nil {
		if address := getStringField(extractedFields, "address", "street_address", "street"); address != "" {
			derived["address"] = address
		}
	}

	// LinkedIn URL aliases
	if extractedFields["linkedin_url"] == nil {
		if url := getStringField(extractedFields, "linkedin_url", "profile_url", "source_url"); url != "" {
			if strings.Contains(url, "linkedin.com/in/") {
				derived["linkedin_url"] = url
			}
		}
		if social, ok := extractedFields["social_links"].(map[string]interface{}); ok {
			if url := getStringFromValue(social["linkedin"]); url != "" {
				derived["linkedin_url"] = url
			}
		}
	}

	return derived
}

func getStringField(fields map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := fields[key]; ok {
			if str := getStringFromValue(value); str != "" {
				return str
			}
		}
	}
	return ""
}

func getMapField(fields map[string]interface{}, keys ...string) map[string]interface{} {
	for _, key := range keys {
		if value, ok := fields[key]; ok {
			if m, ok := value.(map[string]interface{}); ok {
				return m
			}
		}
	}
	return nil
}

func getStringFromValue(value interface{}) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprintf("%v", value))
}

