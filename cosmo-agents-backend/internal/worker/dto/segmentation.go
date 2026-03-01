package dto

import "github.com/google/uuid"

// RecalculateSegmentScoresPayload optionally scopes recalculation to a user or contact.
type RecalculateSegmentScoresPayload struct {
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	ContactID *uuid.UUID `json:"contact_id,omitempty"`
}
