package context

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	ctxService "github.com/rockship/cosmo-agents-go/internal/service/context"
)

type Handler struct {
	ctxSvc *ctxService.Service
}

func NewHandler(ctxSvc *ctxService.Service) *Handler {
	return &Handler{ctxSvc: ctxSvc}
}

// Helper to get user and org IDs from context
func getUserAndOrgID(c fiber.Ctx) (userID, orgID uuid.UUID, err error) {
	userVal := c.Locals("user_id")
	if userVal == nil {
		return uuid.Nil, uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	userID, ok := userVal.(uuid.UUID)
	if !ok {
		return uuid.Nil, uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "invalid user_id")
	}

	orgVal := c.Locals("org_id")
	if orgVal != nil {
		orgID, _ = orgVal.(uuid.UUID)
	}

	return userID, orgID, nil
}

// ============ Organization Context ============

// GetOrgContext returns the organization context
// GET /v1/context/org
func (h *Handler) GetOrgContext(c fiber.Ctx) error {
	_, orgID, err := getUserAndOrgID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	if orgID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no organization"})
	}

	ctx := c.Context()
	orgCtx, err := h.ctxSvc.GetOrgContext(ctx, orgID)
	if err != nil {
		// Return empty context if not found
		return c.JSON(fiber.Map{
			"data": fiber.Map{
				"organization_id": orgID,
			},
		})
	}

	return c.JSON(fiber.Map{"data": orgCtx})
}

// UpdateOrgContext updates the organization context
// PATCH /v1/context/org
func (h *Handler) UpdateOrgContext(c fiber.Ctx) error {
	_, orgID, err := getUserAndOrgID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	if orgID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no organization"})
	}

	var updates map[string]interface{}
	if err := c.Bind().JSON(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	ctx := c.Context()
	if err := h.ctxSvc.UpdateOrgContext(ctx, orgID, updates); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "organization context updated"})
}

// ============ User Context ============

// GetUserContext returns the current user's context
// GET /v1/context/user
func (h *Handler) GetUserContext(c fiber.Ctx) error {
	userID, orgID, err := getUserAndOrgID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	ctx := c.Context()
	userCtx, err := h.ctxSvc.GetUserContext(ctx, userID)
	if err != nil {
		// Return empty context if not found
		return c.JSON(fiber.Map{
			"data": fiber.Map{
				"user_id":         userID,
				"organization_id": orgID,
			},
		})
	}

	return c.JSON(fiber.Map{"data": userCtx})
}

// UpdateUserContext updates the current user's context
// PATCH /v1/context/user
func (h *Handler) UpdateUserContext(c fiber.Ctx) error {
	userID, orgID, err := getUserAndOrgID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var updates map[string]interface{}
	if err := c.Bind().JSON(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	ctx := c.Context()
	if err := h.ctxSvc.UpdateUserContext(ctx, userID, orgID, updates); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "user context updated"})
}

// ============ Merged Context ============

// GetMergedContext returns merged org + user context for agents
// GET /v1/context/merged?session_id=xxx
func (h *Handler) GetMergedContext(c fiber.Ctx) error {
	userID, orgID, err := getUserAndOrgID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	sessionID := c.Query("session_id", "")

	ctx := c.Context()
	merged, err := h.ctxSvc.GetMergedContext(ctx, userID, orgID, sessionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Also return formatted prompt context
	promptContext := h.ctxSvc.FormatContextForPrompt(merged)

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"merged":         merged,
			"prompt_context": promptContext,
		},
	})
}

// ============ Conversation History ============

// GetConversationHistory returns conversation history for a session
// GET /v1/context/history?session_id=xxx&limit=20
func (h *Handler) GetConversationHistory(c fiber.Ctx) error {
	userID, _, err := getUserAndOrgID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	sessionID := c.Query("session_id", "")
	limit := fiber.Query[int](c, "limit", 20)

	ctx := c.Context()
	history, err := h.ctxSvc.GetHistory(ctx, userID, sessionID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"data": history})
}

// ClearConversationHistory clears conversation history for a session
// DELETE /v1/context/history?session_id=xxx
func (h *Handler) ClearConversationHistory(c fiber.Ctx) error {
	userID, _, err := getUserAndOrgID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	sessionID := c.Query("session_id")
	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "session_id required"})
	}

	ctx := c.Context()
	if err := h.ctxSvc.ClearSession(ctx, userID, sessionID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "conversation history cleared"})
}
