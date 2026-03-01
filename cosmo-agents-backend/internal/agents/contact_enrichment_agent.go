package agents

import (
	"context"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/internal/service/intelligence"
)

// ContactEnrichmentAgent generates AI insights and embeddings for a contact.
type ContactEnrichmentAgent struct {
	intel *intelligence.Service
}

func NewContactEnrichmentAgent(intel *intelligence.Service) *ContactEnrichmentAgent {
	return &ContactEnrichmentAgent{intel: intel}
}

// Run executes the enrichment agent (legacy - no context)
func (a *ContactEnrichmentAgent) Run(ctx context.Context, userID, orgID, contactID uuid.UUID, forceRefresh bool) (*v1.ContactEnrichmentResponse, error) {
	return a.intel.EnrichContact(ctx, userID, orgID, contactID, forceRefresh)
}
