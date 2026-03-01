package pagination

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePaginationParamsWithValidation_ErrorHandling(t *testing.T) {
	tests := []struct {
		name               string
		offsetQuery        string
		limitQuery         string
		expectValid        bool
		expectedOffset     int
		expectedLimit      int
		expectedErrorCount int
	}{
		{
			name:           "valid parameters",
			offsetQuery:    "10",
			limitQuery:     "20",
			expectValid:    true,
			expectedOffset: 10,
			expectedLimit:  20,
		},
		{
			name:               "invalid offset",
			offsetQuery:        "invalid",
			limitQuery:         "20",
			expectValid:        false,
			expectedOffset:     0, // defaults to 0
			expectedLimit:      20,
			expectedErrorCount: 1,
		},
		{
			name:               "negative offset",
			offsetQuery:        "-5",
			limitQuery:         "20",
			expectValid:        false,
			expectedOffset:     0, // defaults to 0
			expectedLimit:      20,
			expectedErrorCount: 1,
		},
		{
			name:               "invalid limit",
			offsetQuery:        "10",
			limitQuery:         "invalid",
			expectValid:        false,
			expectedOffset:     10,
			expectedLimit:      25, // defaults to 25
			expectedErrorCount: 1,
		},
		{
			name:               "negative limit",
			offsetQuery:        "10",
			limitQuery:         "-10",
			expectValid:        false,
			expectedOffset:     10,
			expectedLimit:      25, // defaults to 25
			expectedErrorCount: 1,
		},
		{
			name:               "limit exceeds maximum",
			offsetQuery:        "0",
			limitQuery:         "200",
			expectValid:        false,
			expectedOffset:     0,
			expectedLimit:      100, // clamped to max
			expectedErrorCount: 1,
		},
		{
			name:               "multiple errors",
			offsetQuery:        "invalid",
			limitQuery:         "-10",
			expectValid:        false,
			expectedOffset:     0,
			expectedLimit:      25,
			expectedErrorCount: 2,
		},
		{
			name:           "default values",
			offsetQuery:    "",
			limitQuery:     "",
			expectValid:    true,
			expectedOffset: 0,
			expectedLimit:  25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			app.Get("/test", func(c fiber.Ctx) error {
				params, validation := ParsePaginationParamsWithValidation(c)

				assert.Equal(t, tt.expectValid, validation.Valid)
				assert.Equal(t, tt.expectedOffset, params.Offset)
				assert.Equal(t, tt.expectedLimit, params.Limit)

				if !tt.expectValid {
					assert.Len(t, validation.Errors, tt.expectedErrorCount)
				}

				return c.JSON(fiber.Map{
					"valid":  validation.Valid,
					"errors": validation.Errors,
					"offset": params.Offset,
					"limit":  params.Limit,
				})
			})

			url := "/test"
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
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}

func TestValidatePaginationOrError_ErrorResponses(t *testing.T) {
	tests := []struct {
		name           string
		offsetQuery    string
		limitQuery     string
		expectError    bool
		expectedStatus int
	}{
		{
			name:        "valid parameters should not error",
			offsetQuery: "10",
			limitQuery:  "20",
			expectError: false,
		},
		{
			name:           "invalid offset should return 400",
			offsetQuery:    "invalid",
			limitQuery:     "20",
			expectError:    true,
			expectedStatus: 400,
		},
		{
			name:           "invalid limit should return 400",
			offsetQuery:    "10",
			limitQuery:     "invalid",
			expectError:    true,
			expectedStatus: 400,
		},
		{
			name:           "negative values should return 400",
			offsetQuery:    "-5",
			limitQuery:     "-10",
			expectError:    true,
			expectedStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			app.Get("/test", func(c fiber.Ctx) error {
				params, err := ValidatePaginationOrError(c)
				if err != nil {
					return err // This will be the error response
				}

				return c.JSON(fiber.Map{
					"offset": params.Offset,
					"limit":  params.Limit,
				})
			})

			url := "/test"
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

			if tt.expectError {
				assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			} else {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
		})
	}
}

func TestParsePaginationParams_LegacyCompatibility(t *testing.T) {
	// Test that the legacy function still works but silently handles errors
	app := fiber.New()

	app.Get("/test", func(c fiber.Ctx) error {
		params := ParsePaginationParams(c)
		return c.JSON(fiber.Map{
			"offset": params.Offset,
			"limit":  params.Limit,
		})
	})

	// Test with invalid parameters - should not error but use defaults
	req := httptest.NewRequest("GET", "/test?offset=invalid&limit=invalid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPaginationConstants_Security(t *testing.T) {
	// Ensure pagination constants are within reasonable security bounds
	assert.Equal(t, 0, defaultOffset, "Default offset should be 0")
	assert.Equal(t, 25, defaultPageSize, "Default page size should be reasonable")
	assert.Equal(t, MaxPaginationLimit, maxPageSize, "Max page size should match constant")
	assert.LessOrEqual(t, maxPageSize, 1000, "Max page size should not be too large to prevent DoS")
}

func TestCalculatePageInfo_EdgeCases(t *testing.T) {
	tests := []struct {
		name               string
		offset             int
		limit              int
		total              int64
		expectedPage       int
		expectedTotalPages int
	}{
		{
			name:               "first page",
			offset:             0,
			limit:              10,
			total:              100,
			expectedPage:       1,
			expectedTotalPages: 10,
		},
		{
			name:               "middle page",
			offset:             20,
			limit:              10,
			total:              100,
			expectedPage:       3,
			expectedTotalPages: 10,
		},
		{
			name:               "partial last page",
			offset:             0,
			limit:              10,
			total:              95,
			expectedPage:       1,
			expectedTotalPages: 10,
		},
		{
			name:               "empty result",
			offset:             0,
			limit:              10,
			total:              0,
			expectedPage:       1,
			expectedTotalPages: 0,
		},
		{
			name:               "single item",
			offset:             0,
			limit:              10,
			total:              1,
			expectedPage:       1,
			expectedTotalPages: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, totalPages := CalculatePageInfo(tt.offset, tt.limit, tt.total)
			assert.Equal(t, tt.expectedPage, page)
			assert.Equal(t, tt.expectedTotalPages, totalPages)
		})
	}
}
