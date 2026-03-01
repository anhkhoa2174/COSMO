package intent

import (
	"context"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// IntentHandler is the base interface for all intent handlers
type IntentHandler interface {
	// Execute processes the email based on the campaign and intent
	// Returns true if execution was successful, false otherwise
	Execute(ctx context.Context, campaign *domain.Campaign, intent domain.IntentType, email *domain.Email) (bool, error)
}
