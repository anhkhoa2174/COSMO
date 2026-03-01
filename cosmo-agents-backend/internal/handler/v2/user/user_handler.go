package user

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
)

// UserHandler handles V2 user-related requests
type UserHandler struct {
	userRepo *user.UserRepository
}

// NewUserHandler creates a new V2 UserHandler
func NewUserHandler(userRepo *user.UserRepository) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
	}
}

// WhoAmI returns the current authenticated user
// GET /v2/users/me
func (h *UserHandler) WhoAmI(c fiber.Ctx) error {
	// Get user from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized",
		})
	}

	// Fetch user from database
	user, err := h.userRepo.GetDetailByID(c.Context(), userID)
	if err != nil || user == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "User not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   user,
	})
}
