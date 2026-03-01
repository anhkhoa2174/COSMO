package agent

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// GetAuthenticatedUser extracts and validates user ID from Fiber context
func GetAuthenticatedUser(c fiber.Ctx) (uuid.UUID, error) {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return uuid.Nil, fiber.ErrUnauthorized
	}
	return userID, nil
}

// ParseAgentID validates and parses agent ID from URL parameters
func ParseAgentID(c fiber.Ctx) (uuid.UUID, error) {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// ValidateOrganizationAccess checks if user has access to the specified organization
func (h *AgentHandler) ValidateOrganizationAccess(c fiber.Ctx, userID, orgID uuid.UUID) error {
	role, err := h.roleRepo.FindByUserAndOrganization(c.Context(), userID, orgID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, ErrFailedToLoadUserRoles, err.Error(),
		))
	}
	if role == nil {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, ErrAccessDenied, "You don't have permission to access this organization",
		))
	}
	return nil
}

// CheckAgentAccessPermission verifies if user can access the specified agent
func (h *AgentHandler) CheckAgentAccessPermission(c fiber.Ctx, userID, agentID uuid.UUID) (*domain.Agent, error) {
	agent, err := h.agentRepo.FindByID(c.Context(), agentID)
	if err != nil {
		return nil, c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, ErrAgentNotFound, err.Error(),
		))
	}
	if agent == nil {
		return nil, c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, ErrAgentNotFound, "",
		))
	}

	// Check access permissions: owners can always access, others need organization membership
	hasAccess := false
	if agent.UserID == userID {
		hasAccess = true
	} else if agent.OrganizationID != nil {
		role, err := h.roleRepo.FindByUserAndOrganization(c.Context(), userID, *agent.OrganizationID)
		if err != nil {
			return nil, c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
				fiber.StatusInternalServerError, ErrFailedToLoadUserRoles, err.Error(),
			))
		}
		hasAccess = role != nil
	}

	if !hasAccess {
		return nil, c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, ErrAgentNotFound, "",
		))
	}

	return agent, nil
}

// ValidateAndParseFilter validates and processes search filters
func ValidateAndParseFilter(filter map[string]interface{}) (baseRepo.Filter, error) {
	result := make(baseRepo.Filter)

	for k, v := range filter {
		if !AllowedAgentFilterKeys[k] {
			logger.Logger.Warn().Str("filter_key", k).Msg("Attempted to use disallowed filter key")
			continue
		}

		if v == nil {
			continue
		}

		valueStr := fmt.Sprintf("%v", v)
		if valueStr == "" {
			continue
		}

		if len(valueStr) > MaxFilterValueLength {
			return nil, fmt.Errorf("filter '%s' value exceeds maximum length of %d characters", k, MaxFilterValueLength)
		}

		switch k {
		case "organization_id":
			if _, err := uuid.Parse(valueStr); err != nil {
				return nil, fmt.Errorf("organization_id must be a valid UUID")
			}
		case "active":
			switch valueStr {
			case "true", "false", "1", "0":
			default:
				return nil, fmt.Errorf("active must be a boolean value (true/false)")
			}
		}

		result[k] = v
	}

	return result, nil
}
