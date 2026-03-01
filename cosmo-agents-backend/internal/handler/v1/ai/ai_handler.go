package ai

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	aiService "github.com/rockship/cosmo-agents-go/internal/service/ai"
)

// AIHandler manages AI-related HTTP endpoints.
type AIHandler struct {
	companyService *aiService.AICompanyService
}

// NewAIHandler constructs a new AIHandler instance.
func NewAIHandler(companyService *aiService.AICompanyService) *AIHandler {
	return &AIHandler{
		companyService: companyService,
	}
}

// ExtractCompanyInfo handles GET /v1/ai/companies/extract
// @Summary Extract company insights from URL
// @Description Uses AI to extract company description, targeting persona, and value offering from a company website.
// @Tags AI
// @Accept json
// @Produce json
// @Param company_url query string true "Company website URL"
// @Success 200 {object} schema.APIResponse[v1schema.CompanyExtractInfoResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/ai/companies/extract [get]
func (h *AIHandler) ExtractCompanyInfo(c fiber.Ctx) error {
	companyURL := strings.TrimSpace(c.Query("company_url"))
	if companyURL == "" {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "company_url is required", "",
		))
	}

	info, err := h.companyService.ExtractCompanyInfo(c.Context(), companyURL)
	if err != nil {
		switch {
		case errors.Is(err, aiService.ErrInvalidCompanyURL):
			return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
				http.StatusBadRequest, "Invalid company URL", err.Error(),
			))
		case errors.Is(err, aiService.ErrAIClientNotConfigured):
			return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
				http.StatusInternalServerError, "AI provider not configured", "",
			))
		default:
			return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
				http.StatusInternalServerError, "Failed to extract company info", err.Error(),
			))
		}
	}

	response := v1schema.CompanyExtractInfoResponse{
		CompanyDescription:      info.CompanyDescription,
		CompanyTargetingPersona: info.CompanyTargetingPersona,
		ValueOffering:           info.ValueOffering,
	}

	return c.JSON(schema.SuccessResponse(response))
}
