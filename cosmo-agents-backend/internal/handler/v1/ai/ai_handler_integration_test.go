package ai

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAIHandler_Integration tests the actual AIHandler
// Note: This test will have limited coverage due to concrete dependencies
func TestAIHandler_Integration(t *testing.T) {
	t.Run("ExtractCompanyInfo validation logic", func(t *testing.T) {
		// Create handler with nil service to test validation logic only
		handler := NewAIHandler(nil)

		app := fiber.New()
		app.Get("/extract", handler.ExtractCompanyInfo)

		// Test missing company_url parameter
		req := httptest.NewRequest("GET", "/extract", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}
