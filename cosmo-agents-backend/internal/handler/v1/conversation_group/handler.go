// Package conversation_group serves a user's personal groups for AI Inbox
// conversations: create, rename, recolour and delete a group, and file
// conversations in it.
package conversation_group

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	agentRepo "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// Handler serves /v1/conversation-groups.
//
// A group is private to the user who made it. Every group lookup is scoped to
// the caller, and a group that is not theirs is reported as not found rather
// than forbidden, so the endpoints cannot be used to learn which group IDs
// exist.
type Handler struct {
	groups        *conversationRepo.GroupRepository
	conversations *conversationRepo.ConversationRepository
	agents        *agentRepo.AgentRepository
	roles         *roleRepo.RoleRepository
}

// NewHandler creates a Handler.
func NewHandler(
	groups *conversationRepo.GroupRepository,
	conversations *conversationRepo.ConversationRepository,
	agents *agentRepo.AgentRepository,
	roles *roleRepo.RoleRepository,
) *Handler {
	return &Handler{groups: groups, conversations: conversations, agents: agents, roles: roles}
}

type groupRequest struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

type memberRequest struct {
	ConversationID string `json:"conversation_id"`
}

// List handles GET /v1/conversation-groups.
func (h *Handler) List(c fiber.Ctx) error {
	userID, ok := callerID(c)
	if !ok {
		return unauthorized(c)
	}
	groups, err := h.groups.ListByUser(c.Context(), userID)
	if err != nil {
		return serverError(c, "Failed to load groups", err)
	}
	if groups == nil {
		groups = []conversationRepo.GroupWithCount{}
	}
	return c.JSON(schema.SuccessResponse(groups))
}

// Create handles POST /v1/conversation-groups.
func (h *Handler) Create(c fiber.Ctx) error {
	userID, ok := callerID(c)
	if !ok {
		return unauthorized(c)
	}
	var req groupRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err.Error())
	}
	if req.Name == nil {
		return badRequest(c, "Group name is required", "")
	}
	name, ok := domain.NormalizeGroupName(*req.Name)
	if !ok {
		return badRequest(c, "Group name must be 1 to 50 characters", "")
	}
	color := ""
	if req.Color != nil {
		color = *req.Color
	}
	color, ok = domain.NormalizeGroupColor(color)
	if !ok {
		return badRequest(c, "Unknown group colour", "")
	}

	group := &domain.ConversationGroup{UserID: userID, Name: name, Color: color}
	if err := h.groups.Create(c.Context(), group); err != nil {
		if errors.Is(err, conversationRepo.ErrGroupNameTaken) {
			return conflict(c)
		}
		return serverError(c, "Failed to create group", err)
	}
	return c.Status(fiber.StatusCreated).JSON(schema.SuccessResponse(group))
}

// Update handles PATCH /v1/conversation-groups/:id. Name and colour are both
// optional; a field left out keeps its value.
func (h *Handler) Update(c fiber.Ctx) error {
	group, userID, errResp := h.ownedGroup(c)
	if group == nil {
		return errResp
	}
	var req groupRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err.Error())
	}
	if req.Name != nil {
		name, ok := domain.NormalizeGroupName(*req.Name)
		if !ok {
			return badRequest(c, "Group name must be 1 to 50 characters", "")
		}
		group.Name = name
	}
	if req.Color != nil {
		color, ok := domain.NormalizeGroupColor(*req.Color)
		if !ok {
			return badRequest(c, "Unknown group colour", "")
		}
		group.Color = color
	}
	group.UserID = userID
	if err := h.groups.Update(c.Context(), group); err != nil {
		if errors.Is(err, conversationRepo.ErrGroupNameTaken) {
			return conflict(c)
		}
		return serverError(c, "Failed to update group", err)
	}
	return c.JSON(schema.SuccessResponse(group))
}

// Delete handles DELETE /v1/conversation-groups/:id. The conversations in the
// group are not touched; only the group and its memberships go.
func (h *Handler) Delete(c fiber.Ctx) error {
	userID, ok := callerID(c)
	if !ok {
		return unauthorized(c)
	}
	groupID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c, "Invalid group ID", err.Error())
	}
	deleted, err := h.groups.Delete(c.Context(), groupID, userID)
	if err != nil {
		return serverError(c, "Failed to delete group", err)
	}
	if !deleted {
		return notFound(c, "Group not found")
	}
	return c.JSON(schema.SuccessResponse("Group deleted"))
}

// AddConversation handles POST /v1/conversation-groups/:id/conversations.
func (h *Handler) AddConversation(c fiber.Ctx) error {
	group, userID, errResp := h.ownedGroup(c)
	if group == nil {
		return errResp
	}
	var req memberRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err.Error())
	}
	conversationID, err := uuid.Parse(req.ConversationID)
	if err != nil {
		return badRequest(c, "Invalid conversation ID", err.Error())
	}
	visible, err := h.canSeeConversation(c.Context(), userID, conversationID)
	if err != nil {
		return serverError(c, "Failed to check conversation access", err)
	}
	if !visible {
		return notFound(c, "Conversation not found")
	}
	if err := h.groups.AddConversation(c.Context(), group.ID, conversationID); err != nil {
		return serverError(c, "Failed to add conversation to group", err)
	}
	return c.JSON(schema.SuccessResponse("Conversation added to group"))
}

// RemoveConversation handles
// DELETE /v1/conversation-groups/:id/conversations/:conversation_id.
func (h *Handler) RemoveConversation(c fiber.Ctx) error {
	group, _, errResp := h.ownedGroup(c)
	if group == nil {
		return errResp
	}
	conversationID, err := uuid.Parse(c.Params("conversation_id"))
	if err != nil {
		return badRequest(c, "Invalid conversation ID", err.Error())
	}
	if err := h.groups.RemoveConversation(c.Context(), group.ID, conversationID); err != nil {
		return serverError(c, "Failed to remove conversation from group", err)
	}
	return c.JSON(schema.SuccessResponse("Conversation removed from group"))
}

// ownedGroup loads the :id group if it is the caller's. On failure it returns
// a nil group and the response already written.
func (h *Handler) ownedGroup(c fiber.Ctx) (*domain.ConversationGroup, uuid.UUID, error) {
	userID, ok := callerID(c)
	if !ok {
		return nil, uuid.Nil, unauthorized(c)
	}
	groupID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return nil, uuid.Nil, badRequest(c, "Invalid group ID", err.Error())
	}
	group, err := h.groups.FindOwned(c.Context(), groupID, userID)
	if err != nil {
		return nil, uuid.Nil, serverError(c, "Failed to load group", err)
	}
	if group == nil {
		return nil, uuid.Nil, notFound(c, "Group not found")
	}
	return group, userID, nil
}

// canSeeConversation mirrors who can open the conversation in the inbox: its
// owner, the owner of the agent it came through, or a member of that agent's
// organisation. Filing a conversation one cannot see would otherwise confirm
// that its ID exists.
func (h *Handler) canSeeConversation(ctx context.Context, userID, conversationID uuid.UUID) (bool, error) {
	conv, err := h.conversations.FindByID(ctx, conversationID)
	if err != nil || conv == nil {
		return false, nil
	}
	if conv.UserID == userID {
		return true, nil
	}
	if conv.AgentID == nil {
		return false, nil
	}
	agent, err := h.agents.FindByID(ctx, *conv.AgentID)
	if err != nil || agent == nil {
		return false, nil
	}
	if agent.UserID == userID {
		return true, nil
	}
	if agent.OrganizationID == nil {
		return false, nil
	}
	role, err := h.roles.FindByUserAndOrganization(ctx, userID, *agent.OrganizationID)
	if err != nil {
		return false, err
	}
	return role != nil, nil
}

func callerID(c fiber.Ctx) (uuid.UUID, bool) {
	id, ok := c.Locals("user_id").(uuid.UUID)
	return id, ok && id != uuid.Nil
}

func unauthorized(c fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(fiber.StatusUnauthorized, "User not authenticated", ""))
}

func badRequest(c fiber.Ctx, msg, detail string) error {
	return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(fiber.StatusBadRequest, msg, detail))
}

func notFound(c fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(fiber.StatusNotFound, msg, ""))
}

func conflict(c fiber.Ctx) error {
	return c.Status(fiber.StatusConflict).JSON(schema.ErrorResponse(fiber.StatusConflict, "You already have a group with this name", ""))
}

func serverError(c fiber.Ctx, msg string, err error) error {
	return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(fiber.StatusInternalServerError, msg, err.Error()))
}
