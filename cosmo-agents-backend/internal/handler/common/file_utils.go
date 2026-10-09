package common

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// FileUploadHelper provides common file upload utilities
type FileUploadHelper struct{}

// NewFileUploadHelper creates a new file upload helper
func NewFileUploadHelper() *FileUploadHelper {
	return &FileUploadHelper{}
}

// SaveToTempFile saves uploaded file to temp directory with timestamped filename
func (h *FileUploadHelper) SaveToTempFile(file *multipart.FileHeader, prefix string) (string, string, error) {
	// Create temp directory
	tempDir := filepath.Join(os.TempDir(), prefix)
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return "", "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Generate temp filename
	fileName := h.SanitizeFileName(file.Filename)
	// The random part keeps two uploads of the same name in the same second
	// apart; without it one user's import read the other user's file.
	tempFileName := fmt.Sprintf("%s_%s_%s", time.Now().UTC().Format("20060102T150405Z0700"), uuid.NewString()[:8], fileName)
	tempPath := filepath.Join(tempDir, tempFileName)

	// Save file
	if err := h.SaveUploadedFile(file, tempPath); err != nil {
		return "", "", fmt.Errorf("failed to save file: %w", err)
	}

	return tempPath, tempFileName, nil
}

// SaveUploadedFile saves multipart file to destination
func (h *FileUploadHelper) SaveUploadedFile(file *multipart.FileHeader, destPath string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	return nil
}

// SanitizeFileName removes unsafe characters from filename with enhanced security
func (h *FileUploadHelper) SanitizeFileName(name string) string {
	if name == "" {
		return "file"
	}

	// Extract base name and clean the path
	base := filepath.Base(filepath.Clean(name))

	// Check for null bytes and NTFS streams that could lead to path traversal
	if strings.Contains(base, "\x00") || strings.Contains(base, "::") {
		logger.Logger.Warn().Str("filename", base).Msg("Dangerous filename detected, using safe fallback")
		return "file"
	}

	// Validate UTF-8 encoding
	if !utf8.ValidString(base) {
		logger.Logger.Warn().Str("filename", base).Msg("Invalid UTF-8 in filename, using safe fallback")
		return "file"
	}

	// Remove dangerous path traversal attempts
	base = strings.ReplaceAll(base, "..", "_")
	base = strings.ReplaceAll(base, "/", "_")
	base = strings.ReplaceAll(base, "\\", "_")

	// Remove other dangerous characters
	dangerousChars := []string{
		":", "*", "?", "\"", "<", ">", "|", ";", "&", "$", "`",
		"(", ")", "[", "]", "{", "}", "#", "!", "~", "+", "=",
	}

	for _, char := range dangerousChars {
		base = strings.ReplaceAll(base, char, "_")
	}

	// Replace spaces with underscores for consistency
	base = strings.ReplaceAll(base, " ", "_")

	// Remove leading/trailing dots, spaces, and underscores
	base = strings.Trim(base, " ._")

	// Ensure filename is not empty after sanitization
	if base == "" || base == "." || base == ".." {
		return "file"
	}

	// Limit filename length to prevent issues
	if len(base) > 255 {
		base = base[:255]
	}

	return base
}

// ValidateFileType validates file extension against allowed types
func (h *FileUploadHelper) ValidateFileType(file *multipart.FileHeader, allowedExtensions []string) error {
	if len(allowedExtensions) == 0 {
		return nil // No restrictions
	}

	fileName := strings.ToLower(file.Filename)
	for _, ext := range allowedExtensions {
		if strings.HasSuffix(fileName, strings.ToLower(ext)) {
			return nil
		}
	}

	return fmt.Errorf("file type not allowed. Allowed types: %v", allowedExtensions)
}

// ValidateFileSize validates file size against maximum allowed size
func (h *FileUploadHelper) ValidateFileSize(file *multipart.FileHeader, maxSize int64) error {
	if file.Size > maxSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", file.Size, maxSize)
	}
	return nil
}

// GetUploadedFile retrieves file from form with validation
func (h *FileUploadHelper) GetUploadedFile(c fiber.Ctx, fieldName string) (*multipart.FileHeader, error) {
	file, err := c.FormFile(fieldName)
	if err != nil {
		return nil, fmt.Errorf("missing or invalid file field '%s': %w", fieldName, err)
	}
	return file, nil
}

// CleanupTempFile removes temporary file and logs any errors
func (h *FileUploadHelper) CleanupTempFile(filePath string) {
	if err := os.Remove(filePath); err != nil {
		logger.Logger.Warn().Err(err).Str("file_path", filePath).Msg("Failed to cleanup temp file")
	}
}
