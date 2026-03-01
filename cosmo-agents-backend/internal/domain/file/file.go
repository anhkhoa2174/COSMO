package file

import (
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// File represents a stored asset in S3.
type File struct {
	base.Base
	base.SoftDeleteMixin
	base.TimestampMixin

	UserID   *uuid.UUID `gorm:"type:uuid;index:idx_files_user_id" json:"user_id,omitempty"`
	Filename string     `gorm:"type:text" json:"filename"`
	MimeType string     `gorm:"type:text" json:"mime_type"`
	S3Key    string     `gorm:"type:text;uniqueIndex:idx_files_s3_key" json:"s3_key"`
	Size     int64      `gorm:"type:bigint" json:"size"`
}

func (File) TableName() string {
	return "files"
}
