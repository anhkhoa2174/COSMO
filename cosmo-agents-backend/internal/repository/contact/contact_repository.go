package contact

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/core"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	filterPkg "github.com/rockship/cosmo-agents-go/internal/repository/filter"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ContactRepository handles Contact entity operations
type ContactRepository struct {
	*gormpkg.GormRepository[domain.Contact]
	db *gorm.DB
}

// NewContactRepository creates a new contact repository
func NewContactRepository(db *gorm.DB) *ContactRepository {
	return &ContactRepository{
		GormRepository: gormpkg.NewGormRepository[domain.Contact](db),
		db:             db,
	}
}

// FindByIDAndUserID finds a contact by ID and user ID
func (r *ContactRepository) FindByIDAndUserID(ctx context.Context, userID uuid.UUID, contactID uuid.UUID) (*domain.Contact, error) {
	var contact domain.Contact
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND id = ? AND is_deleted = ?", userID, contactID, false).
		// Note: ListContacts.InboundLeadForms preload removed as ListContact model no longer has direct relationships
		First(&contact).Error

	if err != nil {
		return nil, err
	}

	return &contact, nil
}

// FindByUserIDWithPagination finds all contacts for a user with pagination
func (r *ContactRepository) FindByUserIDWithPagination(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Contact, int, error) {
	var contacts []*domain.Contact
	var total int64

	query := r.db.WithContext(ctx).Where("user_id = ? AND is_deleted = ?", userID, false)

	// Count total
	if err := query.Model(&domain.Contact{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get contacts with pagination
	if err := query.
		Offset(offset).
		Limit(limit).
		Order("updated_at DESC, id DESC").
		Find(&contacts).Error; err != nil {
		return nil, 0, err
	}

	return contacts, int(total), nil
}

// FindAllWithPagination finds all contacts (non-deleted) with pagination.
func (r *ContactRepository) FindAllWithPagination(ctx context.Context, offset, limit int) ([]*domain.Contact, int, error) {
	var contacts []*domain.Contact
	var total int64

	query := r.db.WithContext(ctx).Where("is_deleted = ?", false)

	if err := query.Model(&domain.Contact{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Offset(offset).
		Limit(limit).
		Order("updated_at DESC, id DESC").
		Find(&contacts).Error; err != nil {
		return nil, 0, err
	}

	return contacts, int(total), nil
}

// FindByEmail finds a contact by email
func (r *ContactRepository) FindByEmail(ctx context.Context, userID uuid.UUID, email string) (*domain.Contact, error) {
	var contact domain.Contact
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND email = ? AND is_deleted = ?", userID, email, false).
		First(&contact).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &contact, nil
}

// GetByEmail alias for FindByEmail
func (r *ContactRepository) GetByEmail(ctx context.Context, email string, userID uuid.UUID) (*domain.Contact, error) {
	return r.FindByEmail(ctx, userID, email)
}

// GetByLinkedInURL finds a contact by LinkedIn URL
func (r *ContactRepository) GetByLinkedInURL(ctx context.Context, linkedinURL string, userID uuid.UUID) (*domain.Contact, error) {
	var contact domain.Contact
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_deleted = ? AND profile->>'linkedin_url' = ?", userID, false, linkedinURL).
		First(&contact).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &contact, nil
}

// FindBySourceID finds a contact by source and source_id
func (r *ContactRepository) FindBySourceID(ctx context.Context, userID uuid.UUID, source, sourceID string) (*domain.Contact, error) {
	var contact domain.Contact
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND source = ? AND source_id = ? AND is_deleted = ?", userID, source, sourceID, false).
		First(&contact).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &contact, nil
}

// Search finds contacts for a user with optional organization filters and pagination
func (r *ContactRepository) Search(ctx context.Context, userID uuid.UUID, query string, offset, limit int) ([]*domain.Contact, int, error) {
	var contacts []*domain.Contact
	var total int64

	dbQuery := r.db.WithContext(ctx).
		Where("user_id = ? AND is_deleted = ?", userID, false)

	if query != "" {
		searchPattern := "%" + query + "%"
		dbQuery = dbQuery.Where(
			"(name ILIKE ? OR email ILIKE ? OR company ILIKE ?)",
			searchPattern, searchPattern, searchPattern,
		)
	}

	// Count total
	if err := dbQuery.Model(&domain.Contact{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get contacts with pagination
	if err := dbQuery.
		Offset(offset).
		Limit(limit).
		Order("updated_at DESC, id DESC").
		Find(&contacts).Error; err != nil {
		return nil, 0, err
	}

	return contacts, int(total), nil
}

// SearchWithFilter finds contacts with advanced filtering for organizations and custom filters
func (r *ContactRepository) SearchWithFilter(ctx context.Context, userID uuid.UUID, orgIDs []uuid.UUID, filter map[string]interface{}, pagination *baseRepo.PaginationParams) ([]*domain.Contact, int, error) {
	var contacts []*domain.Contact
	var total int64

	if pagination == nil {
		pagination = baseRepo.DefaultPagination()
	}
	pagination.Validate()

	dbQuery := r.db.WithContext(ctx).
		Model(&domain.Contact{}).
		Where("is_deleted = ?", false)

	// Filter by organization_id - all members can see all contacts in their organizations
	// The "Added By" field (user_id) tracks who added each contact
	if len(orgIDs) > 0 {
		dbQuery = dbQuery.Where("organization_id IN ?", orgIDs)
	} else {
		// Fallback: if no organizations, show only user's own contacts
		dbQuery = dbQuery.Where("user_id = ?", userID)
	}

	// Apply additional filters (supports legacy operators like $ilike, $and, $or)
	if len(filter) > 0 {
		builder := filterPkg.NewFilterBuilder(dbQuery)
		dbQuery = builder.Apply(normalizeContactFilter(filter))
	}

	// Count total
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get contacts with pagination
	if err := dbQuery.
		Offset(pagination.Offset).
		Limit(pagination.Limit).
		Order("updated_at DESC, id DESC").
		Find(&contacts).Error; err != nil {
		return nil, 0, err
	}

	return contacts, int(total), nil
}

// FindBySegment returns contacts for a segment with optional filters.
func (r *ContactRepository) FindBySegment(
	ctx context.Context,
	segmentID uuid.UUID,
	filter baseRepo.Filter,
	pagination *baseRepo.PaginationParams,
) ([]*domain.Contact, int, error) {
	var contacts []*domain.Contact
	var total int64

	if pagination == nil {
		pagination = baseRepo.DefaultPagination()
	}
	pagination.Validate()

	query := r.db.WithContext(ctx).
		Model(&domain.Contact{}).
		Joins("JOIN contact_segment_scores css ON css.contact_id = contacts.id").
		Where("contacts.is_deleted = ?", false).
		Where("css.segmentation_id = ?", segmentID).
		Where("css.passes_filters = ?", true)

	filterBuilder := filterPkg.NewFilterBuilder(query)
	query = filterBuilder.Apply(filter)

	if err := query.Select("contacts.id").Distinct().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Select("contacts.*").
		Distinct().
		Offset(pagination.Offset).
		Limit(pagination.Limit).
		Order("contacts.updated_at DESC, contacts.id DESC").
		Find(&contacts).Error; err != nil {
		return nil, 0, err
	}

	return contacts, int(total), nil
}

// normalizeContactFilter prepares contact filters for the generic baseRepo.FilterBuilder.
// - Converts plain string text fields to ILIKE "%value%" for partial matching.
// - Preserves legacy operators ($ilike, $and, $or, etc.) when provided.
// - Treats slices as $in filters for convenience.
func normalizeContactFilter(filter map[string]interface{}) baseRepo.Filter {
	normalized := make(baseRepo.Filter)
	var cityOrFilters []interface{}
	var titleOrFilters []interface{}
	var companyOrFilters []interface{}
	var countryOrFilters []interface{}
	var stateOrFilters []interface{}

	for key, value := range filter {
		// Pass through logical operators untouched
		if key == string(baseRepo.OpAnd) || key == string(baseRepo.OpOr) {
			normalized[key] = value
			continue
		}

		// Map JSONB score paths to raw SQL expression
		if strings.HasPrefix(key, "scores.") {
			normalized[string(baseRepo.OpRaw)] = buildScoreRawExpr(key, value)
			continue
		}
		// Map JSONB ai_insights.* and confirmed_facts.* to raw ILIKE search on text
		if strings.HasPrefix(key, "ai_insights.") || strings.HasPrefix(key, "confirmed_facts.") {
			normalized[string(baseRepo.OpRaw)] = buildJSONTextExpr(strings.Split(key, ".")[0], key, value)
			continue
		}
		// Handle profile.linkedin_url filter - exact match on JSONB field
		if key == "profile.linkedin_url" {
			if strVal, ok := value.(string); ok && strVal != "" {
				normalized[string(baseRepo.OpRaw)] = fmt.Sprintf("LOWER(profile->>'linkedin_url') = LOWER('%s')", strings.ReplaceAll(strVal, "'", "''"))
			}
			continue
		}

		switch v := value.(type) {
		case map[string]interface{}:
			normalized[key] = v
		case string:
			if v == "" {
				continue
			}
			if key == "city" {
				cityOrFilters = buildCityFuzzyFilters(v)
				continue
			}
			if key == "job_title" {
				titleOrFilters = buildTitleFuzzyFilters(v)
				continue
			}
			if key == "company" {
				companyOrFilters = buildCompanyFuzzyFilters(v)
				continue
			}
			if key == "country" {
				countryOrFilters = buildCountryFuzzyFilters(v)
				continue
			}
			if key == "state" {
				stateOrFilters = buildStateFuzzyFilters(v)
				continue
			}
			if isExactMatchField(key) {
				// Exact match for enum-like fields (lifecycle_stage, status, etc.)
				normalized[key] = v
			} else if isTextSearchField(key) {
				// Partial match for text fields (name, company, etc.)
				normalized[key] = map[string]interface{}{string(baseRepo.OpILike): "%" + v + "%"}
			} else {
				normalized[key] = v
			}
		case []string:
			normalized[key] = map[string]interface{}{string(baseRepo.OpIn): toInterfaceSlice(v)}
		case []interface{}:
			normalized[key] = map[string]interface{}{string(baseRepo.OpIn): v}
		default:
			normalized[key] = v
		}
	}

	normalized = appendFuzzyOr(normalized, cityOrFilters)
	normalized = appendFuzzyOr(normalized, titleOrFilters)
	normalized = appendFuzzyOr(normalized, companyOrFilters)
	normalized = appendFuzzyOr(normalized, countryOrFilters)
	normalized = appendFuzzyOr(normalized, stateOrFilters)

	return normalized
}

func isTextSearchField(field string) bool {
	switch field {
	case "name", "first_name", "last_name", "email", "phone", "company", "job_title", "address", "city", "country", "state", "zip":
		return true
	default:
		return false
	}
}

// isExactMatchField returns true for fields that should use exact match (=) instead of ILIKE
func isExactMatchField(field string) bool {
	switch field {
	case "contact_channel", "lifecycle_stage", "context_level", "outreach_decision", "scenario", "next_step", "status":
		return true
	default:
		return false
	}
}

var cityWhitespaceRegex = regexp.MustCompile(`\s+`)
var cityKeepCharsRegex = regexp.MustCompile(`[^a-zA-Z0-9\s]+`)
var cityNormalizeRegex = regexp.MustCompile(`[^a-z]+`)

func buildCityFuzzyFilters(city string) []interface{} {
	cleanLike := strings.TrimSpace(cityKeepCharsRegex.ReplaceAllString(city, " "))
	cleanLike = cityWhitespaceRegex.ReplaceAllString(cleanLike, " ")
	if cleanLike == "" {
		return nil
	}

	filters := []interface{}{
		map[string]interface{}{
			"city": map[string]interface{}{
				string(baseRepo.OpILike): "%" + cleanLike + "%",
			},
		},
	}

	norm := normalizeCityToken(city)
	if norm != "" {
		filters = append(filters, map[string]interface{}{
			string(baseRepo.OpRaw): buildCityRawExpr(norm),
		})
	}

	return filters
}

func normalizeCityToken(city string) string {
	return normalizeAlphaToken(city)
}

func buildTitleFuzzyFilters(title string) []interface{} {
	cleanLike := strings.TrimSpace(cityKeepCharsRegex.ReplaceAllString(title, " "))
	cleanLike = cityWhitespaceRegex.ReplaceAllString(cleanLike, " ")
	if cleanLike == "" {
		return nil
	}

	filters := []interface{}{
		map[string]interface{}{
			"job_title": map[string]interface{}{
				string(baseRepo.OpILike): "%" + cleanLike + "%",
			},
		},
	}

	norm := normalizeAlphaToken(title)
	if norm != "" {
		filters = append(filters, map[string]interface{}{
			string(baseRepo.OpRaw): buildJobTitleRawExpr(norm),
		})
	}

	return filters
}

func buildCompanyFuzzyFilters(company string) []interface{} {
	return buildFieldFuzzyFilters("company", company)
}

func buildCountryFuzzyFilters(country string) []interface{} {
	return buildFieldFuzzyFilters("country", country)
}

func buildStateFuzzyFilters(state string) []interface{} {
	return buildFieldFuzzyFilters("state", state)
}

func normalizeAlphaToken(value string) string {
	normalized := strings.ToLower(value)
	normalized = cityNormalizeRegex.ReplaceAllString(normalized, "")
	return strings.TrimSpace(normalized)
}

func buildCityRawExpr(normalized string) string {
	// Match both normalized city text and initials (e.g., "HCM" -> "Ho Chi Minh").
	return fmt.Sprintf(
		"(regexp_replace(lower(city), '[^a-z]', '', 'g') LIKE '%%%s%%' OR "+
			"array_to_string(ARRAY(SELECT substring(w,1,1) FROM unnest(regexp_split_to_array(regexp_replace(lower(city), '[^a-z]', ' ', 'g'), '\\\\s+')) AS w WHERE w <> ''), '') LIKE '%s%%')",
		normalized,
		normalized,
	)
}

func buildJobTitleRawExpr(normalized string) string {
	return fmt.Sprintf(
		"(regexp_replace(lower(job_title), '[^a-z]', '', 'g') LIKE '%%%s%%' OR "+
			"array_to_string(ARRAY(SELECT substring(w,1,1) FROM unnest(regexp_split_to_array(regexp_replace(lower(job_title), '[^a-z]', ' ', 'g'), '\\\\s+')) AS w WHERE w <> ''), '') LIKE '%s%%')",
		normalized,
		normalized,
	)
}

func buildCompanyRawExpr(normalized string) string {
	return buildFieldRawExpr("company", normalized)
}

func buildFieldFuzzyFilters(field string, value string) []interface{} {
	cleanLike := strings.TrimSpace(cityKeepCharsRegex.ReplaceAllString(value, " "))
	cleanLike = cityWhitespaceRegex.ReplaceAllString(cleanLike, " ")
	if cleanLike == "" {
		return nil
	}

	filters := []interface{}{
		map[string]interface{}{
			field: map[string]interface{}{
				string(baseRepo.OpILike): "%" + cleanLike + "%",
			},
		},
	}

	norm := normalizeAlphaToken(value)
	if norm != "" {
		filters = append(filters, map[string]interface{}{
			string(baseRepo.OpRaw): buildFieldRawExpr(field, norm),
		})
	}

	return filters
}

func buildFieldRawExpr(field string, normalized string) string {
	return fmt.Sprintf(
		"(regexp_replace(lower(%s), '[^a-z]', '', 'g') LIKE '%%%s%%' OR "+
			"array_to_string(ARRAY(SELECT substring(w,1,1) FROM unnest(regexp_split_to_array(regexp_replace(lower(%s), '[^a-z]', ' ', 'g'), '\\\\s+')) AS w WHERE w <> ''), '') LIKE '%s%%')",
		field,
		normalized,
		field,
		normalized,
	)
}

func appendFuzzyOr(base baseRepo.Filter, filters []interface{}) baseRepo.Filter {
	if len(filters) == 0 {
		return base
	}

	orBlock := map[string]interface{}{string(baseRepo.OpOr): filters}
	if len(base) == 0 {
		return baseRepo.Filter{string(baseRepo.OpOr): filters}
	}
	if existingAnd, ok := base[string(baseRepo.OpAnd)].([]interface{}); ok {
		base[string(baseRepo.OpAnd)] = append(existingAnd, orBlock)
		return base
	}
	return baseRepo.Filter{
		string(baseRepo.OpAnd): []interface{}{
			base,
			orBlock,
		},
	}
}

// buildScoreRawExpr builds a raw filter for scores.<key> numeric comparison
func buildScoreRawExpr(key string, val interface{}) map[string]interface{} {
	// support operators map[string]interface{} or exact value
	ops := map[string]interface{}{}
	if m, ok := val.(map[string]interface{}); ok {
		for k, v := range m {
			ops[k] = v
		}
	} else {
		ops["="] = val
	}
	return map[string]interface{}{
		"jsonb_numeric": map[string]interface{}{
			"column": "scores",
			"path":   strings.TrimPrefix(key, "scores."),
			"ops":    ops,
		},
	}
}

// buildJSONTextExpr builds a raw ilike for nested JSONB by casting to text
func buildJSONTextExpr(root string, key string, val interface{}) map[string]interface{} {
	needle := val
	if m, ok := val.(map[string]interface{}); ok {
		if contains, ok := m["contains"]; ok {
			needle = contains
		}
	}
	return map[string]interface{}{
		"jsonb_text": map[string]interface{}{
			"column": root,
			"path":   strings.TrimPrefix(key, root+"."),
			"value":  needle,
		},
	}
}

// buildJSONNumberExpr builds a raw SQL expression for JSONB numeric comparison on scores.*
func buildJSONNumberExpr(root string, key string, val interface{}) map[string]interface{} {
	opVal := map[string]interface{}{}

	// Supported operators: =, >=, >, <=, <
	if m, ok := val.(map[string]interface{}); ok {
		for op, v := range m {
			opVal[op] = v
		}
	} else {
		opVal["="] = val
	}

	return map[string]interface{}{"jsonb_numeric": map[string]interface{}{
		"column": root,
		"path":   strings.TrimPrefix(key, root+"."),
		"ops":    opVal,
	}}
}

func toInterfaceSlice(values []string) []interface{} {
	result := make([]interface{}, len(values))
	for i, v := range values {
		result[i] = v
	}
	return result
}

// BatchCreate creates multiple contacts in a transaction
func (r *ContactRepository) BatchCreate(ctx context.Context, contacts []*domain.Contact) error {
	return r.db.WithContext(ctx).CreateInBatches(contacts, 100).Error
}

// BatchUpdate updates multiple contacts
func (r *ContactRepository) BatchUpdate(ctx context.Context, contacts []*domain.Contact) error {
	return r.Transaction(ctx, func(tx *gorm.DB) error {
		for _, contact := range contacts {
			if err := tx.Save(contact).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// UpsertMany performs a batch upsert of contacts using ON CONFLICT clause
func (r *ContactRepository) UpsertMany(ctx context.Context, contacts []*domain.Contact) error {
	if len(contacts) == 0 {
		return nil
	}

	tx := core.DB(ctx, r.db)

	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "source"}, {Name: "source_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "email", "phone", "company", "job_title", "address", "city", "country", "state", "zip"}),
	}).Clauses(clause.Returning{}).Create(contacts).Error
}

// UpsertBySource creates or updates a contact using (user_id, source, source_id) as unique key.
func (r *ContactRepository) UpsertBySource(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
	var existing domain.Contact
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND source = ? AND source_id = ?", contact.UserID, contact.Source, contact.SourceID).
		First(&existing).Error

	switch {
	case err == gorm.ErrRecordNotFound:
		if err := r.db.WithContext(ctx).Create(contact).Error; err != nil {
			return nil, err
		}
		return contact, nil
	case err != nil:
		return nil, err
	default:
		existing.Name = contact.Name
		existing.Company = contact.Company
		existing.JobTitle = contact.JobTitle
		existing.Address = contact.Address
		existing.City = contact.City
		existing.Country = contact.Country
		existing.State = contact.State
		existing.Zip = contact.Zip
		existing.OrganizationID = contact.OrganizationID
		// Merge profile (email/phone are now stored in profile)
		if len(contact.Profile) > 0 {
			existing.Profile = contact.Profile
		}
		if err := r.db.WithContext(ctx).Save(&existing).Error; err != nil {
			return nil, err
		}
		return &existing, nil
	}
}

// AddToList associates a contact with a list contact (no-op if already associated).
func (r *ContactRepository) AddToList(ctx context.Context, contactID, listContactID uuid.UUID) error {
	association := domain.ListContactAssociation{
		ListContactID: listContactID,
		ContactID:     contactID,
	}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "list_contact_id"}, {Name: "contact_id"}},
			DoNothing: true,
		}).Create(&association).Error
}

// DeleteByIDs soft deletes contacts by IDs (sets is_deleted = true)
func (r *ContactRepository) DeleteByIDs(ctx context.Context, ids []uuid.UUID, userID uuid.UUID, organizationID *uuid.UUID) ([]*domain.Contact, error) {
	var contacts []*domain.Contact

	query := r.db.WithContext(ctx).Where("id IN ? AND is_deleted = ?", ids, false)

	// baseRepo.Filter by user_id OR organization_id
	if organizationID != nil {
		query = query.Where("user_id = ? OR organization_id = ?", userID, *organizationID)
	} else {
		query = query.Where("user_id = ?", userID)
	}

	// Find contacts first
	if err := query.Find(&contacts).Error; err != nil {
		return nil, err
	}

	if len(contacts) == 0 {
		return contacts, nil
	}

	// Soft delete them and update timestamp
	now := time.Now()

	updateQuery := r.db.WithContext(ctx).Model(&domain.Contact{})
	if organizationID != nil {
		updateQuery = updateQuery.Where("id IN ? AND (user_id = ? OR organization_id = ?)", ids, userID, *organizationID)
	} else {
		updateQuery = updateQuery.Where("id IN ? AND user_id = ?", ids, userID)
	}

	if err := updateQuery.Updates(map[string]interface{}{
		"is_deleted": true,
		"updated_at": now,
	}).Error; err != nil {
		return nil, err
	}

	// Update local objects to reflect the deletion
	for i := range contacts {
		contacts[i].IsDeleted = true
		contacts[i].UpdatedAt = now
	}

	return contacts, nil
}

// FindIDsByListContact returns contact IDs belonging to a specific list.
// It enforces ownership by checking user_id/organization_id and excludes soft-deleted contacts.
func (r *ContactRepository) FindIDsByListContact(
	ctx context.Context,
	listContactID uuid.UUID,
	userID uuid.UUID,
	organizationID *uuid.UUID,
	offset, limit int,
) ([]uuid.UUID, error) {
	if limit <= 0 {
		limit = 1000
	}

	query := r.db.WithContext(ctx).
		Table("list_contact_association AS lca").
		Select("c.id").
		Joins("INNER JOIN contacts c ON c.id = lca.contact_id").
		Where("lca.list_contact_id = ? AND c.is_deleted = ?", listContactID, false)

	if organizationID != nil {
		query = query.Where("(c.user_id = ? OR c.organization_id = ?)", userID, *organizationID)
	} else {
		query = query.Where("c.user_id = ?", userID)
	}

	var ids []uuid.UUID
	if err := query.
		Order("c.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&ids).Error; err != nil {
		return nil, err
	}

	return ids, nil
}

// GetBySourceIDs finds contacts by source and source IDs
func (r *ContactRepository) GetBySourceIDs(ctx context.Context, source string, sourceIDs []string) ([]*domain.Contact, error) {
	var contacts []*domain.Contact
	err := r.db.WithContext(ctx).
		Where("source = ? AND source_id IN ? AND is_deleted = ?", source, sourceIDs, false).
		Find(&contacts).Error

	if err != nil {
		return nil, err
	}

	return contacts, nil
}

// GetDistinctFieldValues gets distinct values for a field
func (r *ContactRepository) GetDistinctFieldValues(ctx context.Context, field string, userID uuid.UUID, organizationIDs []uuid.UUID, offset, limit int) ([]interface{}, int64, error) {
	values := make([]interface{}, 0)
	// Validate field name to prevent SQL injection
	if !isValidContactField(field) {
		return nil, 0, fmt.Errorf("Field %s not found", field)
	}

	// Build query with standard Contact columns
	query := r.db.WithContext(ctx).Model(&domain.Contact{}).
		Where("is_deleted = ?", false).
		Where(fmt.Sprintf("%s IS NOT NULL", field)).
		Where(fmt.Sprintf("%s != ?", field), "").
		Where(fmt.Sprintf("%s NOT IN ?", field), []string{"N/A", "null", "None"})

	// baseRepo.Filter by user_id OR organization_id IN (...)
	if len(organizationIDs) > 0 {
		query = query.Where("user_id = ? OR organization_id IN ?", userID, organizationIDs)
	} else {
		query = query.Where("user_id = ?", userID)
	}

	// Count total distinct values
	var total int64
	if err := query.Session(&gorm.Session{NewDB: true}).
		Model(&domain.Contact{}).
		Distinct(field).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get distinct values with pagination. Use Rows() and scan into interface{}
	// to support non-string column types (bool, timestamps, numeric) while
	// handling NULLs safely. Convert []byte -> string for text columns.
	rows, err := query.Session(&gorm.Session{NewDB: true}).
		Model(&domain.Contact{}).
		Select(fmt.Sprintf("DISTINCT %s", field)).
		Order(field). // Sort alphabetically
		Offset(offset * limit).
		Limit(limit).
		Rows()
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var raw interface{}
		if err := rows.Scan(&raw); err != nil {
			return nil, 0, err
		}
		if raw == nil {
			// Skip NULLs (we already filtered IS NOT NULL, but be defensive)
			continue
		}
		switch v := raw.(type) {
		case []byte:
			// Text/varchar comes back as []byte - convert to string
			values = append(values, string(v))
		default:
			// For time.Time, bool, int64, etc. keep native type so JSON
			// marshaling returns proper types to API clients.
			values = append(values, v)
		}
	}

	return values, total, nil
}

// UpdateFields updates specific fields of a contact with access control
// Only updates provided fields (partial update) and verifies user has access
func (r *ContactRepository) UpdateFields(ctx context.Context, contactID uuid.UUID, attributes map[string]interface{}, userID uuid.UUID, organizationID uuid.UUID) (*domain.Contact, error) {
	// Build query with access control
	query := r.db.WithContext(ctx).
		Where("id = ?", contactID).
		Where("is_deleted = ?", false)

	// Access control: user_id = ? OR organization_id = ?
	query = query.Where("user_id = ? OR organization_id = ?", userID, organizationID)

	// Find the contact first to verify it exists and user has access
	var contact domain.Contact
	if err := query.First(&contact).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("Contact not found")
		}
		return nil, err
	}

	// Update only provided fields
	if len(attributes) > 0 {
		if err := r.db.WithContext(ctx).Model(&contact).Updates(attributes).Error; err != nil {
			return nil, err
		}
	}

	// Reload to get fresh timestamps and all fields
	if err := r.db.WithContext(ctx).First(&contact, contactID).Error; err != nil {
		return nil, err
	}

	return &contact, nil
}

// FindByIDs returns contacts mapped by ID.
func (r *ContactRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Contact, error) {
	result := make(map[uuid.UUID]*domain.Contact)
	if len(ids) == 0 {
		return result, nil
	}

	const maxChunk = 500

	for start := 0; start < len(ids); start += maxChunk {
		end := start + maxChunk
		if end > len(ids) {
			end = len(ids)
		}

		var contacts []domain.Contact
		if err := r.db.WithContext(ctx).
			Where("id IN ? AND is_deleted = ?", ids[start:end], false).
			Find(&contacts).Error; err != nil {
			return nil, err
		}

		for i := range contacts {
			result[contacts[i].ID] = &contacts[i]
		}
	}

	return result, nil
}

// isValidContactField validates that the field exists in the Contact model
func isValidContactField(field string) bool {
	validFields := map[string]bool{
		"id":              true,
		"user_id":         true,
		"organization_id": true,
		"source_id":       true,
		"source":          true,
		"hubspot_id":      true,
		"name":            true,
		"first_name":      true,
		"last_name":       true,
		"email":           true,
		"phone":           true,
		"company":         true,
		"job_title":       true,
		"address":         true,
		"city":            true,
		"state":           true,
		"country":         true,
		"zip":             true,
		"do_not_contact":  true,
		"is_deleted":      true,
		"created_at":      true,
		"updated_at":      true,
	}
	return validFields[field]
}

// IsValidContactField exposes the validation function for use by handlers
func IsValidContactField(field string) bool {
	return isValidContactField(field)
}

// Create creates a new contact (interface implementation)
func (r *ContactRepository) Create(ctx context.Context, contact *domain.Contact) error {
	_, err := r.GormRepository.Create(ctx, contact)
	return err
}

// SoftDelete soft deletes a contact (interface implementation)
func (r *ContactRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return r.GormRepository.Delete(ctx, id)
}

// FindByEmailInOrg finds a contact by email within an organization (for duplicate detection)
func (r *ContactRepository) FindByEmailInOrg(ctx context.Context, email string, organizationID uuid.UUID) (*domain.Contact, error) {
	if email == "" {
		return nil, nil
	}

	var contact domain.Contact
	// Email is now stored in profile->>'email' and contact_information
	// Check both for backwards compatibility
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND is_deleted = ?", organizationID, false).
		Where("LOWER(contact_information) = LOWER(?) OR LOWER(profile->>'email') = LOWER(?)", email, email).
		First(&contact).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

// FindByLinkedInURLInOrg finds a contact by LinkedIn URL within an organization (for duplicate detection)
func (r *ContactRepository) FindByLinkedInURLInOrg(ctx context.Context, linkedinURL string, organizationID uuid.UUID) (*domain.Contact, error) {
	if linkedinURL == "" {
		return nil, nil
	}

	// Normalize LinkedIn URL - extract the path part for comparison
	normalizedURL := normalizeLinkedInURL(linkedinURL)
	if normalizedURL == "" {
		return nil, nil
	}

	var contact domain.Contact
	// Search in profile JSONB field for linkedin_url AND contact_information column
	// contact_information stores linkedin_url for LinkedIn source contacts
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND is_deleted = ?", organizationID, false).
		Where("profile->>'linkedin_url' ILIKE ? OR contact_information ILIKE ?", "%"+normalizedURL+"%", "%"+normalizedURL+"%").
		First(&contact).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

// normalizeLinkedInURL extracts the username/path from a LinkedIn URL for comparison
func normalizeLinkedInURL(url string) string {
	url = strings.TrimSpace(url)
	url = strings.TrimSuffix(url, "/")

	// Remove protocol and www
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "www.")

	// Extract path after linkedin.com
	if strings.HasPrefix(url, "linkedin.com/") {
		url = strings.TrimPrefix(url, "linkedin.com/")
	}

	return url
}

// DuplicateCheckResult holds the result of a duplicate check
type DuplicateCheckResult struct {
	Exists          bool            // Whether a duplicate exists
	ExistingContact *domain.Contact // The existing contact if found
	IsSameOwner     bool            // Whether the existing contact belongs to the same user
	MatchField      string          // Which field matched (email, linkedin_url)
}

// CheckDuplicateInOrg checks if a contact already exists in the organization
// Returns the existing contact and whether it belongs to the same owner
func (r *ContactRepository) CheckDuplicateInOrg(ctx context.Context, email, linkedinURL string, organizationID, userID uuid.UUID) (*DuplicateCheckResult, error) {
	result := &DuplicateCheckResult{
		Exists:      false,
		IsSameOwner: false,
	}

	// Check by email first
	if email != "" && email != "N/A" {
		existing, err := r.FindByEmailInOrg(ctx, email, organizationID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			result.Exists = true
			result.ExistingContact = existing
			result.IsSameOwner = existing.UserID == userID
			result.MatchField = "email"
			return result, nil
		}
	}

	// Check by LinkedIn URL
	if linkedinURL != "" {
		existing, err := r.FindByLinkedInURLInOrg(ctx, linkedinURL, organizationID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			result.Exists = true
			result.ExistingContact = existing
			result.IsSameOwner = existing.UserID == userID
			result.MatchField = "linkedin_url"
			return result, nil
		}
	}

	return result, nil
}

// CreateWithOwnershipCheck creates a contact with ownership validation
// Returns error if contact already exists and belongs to a different user
func (r *ContactRepository) CreateWithOwnershipCheck(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
	if contact.OrganizationID == nil {
		// No organization, skip ownership check
		if err := r.Create(ctx, contact); err != nil {
			return nil, err
		}
		return contact, nil
	}

	// Extract LinkedIn URL and email from profile if exists
	linkedinURL := extractLinkedInURL(contact.Profile)
	contactEmail := extractEmailFromProfile(contact.Profile)

	// Check for duplicates
	dupResult, err := r.CheckDuplicateInOrg(ctx, contactEmail, linkedinURL, *contact.OrganizationID, contact.UserID)
	if err != nil {
		return nil, err
	}

	if dupResult.Exists {
		if !dupResult.IsSameOwner {
			// Contact belongs to another BD - reject
			return nil, fmt.Errorf("contact already exists and belongs to another team member (matched by %s)", dupResult.MatchField)
		}
		// Same owner - merge/update
		return r.MergeContact(ctx, dupResult.ExistingContact, contact)
	}

	// No duplicate, create new
	if err := r.Create(ctx, contact); err != nil {
		return nil, err
	}
	return contact, nil
}

// MergeContact merges new contact data into existing contact
func (r *ContactRepository) MergeContact(ctx context.Context, existing, new *domain.Contact) (*domain.Contact, error) {
	// Update fields if new values are provided and different from defaults
	if new.Name != "" && new.Name != "N/A" && (existing.Name == "" || existing.Name == "N/A") {
		existing.Name = new.Name
	}
	// Email and Phone are now stored in profile JSONB - merged below with profile
	if new.Company != "" && new.Company != "N/A" && (existing.Company == "" || existing.Company == "N/A") {
		existing.Company = new.Company
	}
	if new.JobTitle != "" && new.JobTitle != "N/A" && (existing.JobTitle == "" || existing.JobTitle == "N/A") {
		existing.JobTitle = new.JobTitle
	}
	if new.Industry != "" && existing.Industry == "" {
		existing.Industry = new.Industry
	}
	if new.ContactChannel != "" && existing.ContactChannel == "" {
		existing.ContactChannel = new.ContactChannel
	}

	// Merge profile JSONB (includes email and phone)
	if new.Profile != nil {
		existing.Profile = mergeJSONB(existing.Profile, new.Profile)
	}

	// Save merged contact
	if err := r.db.WithContext(ctx).Save(existing).Error; err != nil {
		return nil, err
	}

	return existing, nil
}

// mergeJSONB merges two JSONB objects, preferring non-empty values from new
func mergeJSONB(existing, new base.JSONB) base.JSONB {
	if len(existing) == 0 {
		return new
	}
	if len(new) == 0 {
		return existing
	}

	var existingMap map[string]interface{}
	var newMap map[string]interface{}

	if err := existing.Unmarshal(&existingMap); err != nil {
		return new
	}
	if err := new.Unmarshal(&newMap); err != nil {
		return existing
	}

	// Merge new values into existing
	for k, v := range newMap {
		if v != nil && v != "" {
			if _, exists := existingMap[k]; !exists || existingMap[k] == nil || existingMap[k] == "" {
				existingMap[k] = v
			}
		}
	}

	// Convert back to JSONB using json.Marshal
	var merged base.JSONB
	if err := merged.Marshal(existingMap); err != nil {
		return existing
	}
	return merged
}

// OwnershipCheckResult holds the result of batch ownership check
type OwnershipCheckResult struct {
	ValidContacts    []*domain.Contact // Contacts that can be created/merged
	RejectedContacts []*domain.Contact // Contacts that belong to other users
	RejectedReasons  map[string]string // Map of email/linkedin -> reason
}

// CheckBatchOwnership checks ownership for a batch of contacts
func (r *ContactRepository) CheckBatchOwnership(ctx context.Context, contacts []*domain.Contact, userID uuid.UUID, organizationID uuid.UUID) (*OwnershipCheckResult, error) {
	result := &OwnershipCheckResult{
		ValidContacts:    make([]*domain.Contact, 0),
		RejectedContacts: make([]*domain.Contact, 0),
		RejectedReasons:  make(map[string]string),
	}

	for _, contact := range contacts {
		// Extract LinkedIn URL and email from profile
		linkedinURL := extractLinkedInURL(contact.Profile)
		contactEmail := extractEmailFromProfile(contact.Profile)

		// Check for duplicates
		dupResult, err := r.CheckDuplicateInOrg(ctx, contactEmail, linkedinURL, organizationID, userID)
		if err != nil {
			return nil, err
		}

		if dupResult.Exists && !dupResult.IsSameOwner {
			// Contact belongs to another user - reject
			result.RejectedContacts = append(result.RejectedContacts, contact)
			identifier := contactEmail
			if identifier == "" || identifier == "N/A" {
				identifier = linkedinURL
			}
			result.RejectedReasons[identifier] = fmt.Sprintf("already belongs to another team member (matched by %s)", dupResult.MatchField)
		} else {
			result.ValidContacts = append(result.ValidContacts, contact)
		}
	}

	return result, nil
}

// extractLinkedInURL extracts linkedin_url from profile JSONB
func extractLinkedInURL(profile base.JSONB) string {
	if len(profile) == 0 {
		return ""
	}
	var profileData map[string]interface{}
	if err := profile.Unmarshal(&profileData); err != nil {
		return ""
	}
	if url, ok := profileData["linkedin_url"].(string); ok {
		return url
	}
	return ""
}

// extractEmailFromProfile extracts email from profile JSONB
func extractEmailFromProfile(profile base.JSONB) string {
	if len(profile) == 0 {
		return ""
	}
	var profileData map[string]interface{}
	if err := profile.Unmarshal(&profileData); err != nil {
		return ""
	}
	if email, ok := profileData["email"].(string); ok {
		return email
	}
	return ""
}

// FindByContactInformation finds a contact by contact_information within an organization (primary duplicate detection)
func (r *ContactRepository) FindByContactInformation(ctx context.Context, contactInfo string, organizationID uuid.UUID) (*domain.Contact, error) {
	if contactInfo == "" {
		return nil, nil
	}

	var contact domain.Contact
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND is_deleted = ?", organizationID, false).
		Where("LOWER(contact_information) = LOWER(?)", contactInfo).
		First(&contact).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

// RecalculateAllStatuses recalculates status for all contacts in an organization
// Returns the number of contacts updated
func (r *ContactRepository) RecalculateAllStatuses(ctx context.Context, organizationID *uuid.UUID) (int, error) {
	// Fetch all non-deleted contacts
	query := r.db.WithContext(ctx).Where("is_deleted = ?", false)
	if organizationID != nil {
		query = query.Where("organization_id = ?", *organizationID)
	}

	var contacts []domain.Contact
	if err := query.Find(&contacts).Error; err != nil {
		return 0, err
	}

	updated := 0
	for i := range contacts {
		oldStatus := contacts[i].Status
		contacts[i].CalculateStatus()

		// Only update if status changed
		if contacts[i].Status != oldStatus {
			if err := r.db.WithContext(ctx).Model(&contacts[i]).Updates(map[string]interface{}{
				"status":         contacts[i].Status,
				"missing_fields": contacts[i].MissingFields,
			}).Error; err != nil {
				return updated, err
			}
			updated++
		}
	}

	return updated, nil
}

// UpsertManyWithOwnershipCheck performs batch upsert with ownership validation
// Returns the number of rejected contacts and error details
func (r *ContactRepository) UpsertManyWithOwnershipCheck(ctx context.Context, contacts []*domain.Contact, userID uuid.UUID, organizationID uuid.UUID) (*OwnershipCheckResult, error) {
	if len(contacts) == 0 {
		return &OwnershipCheckResult{}, nil
	}

	// First check ownership for all contacts
	checkResult, err := r.CheckBatchOwnership(ctx, contacts, userID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to check ownership: %w", err)
	}

	// Upsert only valid contacts
	if len(checkResult.ValidContacts) > 0 {
		if err := r.UpsertMany(ctx, checkResult.ValidContacts); err != nil {
			return nil, fmt.Errorf("failed to upsert contacts: %w", err)
		}
	}

	return checkResult, nil
}
