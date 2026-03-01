package validation

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	customValidator "github.com/rockship/cosmo-agents-go/pkg/validator"
)

var validator = customValidator.New()

// ValidateRequest validates a request body and returns error response if validation fails
func ValidateRequest(c fiber.Ctx, request interface{}) error {
	// Bind request body
	if err := c.Bind().JSON(request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Invalid request body",
			err.Error(),
		))
	}

	// Validate
	if err := validator.Validate(request); err != nil {
		if validationErr, ok := err.(customValidator.ValidationErrors); ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status": "error",
				"error": fiber.Map{
					"code":    fiber.StatusBadRequest,
					"message": "Validation failed",
					"details": validationErr.Errors,
				},
			})
		}

		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Validation failed",
			err.Error(),
		))
	}

	return nil
}

// ValidateStruct validates a struct and returns error
func ValidateStruct(s interface{}) error {
	return validator.Validate(s)
}
