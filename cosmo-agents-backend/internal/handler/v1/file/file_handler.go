package file

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	fileRepo "github.com/rockship/cosmo-agents-go/internal/repository/file"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	s3Service "github.com/rockship/cosmo-agents-go/internal/service/s3"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// FileHandler handles file-related HTTP requests
type FileHandler struct {
	fileRepo   *fileRepo.FileRepository
	s3Service  *s3Service.S3Service
	bucketName string
}

// NewFileHandler creates a new FileHandler
func NewFileHandler(fileRepo *fileRepo.FileRepository, s3Service *s3Service.S3Service, bucketName string) *FileHandler {
	return &FileHandler{
		fileRepo:   fileRepo,
		s3Service:  s3Service,
		bucketName: bucketName,
	}
}

// List handles GET /v1/files
// @Summary List user's files
// @Description Get all files belonging to the current user
// @Tags Files
// @Accept json
// @Produce json
// @Param offset query int false "Pagination offset" default(0)
// @Param limit query int false "Pagination limit" default(50)
// @Success 200 {object} map[string]interface{} "List of files"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /v1/files [get]
func (h *FileHandler) List(c fiber.Ctx) error {
	// Get current user from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "Unauthorized", "User ID not found in context",
		))
	}

	// Parse pagination
	offset := 0
	limit := 50

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil {
			offset = val
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil {
			limit = val
		}
	}

	pagination := &baseRepo.PaginationParams{
		Offset: offset,
		Limit:  limit,
	}

	result, err := h.fileRepo.FindByUserID(c.Context(), userID, pagination)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to fetch files", err.Error(),
		))
	}

	// Convert to response
	fileResponses := make([]*v1schema.FileResponse, len(result.List))
	for i, file := range result.List {
		resp := v1schema.ToFileResponse(&file)
		fileResponses[i] = &resp
	}

	paginatedResponse := schema.PaginatedResponse[*v1schema.FileResponse]{
		List:   fileResponses,
		Total:  result.Total,
		Offset: result.Offset,
		Limit:  result.Limit,
	}

	return c.JSON(schema.SuccessResponse(paginatedResponse))
}

// Search handles POST /v1/files/search
// @Summary Search files
// @Description Search files with custom filters
// @Tags Files
// @Accept json
// @Produce json
// @Param body body map[string]interface{} true "Search filters"
// @Success 200 {object} map[string]interface{} "List of files"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /v1/files/search [post]
func (h *FileHandler) Search(c fiber.Ctx) error {
	var body map[string]interface{}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	// Convert body to filter
	filter := baseRepo.Filter{}
	for key, value := range body {
		filter[key] = value
	}

	result, err := h.fileRepo.FindAll(c.Context(), filter, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to search files", err.Error(),
		))
	}

	// Convert to response
	fileResponses := make([]v1schema.FileResponse, len(result.List))
	for i, file := range result.List {
		fileResponses[i] = v1schema.ToFileResponse(&file)
	}

	return c.JSON(schema.SuccessResponse(fileResponses))
}

// UploadToS3 handles POST /v1/files/s3
// @Summary Upload file to S3
// @Description Upload a file to S3 and create a database record
// @Tags Files
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to upload"
// @Success 200 {object} map[string]interface{} "Upload success with file info"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /v1/files/s3 [post]
func (h *FileHandler) UploadToS3(c fiber.Ctx) error {
	// Get current user
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "Unauthorized", "User ID not found in context",
		))
	}

	// Get file from form
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Failed to get file from request", err.Error(),
		))
	}

	// Open file
	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to open file", err.Error(),
		))
	}
	defer file.Close()

	// Generate S3 key with timestamp
	folderName := "examples/"
	timestamp := time.Now().Format("15:04:05_02_01_2006")
	s3Key := fmt.Sprintf("%s%s_%s", folderName, timestamp, fileHeader.Filename)

	// Upload to S3
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	err = h.s3Service.UploadFile(c.Context(), h.bucketName, s3Key, file, contentType, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to upload file to S3", err.Error(),
		))
	}

	// Create database record
	fileRecord := &domain.File{
		UserID:   &userID,
		Filename: fileHeader.Filename,
		MimeType: contentType,
		S3Key:    s3Key,
		Size:     fileHeader.Size,
	}

	if _, err := h.fileRepo.Create(c.Context(), fileRecord); err != nil {
		if cleanupErr := h.s3Service.DeleteObject(c.Context(), h.bucketName, s3Key); cleanupErr != nil {
			logger.Logger.Error().
				Err(cleanupErr).
				Str("s3_key", s3Key).
				Msg("failed to cleanup S3 object after DB persistence error")
		}
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to create file record", err.Error(),
		))
	}

	response := map[string]interface{}{
		"message": "File uploaded successfully",
		"file":    v1schema.ToFileResponse(fileRecord),
	}

	return c.JSON(schema.SuccessResponse(response))
}

// GetFromS3 handles GET /v1/files/s3
// @Summary Get file from S3
// @Description Download a file from S3 by key
// @Tags Files
// @Produce application/octet-stream
// @Param key query string true "S3 object key"
// @Success 200 {file} binary "File content"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /v1/files/s3 [get]
func (h *FileHandler) GetFromS3(c fiber.Ctx) error {
	key := c.Query("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Missing key parameter", "",
		))
	}

	data, err := h.s3Service.GetObject(c.Context(), h.bucketName, key)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to get file from S3", err.Error(),
		))
	}

	// Try to determine content type from file extension
	ext := filepath.Ext(key)
	contentType := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".pdf":
		contentType = "application/pdf"
	case ".txt":
		contentType = "text/plain"
	}

	c.Set("Content-Type", contentType)
	return c.Send(data)
}

// ListFromS3 handles GET /v1/files/s3/list
// @Summary List files from S3
// @Description List all files in S3 bucket with optional prefix filter
// @Tags Files
// @Accept json
// @Produce json
// @Param prefix query string false "S3 key prefix filter"
// @Success 200 {object} map[string]interface{} "List of S3 objects"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /v1/files/s3/list [get]
func (h *FileHandler) ListFromS3(c fiber.Ctx) error {
	prefix := c.Query("prefix", "")

	objects, err := h.s3Service.ListObjects(c.Context(), h.bucketName, prefix)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to list files from S3", err.Error(),
		))
	}

	// Convert to simple response
	objectList := make([]map[string]interface{}, len(objects))
	for i, obj := range objects {
		objectList[i] = map[string]interface{}{
			"key":           *obj.Key,
			"size":          *obj.Size,
			"last_modified": obj.LastModified,
		}
	}

	return c.JSON(schema.SuccessResponse(objectList))
}

// GetPresignedURL handles GET /v1/files/s3/presigned-url
// @Summary Get presigned URL for S3 file
// @Description Generate a presigned URL for downloading a file from S3
// @Tags Files
// @Accept json
// @Produce json
// @Param key query string true "S3 object key"
// @Param expiration query int false "URL expiration in seconds" default(3600)
// @Success 200 {object} map[string]interface{} "Presigned URL"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /v1/files/s3/presigned-url [get]
func (h *FileHandler) GetPresignedURL(c fiber.Ctx) error {
	key := c.Query("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Missing key parameter", "",
		))
	}

	expiration := 3600 // default 1 hour
	if expirationStr := c.Query("expiration"); expirationStr != "" {
		if val, err := strconv.Atoi(expirationStr); err == nil {
			expiration = val
		}
	}

	url, err := h.s3Service.GeneratePresignedURL(c.Context(), h.bucketName, key, time.Duration(expiration)*time.Second)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to generate presigned URL", err.Error(),
		))
	}

	response := map[string]string{
		"presigned_url": url,
	}

	return c.JSON(schema.SuccessResponse(response))
}
