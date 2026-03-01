package v1

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

const (
	// HandlerAI mirrors machine.models.campaign.Handler.AI value.
	HandlerAI = "Let AI reply"
	// HandlerHuman mirrors machine.models.campaign.Handler.HUMAN value.
	HandlerHuman = "Assign to a person"
	// HandlerDraft mirrors machine.models.campaign.Handler.DRAFT value.
	HandlerDraft = "Draft an email"
)

// CampaignResponse matches CampaignCreateResponse in the Python service.
type CampaignResponse struct {
	ID             uuid.UUID              `json:"id"`
	UserID         uuid.UUID              `json:"user_id"`
	Name           *string                `json:"name,omitempty"`
	Playbook       *string                `json:"playbook,omitempty"`
	ListContactID  *uuid.UUID             `json:"list_contact_id,omitempty"`
	OrganizationID *uuid.UUID             `json:"organization_id,omitempty"`
	AgentID        *uuid.UUID             `json:"agent_id,omitempty"`
	TemplateID     *uuid.UUID             `json:"template_id,omitempty"`
	Schedule       *time.Time             `json:"schedule,omitempty"`
	Status         string                 `json:"status"`
	CMetadata      map[string]interface{} `json:"cmetadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// RelationshipAgent matches python schema RelationshipAgent.
type RelationshipAgent struct {
	ID    uuid.UUID              `json:"id"`
	Name  string                 `json:"name"`
	Cmeta map[string]interface{} `json:"cmeta"`
}

// CampaignEntity represents the embedded campaign entity in listings.
type CampaignEntity struct {
	ID             uuid.UUID          `json:"id"`
	UserID         uuid.UUID          `json:"user_id"`
	Playbook       *string            `json:"playbook,omitempty"`
	Name           *string            `json:"name,omitempty"`
	ListContactID  *uuid.UUID         `json:"list_contact_id,omitempty"`
	OrganizationID *uuid.UUID         `json:"organization_id,omitempty"`
	Schedule       *time.Time         `json:"schedule,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
	Status         *string            `json:"status,omitempty"`
	Agent          *RelationshipAgent `json:"agent,omitempty"`
}

// CampaignListItem matches python Listing[CampaignListItem].
type CampaignListItem struct {
	Entity       CampaignEntity `json:"entity"`
	Creator      *string        `json:"creator"`
	Sent         int            `json:"sent"`
	Reply        int            `json:"reply"`
	ReplyRate    float64        `json:"reply_rate"`
	Interested   int            `json:"interested"`
	InterestRate float64        `json:"interest_rate"`
}

// CampaignGetResponse mirrors python CampaignGetResponse (same structure as CampaignListItem).
type CampaignGetResponse = CampaignListItem

// CampaignGetRequest represents POST /v1/campaigns/search request body.
type CampaignGetRequest struct {
	Filter map[string]any `json:"filter,omitempty"`
}

// NotificationRelationshipListItem mirrors python schema.
type NotificationRelationshipListItem struct {
	UserID uuid.UUID `json:"user_id"`
}

// TemplateRelationshipListItem mirrors python schema.
type TemplateRelationshipListItem struct {
	ID        uuid.UUID `json:"id"`
	Category  string    `json:"category"`
	Type      string    `json:"type"`
	SendAfter int       `json:"send_after"`
	Content   string    `json:"content"`
}

// DraftTemplateRelationshipListItem mirrors python schema.
type DraftTemplateRelationshipListItem struct {
	ID     uuid.UUID `json:"id"`
	Intent string    `json:"intent"`
}

// CampaignDetailGetResponse represents GET /v1/campaigns/{campaign_id}.
type CampaignDetailGetResponse struct {
	ID             uuid.UUID                           `json:"id"`
	UserID         uuid.UUID                           `json:"user_id"`
	Playbook       *string                             `json:"playbook"`
	Name           *string                             `json:"name"`
	ListContactID  *uuid.UUID                          `json:"list_contact_id"`
	OrganizationID *uuid.UUID                          `json:"organization_id"`
	Schedule       *time.Time                          `json:"schedule"`
	CreatedAt      time.Time                           `json:"created_at"`
	UpdatedAt      time.Time                           `json:"updated_at"`
	Status         *string                             `json:"status"`
	AgentID        *uuid.UUID                          `json:"agent_id"`
	Notifications  []NotificationRelationshipListItem  `json:"notifications"`
	Templates      []TemplateRelationshipListItem      `json:"templates"`
	DraftTemplates []DraftTemplateRelationshipListItem `json:"draft_templates"`
	CMetadata      map[string]interface{}              `json:"cmetadata"`
}

// CampaignCreateRequest matches python request.
type CampaignCreateRequest struct {
	Name            *string     `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Playbook        string      `json:"playbook" validate:"required"`
	CampaignID      *uuid.UUID  `json:"campaign_id,omitempty"`
	ListContactID   *uuid.UUID  `json:"list_contact_id,omitempty"`
	OrganizationID  *uuid.UUID  `json:"organization_id,omitempty"`
	Schedule        *time.Time  `json:"schedule,omitempty"`
	AgentID         *uuid.UUID  `json:"agent_id,omitempty"`
	Status          *string     `json:"status,omitempty" validate:"omitempty,oneof=active ended paused draft scheduled"`
	NotificationIDs []uuid.UUID `json:"notification_ids,omitempty"`
}

// CampaignUpdateRequest mirrors python update schema.
type CampaignUpdateRequest struct {
	Name            *string                `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Status          *string                `json:"status,omitempty" validate:"omitempty,oneof=active ended paused draft scheduled"`
	Schedule        *time.Time             `json:"schedule,omitempty"`
	ListContactID   *uuid.UUID             `json:"list_contact_id,omitempty"`
	AgentID         *uuid.UUID             `json:"agent_id,omitempty"`
	NotificationIDs []uuid.UUID            `json:"notification_ids,omitempty"`
	CMetadata       map[string]interface{} `json:"cmetadata,omitempty"`
}

// UpdateCampaignMetadataRequest mirrors python: wraps { client: body } operation.
type UpdateCampaignMetadataRequest struct {
	Client map[string]interface{} `json:"client" validate:"required"`
}

// CampaignAssignRequest mirrors python config payload.
type CampaignAssignRequest struct {
	Config []AssignMemberRequest `json:"config" validate:"required,min=1"`
}

// AssignMemberRequest matches python pydantic model.
type AssignMemberRequest struct {
	Who        string      `json:"who" validate:"required,oneof='Let AI reply' 'Assign to a person' 'Draft an email'"`
	IntentType string      `json:"intent_type" validate:"required"`
	Payload    interface{} `json:"payload" validate:"required"` // Pass-through since payload shape varies
}

// CampaignSaveOutreachRequest mirrors python schema.
type CampaignSaveOutreachRequest struct {
	Sequence []OutreachEmailTemplate `json:"sequence" validate:"required,min=1"`
}

// OutreachEmailTemplate matches python schema.
type OutreachEmailTemplate struct {
	Type    string  `json:"type" validate:"required"`
	Subject *string `json:"subject,omitempty"`
	Content *string `json:"content,omitempty"`
}

// FollowUpScheduleRequest matches python CampaignFollowUpSchedule.
type FollowUpScheduleRequest struct {
	FollowUp1Schedule *int `json:"follow_up_1_schedule,omitempty"`
	FollowUp2Schedule *int `json:"follow_up_2_schedule,omitempty"`
}

// DeleteNotificationRequest mirrors python schema.
type DeleteNotificationRequest struct {
	IDs []uuid.UUID `json:"ids" validate:"required,min=1"`
}

// CampaignGenerateSampleResponse mirrors python response for generate endpoint.
type CampaignGenerateSampleResponse struct {
	Intent   string `json:"intent"`
	Response string `json:"response"`
}

// ToCampaignResponse converts domain campaign to API response.
func ToCampaignResponse(campaign *domain.Campaign) CampaignResponse {
	if campaign == nil {
		return CampaignResponse{}
	}

	name := campaign.Name
	playbook := campaign.Playbook
	status := string(campaign.Status)

	return CampaignResponse{
		ID:             campaign.ID,
		UserID:         campaign.UserID,
		Name:           stringPtrOrNil(name),
		Playbook:       stringPtrOrNil(playbook),
		ListContactID:  campaign.ListContactID,
		OrganizationID: campaign.OrganizationID,
		AgentID:        campaign.AgentID,
		TemplateID:     nil,
		Schedule:       campaign.Schedule,
		Status:         status,
		CMetadata:      campaignMetadataToMap(campaign.CMetadata),
		CreatedAt:      campaign.CreatedAt,
		UpdatedAt:      campaign.UpdatedAt,
	}
}

// ToCampaignListItem converts domain campaign and stats to listing item.
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

	status := string(campaign.Status)
	entity.Status = stringPtrOrNil(status)

	// Note: Agent relationship removed to avoid circular imports
	// Agent data should be loaded separately using relations package if needed

	return CampaignListItem{
		Entity:       entity,
		Creator:      creator,
		Sent:         int(sent),
		Reply:        int(reply),
		ReplyRate:    replyRate,
		Interested:   int(interested),
		InterestRate: interestRate,
	}
}

// ToCampaignDetailResponse converts domain campaign with relations to detailed response.
func ToCampaignDetailResponse(campaign *domain.Campaign) CampaignDetailGetResponse {
	if campaign == nil {
		return CampaignDetailGetResponse{}
	}

	status := string(campaign.Status)

	response := CampaignDetailGetResponse{
		ID:             campaign.ID,
		UserID:         campaign.UserID,
		Playbook:       stringPtrOrNil(campaign.Playbook),
		Name:           stringPtrOrNil(campaign.Name),
		ListContactID:  campaign.ListContactID,
		OrganizationID: campaign.OrganizationID,
		Schedule:       campaign.Schedule,
		CreatedAt:      campaign.CreatedAt,
		UpdatedAt:      campaign.UpdatedAt,
		Status:         stringPtrOrNil(status),
		AgentID:        campaign.AgentID,
		CMetadata:      ensureEmptyMap(campaignMetadataToMap(campaign.CMetadata)),
		Notifications:  []NotificationRelationshipListItem{},
		Templates:      []TemplateRelationshipListItem{},
		DraftTemplates: []DraftTemplateRelationshipListItem{},
	}

	// Note: Direct relationships removed to avoid circular imports
	// These fields should be loaded separately using relations package if needed
	response.Notifications = make([]NotificationRelationshipListItem, 0)
	response.Templates = make([]TemplateRelationshipListItem, 0)
	response.DraftTemplates = make([]DraftTemplateRelationshipListItem, 0)

	return response
}

func campaignMetadataToMap(metadata domain.CampaignMetadata) map[string]interface{} {
	data, err := json.Marshal(metadata)
	if err != nil || len(data) == 0 {
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}

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

func stringPtrOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// ensureEmptyMap returns an empty map if the input is nil, otherwise returns the input
func ensureEmptyMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return map[string]interface{}{}
	}
	return m
}

// SearchCampaignsRequest for POST /v1/campaigns/search
type SearchCampaignsRequest struct {
	Filter map[string]interface{} `json:"filter"`
	Offset int                    `json:"offset"`
	Limit  int                    `json:"limit"`
}
