package dto

import (
	"github.com/google/uuid"
)

// ExecuteCampaignPayload represents payload for executing a campaign.
type ExecuteCampaignPayload struct {
	CampaignID uuid.UUID   `json:"campaign_id"`
	UserID     uuid.UUID   `json:"user_id"`
	AgentID    uuid.UUID   `json:"agent_id"`
	ContactIDs []uuid.UUID `json:"contact_ids,omitempty"` // Optional: specific contacts
}

// GenerateEmailPayload represents payload for generating email content using AI.
type GenerateEmailPayload struct {
	CampaignID uuid.UUID              `json:"campaign_id"`
	ContactID  uuid.UUID              `json:"contact_id"`
	TemplateID uuid.UUID              `json:"template_id"`
	TaskID     *uuid.UUID             `json:"task_id,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// ScheduleTasksPayload represents payload for scheduling campaign tasks.
type ScheduleTasksPayload struct {
	CampaignID uuid.UUID   `json:"campaign_id"`
	ContactIDs []uuid.UUID `json:"contact_ids"`
	AgentID    uuid.UUID   `json:"agent_id"`
}
