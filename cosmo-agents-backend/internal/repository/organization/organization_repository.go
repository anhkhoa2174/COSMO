package organization

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rockship/cosmo-agents-go/internal/core"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	filterPkg "github.com/rockship/cosmo-agents-go/internal/repository/filter"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
	"gorm.io/gorm"
)

// OrganizationRepository handles Organization entity operations
type OrganizationRepository struct {
	*gormpkg.GormRepository[domain.Organization]
	db *gorm.DB
}

// NewOrganizationRepository creates a new organization repository
func NewOrganizationRepository(db *gorm.DB) *OrganizationRepository {
	return &OrganizationRepository{
		GormRepository: gormpkg.NewGormRepository[domain.Organization](db),
		db:             db,
	}
}

// FindByUserID finds all organizations for a user
func (r *OrganizationRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Organization, error) {
	var orgs []domain.Organization
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Find(&orgs).Error

	return orgs, err
}

// FindWithUser finds organization with preloaded user
func (r *OrganizationRepository) FindWithUser(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	var org domain.Organization
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("id = ? AND is_deleted = ?", id, false).
		First(&org).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &org, nil
}

// Create creates a new organization
func (r *OrganizationRepository) Create(ctx context.Context, org *domain.Organization) (*domain.Organization, error) {
	tx := core.DB(ctx, r.db)
	err := tx.Create(org).Error
	if err != nil {
		return nil, err
	}
	return org, nil
}

// FindAll finds all organizations with pagination and filters, returning total count
func (r *OrganizationRepository) FindAll(ctx context.Context, filter baseRepo.Filter, pagination *baseRepo.PaginationParams) (*baseRepo.PaginatedResult[domain.Organization], error) {
	var orgs []domain.Organization

	query := r.db.WithContext(ctx)

	if len(filter) > 0 {
		filterBuilder := filterPkg.NewFilterBuilder(query)
		query = filterBuilder.Apply(filter)
	}

	// Count total records matching the filter
	var total int64
	if err := query.Model(&domain.Organization{}).Count(&total).Error; err != nil {
		return nil, err
	}

	// Fetch paginated results
	err := query.
		Offset(pagination.Offset * pagination.Limit).
		Limit(pagination.Limit).
		Order("updated_at DESC").
		Find(&orgs).Error

	if err != nil {
		return nil, err
	}

	return &baseRepo.PaginatedResult[domain.Organization]{
		List:   orgs,
		Total:  total,
		Offset: pagination.Offset,
		Limit:  pagination.Limit,
	}, nil
}

// FindByID finds an organization by ID
func (r *OrganizationRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	var org domain.Organization
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&org).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &org, nil
}

// FindByIDAndUserID finds an organization by ID and User ID
func (r *OrganizationRepository) FindByIDAndUserID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Organization, error) {
	var org domain.Organization
	tx := core.DB(ctx, r.db)
	err := tx.Where("id = ? AND user_id = ? AND is_deleted = ?", id, userID, false).
		First(&org).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &org, nil
}

// Update updates an organization
func (r *OrganizationRepository) Update(ctx context.Context, id uuid.UUID, org *domain.Organization) error {
	return r.db.WithContext(ctx).
		Model(&domain.Organization{}).
		Where("id = ?", id).
		Updates(org).Error
}

// OutreachSettingsForUser returns the raw settings JSON of the organisation the
// user belongs to, or nil when they belong to none or it has never been set.
//
// Membership is resolved through roles rather than through
// FindUserMainOrganization: that helper falls back to organisations the user
// created, and a member must be governed by the cadence of the organisation
// they actually work in. Only the one column is selected — this runs on the
// outreach hot path.
func (r *OrganizationRepository) OutreachSettingsForUser(ctx context.Context, userID uuid.UUID) ([]byte, error) {
	// GORM reads a bare *[]byte destination as "a slice of rows", so the
	// value goes through a one-field struct. An organisation that never saved
	// settings holds NULL, which cannot be scanned into []byte; every AI
	// reply then drafted without the organisation's guidance. Empty settings
	// resolve to the defaults.
	var row struct{ Settings []byte }
	err := r.db.WithContext(ctx).
		Model(&domain.Organization{}).
		Select("COALESCE(organizations.outreach_settings, '{}'::jsonb) AS settings").
		Joins("JOIN roles ON roles.organization_id = organizations.id").
		Where("roles.user_id = ? AND organizations.is_deleted = ?", userID, false).
		Order("CASE WHEN roles.name = 'admin' THEN 0 ELSE 1 END, organizations.created_at").
		Limit(1).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	return row.Settings, nil
}

// UpdateOutreachSettings writes the organisation's outreach cadence.
//
// It is a single-column update rather than Update(): passing a struct there
// would let GORM's zero-value skipping decide which other fields travel, and
// this write must touch nothing but the settings.
func (r *OrganizationRepository) UpdateOutreachSettings(ctx context.Context, id uuid.UUID, settings []byte) error {
	return r.db.WithContext(ctx).
		Model(&domain.Organization{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Update("outreach_settings", settings).Error
}

// FindFirstOrganizationOfUser finds the first organization for a user
func (r *OrganizationRepository) FindFirstOrganizationOfUser(ctx context.Context, userID uuid.UUID) (*domain.Organization, error) {
	var org domain.Organization

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Limit(1).
		First(&org).Error

	if err == nil {
		return &org, nil
	}
	return nil, err
}

func (r *OrganizationRepository) FindUserMainOrganization(ctx context.Context, userID uuid.UUID) (*domain.Organization, error) {
	var org domain.Organization

	// Step 1: Try to find organization where user is admin
	err := r.db.WithContext(ctx).
		Model(&domain.Organization{}).
		Joins("JOIN roles ON roles.organization_id = organizations.id").
		Where("roles.user_id = ? AND roles.name = ? AND organizations.is_deleted = ?", userID, domain.RoleNameAdmin, false).
		Limit(1).
		First(&org).Error

	if err == nil {
		return &org, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	} else {
		// Step 2: Fallback - find first organization created by user
		return r.FindFirstOrganizationOfUser(ctx, userID)
	}
}

func (r *OrganizationRepository) GetManyMembersWithTotal(
	ctx context.Context,
	currentUserId uuid.UUID,
	organizationID string,
	skip int,
	limit int,
) ([]v2schema.OrganizationListItem, int64, error) {
	var organizationIDs []uuid.UUID

	if organizationID != "me" {
		id, err := uuid.Parse(organizationID)
		if err != nil {
			return nil, 0, fmt.Errorf("Failed to parse organization ID: %w", err)
		}
		organizationIDs = []uuid.UUID{id}
	} else {
		var roles []domain.Role
		if err := r.db.WithContext(ctx).
			Where("user_id = ?", currentUserId).
			Find(&roles).Error; err != nil {
			return nil, 0, err
		}

		if len(roles) == 0 {
			return nil, 0, errors.New("no organization found for user")
		}
		for _, r := range roles {
			organizationIDs = append(organizationIDs, r.OrganizationID)
		}
	}

	// TODO: remove if support multiple orgs
	if len(organizationIDs) > 1 {
		organizationIDs = organizationIDs[:1]
	}

	orgID := organizationIDs[0]

	type MemberRow struct {
		UserID                     uuid.UUID
		UserEmail                  string
		UserName                   *string
		UserPicture                *string
		UserPhone                  *string
		UserIsDeleted              bool
		RoleID                     uuid.UUID
		OrganizationID             uuid.UUID
		RoleName                   string
		RoleStatus                 domain.RoleStatus
		RoleJobTitle               *string
		OrgName                    string
		OrgCompanyURL              string
		OrgCompanyDesc             string
		OrgCompanyTargetingPersona pq.StringArray `gorm:"type:text[]"`
		OrgValueOffering           string
	}

	var rows []MemberRow

	query := r.db.WithContext(ctx).
		Table("users").
		Select(`
			users.id as user_id,
			users.email as user_email,
			users.name as user_name,
			users.picture as user_picture,
			users.phone_number as user_phone,
			users.is_deleted as user_is_deleted,
			roles.id as role_id,
			roles.organization_id as organization_id,
			roles.name as role_name,
			roles.status as role_status,
			roles.job_title as role_job_title,
			organizations.id as organization_id,
			organizations.name as org_name,
			organizations.company_url as org_company_url,
			organizations.company_description as org_company_desc,
			organizations.company_targeting_persona as org_company_targeting_persona,
			organizations.value_offering as org_value_offering
		`).
		Joins("JOIN roles ON roles.user_id = users.id").
		Joins("JOIN organizations ON organizations.id = roles.organization_id").
		Where("roles.organization_id = ? AND roles.is_deleted = FALSE", orgID).
		Offset(skip).
		Limit(limit)

	if err := query.Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("query members: %w", err)
	}

	members := make([]v2schema.OrganizationListItem, 0)
	for _, row := range rows {
		persona := []string(row.OrgCompanyTargetingPersona)
		member := v2schema.OrganizationListItem{
			Entity: v2schema.OrganizationMemberEntity{
				ID:          row.UserID,
				Email:       row.UserEmail,
				Name:        row.UserName,
				Picture:     row.UserPicture,
				PhoneNumber: row.UserPhone,
				IsDeleted:   row.UserIsDeleted,
			},
			Role: v2schema.OrganizationRole{
				ID:             row.RoleID,
				OrganizationID: row.OrganizationID,
				Name:           row.RoleName,
				Status:         row.RoleStatus,
				JobTitle:       row.RoleJobTitle,
				Organization: v2schema.OrganizationReadResponse{
					ID:                      row.OrganizationID,
					Name:                    &row.OrgName,
					CompanyURL:              &row.OrgCompanyURL,
					CompanyDescription:      &row.OrgCompanyDesc,
					CompanyTargetingPersona: &persona,
					ValueOffering:           &row.OrgValueOffering,
				},
			},
		}
		members = append(members, member)
	}

	var total int64
	if err := r.db.WithContext(ctx).
		Table("roles").
		Joins("JOIN users ON users.id = roles.user_id").
		Where("roles.organization_id = ? AND roles.is_deleted = FALSE", orgID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count members: %w", err)
	}

	return members, total, nil
}

func (r *OrganizationRepository) SoftDeleteManyByUserID(ctx context.Context, organizationID uuid.UUID, memberIDs []uuid.UUID) error {
	tx := core.DB(ctx, r.db)
	query := tx.Model(&domain.Role{}).
		Where("organization_id = ? AND user_id IN ?", organizationID, memberIDs)

	return query.Updates(map[string]interface{}{"is_deleted": true}).Error
}
