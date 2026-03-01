package contact

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// saveToTempFile saves uploaded file to temp directory
func saveToTempFile(file *multipart.FileHeader) (string, string, error) {
	// Create temp directory
	tempDir := filepath.Join(os.TempDir(), "cosmo-contact-imports")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return "", "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Generate temp filename
	fileName := sanitizeFileName(file.Filename)
	tempFileName := fmt.Sprintf("%s_%s", time.Now().UTC().Format("20060102T150405Z0700"), fileName)
	tempPath := filepath.Join(tempDir, tempFileName)

	// Save file
	if err := saveUploadedFile(file, tempPath); err != nil {
		return "", "", fmt.Errorf("failed to save file: %w", err)
	}

	return tempPath, tempFileName, nil
}

// saveUploadedFile saves multipart file to destination
func saveUploadedFile(file *multipart.FileHeader, destPath string) error {
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

// sanitizeFileName removes unsafe characters from filename
func sanitizeFileName(name string) string {
	base := filepath.Base(name)
	base = strings.ReplaceAll(base, " ", "_")
	return base
}
