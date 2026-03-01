package v1

import (
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// FileResponse represents the API response for a file
type FileResponse struct {
	ID        uuid.UUID  `json:"id"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	Filename  string     `json:"filename"`
	MimeType  string     `json:"mime_type"`
	S3Key     string     `json:"s3_key"`
	Size      int64      `json:"size"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ToFileResponse converts a domain.File to FileResponse
func ToFileResponse(file *domain.File) FileResponse {
	return FileResponse{
		ID:        file.ID,
		UserID:    file.UserID,
		Filename:  file.Filename,
		MimeType:  file.MimeType,
		S3Key:     file.S3Key,
		Size:      file.Size,
		CreatedAt: file.CreatedAt,
		UpdatedAt: file.UpdatedAt,
	}
}
