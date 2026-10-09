package campaign

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	internalworker "github.com/rockship/cosmo-agents-go/internal/worker/dto"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

// Update handles PATCH /v1/campaigns/{id}.
// @Summary Update Campaign
// @Description Update an existing campaign with the provided details.
// @Tags Campaigns
// @Accept json
// @Produce json
// @Param id path string true "Campaign ID"
// @Param body body v1schema.CampaignUpdateRequest true "Campaign Update Request"
// @Security BearerAuth
// @Success 200 {object} schema.APIResponse[string] "Successful Response"
// @Failure 400 {object} schema.APIResponse[any] "Bad Request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 403 {object} schema.APIResponse[any] "Forbidden"
// @Failure 404 {object} schema.APIResponse[any] "Not Found"
// @Failure 500 {object} schema.APIResponse[any] "Internal Server Error"
// @Router /v1/campaigns/{id} [patch]
func (h *Handler) Update(c fiber.Ctx) error {
	campaignID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c, "Invalid campaign ID", err)
	}

	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	primaryOrgID, err := h.roleRepo.FindPrimaryOrganization(c.Context(), userID)
	if err != nil {
		return internalError(c, "Failed to determine primary organization", err)
	}
	if primaryOrgID == nil {
		return unauthorizedAccessResponse(c)
	}

	campaign, err := h.campaignRepo.FindWithRelations(c.Context(), campaignID)
	if err != nil {
		return internalError(c, "Failed to fetch campaign", err)
	}
	if campaign == nil {
		return notFound(c, "Campaign not found")
	}
	if campaign.UserID != userID {
		if campaign.OrganizationID == nil || *campaign.OrganizationID != *primaryOrgID {
			return forbidden(c, "You don't have permission to update this campaign")
		}
	} else if campaign.OrganizationID == nil {
		campaign.OrganizationID = primaryOrgID
	}

	var req v1schema.CampaignUpdateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err)
	}

	if err := v1validation.ValidateStruct(&req); err != nil {
		return badRequest(c, "Validation failed", err)
	}

	finalStatus := campaign.Status
	if req.Status != nil {
		finalStatus = domain.CampaignStatus(*req.Status)
	}

	finalListContactID := campaign.ListContactID
	if req.ListContactID != nil {
		finalListContactID = req.ListContactID
	}

	if req.AgentID != nil {
		ok, err := h.agentUsableBy(c.Context(), *req.AgentID, userID)
		if err != nil {
			return internalError(c, "Failed to verify agent", err)
		}
		if !ok {
			return forbidden(c, "You cannot send from this agent")
		}
	}

	finalAgentID := campaign.AgentID
	if req.AgentID != nil {
		finalAgentID = req.AgentID
	}

	if err := h.validateStatusTransition(c.Context(), finalStatus, finalListContactID, finalAgentID, campaign.ID); err != nil {
		return badRequest(c, "Unable to update campaign status", err)
	}

	previousStatus := campaign.Status
	activating := req.Status != nil && finalStatus == domain.CampaignStatusActive && previousStatus != finalStatus
	// Checked before anything is written: a campaign saved as active whose
	// execution was never queued is stuck, since re-sending "active" is no
	// longer a transition and never enqueues.
	if activating && h.workerClient == nil {
		return internalError(c, "Worker client not configured", errors.New("worker client is nil"))
	}
	updateAttrs := map[string]interface{}{}

	if req.Name != nil {
		updateAttrs["name"] = *req.Name
		campaign.Name = *req.Name
	}
	if req.Status != nil {
		updateAttrs["status"] = finalStatus
		campaign.Status = finalStatus
	}
	if req.Schedule != nil {
		if req.Schedule.Before(time.Now().UTC()) {
			return badRequest(c, "Schedule must be in the future", errors.New("schedule in the past"))
		}
		updateAttrs["schedule"] = *req.Schedule
		campaign.Schedule = req.Schedule
	}
	if req.ListContactID != nil {
		updateAttrs["list_contact_id"] = *req.ListContactID
		campaign.ListContactID = req.ListContactID
	}
	if req.AgentID != nil {
		updateAttrs["agent_id"] = *req.AgentID
		campaign.AgentID = req.AgentID
	}
	if req.CMetadata != nil {
		merged := mergeMaps(metadataToMap(campaign.CMetadata), req.CMetadata)
		updateAttrs["cmetadata"] = merged
		campaign.CMetadata = campaignMetadataFromMap(merged)
	}

	if len(updateAttrs) > 0 {
		if err := h.campaignRepo.UpdateAttributes(c.Context(), campaign.ID, updateAttrs); err != nil {
			return internalError(c, "Failed to update campaign", err)
		}
	}

	if req.NotificationIDs != nil {
		// Note: Campaign.Notifications relationship removed to avoid circular imports
		// Notification data should be loaded separately using relations package if needed
	}

	// When moving to active, enqueue campaign execution.
	if activating {
		if finalAgentID == nil {
			return internalError(c, "Agent is required to execute campaign", errors.New("agent_id is nil"))
		}

		payload := internalworker.ExecuteCampaignPayload{
			CampaignID: campaign.ID,
			UserID:     campaign.UserID,
			AgentID:    *finalAgentID,
		}

		if _, err := h.workerClient.EnqueueCriticalTask(c.Context(), worker.TypeExecuteCampaign, payload); err != nil {
			// The status has to be saved first (the worker refuses a campaign
			// that is not active), so undo it; see the note on activating.
			_ = h.campaignRepo.UpdateAttributes(c.Context(), campaign.ID, map[string]interface{}{"status": previousStatus})
			return internalError(c, "Failed to enqueue campaign execution", err)
		}
	}

	return c.JSON(schema.SuccessResponse(fmt.Sprintf("Update campaign %s successfully", campaignID.String())))
}

// validateStatusTransition validates campaign status transitions
func (h *Handler) validateStatusTransition(ctx context.Context, status domain.CampaignStatus, listContactID *uuid.UUID, agentID *uuid.UUID, campaignID uuid.UUID) error {
	switch status {
	case domain.CampaignStatusActive, domain.CampaignStatusScheduled:
		if listContactID == nil {
			return errors.New("list contact is required")
		}
		count, err := h.countContactsInList(ctx, *listContactID)
		if err != nil {
			return err
		}
		if count == 0 {
			return errors.New("list contact is empty")
		}
		if agentID == nil {
			return errors.New("agent is required")
		}
		templates, err := h.templateRepo.FindByCampaignID(ctx, campaignID)
		if err != nil {
			return err
		}
		if len(templates) == 0 {
			return errors.New("campaign has no templates")
		}
	}
	return nil
}

// countContactsInList counts contacts in a list
func (h *Handler) countContactsInList(ctx context.Context, listID uuid.UUID) (int64, error) {
	return h.listContactRepo.CountContacts(ctx, listID)
}
