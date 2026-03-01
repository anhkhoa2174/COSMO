package v1

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// ContactResponse represents a contact payload returned by create/update endpoints.
type ContactResponse struct {
	ID                uuid.UUID              `json:"id"`
	UserID            uuid.UUID              `json:"user_id"`
	OrganizationID    *uuid.UUID             `json:"organization_id,omitempty"`
	SourceID          *string                `json:"source_id,omitempty"`
	Source            *string                `json:"source,omitempty"`
	HubspotID         *string                `json:"hubspot_id,omitempty"`
	Name              *string                `json:"name,omitempty"`
	Email             *string                `json:"email,omitempty"`
	Phone             *string                `json:"phone,omitempty"`
	Company           *string                `json:"company,omitempty"`
	JobTitle          *string                `json:"job_title,omitempty"`
	Address           *string                `json:"address,omitempty"`
	City              *string                `json:"city,omitempty"`
	Country           *string                `json:"country,omitempty"`
	State             *string                `json:"state,omitempty"`
	Zip               *string                `json:"zip,omitempty"`
	IsDeleted         bool                   `json:"is_deleted,omitempty"`
	CreatedAt         string                 `json:"created_at"`
	UpdatedAt         string                 `json:"updated_at"`
	Tags              map[string]interface{} `json:"tags,omitempty"`
	Profile           map[string]interface{} `json:"profile,omitempty"`
	ConfirmedFacts    map[string]interface{} `json:"confirmed_facts,omitempty"`
	AIInsights        map[string]interface{} `json:"ai_insights,omitempty"`
	InsightValidation map[string]interface{} `json:"insight_validation,omitempty"`
	Scores            map[string]interface{} `json:"scores,omitempty"`
	// Status fields
	Status             string   `json:"status"`                        // "ready" or "pending"
	MissingFields      []string `json:"missing_fields,omitempty"`      // list of missing required fields
	ContactInformation string   `json:"contact_information,omitempty"` // LinkedIn URL or email based on source

	// Outreach context fields
	Industry         string `json:"industry,omitempty"`          // e.g., Fintech, SaaS
	ContactChannel   string `json:"contact_channel,omitempty"`   // e.g., LinkedIn, Email
	LifecycleStage   string `json:"lifecycle_stage,omitempty"`   // new, contacted, replied, etc.
	ContextLevel     string `json:"context_level,omitempty"`     // LOW, MEDIUM, HIGH
	OutreachDecision string `json:"outreach_decision,omitempty"` // INTRO, FOLLOW-UP, NURTURE, HOLD
	Scenario         string `json:"scenario,omitempty"`          // Role-based, Post-reply, etc.
	MessageDraft     string `json:"message_draft,omitempty"`     // Draft message for outreach
	LastOutcome      string `json:"last_outcome,omitempty"`      // Result of last outreach
	NextStep         string `json:"next_step,omitempty"`         // SEND, FOLLOW_UP, SET_MEETING, WAIT, DROP
	Meeting          string `json:"meeting,omitempty"`           // Meeting details if scheduled
	BusinessStage    string `json:"business_stage,omitempty"`    // PRE_SALES, SALES, POST_SALES
}

// ContactListItem mirrors the legacy Python response shape for contact listings.
type ContactListItem struct {
	Entity map[string]interface{} `json:"entity"`
}

// ToContactResponse converts a domain contact to a response payload mirroring the Python API.
func ToContactResponse(contact *domain.Contact) *ContactResponse {
	profile := make(map[string]interface{})
	if len(contact.Profile) > 0 {
		if err := json.Unmarshal(contact.Profile, &profile); err != nil {
			profile = map[string]interface{}{"_unmarshal_error": err.Error()}
		}
	}

	tags := make(map[string]interface{})
	if len(contact.Tags) > 0 {
		if err := json.Unmarshal(contact.Tags, &tags); err != nil {
			tags = map[string]interface{}{"_unmarshal_error": err.Error()}
		}
	}

	confirmedFacts := make(map[string]interface{})
	if len(contact.ConfirmedFacts) > 0 {
		if err := json.Unmarshal(contact.ConfirmedFacts, &confirmedFacts); err != nil {
			confirmedFacts = map[string]interface{}{"_unmarshal_error": err.Error()}
		}
	}

	aiInsights := make(map[string]interface{})
	if len(contact.AIInsights) > 0 {
		if err := json.Unmarshal(contact.AIInsights, &aiInsights); err != nil {
			aiInsights = map[string]interface{}{"_unmarshal_error": err.Error()}
		}
	}

	insightValidation := make(map[string]interface{})
	if len(contact.InsightValidation) > 0 {
		if err := json.Unmarshal(contact.InsightValidation, &insightValidation); err != nil {
			insightValidation = map[string]interface{}{"_unmarshal_error": err.Error()}
		}
	}

	scores := make(map[string]interface{})
	if len(contact.Scores) > 0 {
		if err := json.Unmarshal(contact.Scores, &scores); err != nil {
			scores = map[string]interface{}{"_unmarshal_error": err.Error()}
		}
	}

	source := contact.Source
	sourceID := contact.SourceID

	response := &ContactResponse{
		ID:                contact.ID,
		UserID:            contact.UserID,
		IsDeleted:         contact.IsDeleted,
		CreatedAt:         contact.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:         contact.UpdatedAt.Format(time.RFC3339Nano),
		Tags:              tags,
		Profile:           profile,
		ConfirmedFacts:    confirmedFacts,
		AIInsights:        aiInsights,
		InsightValidation: insightValidation,
		Scores:            scores,
	}

	if contact.OrganizationID != nil {
		response.OrganizationID = contact.OrganizationID
	}
	if source != "" {
		response.Source = &source
	}
	if sourceID != "" {
		response.SourceID = &sourceID
	}
	if contact.HubspotID != nil {
		response.HubspotID = contact.HubspotID
	}
	if contact.Name != "" {
		response.Name = &contact.Name
	}
	// Email and Phone are now stored in profile JSONB
	if email, ok := profile["email"].(string); ok && email != "" {
		response.Email = &email
	}
	if phone, ok := profile["phone"].(string); ok && phone != "" {
		response.Phone = &phone
	}
	if contact.Company != "" {
		response.Company = &contact.Company
	}
	if contact.JobTitle != "" {
		response.JobTitle = &contact.JobTitle
	}
	if contact.Address != "" {
		response.Address = &contact.Address
	}
	if contact.City != "" {
		response.City = &contact.City
	}
	if contact.Country != "" {
		response.Country = &contact.Country
	}
	if contact.State != "" {
		response.State = &contact.State
	}
	if contact.Zip != "" {
		response.Zip = &contact.Zip
	}

	// Set status (default to pending if empty)
	if contact.Status != "" {
		response.Status = contact.Status
	} else {
		response.Status = "pending"
	}

	// Parse missing_fields from JSONB
	if len(contact.MissingFields) > 0 {
		var missingFields []string
		if err := json.Unmarshal(contact.MissingFields, &missingFields); err == nil {
			response.MissingFields = missingFields
		}
	}

	// Set contact_information
	response.ContactInformation = contact.ContactInformation

	// Set outreach context fields
	response.Industry = contact.Industry
	response.ContactChannel = contact.ContactChannel
	response.LifecycleStage = contact.LifecycleStage
	response.ContextLevel = contact.ContextLevel
	response.OutreachDecision = contact.OutreachDecision
	response.Scenario = contact.Scenario
	response.MessageDraft = contact.MessageDraft
	response.LastOutcome = contact.LastOutcome
	response.NextStep = contact.NextStep
	response.Meeting = contact.Meeting
	response.BusinessStage = contact.BusinessStage

	return response
}

// ToContactListItem converts a domain contact to the legacy listing entity format.
func ToContactListItem(contact *domain.Contact) *ContactListItem {
	entity := map[string]interface{}{
		"id":                 contact.ID,
		"user_id":            contact.UserID,
		"organization_id":    contact.OrganizationID,
		"created_at":         contact.CreatedAt,
		"updated_at":         contact.UpdatedAt,
		"inbound_lead_forms": []map[string]interface{}{},
	}

	if contact.SourceID != "" {
		entity["source_id"] = contact.SourceID
	} else {
		entity["source_id"] = nil
	}

	if contact.Source != "" {
		entity["source"] = contact.Source
	} else {
		entity["source"] = nil
	}

	if contact.HubspotID != nil {
		entity["hubspot_id"] = contact.HubspotID
	} else {
		entity["hubspot_id"] = nil
	}

	if contact.Name != "" && contact.Name != domain.NOT_AVAILABLE {
		entity["name"] = contact.Name
	} else {
		entity["name"] = nil
	}

	// Email and Phone are now stored in profile JSONB - will be populated when profile is spread
	entity["email"] = nil
	entity["phone"] = nil

	if contact.Company != "" {
		entity["company"] = contact.Company
	} else {
		entity["company"] = nil
	}
	if contact.JobTitle != "" {
		entity["job_title"] = contact.JobTitle
	} else {
		entity["job_title"] = nil
	}
	if contact.Address != "" {
		entity["address"] = contact.Address
	} else {
		entity["address"] = nil
	}
	if contact.City != "" {
		entity["city"] = contact.City
	} else {
		entity["city"] = nil
	}
	if contact.Country != "" {
		entity["country"] = contact.Country
	} else {
		entity["country"] = nil
	}
	if contact.State != "" {
		entity["state"] = contact.State
	} else {
		entity["state"] = nil
	}
	if contact.Zip != "" {
		entity["zip"] = contact.Zip
	} else {
		entity["zip"] = nil
	}

	if len(contact.Profile) > 0 {
		var profile map[string]interface{}
		if err := json.Unmarshal(contact.Profile, &profile); err == nil {
			for k, v := range profile {
				entity[k] = v
			}
		} else {
			entity["profile_unmarshal_error"] = err.Error()
		}
	}

	if len(contact.Tags) > 0 {
		var tags interface{}
		if err := json.Unmarshal(contact.Tags, &tags); err == nil {
			entity["tags"] = tags
		} else {
			entity["tags_unmarshal_error"] = err.Error()
		}
	}

	// Add status fields
	if contact.Status != "" {
		entity["status"] = contact.Status
	} else {
		entity["status"] = "pending"
	}

	if len(contact.MissingFields) > 0 {
		var missingFields []string
		if err := json.Unmarshal(contact.MissingFields, &missingFields); err == nil {
			entity["missing_fields"] = missingFields
		}
	} else {
		entity["missing_fields"] = []string{}
	}

	// Add contact_information
	entity["contact_information"] = contact.ContactInformation

	// Add outreach context fields
	entity["industry"] = contact.Industry
	entity["contact_channel"] = contact.ContactChannel
	entity["lifecycle_stage"] = contact.LifecycleStage
	entity["context_level"] = contact.ContextLevel
	entity["outreach_decision"] = contact.OutreachDecision
	entity["scenario"] = contact.Scenario
	entity["message_draft"] = contact.MessageDraft
	entity["last_outcome"] = contact.LastOutcome
	entity["next_step"] = contact.NextStep
	entity["meeting"] = contact.Meeting
	entity["business_stage"] = contact.BusinessStage

	return &ContactListItem{Entity: entity}
}

// ContactEnrichmentRequest controls enrichment behavior.
type ContactEnrichmentRequest struct {
	ForceRefresh bool `json:"force_refresh"`
}

// ContactEnrichmentResponse reports enrichment output.
type ContactEnrichmentResponse struct {
	ContactID         uuid.UUID              `json:"contact_id"`
	InsightsGenerated int                    `json:"insights_generated"`
	EmbeddingCreated  bool                   `json:"embedding_created"`
	ConfidenceAvg     float64                `json:"confidence_avg"`
	AIInsights        map[string]interface{} `json:"ai_insights,omitempty"`
}

// CalculateScoresRequest allows optional segmentation filtering.
type CalculateScoresRequest struct {
	SegmentationIDs []uuid.UUID `json:"segmentation_ids"`
}

// SegmentScoreResult describes a single segment fit result.
type SegmentScoreResult struct {
	SegmentationID   uuid.UUID      `json:"segmentation_id"`
	SegmentationName string         `json:"segmentation_name"`
	FitScore         int            `json:"fit_score"`
	ScoreBreakdown   map[string]int `json:"score_breakdown"`
	PassesFilters    bool           `json:"passes_filters"`
}

// CalculateScoresResponse summarizes scoring runs.
type CalculateScoresResponse struct {
	ContactID         uuid.UUID            `json:"contact_id"`
	SegmentsEvaluated int                  `json:"segments_evaluated"`
	SegmentsMatched   int                  `json:"segments_matched"`
	PriorityScore     int                  `json:"priority_score"`
	Scores            []SegmentScoreResult `json:"scores"`
}

// CreateContactRequest represents create contact request
// Supports extra fields for custom fields (model_config = {"extra": "allow"})
type CreateContactRequest struct {
	Name              string                 `json:"name" validate:"required"`
	Email             string                 `json:"email" validate:"omitempty,email"`
	Phone             string                 `json:"phone"`
	Company           string                 `json:"company"`
	JobTitle          string                 `json:"job_title"`
	Address           string                 `json:"address"`
	City              string                 `json:"city"`
	Country           string                 `json:"country"`
	State             string                 `json:"state"`
	Zip               string                 `json:"zip"`
	Profile           map[string]interface{} `json:"profile"`
	DoNotContact      bool                   `json:"do_not_contact"`
	OrganizationID    *uuid.UUID             `json:"organization_id"`
	Tags              map[string]interface{} `json:"tags"`
	ConfirmedFacts    map[string]interface{} `json:"confirmed_facts"`
	AIInsights        map[string]interface{} `json:"ai_insights"`
	InsightValidation map[string]interface{} `json:"insight_validation"`
	Scores            map[string]interface{} `json:"scores"`

	// Outreach context fields
	Industry         string `json:"industry"`
	ContactChannel   string `json:"contact_channel"`
	LifecycleStage   string `json:"lifecycle_stage"`
	ContextLevel     string `json:"context_level"`
	OutreachDecision string `json:"outreach_decision"`
	Scenario         string `json:"scenario"`
	MessageDraft     string `json:"message_draft"`
	LastOutcome      string `json:"last_outcome"`
	NextStep         string `json:"next_step"`
	Meeting          string `json:"meeting"`
	BusinessStage    string `json:"business_stage"`

	// Source tracking fields
	Source      string `json:"source"`       // e.g., "linkedin_extension", "apollo", "csv"
	LinkedInURL string `json:"linkedin_url"` // LinkedIn profile URL

	// ExtraFields captures any additional fields for custom field support
	ExtraFields map[string]interface{} `json:"-"`
}

// UnmarshalJSON custom unmarshaler to capture extra fields for custom field support
func (c *CreateContactRequest) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map to capture all fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Define known fields
	knownFields := map[string]bool{
		"name": true, "email": true, "phone": true, "company": true, "job_title": true,
		"address": true, "city": true, "country": true, "state": true, "zip": true,
		"profile": true, "do_not_contact": true, "organization_id": true, "tags": true,
		"confirmed_facts": true, "ai_insights": true, "insight_validation": true, "scores": true,
		// Outreach context fields
		"industry": true, "contact_channel": true, "lifecycle_stage": true, "context_level": true,
		"outreach_decision": true, "scenario": true, "message_draft": true, "last_outcome": true,
		"next_step": true, "meeting": true, "business_stage": true,
		// Source tracking fields
		"source": true, "linkedin_url": true,
	}

	// Extract extra fields
	c.ExtraFields = make(map[string]interface{})
	for key, value := range raw {
		if !knownFields[key] {
			c.ExtraFields[key] = value
		}
	}

	// Unmarshal into the struct normally (using a type alias to avoid recursion)
	type Alias CreateContactRequest
	aux := &struct{ *Alias }{Alias: (*Alias)(c)}
	return json.Unmarshal(data, aux)
}

// UpdateContactRequest represents update contact request
// Supports extra fields for custom fields (model_config = {"extra": "allow"})
type UpdateContactRequest struct {
	Name              *string                `json:"name,omitempty"`
	Email             *string                `json:"email,omitempty" validate:"omitempty,email"`
	Phone             *string                `json:"phone,omitempty"`
	Company           *string                `json:"company,omitempty"`
	JobTitle          *string                `json:"job_title,omitempty"`
	Address           *string                `json:"address,omitempty"`
	City              *string                `json:"city,omitempty"`
	Country           *string                `json:"country,omitempty"`
	State             *string                `json:"state,omitempty"`
	Zip               *string                `json:"zip,omitempty"`
	Profile           map[string]interface{} `json:"profile,omitempty"`
	DoNotContact      *bool                  `json:"do_not_contact,omitempty"`
	OrganizationID    *uuid.UUID             `json:"organization_id,omitempty"`
	Tags              map[string]interface{} `json:"tags,omitempty"`
	ConfirmedFacts    map[string]interface{} `json:"confirmed_facts,omitempty"`
	AIInsights        map[string]interface{} `json:"ai_insights,omitempty"`
	InsightValidation map[string]interface{} `json:"insight_validation,omitempty"`
	Scores            map[string]interface{} `json:"scores,omitempty"`

	// New system fields for status calculation
	Source             *string `json:"source,omitempty"`
	ContactInformation *string `json:"contact_information,omitempty"`

	// Outreach context fields
	Industry         *string `json:"industry,omitempty"`
	ContactChannel   *string `json:"contact_channel,omitempty"`
	LifecycleStage   *string `json:"lifecycle_stage,omitempty"`
	ContextLevel     *string `json:"context_level,omitempty"`
	OutreachDecision *string `json:"outreach_decision,omitempty"`
	Scenario         *string `json:"scenario,omitempty"`
	MessageDraft     *string `json:"message_draft,omitempty"`
	LastOutcome      *string `json:"last_outcome,omitempty"`
	NextStep         *string `json:"next_step,omitempty"`
	Meeting          *string `json:"meeting,omitempty"`
	BusinessStage    *string `json:"business_stage,omitempty"`

	// ExtraFields captures any additional fields for custom field support
	ExtraFields map[string]interface{} `json:"-"`
}

// UnmarshalJSON custom unmarshaler to capture extra fields for custom field support
func (u *UpdateContactRequest) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map to capture all fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Define known fields
	knownFields := map[string]bool{
		"name": true, "email": true, "phone": true,
		"company": true, "job_title": true, "address": true, "city": true,
		"country": true, "state": true, "zip": true, "profile": true,
		"do_not_contact": true, "organization_id": true, "tags": true,
		"confirmed_facts": true, "ai_insights": true, "insight_validation": true, "scores": true,
		// New system fields for status calculation
		"source": true, "contact_information": true,
		// Outreach context fields
		"industry": true, "contact_channel": true, "lifecycle_stage": true, "context_level": true,
		"outreach_decision": true, "scenario": true, "message_draft": true, "last_outcome": true,
		"next_step": true, "meeting": true, "business_stage": true,
	}

	// Extract extra fields
	u.ExtraFields = make(map[string]interface{})
	for key, value := range raw {
		if !knownFields[key] {
			u.ExtraFields[key] = value
		}
	}

	// Unmarshal into the struct normally (using a type alias to avoid recursion)
	type Alias UpdateContactRequest
	aux := &struct{ *Alias }{Alias: (*Alias)(u)}
	return json.Unmarshal(data, aux)
}

// BatchCreateContactsRequest represents batch create contacts request
type BatchCreateContactsRequest struct {
	Contacts []CreateContactRequest `json:"contacts" validate:"required,min=1,max=1000,dive"`
}

// BatchUpdateContactsRequest represents batch update contacts request
type BatchUpdateContactsRequest struct {
	Updates []ContactUpdate `json:"updates" validate:"required,min=1,max=1000,dive"`
}

// ContactUpdate represents a single contact update in batch
type ContactUpdate struct {
	ID      uuid.UUID            `json:"id" validate:"required"`
	Updates UpdateContactRequest `json:"updates" validate:"required"`
}

// BatchResponse represents batch operation response
type BatchResponse struct {
	Success int      `json:"success"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}

// ListContactsRequest represents list contacts request with filters
type ListContactsRequest struct {
	Offset    int                    `query:"offset"`
	Limit     int                    `query:"limit"`
	Filter    map[string]interface{} `json:"filter"`
	SortBy    string                 `query:"sort_by"`
	SortOrder string                 `query:"sort_order"`
}

// ContactListResponse represents a paginated list of contacts
type ContactListResponse struct {
	Items      []ContactResponse `json:"items"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

// ContactDeleteRequest represents delete contacts request
type ContactDeleteRequest struct {
	IDs []string `json:"ids" validate:"required,min=1"`
}

// ContactFieldValuesRequest represents field values request (query params)
type ContactFieldValuesRequest struct {
	Fields []string `query:"fields" validate:"required,min=1"`
	Offset int      `query:"offset"`
	Limit  int      `query:"limit"`
}

// FieldValueItem represents a single field's distinct values
type FieldValueItem struct {
	List  []interface{} `json:"list"`
	Total int64         `json:"total"`
}

// ContactFieldValuesResponse represents field values response
type ContactFieldValuesResponse map[string]FieldValueItem

// ContactImportCSVResponse represents CSV import response
type ContactImportCSVResponse struct {
	Message string `json:"message"`
}

// ContactSearchRequest represents contact search payload supporting legacy filter keys.
type ContactSearchRequest struct {
	Filter       map[string]interface{} `json:"filter,omitempty"`
	LegacyFilter map[string]interface{} `json:"filter_,omitempty"`
}

// ContactSearchResponse mirrors the legacy Listing response for contact search.
type ContactSearchResponse = schema.PaginatedResponse[*ContactListItem]

// ContactImportHubspotMapping represents HubSpot field mapping
type ContactImportHubspotMapping struct {
	Name     *string `json:"name,omitempty"`
	Email    *string `json:"email,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Company  *string `json:"company,omitempty"`
	JobTitle *string `json:"job_title,omitempty"`
	Address  *string `json:"address,omitempty"`
	City     *string `json:"city,omitempty"`
	Country  *string `json:"country,omitempty"`
	State    *string `json:"state,omitempty"`
	Zip      *string `json:"zip,omitempty"`
	// Support custom fields via map
	CustomFields map[string]string `json:"-"`
}

// ContactImportHubspotRequest represents HubSpot import request
type ContactImportHubspotRequest struct {
	MappingFields ContactImportHubspotMapping `json:"mapping_fields" validate:"required"`
	ListIDs       []string                    `json:"list_ids,omitempty"`
}

// ContactImportHubspotResponse represents HubSpot import response (operation)
type ContactImportHubspotResponse struct {
	ID        uuid.UUID              `json:"id"`
	Name      string                 `json:"name"`
	Status    string                 `json:"status"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Output    interface{}            `json:"output,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// AddResearchFindingRequest represents a research finding to be added to a contact
type AddResearchFindingRequest struct {
	Category     string  `json:"category" validate:"required"`   // Company Intelligence, Technical Stack, etc.
	FieldName    string  `json:"field_name" validate:"required"` // Company Size, Tech Stack, etc.
	Value        string  `json:"value" validate:"required"`      // The actual data found
	Source       *string `json:"source,omitempty"`               // Where the data was found (LinkedIn, Crunchbase, etc.)
	Priority     string  `json:"priority,omitempty"`             // High, Medium, Low
	WhyImportant *string `json:"why_important,omitempty"`        // Why this matters for the deal
}
