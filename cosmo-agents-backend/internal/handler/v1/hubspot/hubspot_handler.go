package hubspot

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	hubspotService "github.com/rockship/cosmo-agents-go/internal/service/hubspot"
)

// HubspotHandler manages HubSpot authorization and sync endpoints.
type HubspotHandler struct {
	service *hubspotService.HubspotIntegrationService
}

// NewHubspotHandler constructs a new HubspotHandler.
func NewHubspotHandler(service *hubspotService.HubspotIntegrationService) *HubspotHandler {
	return &HubspotHandler{service: service}
}

// Authorize handles GET /v1/hubspot/authorize.
// @Summary Get HubSpot authorization URL
// @Description Returns HubSpot OAuth authorization URL for the current user
// @Tags Hubspot
// @Produce json
// @Param redirect_url query string false "Optional redirect URI override"
// @Success 200 {object} schema.APIResponse[v1schema.AuthHubspotResponse]
// @Failure 500 {object} schema.APIResponse[any]
// @Router /v1/hubspot/authorize [get]
func (h *HubspotHandler) Authorize(c fiber.Ctx) error {
	redirect := c.Query("redirect_url", "")

	url, err := h.service.AuthorizationURL(c.Context(), redirect)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
			http.StatusInternalServerError, "Failed to build authorization URL", err.Error(),
		))
	}

	response := v1schema.AuthHubspotResponse{URL: url}
	return c.JSON(schema.SuccessResponse(response))
}

// Callback handles GET /v1/hubspot/callback.
func (h *HubspotHandler) Callback(c fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "Missing authorization code", "",
		))
	}

	redirectURI := c.Query("redirect_uri", "")

	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		if v, alt := c.Locals("userID").(uuid.UUID); alt {
			userID = v
		} else {
			return c.Status(http.StatusUnauthorized).JSON(schema.ErrorResponse(
				http.StatusUnauthorized, "User not authenticated", "",
			))
		}
	}

	result, err := h.service.HandleCallback(c.Context(), userID, code, redirectURI)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
			http.StatusInternalServerError, "Failed to process callback", err.Error(),
		))
	}

	resp := v1schema.AuthHubspotCallbackResponse{
		TokenType:    result.Token.TokenType,
		RefreshToken: result.Token.RefreshToken,
		AccessToken:  result.Token.AccessToken,
		ExpiresIn:    result.Token.ExpiresIn,
	}

	return c.JSON(schema.SuccessResponse(resp))
}

// GetUserInfo handles GET /v1/hubspot/users/me.
func (h *HubspotHandler) GetUserInfo(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		if v, alt := c.Locals("userID").(uuid.UUID); alt {
			userID = v
		} else {
			return c.Status(http.StatusUnauthorized).JSON(schema.ErrorResponse(
				http.StatusUnauthorized, "User not authenticated", "",
			))
		}
	}

	info, err := h.service.GetUserInfo(c.Context(), userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
			http.StatusInternalServerError, "Failed to fetch HubSpot user info", err.Error(),
		))
	}

	resp := v1schema.HubspotUserInfoRead{
		Token:     info.Token,
		User:      info.User,
		HubDomain: info.HubDomain,
		Scopes:    info.Scopes,
		HubID:     info.HubID,
		ClientID:  info.ClientID,
		UserID:    info.UserID,
		TokenType: info.TokenType,
	}

	return c.JSON(schema.SuccessResponse(resp))
}
