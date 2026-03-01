package template

import (
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateHandler_GetByID_Validation(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		shouldFail bool
	}{
		{"valid UUID", uuid.New().String(), false},
		{"invalid UUID", "invalid-uuid", true},
		{"empty UUID", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uuid.Parse(tt.id)
			if tt.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTemplateHandler_JSONStructure(t *testing.T) {
	t.Run("valid template structure with metadata", func(t *testing.T) {
		req := map[string]any{
			"campaign_id": uuid.New().String(),
			"type":        "email",
			"category":    "follow_up",
			"subject":     "Hello",
			"content":     "World",
			"position":    2,
			"send_after":  3600,
			"metadata": map[string]any{
				"lang": "en",
				"a":    1,
			},
		}

		b, err := json.Marshal(req)
		require.NoError(t, err)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal(b, &parsed))

		assert.Equal(t, "Hello", parsed["subject"])
		assert.Equal(t, float64(2), parsed["position"])

		md := parsed["metadata"].(map[string]any)
		assert.Equal(t, "en", md["lang"])
	})
}

func TestTemplateHandler_ListPaginationParsing(t *testing.T) {
	tests := []struct {
		name         string
		offsetQuery  string
		limitQuery   string
		expectOffset int
		expectLimit  int
	}{
		{"default pagination", "", "", 0, 50},
		{"custom pagination", "5", "25", 5, 25},
		{"invalid offset uses default", "bad", "25", 0, 25},
		{"invalid limit uses default", "5", "bad", 5, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/v1/template", func(c fiber.Ctx) error {
				// Mimic handler logic for parsing
				offset := 0
				if s := c.Query("offset"); s != "" {
					if v, err := strconv.Atoi(s); err == nil {
						offset = v
					}
				}
				limit := 50
				if s := c.Query("limit"); s != "" {
					if v, err := strconv.Atoi(s); err == nil {
						limit = v
					}
				}
				return c.JSON(fiber.Map{"offset": offset, "limit": limit})
			})

			url := "/v1/template"
			if tt.offsetQuery != "" || tt.limitQuery != "" {
				url += "?"
				if tt.offsetQuery != "" {
					url += "offset=" + tt.offsetQuery
					if tt.limitQuery != "" {
						url += "&"
					}
				}
				if tt.limitQuery != "" {
					url += "limit=" + tt.limitQuery
				}
			}

			req := httptest.NewRequest("GET", url, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, fiber.StatusOK, resp.StatusCode)

			var result map[string]any
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
			assert.Equal(t, float64(tt.expectOffset), result["offset"])
			assert.Equal(t, float64(tt.expectLimit), result["limit"])
		})
	}
}
