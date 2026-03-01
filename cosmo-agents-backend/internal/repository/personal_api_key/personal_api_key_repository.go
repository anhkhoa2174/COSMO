package personalapikey

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain/personal_api_key"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
)

const KEY_PREFIX = "p_api_key_"

// scanResult is a helper struct to scan string timestamps from SQLite
type scanResult struct {
	ID         string
	CreatedAt  string
	UpdatedAt  string
	UserID     string
	Name       string
	HashedKey  string
	Prefix     string
	ExpiresAt  string
	LastUsedAt *sql.NullString
}

// parseTimestamp parses SQLite timestamp string to time.Time
func parseTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	// Try to parse as RFC3339 first, then fall back to other formats
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t
	}
	// If all formats fail, return zero time
	return time.Time{}
}

// convertToDomain converts scanResult to domain model
func (s *scanResult) convertToDomain() (*personal_api_key.PersonalApiKey, error) {
	// Check if ID is empty (indicates no record found)
	if s.ID == "" {
		return nil, nil
	}

	apiKey := &personal_api_key.PersonalApiKey{}

	apiKey.ID = uuid.MustParse(s.ID)
	apiKey.CreatedAt = parseTimestamp(s.CreatedAt)
	apiKey.UpdatedAt = parseTimestamp(s.UpdatedAt)
	apiKey.UserID = uuid.MustParse(s.UserID)
	apiKey.Name = s.Name
	apiKey.HashedKey = s.HashedKey
	apiKey.Prefix = s.Prefix
	apiKey.ExpiresAt = parseTimestamp(s.ExpiresAt)

	if s.LastUsedAt != nil && s.LastUsedAt.Valid {
		parsed := parseTimestamp(s.LastUsedAt.String)
		apiKey.LastUsedAt = &parsed
	}

	return apiKey, nil
}

// PersonalApiKeyRepository handles personal API key operations
type PersonalApiKeyRepository struct {
	db     *gorm.DB
	secret []byte
}

var ErrMissingPersonalAPIKeySecret = errors.New("personal API key secret is not configured")
var ErrExpiredPersonalAPIKey = errors.New("personal API key expired")

// NewPersonalApiKeyRepository creates a new repository
func NewPersonalApiKeyRepository(db *gorm.DB, secret string) *PersonalApiKeyRepository {
	return &PersonalApiKeyRepository{
		db:     db,
		secret: []byte(secret),
	}
}

func (r *PersonalApiKeyRepository) generateHashKey(rawKey string) (string, error) {
	if len(r.secret) == 0 {
		return "", ErrMissingPersonalAPIKeySecret
	}

	h := hmac.New(sha256.New, r.secret)
	h.Write([]byte(rawKey))
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ValidateRawKey hashes the raw key, looks it up, and checks expiration.
func (r *PersonalApiKeyRepository) ValidateRawKey(ctx context.Context, rawKey string) (*personal_api_key.PersonalApiKey, error) {
	hashedKey, err := r.generateHashKey(rawKey)
	if err != nil {
		return nil, err
	}

	key, err := r.FindByHashedKey(ctx, hashedKey)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, nil
	}

	if time.Now().After(key.ExpiresAt) {
		return nil, ErrExpiredPersonalAPIKey
	}

	return key, nil
}

// GenerateAPIKey generates a new API key with prefix
func (r *PersonalApiKeyRepository) GenerateAPIKey() (rawKey, hashedKey string, err error) {
	// Generate 32 random bytes
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random key: %w", err)
	}
	// Encode to base64
	keyBody := base64.RawURLEncoding.EncodeToString(randomBytes)

	// Raw key format: prefix_key (e.g., "csk_abc123...")
	rawKey = fmt.Sprintf("%s%s", KEY_PREFIX, keyBody)

	// Hash the raw key for storage
	hashedKey, err = r.generateHashKey(rawKey)
	if err != nil {
		return "", "", err
	}

	return rawKey, hashedKey, nil
}

// Create creates a new personal API key
func (r *PersonalApiKeyRepository) Create(ctx context.Context, apiKey *personal_api_key.PersonalApiKey) (*personal_api_key.PersonalApiKey, error) {
	err := r.db.WithContext(ctx).Create(apiKey).Error
	if err != nil {
		return nil, err
	}
	return apiKey, nil
}

// FindByID finds an API key by ID - using scanResult to handle SQLite timestamps
func (r *PersonalApiKeyRepository) FindByID(ctx context.Context, id uuid.UUID) (*personal_api_key.PersonalApiKey, error) {
	var result scanResult

	// Use raw SQL to get string timestamps
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			id,
			created_at,
			updated_at,
			user_id,
			name,
			hashed_key,
			prefix,
			expires_at,
			last_used_at
		FROM personal_api_keys
		WHERE id = ?
	`, id).Scan(&result).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return result.convertToDomain()
}

// FindByUserID finds all API keys for a user - using scanResult to handle SQLite timestamps
func (r *PersonalApiKeyRepository) FindByUserID(ctx context.Context, userID uuid.UUID, pagination *baseRepo.PaginationParams) (*baseRepo.PaginatedResult[personal_api_key.PersonalApiKey], error) {
	var total int64
	var scanResults []scanResult

	// Count total
	countQuery := r.db.WithContext(ctx).Model(&personal_api_key.PersonalApiKey{}).Where("user_id = ?", userID)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, err
	}

	// Build raw SQL query for results
	query := `
		SELECT
			id,
			created_at,
			updated_at,
			user_id,
			name,
			hashed_key,
			prefix,
			expires_at,
			last_used_at
		FROM personal_api_keys
		WHERE user_id = ?
		ORDER BY created_at DESC
	`

	// Add pagination
	if pagination != nil {
		pagination.Validate()
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", pagination.Limit, pagination.Offset)
	}

	err := r.db.WithContext(ctx).Raw(query, userID).Scan(&scanResults).Error
	if err != nil {
		return nil, err
	}

	// Convert scan results to domain models
	keys := make([]personal_api_key.PersonalApiKey, len(scanResults))
	for i, scanResult := range scanResults {
		apiKey, err := scanResult.convertToDomain()
		if err != nil {
			return nil, err
		}
		keys[i] = *apiKey
	}

	offset := 0
	limit := 0
	if pagination != nil {
		offset = pagination.Offset
		limit = pagination.Limit
	}

	return &baseRepo.PaginatedResult[personal_api_key.PersonalApiKey]{
		List:   keys,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}, nil
}

// Delete deletes an API key (hard delete)
func (r *PersonalApiKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&personal_api_key.PersonalApiKey{}, "id = ?", id).Error
}

// FindByHashedKey finds an API key by its hashed value (for authentication) - using scanResult to handle SQLite timestamps
func (r *PersonalApiKeyRepository) FindByHashedKey(ctx context.Context, hashedKey string) (*personal_api_key.PersonalApiKey, error) {
	var result scanResult

	// Use raw SQL to get string timestamps
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			id,
			created_at,
			updated_at,
			user_id,
			name,
			hashed_key,
			prefix,
			expires_at,
			last_used_at
		FROM personal_api_keys
		WHERE hashed_key = ?
	`, hashedKey).Scan(&result).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return result.convertToDomain()
}

// UpdateLastUsed updates the last used timestamp
func (r *PersonalApiKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	// Use SQLite-compatible datetime function
	return r.db.WithContext(ctx).Model(&personal_api_key.PersonalApiKey{}).
		Where("id = ?", id).
		Update("last_used_at", gorm.Expr("CURRENT_TIMESTAMP")).Error
}
