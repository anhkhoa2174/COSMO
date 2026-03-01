package pagination

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

const (
	MaxConversationTypeLength = 50
	MaxPaginationLimit        = 100
)

// Pagination constants (use existing constants from conversation_handler.go)
const (
	defaultOffset   = 0                  // Default offset for pagination
	defaultPageSize = 25                 // Default number of items per page
	maxPageSize     = MaxPaginationLimit // Use existing MaxPaginationLimit constant
)

// PaginationParams holds parsed pagination parameters
type PaginationParams struct {
	Offset int
	Limit  int
}

// ValidationResult holds validation results with errors
type ValidationResult struct {
	Valid  bool
	Errors []string
}

// ParsePaginationParamsWithValidation extracts and validates pagination parameters with error reporting
func ParsePaginationParamsWithValidation(c fiber.Ctx) (PaginationParams, ValidationResult) {
	params := PaginationParams{}
	result := ValidationResult{Valid: true, Errors: []string{}}

	// Parse offset with validation
	offsetParam := c.Query("offset", "0")
	if offsetParam != "" {
		offset, err := strconv.Atoi(offsetParam)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Invalid offset parameter: %s", offsetParam))
			params.Offset = defaultOffset
		} else if offset < 0 {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Offset must be non-negative, got: %d", offset))
			params.Offset = defaultOffset
		} else {
			params.Offset = offset
		}
	} else {
		params.Offset = defaultOffset
	}

	// Parse limit with validation
	limitParam := c.Query("limit", strconv.Itoa(defaultPageSize))
	if limitParam != "" {
		limit, err := strconv.Atoi(limitParam)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Invalid limit parameter: %s", limitParam))
			params.Limit = defaultPageSize
		} else if limit <= 0 {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Limit must be positive, got: %d", limit))
			params.Limit = defaultPageSize
		} else if limit > maxPageSize {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Limit exceeds maximum allowed (%d), got: %d", maxPageSize, limit))
			params.Limit = maxPageSize
		} else {
			params.Limit = limit
		}
	} else {
		params.Limit = defaultPageSize
	}

	return params, result
}

// ParsePaginationParams extracts and validates pagination parameters (legacy version - silent)
func ParsePaginationParams(c fiber.Ctx) PaginationParams {
	params, _ := ParsePaginationParamsWithValidation(c)
	return params
}

// ValidatePaginationOrError validates pagination and returns error response if invalid
func ValidatePaginationOrError(c fiber.Ctx) (PaginationParams, error) {
	params, validation := ParsePaginationParamsWithValidation(c)
	if !validation.Valid {
		return params, c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Invalid pagination parameters",
			fmt.Sprintf("Validation errors: %v", validation.Errors),
		))
	}
	return params, nil
}

// CalculatePageInfo calculates page information for responses
func CalculatePageInfo(offset, limit int, total int64) (page, totalPages int) {
	page = (offset / limit) + 1
	totalPages = int((total + int64(limit) - 1) / int64(limit))
	return page, totalPages
}
