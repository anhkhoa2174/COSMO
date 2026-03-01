package sale_rep

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	saleRepRepo "github.com/rockship/cosmo-agents-go/internal/repository/sale_rep"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	s3Service "github.com/rockship/cosmo-agents-go/internal/service/s3"
)

// SaleRepHandler handles sale rep-related HTTP requests
type SaleRepHandler struct {
	saleRepRepo *saleRepRepo.SaleRepRepository
	userRepo    *userRepo.UserRepository
	s3Service   *s3Service.S3Service
	bucketName  string
}

// NewSaleRepHandler creates a new SaleRepHandler
func NewSaleRepHandler(saleRepRepo *saleRepRepo.SaleRepRepository, userRepo *userRepo.UserRepository, s3Service *s3Service.S3Service, bucketName string) *SaleRepHandler {
	return &SaleRepHandler{
		saleRepRepo: saleRepRepo,
		userRepo:    userRepo,
		s3Service:   s3Service,
		bucketName:  bucketName,
	}
}

// @Summary Create sale rep
// @Description Creates a new sale rep with the provided details
// @Tags Sale-reps V1
// @Accept multipart/form-data
// @Param first_name formData string true "First name"
// @Param last_name formData string true "Last name"
// @Param email formData string true "Email"
// @Param calendar_link formData string false "Calendar link"
// @Param picture formData file false "Profile picture"
// @Success 201 {object} schema.APIResponse[v1schema.SaleRepResponse] "Successfully created sale rep"
// @Failure 400 {object} schema.APIResponse[any] "Bad Request - Invalid input or validation failed"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized - User not authenticated"
// @Failure 409 {object} schema.APIResponse[any] "Conflict - Sale rep with this email already exists"
// @Failure 500 {object} schema.APIResponse[any] "Internal Server Error"
// @Security BearerAuth
// @Router /v1/sale-reps [post]
func (h *SaleRepHandler) Create(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	req := v1schema.CreateSaleRepRequest{
		FirstName: c.FormValue("first_name"),
		LastName:  c.FormValue("last_name"),
		Email:     c.FormValue("email"),
	}

	if v := c.FormValue("calendar_link"); v != "" {
		req.CalendarLink = &v
	}

	if v := c.FormValue("picture"); v != "" {
		req.Picture = &v
	}

	if err := v1validation.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

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

	// Check if sale rep already exists
	if _, err := h.saleRepRepo.GetByUserIDAndEmail(c.Context(), userID, req.Email); err == nil {
		return c.Status(fiber.StatusConflict).JSON(schema.ErrorResponse(
			fiber.StatusConflict,
			"Sale rep with this email already exists",
			"",
		))
	}

	// Create domain sale rep
	saleRep := &domain.SaleRep{
		UserID:         userID,
		OrganizationID: org.ID,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Email:          req.Email,
	}

	if req.CalendarLink != nil {
		saleRep.CalendarLink = *req.CalendarLink
	}
	if req.Picture != nil {
		saleRep.Picture = *req.Picture
	}

	if saleRep, err = h.saleRepRepo.Create(c.Context(), saleRep); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to create sale rep", err.Error(),
		))
	}

	response := v1schema.SaleRepResponse{
		ID:           saleRep.ID,
		UserID:       saleRep.UserID,
		FirstName:    saleRep.FirstName,
		LastName:     saleRep.LastName,
		Email:        saleRep.Email,
		CalendarLink: &saleRep.CalendarLink,
		Picture:      &saleRep.Picture,
		CreatedAt:    saleRep.CreatedAt,
		UpdatedAt:    saleRep.UpdatedAt,
	}

	return c.Status(fiber.StatusCreated).JSON(schema.SuccessResponse(response))
}

// GetByID handles GET /v1/sale-reps/:id
// @Summary Get sale-rep. This API may work incorrectly because python version of this API is getting error, and frontend doesn't use
// @Description Get a sale rep by their ID
// @Tags Sale-reps V1
// @Accept json
// @Produce json
// @Param id path string true "Sale Rep ID"
// @Success 200 {object} schema.APIResponse[v1schema.SaleRepResponse] "Successfully retrieved sale rep"
// @Failure 400 {object} schema.APIResponse[any] "Bad Request - Invalid ID format"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized - User not authenticated"
// @Failure 403 {object} schema.APIResponse[any] "Forbidden - Access denied"
// @Failure 404 {object} schema.APIResponse[any] "Not Found - Sale rep not found"
// @Security BearerAuth
// @Router /v1/sale-reps/{id} [get]
func (h *SaleRepHandler) GetByID(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	id := c.Params("id")
	saleRepID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid sale rep ID", err.Error(),
		))
	}

	saleRep, err := h.saleRepRepo.FindByID(c.Context(), saleRepID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Sale rep not found", err.Error(),
		))
	}

	// Check ownership
	if saleRep.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, "Access denied", "",
		))
	}

	response := v1schema.SaleRepResponse{
		ID:           saleRep.ID,
		UserID:       saleRep.UserID,
		FirstName:    saleRep.FirstName,
		LastName:     saleRep.LastName,
		Email:        saleRep.Email,
		CalendarLink: &saleRep.CalendarLink,
		Picture:      &saleRep.Picture,
		CreatedAt:    saleRep.CreatedAt,
		UpdatedAt:    saleRep.UpdatedAt,
	}

	return c.JSON(schema.SuccessResponse(response))
}

// Search handles POST /v1/sale-reps/search
// @Summary Search sale reps
// @Description Search and filter sale reps with pagination
// @Tags Sale-reps V1
// @Accept json
// @Produce json
// @Param request body v1schema.SaleRepSearchRequest true "Search criteria and filters"
// @Param offset query int false "Pagination offset" default(0)
// @Param limit query int false "Pagination limit, max 100" default(25) maximum(100)
// @Success 200 {object} schema.APIResponse[v1schema.SaleRepSearchResponse] "Successfully retrieved sale reps"
// @Failure 400 {object} schema.APIResponse[any] "Bad Request - Invalid input or validation failed"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized - User not authenticated"
// @Failure 500 {object} schema.APIResponse[any] "Internal Server Error"
// @Security BearerAuth
// @Router /v1/sale-reps/search [post]
func (h *SaleRepHandler) Search(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	var req v1schema.SaleRepSearchRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	if limit > 100 {
		limit = 100
	}

	// Merge user filter
	if req.Filter == nil {
		req.Filter = make(map[string]interface{})
	}

	pagination := baseRepo.PaginationParams{
		Offset: offset * limit, // offset in swagger means page, but offset use in gorm is skip
		Limit:  limit,
	}

	result, total, err := h.saleRepRepo.GetByUserID(c.Context(), userID, req.Filter, &pagination)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to search sale reps", err.Error(),
		))
	}

	list := make([]v1schema.SaleRepListItem, len(result))
	for i, sr := range result {
		if sr.Picture != "" {
			if h.s3Service != nil {
				expiration := 3600 // default 1 hour
				if expirationStr := c.Query("expiration"); expirationStr != "" {
					if val, err := strconv.Atoi(expirationStr); err == nil {
						expiration = val
					}
				}
				url, err := h.s3Service.GeneratePresignedURL(c.Context(), h.bucketName, sr.Picture, time.Duration(expiration)*time.Second)
				if err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
						fiber.StatusInternalServerError, "Failed to generate presigned URL", err.Error(),
					))
				}
				sr.Picture = url
			}

		}
		var calendarLinkPtr *string
		if sr.CalendarLink != "" {
			calendarLinkPtr = &sr.CalendarLink
		}
		var picturePtr *string
		if sr.Picture != "" {
			picturePtr = &sr.Picture
		}
		list[i] = v1schema.SaleRepListItem{
			Entity: v1schema.SaleRepResponse{
				ID:           sr.ID,
				UserID:       sr.UserID,
				FirstName:    sr.FirstName,
				LastName:     sr.LastName,
				Email:        sr.Email,
				CalendarLink: calendarLinkPtr,
				Picture:      picturePtr,
				CreatedAt:    sr.CreatedAt,
				UpdatedAt:    sr.UpdatedAt,
			},
		}
	}

	response := v1schema.SaleRepSearchResponse{
		List:   list,
		Offset: offset,
		Limit:  limit,
		Total:  total,
	}

	return c.JSON(schema.SuccessResponse(response))
}

// Update handles PATCH /v1/sale-reps/:id
// @Summary Update a sale rep
// @Description Update an existing sale rep's information
// @Tags Sale-reps V1
// @Accept json
// @Produce json
// @Param id path string true "Sale Rep ID"
// @Param request body v1schema.UpdateSaleRepRequest true "Sale rep update data"
// @Success 200 {object} schema.APIResponse[v1schema.SaleRepResponse] "Successfully updated sale rep"
// @Failure 400 {object} schema.APIResponse[any] "Bad Request - Invalid input or validation failed"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized - User not authenticated"
// @Failure 403 {object} schema.APIResponse[any] "Forbidden - Access denied"
// @Failure 404 {object} schema.APIResponse[any] "Not Found - Sale rep not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal Server Error"
// @Security BearerAuth
// @Router /v1/sale-reps/{id} [patch]
func (h *SaleRepHandler) Update(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	id := c.Params("id")
	saleRepID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid sale rep ID", err.Error(),
		))
	}

	var req v1schema.UpdateSaleRepRequest
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

	// Get existing sale rep
	existingSaleRep, err := h.saleRepRepo.FindByID(c.Context(), saleRepID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Sale rep not found", err.Error(),
		))
	} else if existingSaleRep == nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Sale rep not found", "",
		))
	}
	// Check ownership
	if existingSaleRep.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, "Access denied", "",
		))
	}

	saleRep := domain.SaleRep{}

	// Update fields
	if req.FirstName != nil {
		saleRep.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		saleRep.LastName = *req.LastName
	}
	if req.CalendarLink != nil {
		saleRep.CalendarLink = *req.CalendarLink
	}
	if req.Picture != nil {
		saleRep.Picture = *req.Picture
	}

	if err := h.saleRepRepo.Update(c.Context(), saleRepID, &saleRep); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to update sale rep", err.Error(),
		))
	}

	response := v1schema.SaleRepResponse{
		ID:           existingSaleRep.ID,
		UserID:       existingSaleRep.UserID,
		FirstName:    existingSaleRep.FirstName,
		LastName:     existingSaleRep.LastName,
		Email:        existingSaleRep.Email,
		CalendarLink: &existingSaleRep.CalendarLink,
		Picture:      &existingSaleRep.Picture,
		CreatedAt:    existingSaleRep.CreatedAt,
		UpdatedAt:    existingSaleRep.UpdatedAt,
	}

	if req.FirstName != nil {
		response.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		response.LastName = *req.LastName
	}
	if req.CalendarLink != nil {
		response.CalendarLink = req.CalendarLink
	}
	if req.Picture != nil {
		response.Picture = req.Picture
	}

	return c.JSON(schema.SuccessResponse(response))
}

// Delete handles DELETE /v1/sale-reps/:id
// @Summary Delete a sale rep
// @Description Delete an existing sale rep
// @Tags Sale-reps V1
// @Accept json
// @Produce json
// @Param id path string true "Sale Rep ID"
// @Success 200 {object} schema.APIResponse[string] "Successfully deleted sale rep"
// @Failure 400 {object} schema.APIResponse[any] "Bad Request - Invalid input or validation failed"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized - User not authenticated"
// @Failure 403 {object} schema.APIResponse[any] "Forbidden - Access denied"
// @Failure 404 {object} schema.APIResponse[any] "Not Found - Sale rep not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal Server Error"
// @Security BearerAuth
// @Router /v1/sale-reps/{id} [delete]
func (h *SaleRepHandler) Delete(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	id := c.Params("id")
	saleRepID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid sale rep ID", err.Error(),
		))
	}

	// Get existing sale rep
	saleRep, err := h.saleRepRepo.FindByID(c.Context(), saleRepID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Sale rep not found", err.Error(),
		))
	}

	// Check ownership
	if saleRep.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, "Access denied", "",
		))
	}

	if err := h.saleRepRepo.HardDelete(c.Context(), saleRepID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to delete sale rep", err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse("Sale rep deleted successfully"))
}
