package user

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1" // used in swagger comments
	userService "github.com/rockship/cosmo-agents-go/internal/service/user"
)

// UserHandlerWithRelations demonstrates how to use the relations package
// to return user data with relationships (organizations, roles, etc.)
type UserHandlerWithRelations struct {
	userRepo                 *userRepo.UserRepository
	userWithRelationsService *userService.UserWithRelationsService
}

// NewUserHandlerWithRelations creates a new user handler with relations support
func NewUserHandlerWithRelations(
	userRepo *userRepo.UserRepository,
	userWithRelationsService *userService.UserWithRelationsService,
) *UserHandlerWithRelations {
	return &UserHandlerWithRelations{
		userRepo:                 userRepo,
		userWithRelationsService: userWithRelationsService,
	}
}

// @Summary Get current user with organizations
// @Description Get information about the currently authenticated user including their organizations
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.APIResponse[v1schema.UserResponse] "User information with organizations"
// @Failure 401 {object} schema.APIResponse[any] "Not authenticated"
// @Failure 404 {object} schema.APIResponse[any] "User not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Router /v1/users/me-with-organizations [get]
func (h *UserHandlerWithRelations) GetCurrentUserWithOrganizations(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Unauthorized",
			"User not authenticated",
		))
	}

	// Use the service to get user with organizations using relations package
	userResponse, err := h.userWithRelationsService.GetUserWithOrganizationsForResponse(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to get user with organizations",
			err.Error(),
		))
	}

	var _ v1schema.UserResponse // force import for swagger
	return c.JSON(schema.SuccessResponse(userResponse))
}

// GetCurrentUserWithFullRelations returns current user with all their relationships
// This demonstrates how to get all user data including organizations, roles, and notifications
//
// @Summary Get current user with full relations
// @Description Get information about the currently authenticated user including all their relationships
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.APIResponse[v1schema.UserResponse] "User information with full relations"
// @Failure 401 {object} schema.APIResponse[any] "Not authenticated"
// @Failure 404 {object} schema.APIResponse[any] "User not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Router /v1/users/me-with-relations [get]
func (h *UserHandlerWithRelations) GetCurrentUserWithFullRelations(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Unauthorized",
			"User not authenticated",
		))
	}

	// Use the service to get user with all relationships using relations package
	userResponse, err := h.userWithRelationsService.GetUserWithFullRelationsForResponse(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to get user with full relations",
			err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(userResponse))
}

// Example: How to integrate this into existing routes
//
// In cmd/server/routes_v1.go, you could add:
//
// // Enhanced user routes with relations support
// userWithRelationsHandler := v1handler.NewUserHandlerWithRelations(
//     deps.Repos.User,
//     service.NewUserWithRelationsService(
//         deps.Repos.User,
//         helperRepo.NewUserWithRelationsRepository(db),
//     ),
// )
//
// // Add to authenticated group
// authenticatedGroup := v1.Group(middleware.Authenticate())
// {
//     // Existing user routes...
//     authenticatedGroup.Get("/users/me-with-organizations", userWithRelationsHandler.GetCurrentUserWithOrganizations)
//     authenticatedGroup.Get("/users/me-with-relations", userWithRelationsHandler.GetCurrentUserWithFullRelations)
// }
//
// This provides endpoints that return the same data structure as before,
// but use the relations package instead of direct domain relationships.
