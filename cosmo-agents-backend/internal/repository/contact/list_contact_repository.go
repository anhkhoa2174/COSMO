package contact

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/rockship/cosmo-agents-go/internal/core"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	filterPkg "github.com/rockship/cosmo-agents-go/internal/repository/filter"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
)

// ListContactRepository handles list contact data operations.
type ListContactRepository struct {
	*gormpkg.GormRepository[domain.ListContact]
	db *gorm.DB
}

func NewListContactRepository(db *gorm.DB) *ListContactRepository {
	return &ListContactRepository{
		GormRepository: gormpkg.NewGormRepository[domain.ListContact](db),
		db:             db,
	}
}

var ErrListContactUnauthorized = errors.New("unauthorized to attach inbound lead form to list")

// FindByID finds a list contact by ID with preloaded relationships
func (r *ListContactRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.ListContact, error) {
	var entity domain.ListContact
	err := r.GormRepository.GetDB().WithContext(ctx).
		// Note: InboundLeadForms preload removed as ListContact model no longer has direct relationships
		Preload("Contacts", "contacts.is_deleted = ?", false).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

// BatchCountContacts returns contact counts for multiple list IDs in a single query
func (r *ListContactRepository) BatchCountContacts(ctx context.Context, listIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	if len(listIDs) == 0 {
		return make(map[uuid.UUID]int64), nil
	}

	var results []struct {
		ListContactID uuid.UUID `json:"list_contact_id"`
		Count         int64     `json:"count"`
	}

	err := r.GormRepository.GetDB().WithContext(ctx).
		Table("list_contact_association as lca").
		Select("lca.list_contact_id, COUNT(*) as count").
		Joins("INNER JOIN contacts c ON c.id = lca.contact_id").
		Where("lca.list_contact_id IN ? AND c.is_deleted = ?", listIDs, false).
		Group("lca.list_contact_id").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[uuid.UUID]int64)
	for _, result := range results {
		counts[result.ListContactID] = result.Count
	}

	// Ensure all listIDs have a count (0 if not found)
	for _, id := range listIDs {
		if _, exists := counts[id]; !exists {
			counts[id] = 0
		}
	}

	return counts, nil
}

// BatchCountInboundForms returns inbound form counts for multiple list IDs in a single query
func (r *ListContactRepository) BatchCountInboundForms(ctx context.Context, listIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	if len(listIDs) == 0 {
		return make(map[uuid.UUID]int64), nil
	}

	var results []struct {
		ListContactID uuid.UUID `json:"list_contact_id"`
		Count         int64     `json:"count"`
	}

	err := r.GormRepository.GetDB().WithContext(ctx).
		Table("inbound_lead_form_list_contact_association as ilflca").
		Select("ilflca.list_contact_id, COUNT(*) as count").
		Joins("INNER JOIN inbound_lead_forms ilf ON ilf.id = ilflca.inbound_lead_form_id").
		Where("ilflca.list_contact_id IN ?", listIDs).
		Group("ilflca.list_contact_id").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[uuid.UUID]int64)
	for _, result := range results {
		counts[result.ListContactID] = result.Count
	}

	// Ensure all listIDs have a count (0 if not found)
	for _, id := range listIDs {
		if _, exists := counts[id]; !exists {
			counts[id] = 0
		}
	}

	return counts, nil
}

// SearchOptimized performs fast search for list contacts without preloading relationships
// Returns only basic list contact info with counts for better performance
func (r *ListContactRepository) SearchOptimized(ctx context.Context, filter baseRepo.Filter, pagination *baseRepo.PaginationParams) (*baseRepo.PaginatedResult[domain.ListContact], error) {
	if pagination == nil {
		pagination = baseRepo.DefaultPagination()
	}
	pagination.Validate()

	// Build query with filters
	query := r.GormRepository.GetDB().WithContext(ctx)
	filterBuilder := filterPkg.NewFilterBuilder(query)
	query = filterBuilder.Apply(filter)

	// Exclude soft-deleted records
	query = query.Where("is_deleted = ?", false)

	// Count total (before pagination)
	var total int64
	if err := query.Model(new(domain.ListContact)).Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply pagination - only select basic fields, no relationships
	var entities []domain.ListContact
	err := query.
		Select("id", "name", "source", "source_id", "hubspot_id", "user_id", "organization_id", "created_at", "updated_at").
		Offset(pagination.Offset).
		Limit(pagination.Limit).
		Order("created_at DESC").
		Find(&entities).Error

	if err != nil {
		return nil, err
	}

	return &baseRepo.PaginatedResult[domain.ListContact]{
		List:   entities,
		Total:  total,
		Offset: pagination.Offset,
		Limit:  pagination.Limit,
	}, nil
}

// FindAll finds all list contacts with preloaded relationships
func (r *ListContactRepository) FindAll(ctx context.Context, filter baseRepo.Filter, pagination *baseRepo.PaginationParams) (*baseRepo.PaginatedResult[domain.ListContact], error) {
	if pagination == nil {
		pagination = baseRepo.DefaultPagination()
	}
	pagination.Validate()

	// Build query with filters
	query := r.GormRepository.GetDB().WithContext(ctx)
	filterBuilder := filterPkg.NewFilterBuilder(query)
	query = filterBuilder.Apply(filter)

	// Exclude soft-deleted records
	query = query.Where("is_deleted = ?", false)

	// Count total (before pagination)
	var total int64
	if err := query.Model(new(domain.ListContact)).Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply pagination and preload
	var entities []domain.ListContact
	err := query.
		// Note: InboundLeadForms preload removed as ListContact model no longer has direct relationships
		Preload("Contacts", "contacts.is_deleted = ?", false).
		Offset(pagination.Offset).
		Limit(pagination.Limit).
		Order("created_at DESC").
		Find(&entities).Error

	if err != nil {
		return nil, err
	}

	return &baseRepo.PaginatedResult[domain.ListContact]{
		List:   entities,
		Total:  total,
		Offset: pagination.Offset,
		Limit:  pagination.Limit,
	}, nil
}

// CountContacts returns the number of contacts linked to a list.
func (r *ListContactRepository) CountContacts(ctx context.Context, listID uuid.UUID) (int64, error) {
	var count int64
	err := r.GormRepository.GetDB().WithContext(ctx).
		Table("list_contact_association as lca").
		Joins("INNER JOIN contacts c ON c.id = lca.contact_id").
		Where("lca.list_contact_id = ? AND c.is_deleted = ?", listID, false).
		Count(&count).Error
	return count, err
}

// CountCampaigns returns the number of campaigns using this list.
func (r *ListContactRepository) CountCampaigns(ctx context.Context, listID uuid.UUID) (int64, error) {
	var count int64
	err := r.GormRepository.GetDB().WithContext(ctx).
		Model(&domain.Campaign{}).
		Where("list_contact_id = ? AND is_deleted = ?", listID, false).
		Count(&count).Error
	return count, err
}

// GetInboundLeadForms retrieves inbound lead forms associated with this list contact
func (r *ListContactRepository) GetInboundLeadForms(ctx context.Context, listID uuid.UUID) ([]*domain.InboundLeadForm, error) {
	// Step 1: Get InboundLeadForm IDs from association table
	var formIDs []uuid.UUID
	err := r.GormRepository.GetDB().WithContext(ctx).
		Table("inbound_lead_form_list_contact_association").
		Where("list_contact_id = ?", listID).
		Pluck("inbound_lead_form_id", &formIDs).Error

	if err != nil {
		return nil, err
	}

	if len(formIDs) == 0 {
		return []*domain.InboundLeadForm{}, nil
	}

	// Step 2: Fetch actual InboundLeadForm records using the IDs
	var forms []*domain.InboundLeadForm
	err = r.GormRepository.GetDB().WithContext(ctx).
		Where("id IN ?", formIDs).
		Find(&forms).Error

	if err != nil {
		return nil, err
	}

	return forms, nil
}

// Upsert creates or updates a list contact.
func (r *ListContactRepository) Upsert(ctx context.Context, listContact *domain.ListContact) error {
	tx := core.DB(ctx, r.db)
	return tx.
		Where("source = ? AND source_id = ? AND user_id = ?",
			listContact.Source,
			listContact.SourceID,
			listContact.UserID,
		).
		Assign(listContact).
		FirstOrCreate(listContact).Error
}

// AddInboundForms links existing inbound lead forms to the given list contact by creating association rows.
func (r *ListContactRepository) AddInboundForms(ctx context.Context, listContactID uuid.UUID, formIDs []uuid.UUID) error {
	if len(formIDs) == 0 {
		return nil
	}

	associations := make([]domain.InboundLeadFormListContactAssociation, 0, len(formIDs))
	for _, fid := range formIDs {
		associations = append(associations, domain.InboundLeadFormListContactAssociation{
			InboundLeadFormID: fid,
			ListContactID:     listContactID,
		})
	}

	// Use ON CONFLICT DO NOTHING to ignore duplicate associations
	return r.GormRepository.GetDB().WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&associations).Error
}

// AddContacts links existing contacts to a list contact (duplicates ignored).
func (r *ListContactRepository) AddContacts(ctx context.Context, listContactID uuid.UUID, contactIDs []uuid.UUID) error {
	if len(contactIDs) == 0 {
		return nil
	}
	tx := core.DB(ctx, r.db)

	associations := make([]domain.ListContactAssociation, 0, len(contactIDs))
	for _, cid := range contactIDs {
		associations = append(associations, domain.ListContactAssociation{
			ListContactID: listContactID,
			ContactID:     cid,
		})
	}

	return tx.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&associations).Error
}

// AddContactsWithOwnership validates ownership before associating contacts.
func (r *ListContactRepository) AddContactsWithOwnership(ctx context.Context, ownerUserID uuid.UUID, listContactID uuid.UUID, contactIDs []uuid.UUID) error {
	return r.GormRepository.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.addContactsWithOwnershipTx(ctx, tx, ownerUserID, listContactID, contactIDs)
	})
}

// AddInboundFormsWithOwnership validates that the caller (ownerUserID) is allowed to modify the
// target list (either owns it or is a member of the owning organization) and then creates the
// inbound lead form <-> list association rows within a transaction. Returns an error if the list
// is not found or the caller is not authorized.
func (r *ListContactRepository) AddInboundFormsWithOwnership(ctx context.Context, ownerUserID uuid.UUID, listContactID uuid.UUID, formIDs []uuid.UUID) error {
	return r.GormRepository.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.addInboundFormsWithOwnershipTx(ctx, tx, ownerUserID, listContactID, formIDs)
	})
}

// AddInboundFormsWithOwnershipTx performs the same logic but uses the provided transaction.
func (r *ListContactRepository) AddInboundFormsWithOwnershipTx(ctx context.Context, tx *gorm.DB, ownerUserID uuid.UUID, listContactID uuid.UUID, formIDs []uuid.UUID) error {
	return r.addInboundFormsWithOwnershipTx(ctx, tx, ownerUserID, listContactID, formIDs)
}

func (r *ListContactRepository) addInboundFormsWithOwnershipTx(ctx context.Context, tx *gorm.DB, ownerUserID uuid.UUID, listContactID uuid.UUID, formIDs []uuid.UUID) error {
	if len(formIDs) == 0 {
		return nil
	}
	if tx == nil {
		return fmt.Errorf("transaction DB is nil")
	}

	session := tx.WithContext(ctx)

	// Fetch the list under transaction
	var list domain.ListContact
	if err := session.Where("id = ? AND is_deleted = ?", listContactID, false).First(&list).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // treat as no-op for not-found (caller can decide to error at service layer)
		}
		return err
	}

	// Check ownership: either user owns the list, or user is member of the list's organization
	allowed := list.UserID == ownerUserID
	if !allowed && list.OrganizationID != nil {
		var cnt int64
		if err := session.Table("roles").Where("organization_id = ? AND user_id = ? AND is_deleted = ?", *list.OrganizationID, ownerUserID, false).Count(&cnt).Error; err != nil {
			return err
		}
		allowed = cnt > 0
	}
	if !allowed {
		return fmt.Errorf("%w: %s", ErrListContactUnauthorized, listContactID.String())
	}

	associations := make([]domain.InboundLeadFormListContactAssociation, 0, len(formIDs))
	for _, fid := range formIDs {
		associations = append(associations, domain.InboundLeadFormListContactAssociation{
			InboundLeadFormID: fid,
			ListContactID:     listContactID,
		})
	}

	return session.Clauses(clause.OnConflict{DoNothing: true}).Create(&associations).Error
}

func (r *ListContactRepository) addContactsWithOwnershipTx(ctx context.Context, tx *gorm.DB, ownerUserID uuid.UUID, listContactID uuid.UUID, contactIDs []uuid.UUID) error {
	if len(contactIDs) == 0 {
		return nil
	}
	if tx == nil {
		return fmt.Errorf("transaction DB is nil")
	}

	session := tx.WithContext(ctx)

	var list domain.ListContact
	if err := session.Where("id = ? AND is_deleted = ?", listContactID, false).First(&list).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	allowed := list.UserID == ownerUserID
	if !allowed && list.OrganizationID != nil {
		var cnt int64
		if err := session.Table("roles").Where("organization_id = ? AND user_id = ? AND is_deleted = ?", *list.OrganizationID, ownerUserID, false).Count(&cnt).Error; err != nil {
			return err
		}
		allowed = cnt > 0
	}
	if !allowed {
		return fmt.Errorf("%w: %s", ErrListContactUnauthorized, listContactID.String())
	}

	contactQuery := session.Model(&domain.Contact{}).
		Where("id IN ? AND is_deleted = ?", contactIDs, false)

	if list.OrganizationID != nil {
		contactQuery = contactQuery.Where("(user_id = ? OR organization_id = ?)", ownerUserID, *list.OrganizationID)
	} else {
		contactQuery = contactQuery.Where("user_id = ?", ownerUserID)
	}

	var count int64
	if err := contactQuery.Count(&count).Error; err != nil {
		return err
	}

	if count != int64(len(contactIDs)) {
		return fmt.Errorf("%w: contact ownership validation failed for list %s", ErrListContactUnauthorized, listContactID.String())
	}

	associations := make([]domain.ListContactAssociation, 0, len(contactIDs))
	for _, cid := range contactIDs {
		associations = append(associations, domain.ListContactAssociation{
			ListContactID: listContactID,
			ContactID:     cid,
		})
	}

	return session.Clauses(clause.OnConflict{DoNothing: true}).Create(&associations).Error
}

// GetByUserID finds all lists for a user.
func (r *ListContactRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]domain.ListContact, int64, error) {
	var lists []domain.ListContact
	var total int64

	// Count total
	if err := r.GormRepository.GetDB().WithContext(ctx).
		Model(&domain.ListContact{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get records
	err := r.GormRepository.GetDB().WithContext(ctx).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&lists).Error

	if err != nil {
		return nil, 0, err
	}

	return lists, total, nil
}

// GetBySource finds all lists from a specific source.
func (r *ListContactRepository) GetBySource(ctx context.Context, source string, limit, offset int) ([]domain.ListContact, int64, error) {
	var lists []domain.ListContact
	var total int64

	// Count total
	if err := r.GormRepository.GetDB().WithContext(ctx).
		Model(&domain.ListContact{}).
		Where("source = ? AND is_deleted = ?", source, false).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get records
	err := r.GormRepository.GetDB().WithContext(ctx).
		Where("source = ? AND is_deleted = ?", source, false).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&lists).Error

	if err != nil {
		return nil, 0, err
	}

	return lists, total, nil
}
