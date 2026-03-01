package inbound_lead_form

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
)

// InboundLeadFormRepository handles inbound lead form persistence.
type InboundLeadFormRepository struct {
	*gormpkg.GormRepository[domain.InboundLeadForm]
	db *gorm.DB
}

// NewInboundLeadFormRepository creates a new repository instance.
func NewInboundLeadFormRepository(db *gorm.DB) *InboundLeadFormRepository {
	return &InboundLeadFormRepository{
		GormRepository: gormpkg.NewGormRepository[domain.InboundLeadForm](db),
		db:             db,
	}
}

// CreateWithListContact persists a form and links it to the provided list contact IDs.
func (r *InboundLeadFormRepository) CreateWithListContact(ctx context.Context, form *domain.InboundLeadForm, listContactIDs []uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(form).Error; err != nil {
			return err
		}

		if len(listContactIDs) == 0 {
			return nil
		}

		associations := make([]domain.InboundLeadFormListContactAssociation, 0, len(listContactIDs))
		for _, listID := range listContactIDs {
			associations = append(associations, domain.InboundLeadFormListContactAssociation{
				InboundLeadFormID: form.ID,
				ListContactID:     listID,
			})
		}

		if err := tx.Create(&associations).Error; err != nil {
			return err
		}

		return nil
	})
}

// CreateWithFields persists a form along with associated fields.
func (r *InboundLeadFormRepository) CreateWithFields(ctx context.Context, form *domain.InboundLeadForm, fields []domain.FormField) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.createWithFieldsTx(ctx, tx, form, fields)
	})
}

// CreateWithFieldsTx persists the form + fields using the provided transaction.
func (r *InboundLeadFormRepository) CreateWithFieldsTx(ctx context.Context, tx *gorm.DB, form *domain.InboundLeadForm, fields []domain.FormField) error {
	return r.createWithFieldsTx(ctx, tx, form, fields)
}

func (r *InboundLeadFormRepository) createWithFieldsTx(ctx context.Context, tx *gorm.DB, form *domain.InboundLeadForm, fields []domain.FormField) error {
	if tx == nil {
		return fmt.Errorf("transaction DB is nil")
	}

	session := tx.WithContext(ctx)
	if err := session.Create(form).Error; err != nil {
		return err
	}

	if len(fields) == 0 {
		return nil
	}

	for i := range fields {
		fields[i].FormID = form.ID
	}

	if err := session.Create(&fields).Error; err != nil {
		return err
	}

	return nil
}

// FindBySlug loads a form by slug with optional relations.
func (r *InboundLeadFormRepository) FindBySlug(ctx context.Context, slug string) (*domain.InboundLeadForm, error) {
	var form domain.InboundLeadForm

	err := r.db.WithContext(ctx).
		Where("slug = ?", slug).
		First(&form).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &form, nil
}

// GetByIdentifier returns a form and preloads associations by slug or UUID.
func (r *InboundLeadFormRepository) GetByIdentifier(ctx context.Context, identifier string) (*domain.InboundLeadForm, error) {
	query := r.db.WithContext(ctx).
		Preload("Fields").
		Preload("Fields.CustomField")

	if id, err := uuid.Parse(identifier); err == nil {
		query = query.Where("id = ?", id)
	} else {
		query = query.Where("slug = ?", identifier)
	}

	var form domain.InboundLeadForm
	if err := query.First(&form).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &form, nil
}

// CreateFormFields batches form fields creation.
func (r *InboundLeadFormRepository) CreateFormFields(ctx context.Context, fields []domain.FormField) error {
	if len(fields) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&fields).Error
}

// DeleteFormFields removes all fields linked to the form ID.
func (r *InboundLeadFormRepository) DeleteFormFields(ctx context.Context, formID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("form_id = ?", formID).
		Delete(&domain.FormField{}).Error
}

// UpdateForm updates base form attributes.
func (r *InboundLeadFormRepository) UpdateForm(ctx context.Context, formID uuid.UUID, attrs map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&domain.InboundLeadForm{}).
		Where("id = ?", formID).
		Updates(attrs).Error
}

// ListByUser retrieves forms owned by a user ordered by creation time desc.
func (r *InboundLeadFormRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.InboundLeadForm, error) {
	var forms []domain.InboundLeadForm
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&forms).Error; err != nil {
		return nil, err
	}
	return forms, nil
}

// ListContactIDs returns IDs of lists associated with the inbound lead form.
func (r *InboundLeadFormRepository) ListContactIDs(ctx context.Context, formID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	if err := r.db.WithContext(ctx).
		Table("inbound_lead_form_list_contact_association").
		Where("inbound_lead_form_id = ?", formID).
		Pluck("list_contact_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}
