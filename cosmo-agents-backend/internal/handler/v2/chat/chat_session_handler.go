// Package chat serves the Ask COSMO conversation history.
//
// The assistant itself runs in the front end, streaming straight from the model
// provider, so this package does not generate anything. Its whole job is to
// remember: open a conversation, list them, read one back, append a completed
// turn, delete one.
package chat

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/schema"

	domain "github.com/rockship/cosmo-agents-go/internal/domain/daily_action"
	dailyActionRepo "github.com/rockship/cosmo-agents-go/internal/repository/daily_action"
)

// Cap on what one turn may store. The assistant's own output is already
// bounded by the model's token limit; this stops a crafted client from using
// the history endpoint as free storage.
const (
	maxMessagesPerAppend = 4
	maxContentChars      = 20_000
)

type Handler struct {
	sessions *dailyActionRepo.ChatSessionRepository
}

func NewHandler(sessions *dailyActionRepo.ChatSessionRepository) *Handler {
	return &Handler{sessions: sessions}
}

func (h *Handler) userID(c fiber.Ctx) (uuid.UUID, bool) {
	if id, ok := c.Locals("user_id").(uuid.UUID); ok {
		return id, true
	}
	// The API-key middleware sets only the camel-case key.
	if id, ok := c.Locals("userID").(uuid.UUID); ok {
		return id, true
	}
	return uuid.Nil, false
}

func (h *Handler) unauthorized(c fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(
		schema.ErrorResponse(fiber.StatusUnauthorized, "Unauthorized", ""),
	)
}

type sessionView struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func viewSession(s domain.ChatSession) sessionView {
	return sessionView{
		ID:        s.ID.String(),
		Title:     s.Title,
		CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// ListSessions returns the caller's conversations, most recent first.
//
// @Summary List Ask COSMO conversations
// @Tags Chat
// @Produce json
// @Success 200 {object} schema.APIResponse[any]
// @Router /v2/chat/sessions [get]
func (h *Handler) ListSessions(c fiber.Ctx) error {
	userID, ok := h.userID(c)
	if !ok {
		return h.unauthorized(c)
	}

	sessions, err := h.sessions.List(c.Context(), userID, 30)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError,
				"Failed to load conversations", err.Error()),
		)
	}

	out := make([]sessionView, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, viewSession(s))
	}
	return c.JSON(schema.SuccessResponse(map[string]any{"sessions": out}))
}

type createSessionRequest struct {
	// FirstQuestion names the conversation. It is optional: a session opened
	// before the first message is typed is simply called "New conversation".
	FirstQuestion string `json:"first_question"`
}

// CreateSession opens a conversation.
//
// @Summary Open an Ask COSMO conversation
// @Tags Chat
// @Accept json
// @Produce json
// @Success 201 {object} schema.APIResponse[any]
// @Router /v2/chat/sessions [post]
func (h *Handler) CreateSession(c fiber.Ctx) error {
	userID, ok := h.userID(c)
	if !ok {
		return h.unauthorized(c)
	}

	var req createSessionRequest
	_ = c.Bind().JSON(&req) // an empty body is a valid request

	session, err := h.sessions.Create(c.Context(), userID, req.FirstQuestion)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError,
				"Failed to open a conversation", err.Error()),
		)
	}
	return c.Status(fiber.StatusCreated).JSON(
		schema.SuccessResponse(viewSession(*session)))
}

type messageView struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// GetMessages reads one conversation back.
//
// @Summary Read an Ask COSMO conversation
// @Tags Chat
// @Produce json
// @Param session_id path string true "Session ID"
// @Success 200 {object} schema.APIResponse[any]
// @Failure 404 {object} schema.APIResponse[any]
// @Router /v2/chat/sessions/{session_id}/messages [get]
func (h *Handler) GetMessages(c fiber.Ctx) error {
	userID, ok := h.userID(c)
	if !ok {
		return h.unauthorized(c)
	}

	sessionID, err := uuid.Parse(c.Params("session_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			schema.ErrorResponse(fiber.StatusBadRequest, "Invalid session id", err.Error()),
		)
	}

	// A session belonging to someone else is reported as missing rather than
	// forbidden, so the endpoint cannot be used to discover that an id exists.
	session, err := h.sessions.Get(c.Context(), userID, sessionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError,
				"Failed to load the conversation", err.Error()),
		)
	}
	if session == nil {
		return c.Status(fiber.StatusNotFound).JSON(
			schema.ErrorResponse(fiber.StatusNotFound, "Conversation not found", ""),
		)
	}

	messages, err := h.sessions.Messages(c.Context(), userID, sessionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError,
				"Failed to load messages", err.Error()),
		)
	}

	out := make([]messageView, 0, len(messages))
	for _, m := range messages {
		out = append(out, messageView{
			Role:      m.Role,
			Content:   m.Content,
			CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return c.JSON(schema.SuccessResponse(map[string]any{
		"session":  viewSession(*session),
		"messages": out,
	}))
}

type appendRequest struct {
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

// AppendMessages records a completed turn.
//
// @Summary Append to an Ask COSMO conversation
// @Tags Chat
// @Accept json
// @Produce json
// @Param session_id path string true "Session ID"
// @Success 204
// @Router /v2/chat/sessions/{session_id}/messages [post]
func (h *Handler) AppendMessages(c fiber.Ctx) error {
	userID, ok := h.userID(c)
	if !ok {
		return h.unauthorized(c)
	}

	sessionID, err := uuid.Parse(c.Params("session_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			schema.ErrorResponse(fiber.StatusBadRequest, "Invalid session id", err.Error()),
		)
	}

	var req appendRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			schema.ErrorResponse(fiber.StatusBadRequest, "Invalid request body", err.Error()),
		)
	}
	if len(req.Messages) == 0 || len(req.Messages) > maxMessagesPerAppend {
		return c.Status(fiber.StatusBadRequest).JSON(
			schema.ErrorResponse(fiber.StatusBadRequest,
				"A turn carries between one and four messages", ""),
		)
	}

	rows := make([]domain.ChatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		// Only the two conversational roles are stored. Accepting `system`
		// would let a client plant instructions that a later turn replays back
		// to the model as though the server had authored them.
		if role != "user" && role != "assistant" {
			return c.Status(fiber.StatusBadRequest).JSON(
				schema.ErrorResponse(fiber.StatusBadRequest,
					"Only user and assistant messages are stored", ""),
			)
		}
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		if len(content) > maxContentChars {
			content = content[:maxContentChars]
		}
		rows = append(rows, domain.ChatMessage{Role: role, Content: content})
	}
	if len(rows) == 0 {
		return c.SendStatus(fiber.StatusNoContent)
	}

	if err := h.sessions.Append(c.Context(), userID, sessionID, rows); err != nil {
		// Append fails when the session does not exist for this user, which is
		// the same answer as "not found" from the caller's point of view.
		return c.Status(fiber.StatusNotFound).JSON(
			schema.ErrorResponse(fiber.StatusNotFound,
				"Conversation not found", err.Error()),
		)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// DeleteSession removes a conversation and its messages.
//
// @Summary Delete an Ask COSMO conversation
// @Tags Chat
// @Param session_id path string true "Session ID"
// @Success 204
// @Router /v2/chat/sessions/{session_id} [delete]
func (h *Handler) DeleteSession(c fiber.Ctx) error {
	userID, ok := h.userID(c)
	if !ok {
		return h.unauthorized(c)
	}

	sessionID, err := uuid.Parse(c.Params("session_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			schema.ErrorResponse(fiber.StatusBadRequest, "Invalid session id", err.Error()),
		)
	}

	if err := h.sessions.Delete(c.Context(), userID, sessionID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError,
				"Failed to delete the conversation", err.Error()),
		)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
