package organization

import (
	"github.com/gofiber/fiber/v3"

	"github.com/rockship/cosmo-agents-go/internal/schema"
	"github.com/rockship/cosmo-agents-go/internal/service/productivity"
)

// GetTeamProductivity reports what the team got done over the last week or
// month.
//
// The endpoint is deliberately not admin-gated. An admin gets a row per
// member; anyone else gets exactly one row — their own. Returning a
// self-scoped report instead of a 403 means the dashboard has one endpoint and
// one shape to render for both roles, and a member cannot widen the result by
// changing a parameter, because the scope is decided from their role and never
// from the request.
//
// @Summary Team productivity
// @Description Per-member activity for the last 7 or 30 days. Admins see the whole team; members see only themselves.
// @Tags Organization
// @Produce json
// @Param organization_id path string true "Organization ID, or 'me'"
// @Param period query string false "7d or 30d (default 30d)"
// @Success 200 {object} schema.APIResponse[any]
// @Router /v2/organizations/{organization_id}/team-productivity [get]
func (h *OrganizationHandler) GetTeamProductivity(c fiber.Ctx) error {
	userID, orgID, isAdmin, errResp := h.resolveOrg(c)
	if errResp != nil {
		return errResp
	}

	if h.productivity == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(
			schema.ErrorResponse(fiber.StatusServiceUnavailable,
				"Productivity reporting is not configured", ""),
		)
	}

	report, err := h.productivity.TeamReport(
		c.Context(), orgID, userID, isAdmin,
		productivity.ParsePeriod(c.Query("period")),
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError,
				"Failed to build productivity report", err.Error()),
		)
	}

	return c.JSON(schema.SuccessResponse(report))
}
