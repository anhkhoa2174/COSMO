package dto

import "github.com/google/uuid"

// ContactOrchestrationPayload controls which agent chain to run for a contact.
type ContactOrchestrationPayload struct {
	ContactID    uuid.UUID `json:"contact_id"`
	UserID       uuid.UUID `json:"user_id"`
	OrgID        uuid.UUID `json:"org_id"`
	Event        string    `json:"event,omitempty"`
	ForceRefresh bool      `json:"force_refresh,omitempty"`
}
