package ai

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"unsafe"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	aiService "github.com/rockship/cosmo-agents-go/internal/service/ai"
)

// MockServiceInterface defines what we need from the service
type MockServiceInterface interface {
	ExtractCompanyInfo(ctx context.Context, companyURL string) (*domain.CompanyInfo, error)
}

// AIHandlerWithInterface allows dependency injection
type AIHandlerWithInterface struct {
	service MockServiceInterface
}

func (h *AIHandlerWithInterface) ExtractCompanyInfo(c fiber.Ctx) error {
	// Copy the exact implementation from ai_handler.go but using interface
	companyURL := strings.TrimSpace(c.Query("company_url"))
	if companyURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    400,
				"message": "company_url is required",
			},
		})
	}

	info, err := h.service.ExtractCompanyInfo(c.Context(), companyURL)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    500,
				"message": "Failed to extract company info",
			},
		})
	}

	response := v1schema.CompanyExtractInfoResponse{
		CompanyDescription:      info.CompanyDescription,
		CompanyTargetingPersona: info.CompanyTargetingPersona,
		ValueOffering:           info.ValueOffering,
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}

// CreateAIHandlerUsingReflection uses unsafe to inject mock service
func CreateAIHandlerUsingReflection(mockService MockServiceInterface) *AIHandler {
	// Create real handler
	handler := NewAIHandler(nil)

	// Use unsafe to replace the service field
	// This is dangerous and only for testing!
	handlerPtr := (*AIHandler)(unsafe.Pointer(handler))

	// Get the mock service as the concrete type
	if concreteService, ok := mockService.(interface {
		Unwrap() *aiService.AICompanyService
	}); ok {
		// If mock provides unwrap, use it
		serviceField := (*aiService.AICompanyService)(unsafe.Pointer(&concreteService))
		*(*uintptr)(unsafe.Pointer(uintptr(unsafe.Pointer(handlerPtr)) + unsafe.Offsetof(handlerPtr.companyService))) = uintptr(unsafe.Pointer(serviceField))
	}

	return handler
}

// TestRealAIHandler tests the actual AIHandler
func TestRealAIHandler(t *testing.T) {
	t.Run("test real handler structure", func(t *testing.T) {
		// Test that we can create and use the handler
		handler := NewAIHandler(nil)
		assert.NotNil(t, handler)
		// companyService will be nil when passed nil, which is expected
		assert.Nil(t, handler.companyService)
	})

	t.Run("test validation with real handler", func(t *testing.T) {
		handler := NewAIHandler(nil)

		app := fiber.New()
		app.Get("/extract", handler.ExtractCompanyInfo)

		// Test missing parameter
		req := httptest.NewRequest("GET", "/extract", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		// Test empty parameter
		req = httptest.NewRequest("GET", "/extract?company_url=", nil)
		resp, err = app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}
