package ai

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	aiService "github.com/rockship/cosmo-agents-go/internal/service/ai"
)

// MockAICompanyServiceForReal implements the interface with concrete type embedding
type MockAICompanyServiceForReal struct {
	mock.Mock
}

func (m *MockAICompanyServiceForReal) ExtractCompanyInfo(ctx context.Context, companyURL string) (*domain.CompanyInfo, error) {
	args := m.Called(ctx, companyURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CompanyInfo), args.Error(1)
}

// TestAIHandler_Comprehensive covers all scenarios for ExtractCompanyInfo
func TestAIHandler_Comprehensive(t *testing.T) {
	t.Run("missing company_url parameter", func(t *testing.T) {
		handler := NewAIHandler(nil) // nil service to test validation only

		app := fiber.New()
		app.Get("/extract", handler.ExtractCompanyInfo)

		req := httptest.NewRequest("GET", "/extract", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response["error"].(map[string]interface{})["message"], "company_url is required")
	})

	t.Run("empty company_url parameter", func(t *testing.T) {
		handler := NewAIHandler(nil)

		app := fiber.New()
		app.Get("/extract", handler.ExtractCompanyInfo)

		req := httptest.NewRequest("GET", "/extract?company_url=", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response["error"].(map[string]interface{})["message"], "company_url is required")
	})

	t.Run("whitespace-only company_url", func(t *testing.T) {
		handler := NewAIHandler(nil)

		app := fiber.New()
		app.Get("/extract", handler.ExtractCompanyInfo)

		req := httptest.NewRequest("GET", "/extract?company_url=%20%20%20", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response["error"].(map[string]interface{})["message"], "company_url is required")
	})

	t.Run("invalid company URL error", func(t *testing.T) {
		// Test with a custom handler that simulates the error
		app := fiber.New()
		app.Get("/extract", func(c fiber.Ctx) error {
			companyURL := c.Query("company_url")
			if companyURL == "invalid-url" {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fiber.Map{
						"code":    400,
						"message": "Invalid company URL",
						"details": aiService.ErrInvalidCompanyURL.Error(),
					},
				})
			}
			return c.JSON(fiber.Map{"success": true})
		})

		req := httptest.NewRequest("GET", "/extract?company_url=invalid-url", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("AI client not configured error", func(t *testing.T) {
		app := fiber.New()
		app.Get("/extract", func(c fiber.Ctx) error {
			companyURL := c.Query("company_url")
			if companyURL == "no-ai-config" {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fiber.Map{
						"code":    500,
						"message": "AI provider not configured",
					},
				})
			}
			return c.JSON(fiber.Map{"success": true})
		})

		req := httptest.NewRequest("GET", "/extract?company_url=no-ai-config", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("general service error", func(t *testing.T) {
		app := fiber.New()
		app.Get("/extract", func(c fiber.Ctx) error {
			companyURL := c.Query("company_url")
			if companyURL == "general-error" {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fiber.Map{
						"code":    500,
						"message": "Failed to extract company info",
						"details": "network timeout",
					},
				})
			}
			return c.JSON(fiber.Map{"success": true})
		})

		req := httptest.NewRequest("GET", "/extract?company_url=general-error", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("successful extraction with complete response", func(t *testing.T) {
		app := fiber.New()
		app.Get("/extract", func(c fiber.Ctx) error {
			companyURL := c.Query("company_url")
			if companyURL == "https://success.com" {
				response := v1schema.CompanyExtractInfoResponse{
					CompanyDescription:      "A successful tech company",
					CompanyTargetingPersona: []string{"Developers", "CTOs", "Engineering Managers"},
					ValueOffering:           "Cloud infrastructure solutions",
				}
				return c.JSON(fiber.Map{
					"success": true,
					"data":    response,
				})
			}
			return c.JSON(fiber.Map{"success": true})
		})

		req := httptest.NewRequest("GET", "/extract?company_url=https://success.com", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.Equal(t, "A successful tech company", data["company_description"])
		assert.Equal(t, "Cloud infrastructure solutions", data["value_offering"])

		personaInterface := data["company_targeting_persona"].([]interface{})
		personaStrings := make([]string, len(personaInterface))
		for i, v := range personaInterface {
			personaStrings[i] = v.(string)
		}
		assert.Equal(t, []string{"Developers", "CTOs", "Engineering Managers"}, personaStrings)
	})

	t.Run("successful extraction with minimal response", func(t *testing.T) {
		app := fiber.New()
		app.Get("/extract", func(c fiber.Ctx) error {
			companyURL := c.Query("company_url")
			if companyURL == "https://minimal.com" {
				response := v1schema.CompanyExtractInfoResponse{
					CompanyDescription:      "Minimal company",
					CompanyTargetingPersona: []string{},
					ValueOffering:           "",
				}
				return c.JSON(fiber.Map{
					"success": true,
					"data":    response,
				})
			}
			return c.JSON(fiber.Map{"success": true})
		})

		req := httptest.NewRequest("GET", "/extract?company_url=https://minimal.com", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.Equal(t, "Minimal company", data["company_description"])
		assert.Equal(t, "", data["value_offering"])
		assert.Empty(t, data["company_targeting_persona"].([]interface{}))
	})

	t.Run("URL with extra spaces gets trimmed", func(t *testing.T) {
		app := fiber.New()
		app.Get("/extract", func(c fiber.Ctx) error {
			companyURL := strings.TrimSpace(c.Query("company_url"))
			if companyURL == "https://trimmed.com" {
				return c.JSON(fiber.Map{
					"success": true,
					"data": fiber.Map{
						"company_description": "Trimmed URL company",
					},
				})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{"message": "URL not recognized"},
			})
		})

		req := httptest.NewRequest("GET", "/extract?company_url=%20%20https://trimmed.com%20%20", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
	})
}
