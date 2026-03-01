package agents

import (
	"context"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/internal/service/intelligence"
)

// SegmentCalculatorAgent evaluates a contact against segments and persists scores.
type SegmentCalculatorAgent struct {
	intel *intelligence.Service
}

func NewSegmentCalculatorAgent(intel *intelligence.Service) *SegmentCalculatorAgent {
	return &SegmentCalculatorAgent{intel: intel}
}

func (a *SegmentCalculatorAgent) Run(ctx context.Context, userID, orgID, contactID uuid.UUID, segmentIDs []uuid.UUID) (*v1.CalculateScoresResponse, error) {
	return a.intel.CalculateSegmentScores(ctx, userID, orgID, contactID, segmentIDs)
}
