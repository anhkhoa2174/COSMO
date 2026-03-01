package dto

import "github.com/google/uuid"

// SyncAgentPayload represents payload for syncing agent credentials.
type SyncAgentPayload struct {
	AgentID uuid.UUID `json:"agent_id"`
	UserID  uuid.UUID `json:"user_id"`
}

// RefreshTokenPayload represents payload for refreshing OAuth token.
type RefreshTokenPayload struct {
	AgentID uuid.UUID `json:"agent_id"`
}

// CheckCredentialsPayload represents payload for checking agent credentials.
type CheckCredentialsPayload struct {
	AgentID uuid.UUID `json:"agent_id"`
}
