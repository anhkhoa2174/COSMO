package campaign

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// GenerateTemplates handles POST /v1/campaigns/{id}/generate.
// @Summary Generate Email Templates
// @Description Generate email templates for a campaign using AI.
// @Tags Campaigns
// @Accept json
// @Produce json
// @Param id path string true "Campaign ID"
// @Param body body v1schema.AIGenerateEmailRequest true "AI Email Generation Request"
// @Success 200 {object} schema.APIResponse[[]v1schema.AIGenerateEmailResponse] "Successful Response"
// @Security BearerAuth
// @Failure 400 {object} schema.APIResponse[any] "Bad Request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 404 {object} schema.APIResponse[any] "Not Found"
// @Failure 500 {object} schema.APIResponse[any] "Internal Server Error"
// @Router /v1/campaigns/{id}/generate [post]
func (h *Handler) GenerateTemplates(c fiber.Ctx) error {
	var req v1schema.AIGenerateEmailRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err)
	}

	if err := v1validation.ValidateStruct(&req); err != nil {
		return badRequest(c, "Validation failed", err)
	}

	// Check if AI service is configured
	if h.aiEmailService == nil {
		return c.Status(http.StatusServiceUnavailable).JSON(schema.ErrorResponse(
			http.StatusServiceUnavailable,
			"AI service not configured",
			nil,
		))
	}

	// Extract campaign context from client_data
	campaignGoal := req.Campaign
	if campaignGoal == "" {
		campaignGoal = "Generate engaging outreach emails"
	}

	// Build context from client_data
	additionalContext := buildContextFromClientData(req.ClientData)

	// Generate templates for common email types using the deprecated method
	// TODO: Replace with proper AI integration when available
	ctx := c.Context()
	userID, _ := userIDFromContext(c)

	aiTemplates, err := h.aiEmailService.GenerateDeprecatedOutreach(ctx, userID, campaignGoal, req.ClientData)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to generate templates: %v", err),
			nil,
		))
	}

	// Convert to expected response format
	var templates []v1schema.AIGenerateEmailResponse
	for _, aiTemplate := range aiTemplates {
		template := v1schema.EmailTemplateItem{
			Type:    aiTemplate.Type,
			Subject: aiTemplate.Subject,
			Content: aiTemplate.Content,
		}

		preview := v1schema.EmailTemplateItem{
			Type:    aiTemplate.Type,
			Subject: replacePlaceholders(aiTemplate.Subject, req.ClientData),
			Content: replacePlaceholders(aiTemplate.Content, req.ClientData),
		}

		templates = append(templates, v1schema.AIGenerateEmailResponse{
			Template: template,
			Preview:  preview,
		})
	}

	// Add additional context to first template if available
	_ = additionalContext // Use context for future enhancement

	return c.JSON(schema.SuccessResponse(templates))
}

// buildContextFromClientData extracts relevant context from client_data map
func buildContextFromClientData(data map[string]interface{}) string {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return ""
	}
	return fmt.Sprintf("Client Data:\n%s", string(jsonBytes))
}

// getStringFromMap safely extracts string value from map
func getStringFromMap(data map[string]interface{}, key, defaultValue string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// replacePlaceholders replaces template variables with actual values
func replacePlaceholders(text string, data map[string]interface{}) string {
	result := text

	placeholders := map[string]string{
		"{{first_name}}":   getStringFromMap(data, "first_name", "John"),
		"{{company_name}}": getStringFromMap(data, "company_name", "Acme Corp"),
		"{{title}}":        getStringFromMap(data, "title", "Manager"),
		"{{industry}}":     getStringFromMap(data, "industry", "Technology"),
		"{{sender_name}}":  getStringFromMap(data, "sender_name", "Sales Team"),
	}

	for placeholder, value := range placeholders {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}
