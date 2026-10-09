package organization

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	"github.com/rockship/cosmo-agents-go/internal/service/autoreply"
	outreachSvc "github.com/rockship/cosmo-agents-go/internal/service/outreach"
)

// outreachSettingsResponse returns both what the admin chose and what is
// actually in force. Sending only the stored values leaves the form unable to
// show what a blank field will fall back to.
type outreachSettingsResponse struct {
	Settings  outreachSvc.OutreachSettings `json:"settings"`
	Effective outreachSvc.Config           `json:"effective"`
	Defaults  outreachSvc.Config           `json:"defaults"`
	Summary   string                       `json:"summary"`

	// AutoReply is reported separately from Effective because it is not part
	// of the cadence value type. SelectableIntents is served rather than
	// hard-coded in the client so the form can never offer an intent the
	// server would refuse to save.
	AutoReply         autoReplyView `json:"auto_reply"`
	SelectableIntents []string      `json:"auto_reply_selectable_intents"`

	// MergeTags are the fields a fixed reply may use, served for the same
	// reason as SelectableIntents: the editor must not offer a field the
	// server would refuse to save.
	MergeTags []string `json:"auto_reply_merge_tags"`
}

type autoReplyView struct {
	Enabled       bool     `json:"enabled"`
	Intents       []string `json:"intents"`
	MinConfidence float64  `json:"min_confidence"`
	DailyCap      int      `json:"daily_cap"`
	Summary       string   `json:"summary"`

	// Contents is what each intent's reply says, keyed like Intents.
	Contents map[string]autoreply.Content `json:"contents"`
}

func viewAutoReply(r autoreply.Resolved) autoReplyView {
	intents := make([]string, 0, len(r.Intents))
	for i := range r.Intents {
		intents = append(intents, string(i))
	}
	sort.Strings(intents)
	contents := make(map[string]autoreply.Content, len(r.Contents))
	for i, c := range r.Contents {
		contents[string(i)] = c
	}
	return autoReplyView{
		Contents:      contents,
		Enabled:       r.Enabled,
		Intents:       intents,
		MinConfidence: r.MinConfidence,
		DailyCap:      r.DailyCap,
		Summary:       r.Describe(),
	}
}

func selectableIntents() []string {
	out := []string{}
	for _, i := range autoreply.Selectable() {
		out = append(out, string(i))
	}
	return out
}

// resolveOrg finds the organisation the request is about and the caller's role
// in it. Endpoints that admins and members both use call this; endpoints that
// only admins may touch call requireAdmin instead.
func (h *OrganizationHandler) resolveOrg(c fiber.Ctx) (uuid.UUID, uuid.UUID, bool, error) {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return uuid.Nil, uuid.Nil, false, c.Status(fiber.StatusUnauthorized).JSON(
			schema.ErrorResponse(fiber.StatusUnauthorized, "Unauthorized", ""),
		)
	}

	// The rest of the organisation API accepts "me" for "the organisation I
	// belong to"; these endpoints use the same shorthand.
	var orgID uuid.UUID
	if raw := c.Params("organization_id"); raw == "me" {
		org, err := h.orgRepo.FindUserMainOrganization(c.Context(), userID)
		if err != nil || org == nil {
			return uuid.Nil, uuid.Nil, false, c.Status(fiber.StatusNotFound).JSON(
				schema.ErrorResponse(fiber.StatusNotFound, "No organization for this user", ""),
			)
		}
		orgID = org.ID
	} else {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return uuid.Nil, uuid.Nil, false, c.Status(fiber.StatusBadRequest).JSON(
				schema.ErrorResponse(fiber.StatusBadRequest, "Invalid organization id", err.Error()),
			)
		}
		orgID = parsed
	}

	role, err := h.roleRepo.FindByUserAndOrganization(c.Context(), userID, orgID)
	if err != nil {
		return uuid.Nil, uuid.Nil, false, c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to verify permissions", err.Error()),
		)
	}
	// A caller with no role in the organisation is told it does not exist
	// rather than that they lack access, so the endpoint cannot be used to
	// probe for organisation ids.
	if role == nil {
		return uuid.Nil, uuid.Nil, false, c.Status(fiber.StatusNotFound).JSON(
			schema.ErrorResponse(fiber.StatusNotFound, "Organization not found", ""),
		)
	}

	return userID, orgID, role.Name == domain.RoleNameAdmin, nil
}

// requireAdmin resolves the target organisation and refuses anyone who is not
// an admin of it. Cadence changes affect every campaign in the account, so
// they are not a member-level action.
func (h *OrganizationHandler) requireAdmin(c fiber.Ctx) (uuid.UUID, uuid.UUID, error) {
	userID, orgID, isAdmin, errResp := h.resolveOrg(c)
	if errResp != nil {
		return uuid.Nil, uuid.Nil, errResp
	}
	if !isAdmin {
		return uuid.Nil, uuid.Nil, c.Status(fiber.StatusForbidden).JSON(
			schema.ErrorResponse(fiber.StatusForbidden, "Admin role required",
				"Only an organization admin can change outreach timing"),
		)
	}
	return userID, orgID, nil
}

// GetOutreachSettings returns the organisation's cadence.
//
// @Summary Get outreach timing settings
// @Tags Organization
// @Produce json
// @Param organization_id path string true "Organization ID"
// @Success 200 {object} schema.APIResponse[any]
// @Router /v2/organizations/{organization_id}/outreach-settings [get]
func (h *OrganizationHandler) GetOutreachSettings(c fiber.Ctx) error {
	_, orgID, errResp := h.requireAdmin(c)
	if errResp != nil {
		return errResp
	}

	org, err := h.orgRepo.FindByID(c.Context(), orgID)
	if err != nil || org == nil {
		return c.Status(fiber.StatusNotFound).JSON(
			schema.ErrorResponse(fiber.StatusNotFound, "Organization not found", ""),
		)
	}

	var stored outreachSvc.OutreachSettings
	if len(org.OutreachSettings) > 0 {
		_ = json.Unmarshal(org.OutreachSettings, &stored)
	}
	effective := outreachSvc.ConfigFrom(org.OutreachSettings)

	return c.JSON(schema.SuccessResponse(outreachSettingsResponse{
		Settings:          stored,
		Effective:         effective,
		Defaults:          outreachSvc.DefaultConfig(),
		Summary:           effective.Describe(),
		AutoReply:         viewAutoReply(outreachSvc.AutoReplyFrom(org.OutreachSettings)),
		SelectableIntents: selectableIntents(),
		MergeTags:         autoreply.MergeTags,
	}))
}

// UpdateOutreachSettings stores a new cadence.
//
// @Summary Update outreach timing settings
// @Tags Organization
// @Accept json
// @Produce json
// @Param organization_id path string true "Organization ID"
// @Success 200 {object} schema.APIResponse[any]
// @Failure 403 {object} schema.APIResponse[any] "Only admins may change timing"
// @Failure 422 {object} schema.APIResponse[any] "A value is out of range"
// @Router /v2/organizations/{organization_id}/outreach-settings [put]
func (h *OrganizationHandler) UpdateOutreachSettings(c fiber.Ctx) error {
	_, orgID, errResp := h.requireAdmin(c)
	if errResp != nil {
		return errResp
	}

	var req outreachSvc.OutreachSettings
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			schema.ErrorResponse(fiber.StatusBadRequest, "Invalid request body", err.Error()),
		)
	}

	// Reject the whole request rather than storing the valid half: a cadence
	// half-applied is harder to reason about than one rejected outright.
	if problems := req.Validate(); len(problems) > 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(
			schema.ErrorResponse(fiber.StatusUnprocessableEntity,
				"Invalid outreach timing", strings.Join(problems, "; ")),
		)
	}

	encoded, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to encode settings", err.Error()),
		)
	}

	if err := h.orgRepo.UpdateOutreachSettings(c.Context(), orgID, encoded); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to save settings", err.Error()),
		)
	}

	effective := outreachSvc.ConfigFrom(encoded)
	return c.JSON(schema.SuccessResponse(outreachSettingsResponse{
		Settings:          req,
		Effective:         effective,
		Defaults:          outreachSvc.DefaultConfig(),
		Summary:           effective.Describe(),
		AutoReply:         viewAutoReply(outreachSvc.AutoReplyFrom(encoded)),
		SelectableIntents: selectableIntents(),
		MergeTags:         autoreply.MergeTags,
	}))
}
