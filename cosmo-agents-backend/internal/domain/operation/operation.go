package operation

import (
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// OperationStatus indicates progress state.
type OperationStatus string

const (
	OperationStatusInProgress OperationStatus = "in_progress"
	OperationStatusSuccess    OperationStatus = "success"
	OperationStatusFailed     OperationStatus = "failed"
)

// Operation tracks async operations.
type Operation struct {
	base.Base
	base.TimestampMixin

	Name   string          `gorm:"type:text;not null" json:"name"`
	Status OperationStatus `gorm:"type:text;not null;default:'in_progress'" json:"status"`
	Input  base.JSONB      `gorm:"type:jsonb" json:"input,omitempty"`
	Output base.JSONB      `gorm:"type:jsonb" json:"output,omitempty"`
}

func (Operation) TableName() string {
	return "operations"
}
