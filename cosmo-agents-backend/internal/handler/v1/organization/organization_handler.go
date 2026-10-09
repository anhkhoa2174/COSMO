package organization

import (
	"context"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/core"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"github.com/rockship/cosmo-agents-go/internal/repository/organization"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	orgService "github.com/rockship/cosmo-agents-go/internal/service/organization"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	customValidator "github.com/rockship/cosmo-agents-go/pkg/validator"
	"gorm.io/gorm"
)

// OrganizationHandler handles organization endpoints
type OrganizationHandler struct {
	session  core.Session
	repo     *organization.OrganizationRepository
	userRepo *user.UserRepository
	roleRepo *roleRepo.RoleRepository
	service  *orgService.Service
}

// NewOrganizationHandler creates a new organization handler
func NewOrganizationHandler(
	session core.Session,
	repo *organization.OrganizationRepository,
	userRepo *user.UserRepository,
	roleRepo *roleRepo.RoleRepository,
	service *orgService.Service,
) *OrganizationHandler {
	return &OrganizationHandler{
		session:  session,
		repo:     repo,
		userRepo: userRepo,
		roleRepo: roleRepo,
		service:  service,
	}
}

// ListOrganizations handles GET /v1/organization
// @Summary List organizations
// @Tags Organizations V1
// @Param offset query int false "Offset"
// @Param limit query int false "Limit"
// @Success 200 {object} schema.APIResponse[schema.PaginatedResponse[v1schema.OrganizationResponse]]
// @Security BearerAuth
// @Router /v1/organizations [get]
func (h *OrganizationHandler) ListOrganizations(c fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, ErrUserNotAuthenticated.Error(), "",
		))
	}

	// Parse pagination params
	offsetInt, _ := strconv.Atoi(c.Query("offset", "0"))
	limitInt, _ := strconv.Atoi(c.Query("limit", "50"))

	pagination := &baseRepo.PaginationParams{
		Offset: offsetInt,
		Limit:  limitInt,
	}
	pagination.Validate()

	// Fetch organizations using service
	result, err := h.service.GetOrganizationsByUserID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to fetch organizations", err.Error()),
		)
	}

	// Apply pagination to results
	total := len(result)
	start := pagination.Offset
	if start > total {
		start = total
	}
	end := start + pagination.Limit
	if end > total {
		end = total
	}

	var paginatedResult []domain.Organization
	if start < end {
		paginatedResult = result[start:end]
	} else {
		paginatedResult = []domain.Organization{}
	}

	// Convert to response DTOs
	responses := make([]*v1schema.OrganizationResponse, len(paginatedResult))
	for i, org := range paginatedResult {
		responses[i] = v1schema.ToOrganizationResponse(&org)
	}

	paginatedResponse := schema.PaginatedResponse[*v1schema.OrganizationResponse]{
		List:   responses,
		Total:  int64(total),
		Offset: pagination.Offset,
		Limit:  pagination.Limit,
	}

	return c.JSON(schema.SuccessResponse(paginatedResponse))
}

// CreateOrganization handles POST /v1/organizations
// @Summary Create organization
// @Description Creates a new organization and assigns the creator as admin
// @Tags Organizations V1
// @Accept json
// @Produce json
// @Param body body v1schema.CreateOrganizationRequest true "Organization data"
// @Success 200 {object} schema.APIResponse[v1schema.OrganizationResponse] "Successfully created organization"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/organizations [post]
func (h *OrganizationHandler) CreateOrganization(c fiber.Ctx) error {
	// Get user from context
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, ErrUserNotAuthenticated.Error(), "",
		))
	}

	// Parse request
	var req v1schema.CreateOrganizationRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}
	if err := v1validation.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

	var orgOutput *domain.Organization

	// Create organization
	if err := h.executeInTransaction(c.Context(), func(txCtx context.Context) error {
		org := &domain.Organization{
			UserID:                  &userID,
			Name:                    req.Name,
			CompanyURL:              req.CompanyURL,
			CompanyDescription:      req.CompanyDescription,
			CompanyTargetingPersona: req.CompanyTargetingPersona,
			ValueOffering:           req.ValueOffering,
			CRM:                     req.CRM,
			LeadHandling:            req.LeadHandling,
			LeadHandlingOther:       req.LeadHandlingOther,
		}

		if _, err := h.repo.Create(txCtx, org); err != nil {
			return err
		}

		// Create admin role for the creator
		role := &domain.Role{
			UserID:         userID,
			OrganizationID: org.ID,
			Name:           domain.RoleNameAdmin,
			Status:         domain.RoleStatusActive,
		}
		if _, err := h.roleRepo.Upsert(txCtx, role); err != nil {
			return err
		}

		orgOutput = org
		return nil
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
				fiber.StatusNotFound, "Failed to create organization", err.Error(),
			))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to create organization", err.Error()),
		)
	}

	// Convert to response
	response := v1schema.ToOrganizationResponse(orgOutput)
	return c.JSON(schema.SuccessResponse(response))
}

// UpdateOrganization handles PATCH /v1/organizations/:id
// @Summary Update organization
// @Description Updates an organization by ID (admin only)
// @Tags Organizations V1
// @Accept json
// @Produce json
// @Param id path string true "Organization ID (UUID)"
// @Param body body v1schema.UpdateOrganizationRequest true "Updated organization data"
// @Success 200 {object} schema.APIResponse[v1schema.OrganizationResponse] "Successfully updated organization"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 403 {object} schema.APIResponse[any] "Forbidden - admin role required"
// @Failure 404 {object} schema.APIResponse[any] "Organization not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/organizations/{id} [patch]
func (h *OrganizationHandler) UpdateOrganization(c fiber.Ctx) error {
	// Get user from context
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, ErrUserNotAuthenticated.Error(), "",
		))
	}

	// Parse ID
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			schema.ErrorResponse(fiber.StatusBadRequest, "Invalid organization ID", err.Error()),
		)
	}

	// Parse request
	var req v1schema.UpdateOrganizationRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			schema.ErrorResponse(fiber.StatusBadRequest, "Invalid request body", err.Error()),
		)
	}
	if err := v1validation.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

	// Verify user is admin of organization
	role, err := h.roleRepo.FindByUserAndOrganization(c.Context(), userID, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to verify permissions", err.Error()),
		)
	}

	if role == nil || role.Name != domain.RoleNameAdmin {
		return c.Status(fiber.StatusForbidden).JSON(
			schema.ErrorResponse(fiber.StatusForbidden, "Admin role required", ""),
		)
	}

	// Fetch existing organization
	org, err := h.repo.FindByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to fetch organization", err.Error()),
		)
	} else if org == nil {
		return c.Status(fiber.StatusNotFound).JSON(
			schema.ErrorResponse(fiber.StatusNotFound, "Organization not found", ""),
		)
	}

	// Update fields
	if req.Name != nil {
		org.Name = *req.Name
	}
	if req.CompanyURL != nil {
		org.CompanyURL = *req.CompanyURL
	}
	if req.CompanyDescription != nil {
		org.CompanyDescription = *req.CompanyDescription
	}
	if req.CompanyTargetingPersona != nil {
		org.CompanyTargetingPersona = req.CompanyTargetingPersona
	}
	if req.ValueOffering != nil {
		org.ValueOffering = *req.ValueOffering
	}
	if req.CRM != nil {
		org.CRM = *req.CRM
	}
	if req.LeadHandling != nil {
		org.LeadHandling = req.LeadHandling
	}
	if req.LeadHandlingOther != nil {
		org.LeadHandlingOther = *req.LeadHandlingOther
	}

	// Save
	if err := h.repo.Update(c.Context(), org.ID, org); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to update organization", err.Error()),
		)
	}

	// Convert to response
	response := v1schema.ToOrganizationResponse(org)

	return c.JSON(schema.SuccessResponse(response))
}

// AddMember handles POST /v1/organizations/:id/members
// @Summary Add member to organization
// @Description Adds a new member to an organization (admin only)
// @Tags Organizations V1
// @Accept json
// @Produce json
// @Param id path string true "Organization ID (UUID) or 'me' for current user's organization"
// @Param body body v1schema.OrganizationMemberCreateRequest true "Member data"
// @Success 200 {object} schema.APIResponse[v1schema.RoleResponse] "Successfully added member"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 403 {object} schema.APIResponse[any] "Forbidden - admin role required"
// @Failure 404 {object} schema.APIResponse[any] "Organization not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/organizations/{id}/members [post]
func (h *OrganizationHandler) AddMember(c fiber.Ctx) error {
	// Get user from context
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, ErrUserNotAuthenticated.Error(), "",
		))
	}

	// Parse ID (can be "me" or UUID)
	idStr := c.Params("id")

	// Parse request
	var req v1schema.OrganizationMemberCreateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			schema.ErrorResponse(fiber.StatusBadRequest, "Invalid request body", err.Error()),
		)
	}
	if err := v1validation.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

	var orgID uuid.UUID
	var err error

	if idStr == "me" {
		// Get user's organization
		org, err := h.userRepo.FindUserMainOrganization(c.Context(), userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(
					schema.ErrorResponse(fiber.StatusNotFound, "User has no organization", ""),
				)
			}
			return c.Status(fiber.StatusInternalServerError).JSON(
				schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to find main organization", err.Error()),
			)
		}
		orgID = org.ID
	} else {
		orgID, err = uuid.Parse(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(
				schema.ErrorResponse(fiber.StatusBadRequest, "Invalid organization ID", err.Error()),
			)
		}
	}

	// Verify current user is admin of organization
	role, err := h.roleRepo.FindByUserAndOrganization(c.Context(), userID, orgID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to verify permissions", err.Error()),
		)
	}

	if role == nil || role.Name != domain.RoleNameAdmin {
		return c.Status(fiber.StatusForbidden).JSON(
			schema.ErrorResponse(fiber.StatusForbidden, "Admin role required", ""),
		)
	}

	var memberOutput *domain.User
	if err := h.executeInTransaction(c.Context(), func(txCtx context.Context) error {

		member, err := h.userRepo.UpsertByEmail(txCtx, &domain.User{
			Email:    req.Email,
			Name:     req.Name,
			JobTitle: req.JobTitle,
		})
		if err != nil {
			return err
		}

		// Create role for new member
		var roleName domain.RoleName
		if req.Role == "admin" {
			roleName = domain.RoleNameAdmin
		} else {
			roleName = domain.RoleNameMember
		}

		newRole := &domain.Role{
			UserID:         member.ID,
			OrganizationID: orgID,
			Name:           roleName,
			Status:         domain.RoleStatusActive, // Do not need job_title here because old code not use job_title, too
		}

		createdRole, err := h.roleRepo.Upsert(txCtx, newRole)

		if err != nil {
			return err
		}
		if createdRole == nil {
			return errors.New("role creation returned nil")
		}

		memberOutput = member
		return nil
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(
				schema.ErrorResponse(fiber.StatusNotFound, "Failed to add member", err.Error()),
			)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to add member", err.Error()),
		)
	}

	response := v1schema.ToUserResponse(memberOutput)
	return c.JSON(schema.SuccessResponse(response))
}

// AssignUser handles POST /v1/organizations/assign
// @Summary Assign user to organization
// @Description Creates a role linking a user to an organization
// @Tags Organizations V1
// @Accept json
// @Produce json
// @Param body body v1schema.RoleCreateRequest true "Role assignment data"
// @Success 200 {object} schema.APIResponse[v1schema.RoleResponse] "Successfully assigned user"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/organizations/assign [post]
func (h *OrganizationHandler) AssignUser(c fiber.Ctx) error {
	// Get user from context
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, ErrUserNotAuthenticated.Error(), "",
		))
	}

	// Parse request
	var req v1schema.RoleCreateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			schema.ErrorResponse(fiber.StatusBadRequest, "Invalid request body", err.Error()),
		)
	}
	if err := v1validation.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

	var roleOutput *domain.Role

	if err := h.executeInTransaction(c.Context(), func(txCtx context.Context) error {
		// Check if organization exists
		org, err := h.repo.FindByIDAndUserID(txCtx, req.OrganizationID, userID)
		if err != nil {
			return err
		} else if org == nil {
			return errors.New("organization not found")
		}

		// Find user by email or create if not exists
		user, err := h.userRepo.FindByEmail(txCtx, req.Email)
		if err != nil {
			return err
		}

		var assignedUserID uuid.UUID
		if user == nil {
			// Create new user
			newUser := &domain.User{
				Email: req.Email,
			}
			createdUser, err := h.userRepo.Create(txCtx, newUser)
			if err != nil {
				return err
			}
			assignedUserID = createdUser.ID
		} else {
			assignedUserID = user.ID
		}

		// Parse role name
		var roleName domain.RoleName
		if req.Name == "admin" {
			roleName = domain.RoleNameAdmin
		} else {
			roleName = domain.RoleNameMember
		}

		// Create role
		role := &domain.Role{
			UserID:         assignedUserID,
			OrganizationID: req.OrganizationID,
			Name:           roleName,
			JobTitle:       req.JobTitle,
			Status:         domain.RoleStatusActive,
		}

		createdRole, err := h.roleRepo.Upsert(txCtx, role)
		if err != nil {
			return err
		}

		roleOutput = createdRole
		return nil
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
				fiber.StatusNotFound, "Failed to assign user", err.Error(),
			))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError, "Failed to assign user", err.Error()),
		)
	}

	// Convert to response
	response := v1schema.ToRoleResponse(roleOutput)
	return c.JSON(schema.SuccessResponse(response))
}

func (h *OrganizationHandler) executeInTransaction(
	ctx context.Context,
	fn func(context.Context) error,
) error {
	session, err := h.session.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			_ = session.Rollback() // Just ignore error, do not prevent panic
			panic(r)
		}
	}()

	if err := fn(session.Context()); err != nil {
		if err := session.Rollback(); err != nil {
			logger.Logger.Error().Err(err).Msg("rollback failed after transaction error")
		}
		return err
	}

	if err := session.Commit(); err != nil {
		logger.Logger.Error().Err(err).Msg("session commit failed")
		return err
	}

	return nil
}

// Helper functions for organization handler package

var (
	ErrUserNotAuthenticated = errors.New("user not authenticated")
)

var validator = customValidator.New()

// ValidateStruct validates a struct and returns error
func ValidateStruct(s interface{}) error {
	return validator.Validate(s)
}
