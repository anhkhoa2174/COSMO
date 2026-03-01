package intelligence

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	_ "github.com/rockship/cosmo-agents-go/internal/schema"           // imported for swagger
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1" // imported for swagger
)

// GenerateMeetingBrief generates a pre-meeting brief for a contact
// @Summary Generate pre-meeting brief
// @Description Generates talking points, discovery questions, and risk flags for an upcoming meeting
// @Tags Intelligence
// @Accept json
// @Produce json
// @Param id path string true "Contact ID"
// @Success 200 {object} schema.APIResponse[v1schema.MeetingBriefResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/contacts/{id}/generate-meeting-brief [post]
func (h *Handler) GenerateMeetingBrief(c fiber.Ctx) error {
	user, orgID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	contactID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "invalid contact id", err)
	}

	// Generate meeting brief
	brief, err := h.svc.GenerateMeetingBrief(c.Context(), contactID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "failed to generate meeting brief", err)
	}

	// Log for debugging
	_ = user
	_ = orgID
	var _ v1schema.MeetingBriefResponse // force import for swagger

	return h.responseHelper.Success(c, brief)
}
