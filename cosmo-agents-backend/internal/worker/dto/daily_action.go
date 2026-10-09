package dto

import "github.com/google/uuid"

// GenerateActionsPayload is the Asynq task payload for daily_action:generate.
type GenerateActionsPayload struct {
	UserID       uuid.UUID `json:"user_id"`
	Language     string    `json:"language"`
	ForceRefresh bool      `json:"force_refresh"`
}
