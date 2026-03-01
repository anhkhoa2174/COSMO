package skills

import (
	"context"
	"time"

	"github.com/google/uuid"
	intRepo "github.com/rockship/cosmo-agents-go/internal/repository/interaction"
	"github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// InteractionLogSkill logs interactions and fetches history.
type InteractionLogSkill struct {
	repo *intRepo.Repository
}

func NewInteractionLogSkill(repo *intRepo.Repository) *InteractionLogSkill {
	return &InteractionLogSkill{repo: repo}
}

func (s *InteractionLogSkill) LogInteraction(ctx context.Context, req v1.CreateInteractionRequest) (*v1.InteractionResponse, error) {
	if req.OccurredAt == nil {
		now := time.Now()
		req.OccurredAt = &now
	}
	model := v1.ToInteractionModel(req)
	if _, err := s.repo.Create(ctx, model); err != nil {
		return nil, err
	}
	return v1.ToInteractionResponse(model), nil
}

func (s *InteractionLogSkill) ListByContact(ctx context.Context, contactID uuid.UUID, limit int) ([]*v1.InteractionResponse, error) {
	items, err := s.repo.ListByContact(ctx, contactID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.InteractionResponse, 0, len(items))
	for _, it := range items {
		out = append(out, v1.ToInteractionResponse(it))
	}
	return out, nil
}
