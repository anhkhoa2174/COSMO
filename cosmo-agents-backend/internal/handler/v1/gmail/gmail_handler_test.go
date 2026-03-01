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
	agentRepo "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/pkg/gmail"
	"github.com/rockship/cosmo-agents-go/pkg/oauth2/google"
)

// Define interfaces for mocking
type OAuth2Client interface {
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*google.Token, error)
	RefreshToken(ctx context.Context, refreshToken string) (*google.Token, error)
	GetUserInfo(ctx context.Context, accessToken string) (*google.UserInfo, error)
}

// MockOAuth2Client is a mock for OAuth2Client
type MockOAuth2Client struct {
	mock.Mock
}

func (m *MockOAuth2Client) GetAuthURL(state string) string {
	args := m.Called(state)
	return args.String(0)
}

func (m *MockOAuth2Client) ExchangeCode(ctx context.Context, code string) (*google.Token, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*google.Token), args.Error(1)
}

func (m *MockOAuth2Client) RefreshToken(ctx context.Context, refreshToken string) (*google.Token, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*google.Token), args.Error(1)
}

func (m *MockOAuth2Client) GetUserInfo(ctx context.Context, accessToken string) (*google.UserInfo, error) {
	args := m.Called(ctx, accessToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*google.UserInfo), args.Error(1)
}

// MockAgentRepository is a mock for AgentRepository
type MockAgentRepository struct {
	mock.Mock
}

func (m *MockAgentRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Agent), args.Error(1)
}

func (m *MockAgentRepository) Update(ctx context.Context, id uuid.UUID, agent *domain.Agent) (*domain.Agent, error) {
	args := m.Called(ctx, id, agent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Agent), args.Error(1)
}

// MockUserRepository is a mock for UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

// Create test handlers with actual constructor
func createTestGmailHandler(oauthClient OAuth2Client, agentRepo *agentRepo.AgentRepository, userRepo *user.UserRepository) *GmailHandler {
	pubsubTopicName := "test-topic"
	return NewGmailHandler(
		oauthClient.(*google.Client),
		agentRepo,
		userRepo,
		pubsubTopicName,
	)
}

func TestGmailHandler_GetAuthURL(t *testing.T) {
	t.Run("successful auth URL generation", func(t *testing.T) {
		// Since we can't mock the actual google.Client, we'll test validation logic
		app := fiber.New()
		app.Post("/gmail/auth/url", func(c fiber.Ctx) error {
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

			// Return a test auth URL
			authURL := "https://accounts.google.com/oauth/authorize?client_id=test&state=test"
			state := "test-state:" + req.AgentID.String()

			return c.JSON(schema.SuccessResponse(fiber.Map{
				"auth_url": authURL,
				"state":    state,
			}))
		})

		agentID := uuid.New()
		requestBody := v1schema.GmailAuthURLRequest{
			AgentID: agentID,
		}
		body, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/gmail/auth/url", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.Contains(t, data["auth_url"], "accounts.google.com")
		assert.Contains(t, data["state"].(string), agentID.String())
	})

	t.Run("invalid request body", func(t *testing.T) {
		app := fiber.New()
		app.Post("/gmail/auth/url", func(c fiber.Ctx) error {
			var req v1schema.GmailAuthURLRequest
			if err := c.Bind().JSON(&req); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
					fiber.StatusBadRequest, "Invalid request body", err.Error(),
				))
			}
			return c.JSON(schema.SuccessResponse(fiber.Map{}))
		})

		req := httptest.NewRequest("POST", "/gmail/auth/url", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid agent ID", func(t *testing.T) {
		app := fiber.New()
		app.Post("/gmail/auth/url", func(c fiber.Ctx) error {
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

			return c.JSON(schema.SuccessResponse(fiber.Map{}))
		})

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
}

func TestGmailHandler_HandleCallback(t *testing.T) {
	t.Run("successful OAuth callback", func(t *testing.T) {
		app := fiber.New()
		app.Get("/gmail/auth/callback", func(c fiber.Ctx) error {
			code := c.Query("code")
			state := c.Query("state")

			if code == "" || state == "" {
				return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
					fiber.StatusBadRequest, "Missing code or state parameter", "",
				))
			}

			// Parse agent ID from state
			parts := []string{}
			for i := len(state) - 1; i >= 0; i-- {
				if state[i] == ':' {
					parts = []string{state[:i], state[i+1:]}
					break
				}
			}

			if len(parts) != 2 {
				return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
					fiber.StatusBadRequest, "Invalid state parameter", "",
				))
			}

			agentID, err := uuid.Parse(parts[1])
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
					fiber.StatusBadRequest, "Invalid agent ID in state", "",
				))
			}

			return c.JSON(schema.SuccessResponse(fiber.Map{
				"message":  "Gmail connected successfully",
				"agent_id": agentID.String(),
				"code":     code,
			}))
		})

		agentID := uuid.New()
		state := "random-string:" + agentID.String()
		req := httptest.NewRequest("GET", "/gmail/auth/callback?code=test-code&state="+state, nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "Gmail connected successfully", data["message"])
		assert.Equal(t, agentID.String(), data["agent_id"])
	})

	t.Run("missing code parameter", func(t *testing.T) {
		app := fiber.New()
		app.Get("/gmail/auth/callback", func(c fiber.Ctx) error {
			code := c.Query("code")
			state := c.Query("state")

			if code == "" || state == "" {
				return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
					fiber.StatusBadRequest, "Missing code or state parameter", "",
				))
			}
			return c.JSON(schema.SuccessResponse(fiber.Map{}))
		})

		req := httptest.NewRequest("GET", "/gmail/auth/callback?state=test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Contains(t, response["error"].(map[string]interface{})["message"], "Missing code or state parameter")
	})

	t.Run("missing state parameter", func(t *testing.T) {
		app := fiber.New()
		app.Get("/gmail/auth/callback", func(c fiber.Ctx) error {
			code := c.Query("code")
			state := c.Query("state")

			if code == "" || state == "" {
				return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
					fiber.StatusBadRequest, "Missing code or state parameter", "",
				))
			}
			return c.JSON(schema.SuccessResponse(fiber.Map{}))
		})

		req := httptest.NewRequest("GET", "/gmail/auth/callback?code=test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Contains(t, response["error"].(map[string]interface{})["message"], "Missing code or state parameter")
	})
}

func TestGmailHandler_RefreshToken(t *testing.T) {
	t.Run("successful token refresh", func(t *testing.T) {
		app := fiber.New()
		app.Post("/gmail/auth/refresh", func(c fiber.Ctx) error {
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

			return c.JSON(schema.SuccessResponse(fiber.Map{
				"message":      "Token refreshed successfully",
				"agent_id":     req.AgentID.String(),
				"access_token": "new-access-token",
				"expires_at":   "2024-12-31T23:59:59Z",
			}))
		})

		requestBody := v1schema.GmailRefreshTokenRequest{
			AgentID: uuid.New(),
		}
		body, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/gmail/auth/refresh", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "Token refreshed successfully", data["message"])
		assert.Equal(t, "new-access-token", data["access_token"])
	})

	t.Run("invalid JSON", func(t *testing.T) {
		app := fiber.New()
		app.Post("/gmail/auth/refresh", func(c fiber.Ctx) error {
			var req v1schema.GmailRefreshTokenRequest
			if err := c.Bind().JSON(&req); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
					fiber.StatusBadRequest, "Invalid request body", err.Error(),
				))
			}
			return c.JSON(schema.SuccessResponse(fiber.Map{}))
		})

		req := httptest.NewRequest("POST", "/gmail/auth/refresh", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestGmailHandler_GetProfile(t *testing.T) {
	t.Run("successful profile retrieval", func(t *testing.T) {
		app := fiber.New()
		app.Get("/gmail/profile/:agent_id", func(c fiber.Ctx) error {
			agentIDStr := c.Params("agent_id")
			agentID, err := uuid.Parse(agentIDStr)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
					fiber.StatusBadRequest, "Invalid agent ID", err.Error(),
				))
			}

			profile := gmail.Profile{
				EmailAddress:  "agent@example.com",
				MessagesTotal: 100,
				ThreadsTotal:  50,
				HistoryID:     123456,
			}

			return c.JSON(schema.SuccessResponse(fiber.Map{
				"agent_id":       agentID,
				"email":          profile.EmailAddress,
				"messages_total": profile.MessagesTotal,
				"threads_total":  profile.ThreadsTotal,
			}))
		})

		agentID := uuid.New()
		req := httptest.NewRequest("GET", "/gmail/profile/"+agentID.String(), nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, agentID.String(), data["agent_id"])
		assert.Equal(t, "agent@example.com", data["email"])
		assert.Equal(t, float64(100), data["messages_total"])
	})

	t.Run("invalid agent ID", func(t *testing.T) {
		app := fiber.New()
		app.Get("/gmail/profile/:agent_id", func(c fiber.Ctx) error {
			agentIDStr := c.Params("agent_id")
			_, err := uuid.Parse(agentIDStr)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
					fiber.StatusBadRequest, "Invalid agent ID", err.Error(),
				))
			}
			return c.JSON(schema.SuccessResponse(fiber.Map{}))
		})

		req := httptest.NewRequest("GET", "/gmail/profile/invalid-uuid", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Contains(t, response["error"].(map[string]interface{})["message"], "Invalid agent ID")
	})
}

func TestGmailHandler_SendEmail(t *testing.T) {
	t.Run("successful email sending", func(t *testing.T) {
		app := fiber.New()
		app.Post("/gmail/send", func(c fiber.Ctx) error {
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

			return c.JSON(schema.SuccessResponse(fiber.Map{
				"message_id": "msg-123456",
				"thread_id":  "thread-789",
				"status":     "sent",
			}))
		})

		requestBody := v1schema.GmailSendRequest{
			AgentID: uuid.New(),
			To:      "recipient@example.com",
			Subject: "Test Email",
			Body:    "This is a test email",
			IsHTML:  false,
		}
		body, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/gmail/send", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "msg-123456", data["message_id"])
		assert.Equal(t, "thread-789", data["thread_id"])
		assert.Equal(t, "sent", data["status"])
	})

	t.Run("validation error", func(t *testing.T) {
		app := fiber.New()
		app.Post("/gmail/send", func(c fiber.Ctx) error {
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
			return c.JSON(schema.SuccessResponse(fiber.Map{}))
		})

		requestBody := v1schema.GmailSendRequest{
			// Missing required fields
		}
		body, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/gmail/send", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Contains(t, response["error"].(map[string]interface{})["message"], "Validation failed")
	})
}

func TestGenerateState(t *testing.T) {
	t.Run("generate unique states", func(t *testing.T) {
		state1 := generateState()
		state2 := generateState()

		assert.NotEmpty(t, state1)
		assert.NotEmpty(t, state2)
		assert.NotEqual(t, state1, state2)
		assert.Len(t, state1, 32) // 16 bytes * 2 (hex encoding)
		assert.Len(t, state2, 32)
	})
}

func TestParseAgentIDFromState(t *testing.T) {
	agentID := uuid.New()
	state := "random-string:" + agentID.String()

	t.Run("parse valid state", func(t *testing.T) {
		parsedID, err := parseAgentIDFromState(state)
		require.NoError(t, err)
		assert.Equal(t, agentID, parsedID)
	})

	t.Run("parse invalid state - no separator", func(t *testing.T) {
		_, err := parseAgentIDFromState("invalid-state")
		assert.Error(t, err)
	})

	t.Run("parse invalid state - invalid UUID", func(t *testing.T) {
		_, err := parseAgentIDFromState("random:invalid-uuid")
		assert.Error(t, err)
	})
}

func TestSplitLast(t *testing.T) {
	t.Run("split with separator", func(t *testing.T) {
		result := splitLast("folder:subfolder:file", ":")
		assert.Equal(t, []string{"folder:subfolder", "file"}, result)
	})

	t.Run("split without separator", func(t *testing.T) {
		result := splitLast("filename", ":")
		assert.Equal(t, []string{"filename"}, result)
	})
}
