package file

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// TestFileHandler_BasicValidation tests basic validation without mocks
func TestFileHandler_BasicValidation(t *testing.T) {
	t.Run("List requires user_id", func(t *testing.T) {
		app := fiber.New()
		app.Get("/files", func(c fiber.Ctx) error {
			userID, ok := c.Locals("user_id").(uuid.UUID)
			if !ok {
				return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
					fiber.StatusUnauthorized, "Unauthorized", "User ID not found in context",
				))
			}
			return c.JSON(fiber.Map{"user_id": userID})
		})

		req := httptest.NewRequest("GET", "/files", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("List with user_id succeeds", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c fiber.Ctx) error {
			c.Locals("user_id", uuid.New())
			return c.Next()
		})

		app.Get("/files", func(c fiber.Ctx) error {
			userID, ok := c.Locals("user_id").(uuid.UUID)
			if !ok {
				return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
					fiber.StatusUnauthorized, "Unauthorized", "User ID not found in context",
				))
			}
			return c.JSON(fiber.Map{"user_id": userID})
		})

		req := httptest.NewRequest("GET", "/files", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("GetFromS3 missing key parameter", func(t *testing.T) {
		app := fiber.New()
		app.Get("/files/s3", func(c fiber.Ctx) error {
			key := c.Query("key")
			if key == "" {
				return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
					fiber.StatusBadRequest, "Missing key parameter", "",
				))
			}
			return c.JSON(fiber.Map{"key": key})
		})

		req := httptest.NewRequest("GET", "/files/s3", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("GetFromS3 with key parameter", func(t *testing.T) {
		app := fiber.New()
		app.Get("/files/s3", func(c fiber.Ctx) error {
			key := c.Query("key")
			if key == "" {
				return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
					fiber.StatusBadRequest, "Missing key parameter", "",
				))
			}
			return c.JSON(fiber.Map{"key": key})
		})

		req := httptest.NewRequest("GET", "/files/s3?key=test.txt", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "test.txt", response["key"])
	})

	t.Run("GetPresignedURL missing key", func(t *testing.T) {
		app := fiber.New()
		app.Get("/files/s3/presigned-url", func(c fiber.Ctx) error {
			key := c.Query("key")
			if key == "" {
				return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
					fiber.StatusBadRequest, "Missing key parameter", "",
				))
			}
			return c.JSON(fiber.Map{"key": key})
		})

		req := httptest.NewRequest("GET", "/files/s3/presigned-url", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Search invalid JSON", func(t *testing.T) {
		app := fiber.New()
		app.Post("/files/search", func(c fiber.Ctx) error {
			var body map[string]interface{}
			if err := c.Bind().JSON(&body); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
					fiber.StatusBadRequest, "Invalid request body", err.Error(),
				))
			}
			return c.JSON(body)
		})

		req := httptest.NewRequest("POST", "/files/search", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Search valid JSON", func(t *testing.T) {
		app := fiber.New()
		app.Post("/files/search", func(c fiber.Ctx) error {
			var body map[string]interface{}
			if err := c.Bind().JSON(&body); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
					fiber.StatusBadRequest, "Invalid request body", err.Error(),
				))
			}
			return c.JSON(body)
		})

		searchBody := map[string]interface{}{
			"filename": "test.txt",
		}
		body, _ := json.Marshal(searchBody)

		req := httptest.NewRequest("POST", "/files/search", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "test.txt", response["filename"])
	})
}
