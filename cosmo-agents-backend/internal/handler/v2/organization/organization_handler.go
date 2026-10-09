package organization

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/core"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	"github.com/rockship/cosmo-agents-go/internal/repository/organization"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
	"github.com/rockship/cosmo-agents-go/internal/service/productivity"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

// OrganizationHandler handles V2 organization-related requests
type OrganizationHandler struct {
	session      core.Session
	orgRepo      *organization.OrganizationRepository
	userRepo     *user.UserRepository
	roleRepo     *roleRepo.RoleRepository
	workerClient workerClient

	// Optional: set with WithProductivity. Left nil the team-productivity
	// endpoint reports itself unconfigured rather than panicking.
	productivity *productivity.Service
}

// WithProductivity enables the team-productivity endpoint.
func (h *OrganizationHandler) WithProductivity(svc *productivity.Service) *OrganizationHandler {
	h.productivity = svc
	return h
}

type workerClient interface {
	EnqueueLowPriorityTask(ctx context.Context, taskType string, payload interface{}, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// NewOrganizationHandler creates a new V2 OrganizationHandler
func NewOrganizationHandler(
	session core.Session,
	orgRepo *organization.OrganizationRepository,
	userRepo *user.UserRepository,
	roleRepo *roleRepo.RoleRepository,
	workerClient workerClient,
) *OrganizationHandler {
	return &OrganizationHandler{
		session:      session,
		orgRepo:      orgRepo,
		userRepo:     userRepo,
		roleRepo:     roleRepo,
		workerClient: workerClient,
	}
}

// GetDetail returns organization details
// @Summary Get organization details
// @Description Get details of a specific organization by ID
// @Tags Organizations V2
// @Accept json
// @Produce json
// @Param organization_id path string true "Organization ID" format(uuid)
// @Success 200 {object} schema.APIResponse[v2schema.OrganizationReadResponse] "Successfully retrieved organization"
// @Failure 400 {object} schema.APIResponse[any] "Invalid organization ID"
// @Failure 404 {object} schema.APIResponse[any] "Organization not found"
// @Security BearerAuth
// @Router /v2/organizations/{organization_id} [get]
func (h *OrganizationHandler) GetDetail(c fiber.Ctx) error {
	// Get user from context
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(
			schema.ErrorResponse(fiber.StatusUnauthorized, "Unauthorized", ""),
		)
	}

	organizationID, err := uuid.Parse(c.Params("organization_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Invalid organization ID",
			err.Error(),
		))
	}

	org, err := h.orgRepo.FindByID(c.Context(), organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
				fiber.StatusNotFound,
				"Organization not found",
				"",
			))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to find organization",
			err.Error(),
		))
	}

	response := v2schema.OrganizationReadResponse{
		ID:                      org.ID,
		Name:                    &org.Name,
		CompanyURL:              &org.CompanyURL,
		CompanyDescription:      &org.CompanyDescription,
		CompanyTargetingPersona: (*[]string)(&org.CompanyTargetingPersona),
		ValueOffering:           &org.ValueOffering,
	}

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(response))
}

// InviteMember invites a member to the organization
// @Summary Invite a member to the organization
// @Description Invites a user to join the organization with the specified role
// @Tags Organizations V2
// @Accept json
// @Produce json
// @Param organization_id path string true "Organization ID or 'me' for current user's organization"
// @param request body v2schema.OrganizationMemberCreateRequest true "Invitation details"
// @Success 200 {object} schema.APIResponse[v2schema.OrganizationMemberCreateResponse] "Successfully invited member"
// @Success 200 {object} schema.APIResponse[any] "User not found, invitation required"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request body or organization ID"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 403 {object} schema.APIResponse[any] "Forbidden - Only admins can invite members"
// @Failure 404 {object} schema.APIResponse[any] "User or organization not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v2/organizations/{organization_id}/members/invite [post]
// InviteMember invites a member to the organization
// POST /v2/organizations/{organization_id}/members/invite
func (h *OrganizationHandler) InviteMember(c fiber.Ctx) error {
	organizationIDParam := c.Params("organization_id")

	// Get current user from context
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Unauthorized",
			"",
		))
	}

	var req v2schema.OrganizationMemberCreateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Invalid request body",
			err.Error(),
		))
	}
	if err := v1validation.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

	// Parse or resolve organization ID
	var organizationID uuid.UUID
	if organizationIDParam == "me" {
		// Get user's main organization
		org, err := h.userRepo.FindUserMainOrganization(c.Context(), userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
					fiber.StatusNotFound,
					"No organization found for user",
					"",
				))
			}
			return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
				fiber.StatusInternalServerError,
				"Failed to get user's organization",
				err.Error(),
			))
		}
		organizationID = org.ID
	} else {
		parsedID, err := uuid.Parse(organizationIDParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest,
				"Invalid organization ID",
				err.Error(),
			))
		}
		organizationID = parsedID
	}

	// Check if current user is admin of the organization
	role, err := h.roleRepo.FindByUserAndOrganization(c.Context(), userID, organizationID)
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

	// Build state parameters for OAuth flow
	stateParams := url.Values{}
	stateParams.Set("organization_id", organizationID.String())
	if req.Role != nil {
		stateParams.Set("role", string(*req.Role))
	} else {
		stateParams.Set("role", string(domain.RoleNameMember))
	}
	stateParams.Set("email", req.Email)
	if req.JobTitle != nil {
		stateParams.Set("job_title", *req.JobTitle)
	}
	state := url.QueryEscape(stateParams.Encode())

	// Build redirect URI
	fullURL := string(c.Request().URI().FullURI())
	urlSplit := strings.Split(fullURL, "/v2")
	baseURL := urlSplit[0]

	var redirectURI string
	if req.RedirectURI == nil {
		callbackURL := baseURL + "/v2/auth/members/oauth2callback"
		redirectURI = baseURL + "/v2/auth/members/invite-callback?redirect_uri=" + url.QueryEscape(callbackURL) + "&state=" + state
	} else {
		redirectURI = *req.RedirectURI + "?state=" + state
	}

	// Find or create the invited user
	user, err := h.userRepo.FindByEmail(c.Context(), req.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to find user",
			err.Error(),
		))
	}

	// Check if user already belongs to another organization
	if user != nil {
		// TODO: remove this if support multiple orgs
		org, err := h.userRepo.FindFirstOrganizationOfUser(c.Context(), user.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
				fiber.StatusInternalServerError,
				"Failed to check user's organization",
				err.Error(),
			))
		}

		if org != nil {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest,
				"User already exists in another organization",
				fmt.Sprintf("User with email %s already exists in another organization", req.Email),
			))
		}

		role, err := h.roleRepo.FindByUserAndOrganization(c.Context(), user.ID, organizationID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
				fiber.StatusInternalServerError,
				"Failed to check user's role",
				err.Error(),
			))
		}

		if role != nil {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest,
				"User already exists in another organization",
				fmt.Sprintf("User with email %s already exists in another organization", req.Email),
			))
		}
	}

	// Create or update user
	userDataToUpdate := domain.User{}
	if user != nil {
		// For case update, we only use email, name, but add old job title to prevent update to null (maybe)
		userDataToUpdate = domain.User{
			Email:    req.Email,
			Name:     *req.Name,
			JobTitle: user.JobTitle,
		}
	} else {
		// For case create, we only use email, name like python code
		userDataToUpdate = domain.User{
			Email: req.Email,
			Name:  *req.Name,
		}
	}
	var roleOutput *domain.Role
	var userOutput *domain.User

	if err := h.executeInTransaction(c.Context(), func(txCtx context.Context) error {
		user, err = h.userRepo.UpsertByEmail(txCtx, &userDataToUpdate)
		if err != nil {
			return err
		}

		// Create or update role for the invited user
		roleName := domain.RoleNameMember
		if req.Role != nil {
			roleName = *req.Role
		}

		newRole := &domain.Role{
			UserID:         user.ID,
			OrganizationID: organizationID,
			Name:           roleName,
			Status:         domain.RoleStatusPending,
		}

		if req.JobTitle != nil {
			newRole.JobTitle = *req.JobTitle
		}

		createdRole, err := h.roleRepo.Upsert(txCtx, newRole)
		if err != nil {
			return err
		}

		// Prepare email payload
		payload := v2schema.InviteMemberPayload{
			MemberName: *req.Name,
			Email:      req.Email,
			URL:        redirectURI,
		}

		// Enqueue email task within the same transaction to ensure atomicity
		if h.workerClient == nil {
			return fmt.Errorf("worker client not configured")
		}

		if _, err := h.workerClient.EnqueueLowPriorityTask(txCtx, worker.TypeSendInviteMemberEmail, payload); err != nil {
			return fmt.Errorf("failed to enqueue invitation email: %w", err)
		}

		userOutput = user
		roleOutput = createdRole
		return nil
	}); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to invite member",
			err.Error(),
		))
	}

	response := v2schema.OrganizationMemberCreateResponse{
		InvitedUser: userOutput,
		Role:        roleOutput,
	}

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(response))
}

// SearchMembers searches organization members
// @Summary Search organization members
// @Description Search and filter organization members with pagination support
// @Tags Organizations V2
// @Accept json
// @Produce json
// @Param organization_id path string true "Organization ID or 'me' for current user's organization"
// @Param offset query int false "Pagination offset" default(0)
// @Param limit query int false "Pagination limit (max 100)" default(25) maximum(100)
// @Param request body v2schema.OrganizationMemberSearchRequest true "Search criteria"
// @Success 200 {object} schema.APIResponse[v2schema.OrganizationMemberSearchResponse] "Successfully retrieved members"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request or organization ID"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 404 {object} schema.APIResponse[any] "Organization or user not found"
// @Security BearerAuth
// @Router /v2/organizations/{organization_id}/members/search [post]
func (h *OrganizationHandler) SearchMembers(c fiber.Ctx) error {
	organizationIDParam := c.Params("organization_id")

	// Get current user from context
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Unauthorized",
			"",
		))
	}

	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	if limit > 100 {
		limit = 100
	}

	var req v2schema.OrganizationMemberSearchRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Invalid request body",
			err.Error(),
		))
	}
	if err := v1validation.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

	members, total, err := h.orgRepo.GetManyMembersWithTotal(
		c.Context(),
		userID,
		organizationIDParam,
		offset*limit,
		limit,
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to get members",
			err.Error(),
		))
	}

	response := v2schema.OrganizationMemberSearchResponse{
		List:   members,
		Offset: offset,
		Limit:  limit,
		Total:  total,
	}

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(response))
}

// RemoveMembers removes members from organization
// @Summary Remove members from organization
// @Description Removes one or more members from the specified organization. Only organization admins can perform this action.
// @Tags Organizations V2
// @Accept json
// @Produce json
// @Param organization_id path string true "Organization ID or 'me' for current user's organization"
// @param request body v2schema.OrganizationMemberDeleteRequest true "Member IDs to remove"
// @Success 200 {object} schema.APIResponse[any] "Successfully removed members"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request body or organization ID"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 403 {object} schema.APIResponse[any] "Forbidden - Only admins can remove members"
// @Security BearerAuth
// @Router /v2/organizations/{organization_id}/members/remove [delete]
func (h *OrganizationHandler) RemoveMembers(c fiber.Ctx) error {
	organizationIDParam := c.Params("organization_id")

	// Get current user from context
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Unauthorized",
			"",
		))
	}

	var req v2schema.OrganizationMemberDeleteRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Invalid request body",
			err.Error(),
		))
	}
	if err := v1validation.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

	// Parse or resolve organization ID
	var organizationID uuid.UUID
	if organizationIDParam == "me" {
		// Get user's main organization
		org, err := h.userRepo.FindUserMainOrganization(c.Context(), userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
					fiber.StatusNotFound,
					"No organization found for user",
					"",
				))
			}
			return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
				fiber.StatusInternalServerError,
				"Failed to get user's organization",
				err.Error(),
			))
		}
		organizationID = org.ID
	} else {
		parsedID, err := uuid.Parse(organizationIDParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest,
				"Invalid organization ID",
				err.Error(),
			))
		}
		organizationID = parsedID
	}

	// Check if current user is admin of the organization
	role, err := h.roleRepo.FindByUserAndOrganization(c.Context(), userID, organizationID)
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

	// Soft delete roles for the specified members
	if err := h.executeInTransaction(c.Context(), func(txCtx context.Context) error {
		if len(req.MemberIDs) > 0 {
			err := h.orgRepo.SoftDeleteManyByUserID(txCtx, organizationID, req.MemberIDs)
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to remove members",
			err.Error(),
		))
	}

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse("Deleted Successfully"))
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
