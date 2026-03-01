package campaign

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	filterPkg "github.com/rockship/cosmo-agents-go/internal/repository/filter"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
)

// CampaignRepository handles Campaign entity operations
type CampaignRepository struct {
	*gormpkg.GormRepository[domain.Campaign]
	db *gorm.DB
}

// NewCampaignRepository creates a new campaign repository
func NewCampaignRepository(db *gorm.DB) *CampaignRepository {
	return &CampaignRepository{
		GormRepository: gormpkg.NewGormRepository[domain.Campaign](db),
		db:             db,
	}
}

// FindByUserID finds all campaigns for a user with pagination
func (r *CampaignRepository) FindByUserID(ctx context.Context, userID uuid.UUID, pagination *baseRepo.PaginationParams) (*baseRepo.PaginatedResult[domain.Campaign], error) {
	filter := baseRepo.Filter{
		"user_id":    userID,
		"is_deleted": false,
	}
	return r.FindAll(ctx, filter, pagination)
}

// FindByStatus finds campaigns by status
func (r *CampaignRepository) FindByStatus(ctx context.Context, userID uuid.UUID, status domain.CampaignStatus, pagination *baseRepo.PaginationParams) (*baseRepo.PaginatedResult[domain.Campaign], error) {
	filter := baseRepo.Filter{
		"user_id":    userID,
		"status":     status,
		"is_deleted": false,
	}
	return r.FindAll(ctx, filter, pagination)
}

// FindWithRelations finds campaign with preloaded relations
func (r *CampaignRepository) FindWithRelations(ctx context.Context, id uuid.UUID) (*domain.Campaign, error) {
	var campaign domain.Campaign
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&campaign).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &campaign, nil
}

// UpdateStatus updates campaign status
func (r *CampaignRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.CampaignStatus) error {
	return r.db.WithContext(ctx).
		Model(&domain.Campaign{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// UpdateAttributes updates a campaign with the provided attribute map.
func (r *CampaignRepository) UpdateAttributes(ctx context.Context, id uuid.UUID, attrs map[string]interface{}) error {
	if len(attrs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&domain.Campaign{}).
		Where("id = ?", id).
		Updates(attrs).Error
}

// CampaignWithStats represents a campaign row enriched with statistics.
type CampaignWithStats struct {
	domain.Campaign
	Creator        sql.NullString `gorm:"column:creator"`
	Sent           int64          `gorm:"column:sent"`
	Reply          int64          `gorm:"column:reply"`
	ReplyRate      float64        `gorm:"column:reply_rate"`
	Interested     int64          `gorm:"column:interested"`
	InterestRate   float64        `gorm:"column:interest_rate"`
	AgentName      sql.NullString `gorm:"column:agent_name"`
	AgentCMetadata domain.JSONB   `gorm:"column:agent_cmetadata"`
}

// FindWithStats returns campaigns owned by a user along with aggregate statistics.
func (r *CampaignRepository) FindWithStats(
	ctx context.Context,
	userID uuid.UUID,
	organizationID *uuid.UUID,
	filter baseRepo.Filter,
	pagination *baseRepo.PaginationParams,
) ([]CampaignWithStats, int64, error) {
	if pagination == nil {
		pagination = baseRepo.DefaultPagination()
	}
	pagination.Validate()

	intentInterested := domain.IntentInterested

	baseQuery := r.db.WithContext(ctx).
		Model(&domain.Campaign{}).
		Table("campaigns").
		Where("campaigns.is_deleted = ?", false).
		Joins("LEFT JOIN users ON users.id = campaigns.user_id AND users.is_deleted = ?", false).
		Joins("LEFT JOIN conversations ON conversations.campaign_id = campaigns.id AND conversations.is_deleted = ?", false).
		Joins("LEFT JOIN agents ON agents.id = campaigns.agent_id AND agents.is_deleted = ?", false)

	if organizationID != nil {
		baseQuery = baseQuery.Where(
			"(campaigns.user_id = ? OR campaigns.organization_id = ?)",
			userID,
			*organizationID,
		)
	} else {
		baseQuery = baseQuery.Where("campaigns.user_id = ?", userID)
	}

	filterBuilder := filterPkg.NewFilterBuilder(baseQuery)
	baseQuery = filterBuilder.Apply(filter)

	var total int64
	if err := baseQuery.Distinct("campaigns.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	selectClause := `
		campaigns.*,
		users.name AS creator,
		COUNT(conversations.id) AS sent,
		SUM(CASE WHEN conversations.replied THEN 1 ELSE 0 END) AS reply,
		CASE WHEN COUNT(conversations.id) = 0 THEN 0
			 ELSE SUM(CASE WHEN conversations.replied THEN 1 ELSE 0 END)::float / COUNT(conversations.id) END AS reply_rate,
		SUM(CASE WHEN conversations.intents @> ARRAY[?]::varchar[] THEN 1 ELSE 0 END) AS interested,
		CASE WHEN SUM(CASE WHEN conversations.replied THEN 1 ELSE 0 END) = 0 THEN 0
			 ELSE SUM(CASE WHEN conversations.intents @> ARRAY[?]::varchar[] THEN 1 ELSE 0 END)::float /
				  SUM(CASE WHEN conversations.replied THEN 1 ELSE 0 END) END AS interest_rate,
		MAX(agents.name) AS agent_name,
		COALESCE((array_agg(agents.cmetadata) FILTER (WHERE agents.cmetadata IS NOT NULL))[1], '{}'::jsonb) AS agent_cmetadata
	`

	var rows []CampaignWithStats
	err := baseQuery.
		Select(selectClause, intentInterested, intentInterested).
		Group("campaigns.id, users.name").
		Order("campaigns.updated_at DESC").
		Offset(pagination.Offset).
		Limit(pagination.Limit).
		Scan(&rows).Error

	if err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}
