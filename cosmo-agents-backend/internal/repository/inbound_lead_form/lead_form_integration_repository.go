package inbound_lead_form

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
)

// LeadFormIntegrationRepository handles persistence for lead form integrations.
type LeadFormIntegrationRepository struct {
	*gormpkg.GormRepository[domain.LeadFormIntegration]
	db *gorm.DB
}

// NewLeadFormIntegrationRepository creates a new repository instance.
func NewLeadFormIntegrationRepository(db *gorm.DB) *LeadFormIntegrationRepository {
	return &LeadFormIntegrationRepository{
		GormRepository: gormpkg.NewGormRepository[domain.LeadFormIntegration](db),
		db:             db,
	}
}

// CreateIntegration inserts an integration with config.
func (r *LeadFormIntegrationRepository) CreateIntegration(ctx context.Context, integration *domain.LeadFormIntegration) error {
	return r.db.WithContext(ctx).Create(integration).Error
}

// CreateFieldMappings inserts mapping batch.
func (r *LeadFormIntegrationRepository) CreateFieldMappings(ctx context.Context, mappings []domain.LeadFieldMapping) error {
	if len(mappings) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&mappings).Error
}

// DeleteFieldMappings removes mappings for integration.
func (r *LeadFormIntegrationRepository) DeleteFieldMappings(ctx context.Context, integrationID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("form_integration_id = ?", integrationID).
		Delete(&domain.LeadFieldMapping{}).Error
}

// GetWithMappings loads integration including mappings.
func (r *LeadFormIntegrationRepository) GetWithMappings(ctx context.Context, id uuid.UUID) (*domain.LeadFormIntegration, error) {
	var integration domain.LeadFormIntegration
	if err := r.db.WithContext(ctx).
		Preload("FieldMappings").
		Preload("FieldMappings.CustomField").
		First(&integration, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &integration, nil
}
