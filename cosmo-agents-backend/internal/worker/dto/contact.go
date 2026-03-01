package dto

import "github.com/google/uuid"

// EnrichContactPayload represents payload for enriching contact data.
type EnrichContactPayload struct {
	ContactID uuid.UUID `json:"contact_id"`
	UserID    uuid.UUID `json:"user_id"`
	Source    string    `json:"source"` // e.g., "linkedin", "clearbit", "hunter"
}

// SyncContactPayload represents payload for syncing contact with external system.
type SyncContactPayload struct {
	ContactID uuid.UUID `json:"contact_id"`
	UserID    uuid.UUID `json:"user_id"`
	System    string    `json:"system"` // e.g., "hubspot", "salesforce"
}
