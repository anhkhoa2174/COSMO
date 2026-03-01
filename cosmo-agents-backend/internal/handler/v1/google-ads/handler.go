package googleads

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// Handler handles Google Ads webhook-related HTTP requests following clean architecture
type Handler struct {
	useCase GoogleAdsUseCase
}

// NewHandler creates a new Google Ads handler
func NewHandler(useCase GoogleAdsUseCase) *Handler {
	return &Handler{
		useCase: useCase,
	}
}

// GenerateWebhook handles POST /v1/google-ads/generate-webhook
// @Summary Generate Google Ads webhook URL
// @Description Creates a webhook endpoint for Google Ads lead forms
// @Tags Google Ads
// @Accept json
// @Produce json
// @Param request body v1schema.CreateGoogleAdsWebhookRequest true "Webhook creation request"
// @Success 200 {object} schema.APIResponse[map[string]string] "Webhook URL generated successfully"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 404 {object} schema.APIResponse[any] "Campaign not found"
// @Failure 409 {object} schema.APIResponse[any] "Webhook slug already exists"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/google-ads/generate-webhook [post]
func (h *Handler) GenerateWebhook(c fiber.Ctx) error {
	// Extract user ID from context
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		if v, okAlt := c.Locals("userID").(uuid.UUID); okAlt {
			userID = v
		} else {
			return c.Status(http.StatusUnauthorized).JSON(schema.ErrorResponse(
				http.StatusUnauthorized, "User not authenticated", "",
			))
		}
	}

	// Parse request body
	var req v1schema.CreateGoogleAdsWebhookRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	// Validate request
	if err := validateStruct(req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

	// Call use case
	resp, err := h.useCase.GenerateWebhook(c.Context(), GenerateWebhookRequest{
		CampaignID:    req.CampaignID,
		ContactListID: req.ContactListID,
		Name:          req.Name,
		Slug:          req.Slug,
		UserID:        userID,
	})

	if err != nil {
		logger.Logger.Error().Err(err).Msg("failed to generate webhook")

		switch {
		case errors.Is(err, ErrCampaignNotFound):
			return c.Status(http.StatusNotFound).JSON(schema.ErrorResponse(
				http.StatusNotFound, "Campaign not found", err.Error(),
			))
		case errors.Is(err, ErrWebhookSlugExists):
			return c.Status(http.StatusConflict).JSON(schema.ErrorResponse(
				http.StatusConflict, "Webhook slug already exists", "",
			))
		case errors.Is(err, ErrContactListNotFound):
			return c.Status(http.StatusNotFound).JSON(schema.ErrorResponse(
				http.StatusNotFound, "Contact list not found", err.Error(),
			))
		default:
			return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
				http.StatusInternalServerError, "Failed to generate webhook", err.Error(),
			))
		}
	}

	return c.JSON(schema.SuccessResponse(fiber.Map{
		"url": resp.WebhookURL,
	}))
}

// ProcessWebhook handles POST /v1/google-ads/:slug/webhook
// @Summary Process Google Ads webhook
// @Description Processes incoming webhook data from Google Ads lead forms
// @Tags Google Ads
// @Accept json
// @Produce json
// @Param slug path string true "Webhook slug"
// @Param payload body v1schema.GoogleAdsWebhookPayload true "Webhook payload"
// @Security BearerAuth
// @Success 200 {object} schema.APIResponse[map[string]string] "Webhook processed successfully"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 404 {object} schema.APIResponse[any] "Form not found"
// @Failure 409 {object} schema.APIResponse[any] "Campaign not active or missing agent"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Router /v1/google-ads/{slug}/webhook [post]
func (h *Handler) ProcessWebhook(c fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "Missing slug parameter", "",
		))
	}

	// Parse request body
	var payload v1schema.GoogleAdsWebhookPayload
	if err := c.Bind().JSON(&payload); err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "Invalid webhook payload", err.Error(),
		))
	}

	// Convert to use case request
	columns := make([]ColumnEntry, len(payload.UserColumnData))
	for i, col := range payload.UserColumnData {
		columns[i] = ColumnEntry{
			ColumnID:    col.ColumnID,
			StringValue: col.StringValue,
		}
	}

	// Call use case
	err := h.useCase.ProcessWebhook(c.Context(), ProcessWebhookRequest{
		Slug:           slug,
		LeadID:         payload.LeadID,
		UserColumnData: columns,
	})

	if err != nil {
		logger.Logger.Error().Err(err).Str("slug", slug).Msg("failed to process webhook")

		switch {
		case errors.Is(err, ErrInboundFormNotFound):
			return c.Status(http.StatusNotFound).JSON(schema.ErrorResponse(
				http.StatusNotFound, "Inbound form not found", "",
			))
		case errors.Is(err, ErrCampaignNotFound):
			return c.Status(http.StatusNotFound).JSON(schema.ErrorResponse(
				http.StatusNotFound, "Campaign not found", "",
			))
		case errors.Is(err, ErrCampaignInactive):
			// Return success but log that campaign is inactive
			return c.JSON(schema.SuccessResponse(fiber.Map{
				"message": "Campaign not active; webhook ignored",
			}))
		case errors.Is(err, ErrMissingAgent):
			return c.Status(http.StatusConflict).JSON(schema.ErrorResponse(
				http.StatusConflict, "Campaign missing agent", "",
			))
		default:
			return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
				http.StatusInternalServerError, "Failed to process webhook", err.Error(),
			))
		}
	}

	return c.JSON(schema.SuccessResponse(fiber.Map{
		"message": "Webhook processed successfully",
	}))
}
