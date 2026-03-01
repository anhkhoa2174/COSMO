package v2

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// CampaignSearchRequest represents the request for POST /v2/campaigns/search
type CampaignSearchRequest struct {
	Filter map[string]interface{} `json:"filter"`
}

// RelationshipAgent represents an agent with minimal metadata
type RelationshipAgent struct {
	ID        uuid.UUID              `json:"id"`
	Name      string                 `json:"name"`
	CMetadata map[string]interface{} `json:"cmetadata"`
}

// CampaignEntity represents a campaign entity in search results
type CampaignEntity struct {
	ID             uuid.UUID          `json:"id"`
	UserID         uuid.UUID          `json:"user_id"`
	Playbook       *string            `json:"playbook"`
	Name           *string            `json:"name"`
	ListContactID  *uuid.UUID         `json:"list_contact_id"`
	OrganizationID *uuid.UUID         `json:"organization_id"`
	Schedule       *time.Time         `json:"schedule"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
	Status         *string            `json:"status"`
	Agent          *RelationshipAgent `json:"agent"`
}

// CampaignListItem represents a single campaign in search results
type CampaignListItem struct {
	Entity       CampaignEntity     `json:"entity"`
	Creator      *string            `json:"creator"`
	Agent        *RelationshipAgent `json:"agent"`
	Sent         int                `json:"sent"`
	Reply        int                `json:"reply"`
	ReplyRate    float64            `json:"reply_rate"`
	Interested   int                `json:"interested"`
	InterestRate float64            `json:"interest_rate"`
}

// CampaignSearchResponse represents the response for POST /v2/campaigns/search
type CampaignSearchResponse = schema.PaginatedResponse[CampaignListItem]

// CampaignGenerateTemplateRequest represents the request for POST /v2/campaigns/{campaign_id}/templates
type CampaignGenerateTemplateRequest struct {
	Prompt       *string                `json:"prompt"`
	DocumentGids []string               `json:"document_gids"`
	ContactData  map[string]interface{} `json:"contact_data"`
	Tone         *string                `json:"tone"`
}

// EmailTemplateStructure represents the generated email template structure
type EmailTemplateStructure struct {
	Category string `json:"category"`
	Type     string `json:"type"`
	Subject  string `json:"subject"`
	Content  string `json:"content"`
}

// CampaignGenerateTemplateResponse represents the response for POST /v2/campaigns/{campaign_id}/templates
type CampaignGenerateTemplateResponse struct {
	ID       uuid.UUID              `json:"id"`
	Template EmailTemplateStructure `json:"template"`
}

// CampaignRegenerateTemplateRequest represents the request for POST /v2/campaigns/{campaign_id}/templates/{template_id}
type CampaignRegenerateTemplateRequest struct {
	Prompt       string                 `json:"prompt"`
	ContactData  map[string]interface{} `json:"contact_data"`
	DocumentGids []string               `json:"document_gids"`
	Tone         *string                `json:"tone"`
}

// CampaignRegenerateTemplateResponse represents the response for POST /v2/campaigns/{campaign_id}/templates/{template_id}
type CampaignRegenerateTemplateResponse = CampaignGenerateTemplateResponse

// CampaignDraftTemplateDetail represents the template detail in draft template response
type CampaignDraftTemplateDetail struct {
	Type    string `json:"type"`
	Subject string `json:"subject"`
	Content string `json:"content"`
}

// CampaignDraftTemplateCreateResponse represents the response for GET /v2/campaign/{campaign_id}/draft-templates
type CampaignDraftTemplateCreateResponse struct {
	ID         uuid.UUID                   `json:"id"`
	Intent     domain.IntentType           `json:"intent"`
	CampaignID uuid.UUID                   `json:"campaign_id"`
	TemplateID uuid.UUID                   `json:"template_id"`
	CreatedAt  time.Time                   `json:"created_at"`
	UpdatedAt  time.Time                   `json:"updated_at"`
	Template   CampaignDraftTemplateDetail `json:"template"`
}

// CampaignGenerateSampleResponseRequest represents the request for POST /v2/campaign/{campaign_id}/generate-sample-response
type CampaignGenerateSampleResponseRequest struct {
	Intent           domain.IntentType      `json:"intent"`
	ContactData      map[string]interface{} `json:"contact_data"`
	OutreachTemplate string                 `json:"outreach_template"`
}

// CampaignGenerateSampleResponseResponse represents the response for POST /v2/campaign/{campaign_id}/generate-sample-response
type CampaignGenerateSampleResponseResponse struct {
	Intent   domain.IntentType `json:"intent"`
	Response string            `json:"response"`
}

// CampaignClassifySampleResponseRequest represents the request for POST /v2/campaign/{campaign_id}/classify-sample-response
type CampaignClassifySampleResponseRequest struct {
	Response string `json:"response"`
}

// HardCodeMail represents an email in conversation history
type HardCodeMail struct {
	ToEmail   string `json:"to_email"`
	Content   string `json:"content"`
	Status    string `json:"status"`
	FromEmail string `json:"from_email"`
	Subject   string `json:"subject"`
}

// CampaignGenerateReplyRequest represents the request for POST /v2/campaigns/{campaign_id}/generate-reply
type CampaignGenerateReplyRequest struct {
	Conversation []HardCodeMail     `json:"conversation"`
	Intent       *domain.IntentType `json:"intent"`
}

// Helper functions

// stringPtrOrNil returns nil if value is empty, otherwise returns pointer to value
func stringPtrOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// ToCampaignListItem converts domain campaign and stats to v2 listing item
func ToCampaignListItem(
	campaign *domain.Campaign,
	creator *string,
	sent, reply, interested int64,
	replyRate, interestRate float64,
) CampaignListItem {
	if campaign == nil {
		return CampaignListItem{}
	}

	entity := CampaignEntity{
		ID:             campaign.ID,
		UserID:         campaign.UserID,
		Playbook:       stringPtrOrNil(campaign.Playbook),
		Name:           stringPtrOrNil(campaign.Name),
		ListContactID:  campaign.ListContactID,
		OrganizationID: campaign.OrganizationID,
		Schedule:       campaign.Schedule,
		CreatedAt:      campaign.CreatedAt,
		UpdatedAt:      campaign.UpdatedAt,
	}

	// Convert CampaignStatus to string
	status := string(campaign.Status)
	entity.Status = stringPtrOrNil(status)

	// Note: Agent relationship removed to avoid circular imports
	// Agent data should be loaded separately using relations package if needed
	entity.Agent = nil

	return CampaignListItem{
		Entity:       entity,
		Creator:      creator,
		Agent:        entity.Agent,
		Sent:         int(sent),
		Reply:        int(reply),
		ReplyRate:    replyRate,
		Interested:   int(interested),
		InterestRate: interestRate,
	}
}

// Helper functions

func jsonBytesToMap(data domain.JSONB) map[string]interface{} {
	if len(data) == 0 {
		return nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}
