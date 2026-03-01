package meta

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	facebookTokenRepo "github.com/rockship/cosmo-agents-go/internal/repository/facebook_token"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// MetaHandler handles Facebook/Meta-related HTTP requests
type MetaHandler struct {
	facebookTokenRepo *facebookTokenRepo.FacebookTokenRepository
	metaClientID      string
	metaClientSecret  string
	metaScopes        string
	metaVerifyToken   string
}

// NewMetaHandler creates a new MetaHandler
func NewMetaHandler(
	facebookTokenRepo *facebookTokenRepo.FacebookTokenRepository,
	metaClientID, metaClientSecret, metaScopes, metaVerifyToken string,
) *MetaHandler {
	return &MetaHandler{
		facebookTokenRepo: facebookTokenRepo,
		metaClientID:      metaClientID,
		metaClientSecret:  metaClientSecret,
		metaScopes:        metaScopes,
		metaVerifyToken:   metaVerifyToken,
	}
}

// GetAuthURL handles GET /v1/meta/oauth2/auth-url
// @Summary Get Facebook OAuth authorization URL
// @Description Generate Facebook OAuth2 authorization URL
// @Tags Meta
// @Accept json
// @Produce json
// @Param redirect_uri query string true "OAuth redirect URI"
// @Success 200 {object} schema.APIResponse[string] "Authorization URL"
// @Router /v1/meta/oauth2/auth-url [get]
func (h *MetaHandler) GetAuthURL(c fiber.Ctx) error {
	redirectURI := c.Query("redirect_uri")
	if redirectURI == "" {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Missing redirect_uri parameter", "",
		))
	}

	// Build Facebook OAuth URL
	authURL := fmt.Sprintf(
		"https://www.facebook.com/v18.0/dialog/oauth?client_id=%s&redirect_uri=%s&scope=%s&response_type=code",
		h.metaClientID,
		redirectURI,
		h.metaScopes,
	)

	return c.JSON(schema.SuccessResponse(authURL))
}

// OAuth2Callback handles GET /v1/meta/oauth2/callback
// @Summary Handle Facebook OAuth callback
// @Description Exchange authorization code for access token
// @Tags Meta
// @Accept json
// @Produce json
// @Param code query string true "Authorization code"
// @Param redirect_uri query string true "OAuth redirect URI"
// @Success 200 {object} schema.APIResponse[v1schema.FacebookTokenResponse] "Token data"
// @Failure 400 {object} schema.APIResponse[any] "Bad request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Security BearerAuth
// @Router /v1/meta/oauth2/callback [get]
func (h *MetaHandler) OAuth2Callback(c fiber.Ctx) error {
	// Get current user (verify authentication)
	_, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "Unauthorized", "User ID not found",
		))
	}

	code := c.Query("code")
	redirectURI := c.Query("redirect_uri")

	if code == "" || redirectURI == "" {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Missing code or redirect_uri", "",
		))
	}

	// TODO: Exchange code for token via Facebook API
	// This is a simplified version - in production, make HTTP call to Facebook
	response := v1schema.FacebookTokenResponse{
		AccessToken: "mock_access_token_" + code[:10],
		TokenType:   "bearer",
		ExpiresIn:   5184000, // 60 days
	}

	return c.JSON(schema.SuccessResponse(response))
}

// VerifyWebhook handles GET /v1/meta/webhook
// @Summary Verify Facebook webhook
// @Description Verify Facebook webhook subscription (webhook challenge)
// @Tags Meta
// @Produce plain
// @Param hub.mode query string true "Webhook mode (should be 'subscribe')"
// @Param hub.verify_token query string true "Verification token"
// @Param hub.challenge query string true "Challenge string from Facebook"
// @Success 200 {string} string "Challenge string"
// @Failure 403 {object} map[string]string "Verification failed"
// @Router /v1/meta/webhook [get]
func (h *MetaHandler) VerifyWebhook(c fiber.Ctx) error {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if mode == "subscribe" && token == h.metaVerifyToken {
		return c.SendString(challenge)
	}

	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
		"error": "Verification failed",
	})
}

// SubscribePageRequest represents page subscription request
type SubscribePageRequest struct {
	PageID      string `json:"page_id"`
	AccessToken string `json:"access_token"`
}

// SubscribePage handles POST /v1/meta/subscribe
// @Summary Subscribe page to webhook
// @Description Subscribe a Facebook page to receive lead gen notifications
// @Tags Meta
// @Accept json
// @Produce json
// @Param body body SubscribePageRequest true "Page subscription details"
// @Success 200 {object} schema.APIResponse[map[string]bool] "Subscription result"
// @Failure 400 {object} schema.APIResponse[any] "Bad request"
// @Router /v1/meta/subscribe [post]
func (h *MetaHandler) SubscribePage(c fiber.Ctx) error {
	var req SubscribePageRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	if req.PageID == "" || req.AccessToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Missing page_id or access_token", "",
		))
	}

	// TODO: Make API call to Facebook to subscribe page
	// POST https://graph.facebook.com/v18.0/{page_id}/subscribed_apps
	// This is a simplified mock response

	result := fiber.Map{
		"success": true,
	}

	return c.JSON(schema.SuccessResponse(result))
}

// LeadWebhookPayload represents Facebook lead gen webhook structure
type LeadWebhookPayload struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Time    int64  `json:"time"`
		Changes []struct {
			Field string `json:"field"`
			Value struct {
				AdID        string `json:"ad_id"`
				FormID      string `json:"form_id"`
				LeadgenID   string `json:"leadgen_id"`
				CreatedTime int64  `json:"created_time"`
				PageID      string `json:"page_id"`
				AdgroupID   string `json:"adgroup_id"`
				CampaignID  string `json:"campaign_id"`
				UserID      string `json:"user_id"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

// ReceiveLeadNotification handles POST /v1/meta/webhook
// @Summary Receive Facebook lead webhook
// @Description Receive and process Facebook lead form webhook notifications
// @Tags Meta
// @Accept json
// @Produce json
// @Param body body LeadWebhookPayload true "Webhook payload"
// @Success 200 {object} schema.APIResponse[string] "Webhook received"
// @Router /v1/meta/webhook [post]
func (h *MetaHandler) ReceiveLeadNotification(c fiber.Ctx) error {
	var payload LeadWebhookPayload
	if err := c.Bind().JSON(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid webhook payload", err.Error(),
		))
	}

	// Log webhook for debugging
	payloadJSON, _ := json.Marshal(payload)
	fmt.Printf("Received Facebook lead webhook: %s\n", string(payloadJSON))

	// Check if this is a leadgen webhook
	if !isLeadGenWebhook(&payload) {
		return c.JSON(schema.SuccessResponse(fiber.Map{}))
	}

	// TODO: Process lead webhook
	// - Extract lead data from payload
	// - Fetch full lead details from Facebook API
	// - Create contact in database
	// - Trigger any workflows

	return c.JSON(schema.SuccessResponse("Received lead successfully"))
}

// isLeadGenWebhook checks if webhook is for lead gen event
func isLeadGenWebhook(payload *LeadWebhookPayload) bool {
	if payload.Object != "page" {
		return false
	}

	if len(payload.Entry) == 0 {
		return false
	}

	entry := payload.Entry[0]
	if len(entry.Changes) == 0 {
		return false
	}

	return entry.Changes[0].Field == "leadgen"
}
