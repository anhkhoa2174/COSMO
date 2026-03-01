package interaction

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/handler"
	"github.com/rockship/cosmo-agents-go/internal/middleware"
	"github.com/rockship/cosmo-agents-go/internal/repository/interaction"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

type Handler struct {
	responseHelper *handler.ResponseHelper
	authHelper     *middleware.AuthHelper
	repo           *interaction.Repository
	workerClient   *worker.Client
}

func New(repo *interaction.Repository, userRepo *userRepo.UserRepository, roleRepo *roleRepo.RoleRepository, workerClient *worker.Client) *Handler {
	return &Handler{
		responseHelper: handler.NewResponseHelper(),
		authHelper:     middleware.NewAuthHelper(userRepo, roleRepo),
		repo:           repo,
		workerClient:   workerClient,
	}
}

// Create logs an interaction
func (h *Handler) Create(c fiber.Ctx) error {
	user, organizationID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}
	var req v1schema.CreateInteractionRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}
	model := v1schema.ToInteractionModel(req)
	if _, err := h.repo.Create(c.Context(), model); err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to log interaction", err)
	}

	if h.workerClient != nil {
		payload := map[string]interface{}{
			"contact_id": model.ContactID,
			"user_id":    user.ID,
			"org_id":     organizationID,
			"event":      "interaction_logged",
		}
		_, _ = h.workerClient.EnqueueTask(context.Background(), worker.TypeOrchestrateContact, payload)
	}

	return h.responseHelper.Success(c, v1schema.ToInteractionResponse(model))
}

// ListByContact returns recent interactions for a contact
func (h *Handler) ListByContact(c fiber.Ctx) error {
	_, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}
	contactID, err := uuid.Parse(c.Params("contact_id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact id", err)
	}
	limit := 50
	if lStr := c.Query("limit"); lStr != "" {
		if v, err := strconv.Atoi(lStr); err == nil && v > 0 {
			limit = v
		}
	}
	items, err := h.repo.ListByContact(c.Context(), contactID, limit)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to list interactions", err)
	}
	resp := make([]*v1schema.InteractionResponse, 0, len(items))
	for _, it := range items {
		resp = append(resp, v1schema.ToInteractionResponse(it))
	}
	return h.responseHelper.Success(c, resp)
}

// health ping
func (h *Handler) Ping(c fiber.Ctx) error {
	return c.JSON(map[string]any{"status": "ok", "ts": time.Now()})
}
