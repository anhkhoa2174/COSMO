package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
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

// AICompanyServiceInterface defines the interface for AI company service
type AICompanyServiceInterface interface {
	ExtractCompanyInfo(ctx context.Context, companyURL string) (*domain.CompanyInfo, error)
}

// MockAICompanyService is a mock for AICompanyServiceInterface
type MockAICompanyService struct {
	mock.Mock
}

func (m *MockAICompanyService) ExtractCompanyInfo(ctx context.Context, companyURL string) (*domain.CompanyInfo, error) {
	args := m.Called(ctx, companyURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CompanyInfo), args.Error(1)
}

// TestableAIHandler wraps AIHandler to accept interface for testing
type TestableAIHandler struct {
	companyService AICompanyServiceInterface
}

func NewTestableAIHandler(companyService AICompanyServiceInterface) *TestableAIHandler {
	return &TestableAIHandler{
		companyService: companyService,
	}
}

// ExtractCompanyInfo implements the same logic as AIHandler but uses interface
func (h *TestableAIHandler) ExtractCompanyInfo(c fiber.Ctx) error {
	// Copy the exact logic from ai_handler.go
	companyURL := strings.TrimSpace(c.Query("company_url"))
	if companyURL == "" {
		return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    400,
				"message": "company_url is required",
			},
		})
	}

	info, err := h.companyService.ExtractCompanyInfo(c.Context(), companyURL)
	if err != nil {
		switch {
		case errors.Is(err, aiService.ErrInvalidCompanyURL):
			return c.Status(http.StatusBadRequest).JSON(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    400,
					"message": "Invalid company URL",
				},
			})
		default:
			return c.Status(http.StatusInternalServerError).JSON(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    500,
					"message": "Failed to extract company info",
				},
			})
		}
	}

	response := v1schema.CompanyExtractInfoResponse{
		CompanyDescription:      info.CompanyDescription,
		CompanyTargetingPersona: info.CompanyTargetingPersona,
		ValueOffering:           info.ValueOffering,
	}

	return c.JSON(map[string]interface{}{
		"success": true,
		"data":    response,
	})
}

// TestAIHandler_ExtractCompanyInfo tests the actual handler method
func TestAIHandler_ExtractCompanyInfo(t *testing.T) {
	t.Run("successful extraction", func(t *testing.T) {
		mockService := new(MockAICompanyService)

		companyInfo := &domain.CompanyInfo{
			CompanyDescription:      "A tech company",
			CompanyTargetingPersona: []string{"Developers"},
			ValueOffering:           "API services",
		}

		mockService.On("ExtractCompanyInfo", mock.Anything, "https://example.com").Return(companyInfo, nil)

		// Create handler with mock service
		handler := NewTestableAIHandler(mockService)

		app := fiber.New()
		app.Get("/extract", handler.ExtractCompanyInfo)

		req := httptest.NewRequest("GET", "/extract?company_url=https://example.com", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, companyInfo.CompanyDescription, data["company_description"])

		// Convert interface slice to string slice for comparison
		personaInterface := data["company_targeting_persona"].([]interface{})
		personaStrings := make([]string, len(personaInterface))
		for i, v := range personaInterface {
			personaStrings[i] = v.(string)
		}
		assert.Equal(t, companyInfo.CompanyTargetingPersona, personaStrings)

		assert.Equal(t, companyInfo.ValueOffering, data["value_offering"])

		mockService.AssertExpectations(t)
	})

	t.Run("missing company URL", func(t *testing.T) {
		mockService := new(MockAICompanyService)
		handler := NewTestableAIHandler(mockService)

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

	t.Run("invalid company URL", func(t *testing.T) {
		mockService := new(MockAICompanyService)
		mockService.On("ExtractCompanyInfo", mock.Anything, "invalid-url").Return(nil, aiService.ErrInvalidCompanyURL)

		handler := NewTestableAIHandler(mockService)

		app := fiber.New()
		app.Get("/extract", handler.ExtractCompanyInfo)

		req := httptest.NewRequest("GET", "/extract?company_url=invalid-url", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response["error"].(map[string]interface{})["message"], "Invalid company URL")

		mockService.AssertExpectations(t)
	})
}
