package v3

import (
	"time"

	"github.com/google/uuid"
)

// OperationResponse represents the payload returned by GET /v3/operations/{operation_id}
type OperationResponse struct {
	ID        uuid.UUID              `json:"id"`
	Name      string                 `json:"name"`
	Status    string                 `json:"status"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Output    map[string]interface{} `json:"output,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}
