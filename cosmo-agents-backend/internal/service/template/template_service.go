package template

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// TemplateRepository interface defines the contract for template repository operations
type TemplateRepository interface {
	Create(ctx context.Context, template *domain.Template) (*domain.Template, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Template, error)
	FindByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*domain.Template, error)
	Update(ctx context.Context, id uuid.UUID, template *domain.Template) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Template, int64, error)
	FindByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]*domain.Template, error)
	FindByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Template, error)
	ReorderTemplates(ctx context.Context, positions map[uuid.UUID]float64) error
	AddKnowledge(ctx context.Context, templateID uuid.UUID, knowledge *domain.Knowledge) error
}

// TemplateService provides business logic for template operations
type TemplateService struct {
	templateRepo TemplateRepository
}

// NewTemplateService creates a new template service
func NewTemplateService(templateRepo TemplateRepository) *TemplateService {
	return &TemplateService{
		templateRepo: templateRepo,
	}
}

// GetTemplatesByUserID retrieves templates for a specific user with pagination
func (s *TemplateService) GetTemplatesByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Template, int64, error) {
	if offset < 0 {
		return nil, 0, fmt.Errorf("invalid offset: %d cannot be negative", offset)
	}
	if limit == 0 {
		limit = 50 // Default limit
	}
	if limit < 0 {
		return nil, 0, fmt.Errorf("invalid limit: %d cannot be negative", limit)
	}
	if limit > 1000 {
		return nil, 0, fmt.Errorf("limit too large: %d exceeds maximum allowed (1000)", limit)
	}
	return s.templateRepo.FindByUserID(ctx, userID, offset, limit)
}

// GetTemplatesByCampaignID retrieves templates for a specific campaign
func (s *TemplateService) GetTemplatesByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]*domain.Template, error) {
	return s.templateRepo.FindByCampaignID(ctx, campaignID)
}

// GetTemplateByID retrieves a template by ID
func (s *TemplateService) GetTemplateByID(ctx context.Context, id uuid.UUID) (*domain.Template, error) {
	return s.templateRepo.FindByID(ctx, id)
}

// GetTemplateByIDAndUserID retrieves a template by ID with user authorization check
func (s *TemplateService) GetTemplateByIDAndUserID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Template, error) {
	return s.templateRepo.FindByIDAndUserID(ctx, id, userID)
}

// CreateTemplate creates a new template
func (s *TemplateService) CreateTemplate(ctx context.Context, template *domain.Template) (*domain.Template, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}
	if template == nil {
		return nil, fmt.Errorf("template cannot be nil")
	}
	if template.UserID == uuid.Nil {
		return nil, fmt.Errorf("user ID is required")
	}
	if template.Subject == "" {
		return nil, fmt.Errorf("subject is required")
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return s.templateRepo.Create(ctx, template)
}

// UpdateTemplate updates an existing template
func (s *TemplateService) UpdateTemplate(ctx context.Context, template *domain.Template) (*domain.Template, error) {
	if template == nil {
		return nil, fmt.Errorf("template cannot be nil")
	}
	err := s.templateRepo.Update(ctx, template.ID, template)
	if err != nil {
		return nil, err
	}
	return s.templateRepo.FindByID(ctx, template.ID)
}

// DeleteTemplate deletes a template
func (s *TemplateService) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	return s.templateRepo.Delete(ctx, id)
}

// ReorderTemplates updates the position order of templates
func (s *TemplateService) ReorderTemplates(ctx context.Context, positions map[uuid.UUID]float64) error {
	return s.templateRepo.ReorderTemplates(ctx, positions)
}

// AddKnowledgeToTemplate adds knowledge to a template with security checks
func (s *TemplateService) AddKnowledgeToTemplate(ctx context.Context, templateID uuid.UUID, knowledge *domain.Knowledge) error {
	if knowledge == nil {
		return fmt.Errorf("knowledge cannot be nil")
	}
	return s.templateRepo.AddKnowledge(ctx, templateID, knowledge)
}

// GetTemplatesByIDs retrieves multiple templates by their IDs
func (s *TemplateService) GetTemplatesByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Template, error) {
	return s.templateRepo.FindByIDs(ctx, ids)
}
