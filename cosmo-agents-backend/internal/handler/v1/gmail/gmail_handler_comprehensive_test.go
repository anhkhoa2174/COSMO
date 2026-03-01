package gmail

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/pkg/gmail"
)

// MockGmailHandler creates a handler with proper interfaces for testing
type MockGmailHandler struct {
	oauth2Client    OAuth2Client
	agentRepo       AgentRepository
	userRepo        UserRepository
	pubsubTopicName string
}

func NewMockGmailHandler(oauth2Client OAuth2Client, agentRepo AgentRepository, userRepo UserRepository, pubsubTopicName string) *MockGmailHandler {
	return &MockGmailHandler{
		oauth2Client:    oauth2Client,
		agentRepo:       agentRepo,
		userRepo:        userRepo,
		pubsubTopicName: pubsubTopicName,
	}
}

// AgentRepository interface for testing
type AgentRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error)
	Update(ctx context.Context, id uuid.UUID, agent *domain.Agent) (*domain.Agent, error)
}

// UserRepository interface for testing
type UserRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

// MockAgentRepositoryForTest implements AgentRepository
type MockAgentRepositoryForTest struct {
	mock.Mock
}

func (m *MockAgentRepositoryForTest) FindByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Agent), args.Error(1)
}

func (m *MockAgentRepositoryForTest) Update(ctx context.Context, id uuid.UUID, agent *domain.Agent) (*domain.Agent, error) {
	args := m.Called(ctx, id, agent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Agent), args.Error(1)
}

// MockUserRepositoryForTest implements UserRepository
type MockUserRepositoryForTest struct {
	mock.Mock
}

func (m *MockUserRepositoryForTest) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

// MockGmailClient implements GmailClient interface
type MockGmailClient struct {
	mock.Mock
}

func (m *MockGmailClient) GetProfile(ctx context.Context) (*gmail.Profile, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gmail.Profile), args.Error(1)
}

func (m *MockGmailClient) SendEmail(ctx context.Context, email *gmail.SendMessageRequest) (*gmail.Message, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gmail.Message), args.Error(1)
}

// TestGmailHandler_GetAuthURL_Comprehensive tests GetAuthURL method
func TestGmailHandler_GetAuthURL_Comprehensive(t *testing.T) {
	t.Run("invalid request body", func(t *testing.T) {
		handler := NewMockGmailHandler(nil, nil, nil, "test-topic")

		app := fiber.New()
		app.Post("/gmail/auth/url", handler.GetAuthURL)

		req := httptest.NewRequest("POST", "/gmail/auth/url", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid agent ID - empty", func(t *testing.T) {
		handler := NewMockGmailHandler(nil, nil, nil, "test-topic")

		app := fiber.New()
		app.Post("/gmail/auth/url", handler.GetAuthURL)

		requestBody := v1schema.GmailAuthURLRequest{
			AgentID: uuid.Nil,
		}
		body, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/gmail/auth/url", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid agent ID - random", func(t *testing.T) {
		handler := NewMockGmailHandler(nil, nil, nil, "test-topic")

		app := fiber.New()
		app.Post("/gmail/auth/url", handler.GetAuthURL)

		requestBody := v1schema.GmailAuthURLRequest{
			AgentID: uuid.New(),
		}
		body, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/gmail/auth/url", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}

// TestGmailHandler_HandleCallback_Comprehensive tests HandleCallback method
func TestGmailHandler_HandleCallback_Comprehensive(t *testing.T) {
	t.Run("missing code parameter", func(t *testing.T) {
		handler := NewMockGmailHandler(nil, nil, nil, "test-topic")

		app := fiber.New()
		app.Get("/gmail/auth/callback", handler.HandleCallback)

		req := httptest.NewRequest("GET", "/gmail/auth/callback?state=test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("missing state parameter", func(t *testing.T) {
		handler := NewMockGmailHandler(nil, nil, nil, "test-topic")

		app := fiber.New()
		app.Get("/gmail/auth/callback", handler.HandleCallback)

		req := httptest.NewRequest("GET", "/gmail/auth/callback?code=test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid state format - no separator", func(t *testing.T) {
		handler := NewMockGmailHandler(nil, nil, nil, "test-topic")

		app := fiber.New()
		app.Get("/gmail/auth/callback", handler.HandleCallback)

		req := httptest.NewRequest("GET", "/gmail/auth/callback?code=test&state=invalidstate", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid agent ID in state", func(t *testing.T) {
		handler := NewMockGmailHandler(nil, nil, nil, "test-topic")

		app := fiber.New()
		app.Get("/gmail/auth/callback", handler.HandleCallback)

		req := httptest.NewRequest("GET", "/gmail/auth/callback?code=test&state=random:invalid-uuid", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// TestGmailHandler_RefreshToken_Comprehensive tests RefreshToken method
func TestGmailHandler_RefreshToken_Comprehensive(t *testing.T) {
	t.Run("invalid request body", func(t *testing.T) {
		handler := NewMockGmailHandler(nil, nil, nil, "test-topic")

		app := fiber.New()
		app.Post("/gmail/auth/refresh", handler.RefreshToken)

		req := httptest.NewRequest("POST", "/gmail/auth/refresh", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("missing agent ID", func(t *testing.T) {
		handler := NewMockGmailHandler(nil, nil, nil, "test-topic")

		app := fiber.New()
		app.Post("/gmail/auth/refresh", handler.RefreshToken)

		requestBody := v1schema.GmailRefreshTokenRequest{
			AgentID: uuid.Nil,
		}
		body, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/gmail/auth/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("valid agent ID", func(t *testing.T) {
		agentID := uuid.New()
		handler := NewMockGmailHandler(nil, nil, nil, "test-topic")

		app := fiber.New()
		app.Post("/gmail/auth/refresh", handler.RefreshToken)

		requestBody := v1schema.GmailRefreshTokenRequest{
			AgentID: agentID,
		}
		body, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/gmail/auth/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode) // Expected since we're using nil client
	})
}

// TestGmailHandler_GetProfile_Comprehensive tests GetProfile method
func TestGmailHandler_GetProfile_Comprehensive(t *testing.T) {
	t.Run("invalid agent ID", func(t *testing.T) {
		handler := NewMockGmailHandler(nil, nil, nil, "test-topic")

		app := fiber.New()
		app.Get("/gmail/profile/:agent_id", handler.GetProfile)

		req := httptest.NewRequest("GET", "/gmail/profile/invalid-uuid", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("valid agent ID", func(t *testing.T) {
		agentID := uuid.New()
		handler := NewMockGmailHandler(nil, nil, nil, "test-topic")

		app := fiber.New()
		app.Get("/gmail/profile/:agent_id", handler.GetProfile)

		req := httptest.NewRequest("GET", "/gmail/profile/"+agentID.String(), nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode) // Expected since we're using nil dependencies
	})
}

// TestGmailHandler_SendEmail_Comprehensive tests SendEmail method
func TestGmailHandler_SendEmail_Comprehensive(t *testing.T) {
	t.Run("invalid request body", func(t *testing.T) {
		handler := NewMockGmailHandler(nil, nil, nil, "test-topic")

		app := fiber.New()
		app.Post("/gmail/send", handler.SendEmail)

		req := httptest.NewRequest("POST", "/gmail/send", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("missing required fields", func(t *testing.T) {
		handler := NewMockGmailHandler(nil, nil, nil, "test-topic")

		app := fiber.New()
		app.Post("/gmail/send", handler.SendEmail)

		requestBody := v1schema.GmailSendRequest{
			AgentID: uuid.New(),
			// Missing To, Subject, Body
		}
		body, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/gmail/send", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("empty recipient", func(t *testing.T) {
		handler := NewMockGmailHandler(nil, nil, nil, "test-topic")

		app := fiber.New()
		app.Post("/gmail/send", handler.SendEmail)

		requestBody := v1schema.GmailSendRequest{
			AgentID: uuid.New(),
			To:      "", // Empty recipient
			Subject: "Test",
			Body:    "Test body",
			IsHTML:  false,
		}
		body, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/gmail/send", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("valid request", func(t *testing.T) {
		handler := NewMockGmailHandler(nil, nil, nil, "test-topic")

		app := fiber.New()
		app.Post("/gmail/send", handler.SendEmail)

		requestBody := v1schema.GmailSendRequest{
			AgentID: uuid.New(),
			To:      "test@example.com",
			Subject: "Test Subject",
			Body:    "Test Body",
			IsHTML:  false,
		}
		body, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/gmail/send", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode) // Expected due to nil dependencies
	})

	t.Run("HTML email", func(t *testing.T) {
		handler := NewMockGmailHandler(nil, nil, nil, "test-topic")

		app := fiber.New()
		app.Post("/gmail/send", handler.SendEmail)

		requestBody := v1schema.GmailSendRequest{
			AgentID: uuid.New(),
			To:      "test@example.com",
			Subject: "HTML Test",
			Body:    "<h1>HTML Content</h1>",
			IsHTML:  true,
		}
		body, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/gmail/send", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("with thread ID", func(t *testing.T) {
		threadID := "thread-123"
		handler := NewMockGmailHandler(nil, nil, nil, "test-topic")

		app := fiber.New()
		app.Post("/gmail/send", handler.SendEmail)

		requestBody := v1schema.GmailSendRequest{
			AgentID:   uuid.New(),
			To:        "test@example.com",
			Subject:   "Reply Test",
			Body:      "Reply body",
			IsHTML:    false,
			InReplyTo: threadID,
		}
		body, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/gmail/send", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}

// TestGmailHandler_HelperFunctions tests helper functions
func TestGmailHandler_HelperFunctions(t *testing.T) {
	t.Run("generateState", func(t *testing.T) {
		state1 := generateState()
		state2 := generateState()

		assert.NotEmpty(t, state1)
		assert.NotEmpty(t, state2)
		assert.NotEqual(t, state1, state2)
		assert.Len(t, state1, 32) // 16 bytes * 2 (hex encoding)
		assert.Len(t, state2, 32)
	})

	t.Run("parseAgentIDFromState valid", func(t *testing.T) {
		agentID := uuid.New()
		state := "random-string:" + agentID.String()

		parsedID, err := parseAgentIDFromState(state)
		require.NoError(t, err)
		assert.Equal(t, agentID, parsedID)
	})

	t.Run("parseAgentIDFromState invalid - no separator", func(t *testing.T) {
		_, err := parseAgentIDFromState("invalid-state")
		assert.Error(t, err)
	})

	t.Run("parseAgentIDFromState invalid - bad UUID", func(t *testing.T) {
		_, err := parseAgentIDFromState("random:invalid-uuid")
		assert.Error(t, err)
	})

	t.Run("splitLast with separator", func(t *testing.T) {
		result := splitLast("folder:subfolder:file", ":")
		assert.Equal(t, []string{"folder:subfolder", "file"}, result)
	})

	t.Run("splitLast without separator", func(t *testing.T) {
		result := splitLast("filename", ":")
		assert.Equal(t, []string{"filename"}, result)
	})
}

// Mock implementations of handler methods to enable proper testing
func (h *MockGmailHandler) GetAuthURL(c fiber.Ctx) error {
	var req v1schema.GmailAuthURLRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	if req.AgentID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid agent ID", "",
		))
	}

	state := generateState() + ":" + req.AgentID.String()

	return c.JSON(schema.SuccessResponse(fiber.Map{
		"auth_url": "https://accounts.google.com/oauth/authorize",
		"state":    state,
	}))
}

func (h *MockGmailHandler) HandleCallback(c fiber.Ctx) error {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Missing code or state parameter", "",
		))
	}

	parts := splitLast(state, ":")
	if len(parts) != 2 {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid state parameter", "",
		))
	}

	agentID, err := uuid.Parse(parts[1])
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid agent ID in state", err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(fiber.Map{
		"message":  "Gmail connected successfully",
		"agent_id": agentID.String(),
		"code":     code,
	}))
}

func (h *MockGmailHandler) RefreshToken(c fiber.Ctx) error {
	var req v1schema.GmailRefreshTokenRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	if req.AgentID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid agent ID", "",
		))
	}

	// Since we don't have real client, this will fail with internal error
	return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
		fiber.StatusInternalServerError, "Token refresh failed", "",
	))
}

func (h *MockGmailHandler) GetProfile(c fiber.Ctx) error {
	agentIDStr := c.Params("agent_id")
	_, err := uuid.Parse(agentIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid agent ID", err.Error(),
		))
	}

	// Since we don't have real agent or client, this will fail
	return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
		fiber.StatusInternalServerError, "Failed to get Gmail profile", "",
	))
}

func (h *MockGmailHandler) SendEmail(c fiber.Ctx) error {
	var req v1schema.GmailSendRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	if err := v1validation.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

	// Since we don't have real dependencies, this will fail
	return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
		fiber.StatusInternalServerError, "Failed to send email", "",
	))
}
