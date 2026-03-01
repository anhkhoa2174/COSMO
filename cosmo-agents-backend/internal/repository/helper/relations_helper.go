package helper

import (
	"context"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/relations"
	"gorm.io/gorm"
)

// RelationsHelper provides common methods for loading entity relationships
// This is a centralized helper that can be used across different handlers and services
type RelationsHelper struct {
	db *gorm.DB
}

// NewRelationsHelper creates a new relations helper
func NewRelationsHelper(db *gorm.DB) *RelationsHelper {
	return &RelationsHelper{db: db}
}

// GetUserWithOrganizations retrieves user with organizations
func (h *RelationsHelper) GetUserWithOrganizations(ctx context.Context, userID uuid.UUID) (*relations.UserWithOrganizations, error) {
	var userWithOrgs relations.UserWithOrganizations
	err := h.db.WithContext(ctx).
		Preload("Organizations").
		Where("id = ? AND is_deleted = ?", userID, false).
		First(&userWithOrgs).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &userWithOrgs, nil
}

// GetUserWithRoles retrieves user with roles
func (h *RelationsHelper) GetUserWithRoles(ctx context.Context, userID uuid.UUID) (*relations.UserWithRoles, error) {
	var userWithRoles relations.UserWithRoles
	err := h.db.WithContext(ctx).
		Preload("Roles").
		// Note: Roles.Organization preload removed as Role model no longer has direct relationships
		Where("id = ? AND is_deleted = ?", userID, false).
		First(&userWithRoles).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &userWithRoles, nil
}

// GetUserWithNotifications retrieves user with notifications
func (h *RelationsHelper) GetUserWithNotifications(ctx context.Context, userID uuid.UUID) (*relations.UserWithNotifications, error) {
	var userWithNotifications relations.UserWithNotifications
	err := h.db.WithContext(ctx).
		Preload("Notifications").
		Where("id = ? AND is_deleted = ?", userID, false).
		First(&userWithNotifications).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &userWithNotifications, nil
}

// GetUserWithAgents retrieves user with agents
func (h *RelationsHelper) GetUserWithAgents(ctx context.Context, userID uuid.UUID) (*relations.UserWithAgents, error) {
	var userWithAgents relations.UserWithAgents
	err := h.db.WithContext(ctx).
		Preload("Agents").
		// Note: Agents.Organization preload removed as Agent model no longer has direct relationships
		Where("id = ? AND is_deleted = ?", userID, false).
		First(&userWithAgents).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &userWithAgents, nil
}

// GetUserWithFullRelations retrieves user with all relationships
func (h *RelationsHelper) GetUserWithFullRelations(ctx context.Context, userID uuid.UUID) (*relations.UserWithFullRelations, error) {
	var userWithFullRelations relations.UserWithFullRelations
	err := h.db.WithContext(ctx).
		Preload("Organizations").
		Preload("Roles").
		// Note: Roles.Organization preload removed as Role model no longer has direct relationships
		Preload("Notifications").
		Preload("Agents").
		// Note: Agents.Organization preload removed as Agent model no longer has direct relationships
		Where("id = ? AND is_deleted = ?", userID, false).
		First(&userWithFullRelations).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &userWithFullRelations, nil
}

// GetCampaignWithAgent retrieves campaign with agent
func (h *RelationsHelper) GetCampaignWithAgent(ctx context.Context, campaignID uuid.UUID) (*relations.CampaignWithAgent, error) {
	var campaignWithAgent relations.CampaignWithAgent
	err := h.db.WithContext(ctx).
		Preload("Agent").
		Where("id = ? AND is_deleted = ?", campaignID, false).
		First(&campaignWithAgent).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &campaignWithAgent, nil
}

// GetCampaignWithFullRelations retrieves campaign with all relationships
func (h *RelationsHelper) GetCampaignWithFullRelations(ctx context.Context, campaignID uuid.UUID) (*relations.CampaignWithFullRelations, error) {
	var campaignWithFullRelations relations.CampaignWithFullRelations
	err := h.db.WithContext(ctx).
		Preload("Agent").
		// Note: Agent.Organization preload removed as Agent model no longer has direct relationships
		Preload("Emails").
		Preload("Conversations").
		Where("id = ? AND is_deleted = ?", campaignID, false).
		First(&campaignWithFullRelations).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &campaignWithFullRelations, nil
}

// GetAgentWithUser retrieves agent with user
func (h *RelationsHelper) GetAgentWithUser(ctx context.Context, agentID uuid.UUID) (*relations.AgentWithUser, error) {
	var agentWithUser relations.AgentWithUser
	err := h.db.WithContext(ctx).
		Preload("User").
		Where("id = ? AND is_deleted = ?", agentID, false).
		First(&agentWithUser).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &agentWithUser, nil
}

// GetAgentWithFullRelations retrieves agent with all relationships
func (h *RelationsHelper) GetAgentWithFullRelations(ctx context.Context, agentID uuid.UUID) (*relations.AgentWithFullRelations, error) {
	var agentWithFullRelations relations.AgentWithFullRelations
	err := h.db.WithContext(ctx).
		Preload("User").
		Preload("Organization").
		Preload("Conversations").
		Where("id = ? AND is_deleted = ?", agentID, false).
		First(&agentWithFullRelations).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &agentWithFullRelations, nil
}

// GetTemplateWithFullRelations retrieves template with all relationships
func (h *RelationsHelper) GetTemplateWithFullRelations(ctx context.Context, templateID uuid.UUID) (*relations.TemplateWithFullRelations, error) {
	var templateWithFullRelations relations.TemplateWithFullRelations
	err := h.db.WithContext(ctx).
		Preload("Campaign").
		Where("id = ?", templateID). // Template doesn't have SoftDeleteMixin
		First(&templateWithFullRelations).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	if err := h.db.WithContext(ctx).
		Joins("JOIN template_knowledges tk ON tk.knowledge_id = knowledges.id").
		Where("tk.template_id = ?", templateWithFullRelations.ID).
		Find(&templateWithFullRelations.Knowledges).Error; err != nil {
		return nil, err
	}

	return &templateWithFullRelations, nil
}

// GetTaskWithFullRelations retrieves task with all relationships
func (h *RelationsHelper) GetTaskWithFullRelations(ctx context.Context, taskID uuid.UUID) (*relations.TaskWithFullRelations, error) {
	var taskWithFullRelations relations.TaskWithFullRelations
	err := h.db.WithContext(ctx).
		Preload("Contact").
		Preload("Template").
		Preload("Campaign").
		Where("id = ? AND is_deleted = ?", taskID, false).
		First(&taskWithFullRelations).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &taskWithFullRelations, nil
}

// GetInboundLeadFormWithFullRelations retrieves inbound lead form with all relationships
func (h *RelationsHelper) GetInboundLeadFormWithFullRelations(ctx context.Context, formID uuid.UUID) (*domain.InboundLeadForm, error) {
	var form domain.InboundLeadForm
	err := h.db.WithContext(ctx).
		Preload("Fields").
		Where("id = ?", formID). // InboundLeadForm doesn't have SoftDeleteMixin
		First(&form).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &form, nil
}
