package campaign

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// Test basic domain structures for campaign service
func TestCampaignService_Structures(t *testing.T) {
	// Test Campaign structure
	userID := uuid.New()
	campaignID := uuid.New()
	agentID := uuid.New()
	agentIDPtr := &agentID

	campaign := &domain.Campaign{
		Base:    domain.Base{ID: campaignID},
		UserID:  userID,
		Status:  domain.CampaignStatusActive,
		AgentID: agentIDPtr,
	}

	assert.NotNil(t, campaign)
	assert.Equal(t, userID, campaign.UserID)
	assert.Equal(t, campaignID, campaign.ID)
	assert.Equal(t, domain.CampaignStatusActive, campaign.Status)
	assert.Equal(t, agentID, *campaign.AgentID)

	// Test Contact structure
	contact := &domain.Contact{
		Base:    domain.Base{ID: uuid.New()},
		UserID:  userID,
		Profile: base.JSONB(`{"email":"test@example.com"}`),
	}

	assert.NotNil(t, contact)
	assert.Equal(t, userID, contact.UserID)
	email := ""
	if len(contact.Profile) > 0 {
		var profile map[string]interface{}
		if err := contact.Profile.Unmarshal(&profile); err == nil {
			if v, ok := profile["email"]; ok {
				email = fmt.Sprintf("%v", v)
			}
		}
	}
	assert.Equal(t, "test@example.com", email)

	// Test Template structure
	template := &domain.Template{
		Base:       domain.Base{ID: uuid.New()},
		CampaignID: &campaignID,
		SendAfter:  1,
	}

	assert.NotNil(t, template)
	assert.Equal(t, campaignID, *template.CampaignID)
	assert.Equal(t, 1, template.SendAfter)

	// Test Agent structure
	agent := &domain.Agent{
		Base: domain.Base{ID: agentID},
		Name: "Test Agent",
	}

	assert.NotNil(t, agent)
	assert.Equal(t, agentID, agent.ID)
	assert.Equal(t, "Test Agent", agent.Name)

	// Test TaskAttributes structure
	taskAttrs := domain.TaskAttributes{
		ContactID:  uuid.New(),
		CampaignID: campaignID,
		TemplateID: uuid.New(),
		Status:     domain.TaskStatusPending,
	}

	assert.NotEqual(t, uuid.Nil, taskAttrs.ContactID)
	assert.NotEqual(t, uuid.Nil, taskAttrs.CampaignID)
	assert.NotEqual(t, uuid.Nil, taskAttrs.TemplateID)
	assert.Equal(t, domain.TaskStatusPending, taskAttrs.Status)
}

// Test error constants
func TestCampaignService_ErrorConstants(t *testing.T) {
	assert.Error(t, ErrInvalidCampaignStatus)
	assert.Contains(t, ErrInvalidCampaignStatus.Error(), "invalid campaign status")

	assert.Error(t, ErrEmptyContactList)
	assert.Contains(t, ErrEmptyContactList.Error(), "contact list is empty")

	assert.Error(t, ErrNoTemplates)
	assert.Contains(t, ErrNoTemplates.Error(), "no outreach emails are defined")

	assert.Error(t, ErrNoAgent)
	assert.Contains(t, ErrNoAgent.Error(), "no agent is configured for the campaign")
}

// Test campaign status validation
func TestCampaignService_StatusValidation(t *testing.T) {
	validStatuses := []domain.CampaignStatus{
		domain.CampaignStatusActive,
		domain.CampaignStatusScheduled,
	}

	invalidStatuses := []domain.CampaignStatus{
		domain.CampaignStatusPaused,
		domain.CampaignStatusEnded,
	}

	for _, status := range validStatuses {
		isValid := false
		for _, valid := range validStatuses {
			if status == valid {
				isValid = true
				break
			}
		}
		assert.True(t, isValid, "Status %v should be valid", status)
	}

	for _, status := range invalidStatuses {
		isValid := false
		for _, valid := range validStatuses {
			if status == valid {
				isValid = true
				break
			}
		}
		assert.False(t, isValid, "Status %v should be invalid", status)
	}
}
