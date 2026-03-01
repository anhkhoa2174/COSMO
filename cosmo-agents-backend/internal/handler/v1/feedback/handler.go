package feedback

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/handler"
	"github.com/rockship/cosmo-agents-go/internal/middleware"
	feedbackRepo "github.com/rockship/cosmo-agents-go/internal/repository/feedback"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

type Handler struct {
	responseHelper *handler.ResponseHelper
	authHelper     *middleware.AuthHelper
	repo           *feedbackRepo.Repository
}

func New(repo *feedbackRepo.Repository, userRepo *userRepo.UserRepository, roleRepo *roleRepo.RoleRepository) *Handler {
	return &Handler{
		responseHelper: handler.NewResponseHelper(),
		authHelper:     middleware.NewAuthHelper(userRepo, roleRepo),
		repo:           repo,
	}
}

// Create score adjustment feedback
func (h *Handler) ScoreAdjust(c fiber.Ctx) error {
	user, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}
	var req v1schema.CreateScoreFeedbackRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}
	data := map[string]interface{}{
		"field_path":     req.FieldPath,
		"original_score": req.OriginalScore,
		"adjusted_score": req.AdjustedScore,
		"reason":         req.Reason,
	}
	f := &domain.UserFeedback{
		UserID:       user.ID,
		EntityType:   "contact",
		EntityID:     req.ContactID,
		FeedbackType: "score_adjustment",
		FeedbackData: marshalMap(data),
		Applied:      true,
		AppliedAt:    ptrTime(time.Now()),
	}
	if _, err := h.repo.Create(c.Context(), f); err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to store feedback", err)
	}
	return h.responseHelper.Success(c, v1schema.ToFeedbackResponse(f))
}

// Create insight validation feedback
func (h *Handler) InsightValidation(c fiber.Ctx) error {
	user, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}
	var req v1schema.CreateInsightFeedbackRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}
	data := map[string]interface{}{
		"insight_type":   req.InsightType,
		"insight_text":   req.InsightText,
		"validation":     req.Validation,
		"confirmed_data": req.ConfirmedData,
	}
	f := &domain.UserFeedback{
		UserID:       user.ID,
		EntityType:   "contact",
		EntityID:     req.ContactID,
		FeedbackType: "insight_validation",
		FeedbackData: marshalMap(data),
		Applied:      true,
		AppliedAt:    ptrTime(time.Now()),
	}
	if _, err := h.repo.Create(c.Context(), f); err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to store feedback", err)
	}
	return h.responseHelper.Success(c, v1schema.ToFeedbackResponse(f))
}

// Create custom fact feedback
func (h *Handler) CustomFact(c fiber.Ctx) error {
	user, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}
	var req v1schema.CreateCustomFactRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}
	data := map[string]interface{}{
		"fact_type": req.FactType,
		"fact_data": req.FactData,
	}
	f := &domain.UserFeedback{
		UserID:       user.ID,
		EntityType:   "contact",
		EntityID:     req.ContactID,
		FeedbackType: "custom_fact",
		FeedbackData: marshalMap(data),
		Applied:      true,
		AppliedAt:    ptrTime(time.Now()),
	}
	if _, err := h.repo.Create(c.Context(), f); err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to store feedback", err)
	}
	return h.responseHelper.Success(c, v1schema.ToFeedbackResponse(f))
}

// List feedback by user
func (h *Handler) List(c fiber.Ctx) error {
	user, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}
	items, err := h.repo.ListByUser(c.Context(), user.ID, 50)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to list feedback", err)
	}
	resp := make([]*v1schema.FeedbackResponse, 0, len(items))
	for _, it := range items {
		resp = append(resp, v1schema.ToFeedbackResponse(it))
	}
	return h.responseHelper.Success(c, resp)
}

func marshalMap(m map[string]interface{}) domain.JSONB {
	var jb domain.JSONB
	_ = jb.Marshal(m)
	return jb
}

func ptrTime(t time.Time) *time.Time { return &t }
