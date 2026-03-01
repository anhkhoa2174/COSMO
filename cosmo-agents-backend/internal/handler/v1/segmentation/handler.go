package segmentation

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/handler"
	"github.com/rockship/cosmo-agents-go/internal/middleware"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	"github.com/rockship/cosmo-agents-go/internal/repository/segmentation"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

type Handler struct {
	responseHelper *handler.ResponseHelper
	authHelper     *middleware.AuthHelper
	segRepo        *segmentation.SegmentationRepository
	scoreRepo      *segmentation.ScoreRepository
	contactRepo    *contactRepo.ContactRepository
}

func NewHandler(
	segRepo *segmentation.SegmentationRepository,
	scoreRepo *segmentation.ScoreRepository,
	contactRepo *contactRepo.ContactRepository,
	userRepo *userRepo.UserRepository,
	roleRepo *roleRepo.RoleRepository,
) *Handler {
	return &Handler{
		responseHelper: handler.NewResponseHelper(),
		authHelper:     middleware.NewAuthHelper(userRepo, roleRepo),
		segRepo:        segRepo,
		scoreRepo:      scoreRepo,
		contactRepo:    contactRepo,
	}
}

// Create segmentation
func (h *Handler) Create(c fiber.Ctx) error {
	user, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	var req v1schema.CreateSegmentationRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	seg := v1schema.ToSegmentationModel(req, user.ID)
	if _, err := h.segRepo.Create(c.Context(), seg); err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to create segmentation", err)
	}
	return h.responseHelper.Success(c, v1schema.ToSegmentationResponse(seg))
}

// List segmentations
func (h *Handler) List(c fiber.Ctx) error {
	user, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}
	onlyActive := c.Query("active") == "true"
	segs, err := h.segRepo.List(c.Context(), user.ID, onlyActive)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to list segmentations", err)
	}
	resp := make([]*v1schema.SegmentationResponse, 0, len(segs))
	for _, s := range segs {
		resp = append(resp, v1schema.ToSegmentationResponse(s))
	}
	return h.responseHelper.Success(c, resp)
}

// Get returns a single segmentation by ID
func (h *Handler) Get(c fiber.Ctx) error {
	_, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	segID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid segmentation id", err)
	}

	seg, err := h.segRepo.GetByID(c.Context(), segID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to get segmentation", err)
	}
	if seg == nil {
		return h.responseHelper.NotFound(c, "Segmentation not found", nil)
	}

	return h.responseHelper.Success(c, v1schema.ToSegmentationResponse(seg))
}

// Upsert score for a contact in a segment
func (h *Handler) UpsertScore(c fiber.Ctx) error {
	_, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	segID, err := uuid.Parse(c.Params("segmentation_id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid segmentation id", err)
	}
	contactID, err := uuid.Parse(c.Params("contact_id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact id", err)
	}

	var req v1schema.UpsertSegmentScoreRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	model := v1schema.ToSegmentScoreModel(req, contactID, segID)
	if err := h.scoreRepo.UpsertScore(c.Context(), model); err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to upsert score", err)
	}
	return h.responseHelper.Success(c, v1schema.ToSegmentScoreResponse(model))
}

// List scores for a contact
func (h *Handler) ListScores(c fiber.Ctx) error {
	_, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}
	contactID, err := uuid.Parse(c.Params("contact_id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact id", err)
	}

	scores, err := h.scoreRepo.ListByContact(c.Context(), contactID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to list scores", err)
	}
	resp := make([]*v1schema.SegmentScoreResponse, 0, len(scores))
	for _, s := range scores {
		resp = append(resp, v1schema.ToSegmentScoreResponse(s))
	}
	return h.responseHelper.Success(c, resp)
}

// Delete segmentation
func (h *Handler) Delete(c fiber.Ctx) error {
	user, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	segID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid segmentation id", err)
	}

	deleted, err := h.segRepo.DeleteByID(c.Context(), segID, user.ID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to delete segmentation", err)
	}
	if !deleted {
		return h.responseHelper.NotFound(c, "Segmentation not found", nil)
	}

	return h.responseHelper.Success(c, fiber.Map{"message": "Segmentation deleted successfully"})
}

// GetContacts returns contacts in a segment
func (h *Handler) GetContacts(c fiber.Ctx) error {
	_, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	segID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid segmentation id", err)
	}

	// Parse pagination params
	limit := 100
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	pagination := &baseRepo.PaginationParams{
		Limit:  limit,
		Offset: offset,
	}

	// Use FindBySegment from contact repository
	contacts, total, err := h.contactRepo.FindBySegment(c.Context(), segID, nil, pagination)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to get contacts", err)
	}

	// Convert to response format
	resp := make([]fiber.Map, 0, len(contacts))
	for _, contact := range contacts {
		// Extract email from profile
		var email string
		if len(contact.Profile) > 0 {
			var profile map[string]interface{}
			if err := json.Unmarshal(contact.Profile, &profile); err == nil {
				if e, ok := profile["email"].(string); ok {
					email = e
				}
			}
		}
		resp = append(resp, fiber.Map{
			"id":        contact.ID,
			"email":     email,
			"name":      contact.Name,
			"company":   contact.Company,
			"job_title": contact.JobTitle,
		})
	}

	return h.responseHelper.Success(c, fiber.Map{
		"contacts": resp,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}
