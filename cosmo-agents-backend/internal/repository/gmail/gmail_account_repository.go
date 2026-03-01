package gmail

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/google_token_store"

	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
	googleoauth "github.com/rockship/cosmo-agents-go/pkg/oauth2/google"
)

// GmailAccountRepository handles database operations for Gmail accounts
type GmailAccountRepository struct {
	*gormpkg.GormRepository[domain.GmailAccount]
}

// NewGmailAccountRepository creates a new GmailAccountRepository instance
func NewGmailAccountRepository(db *gorm.DB) *GmailAccountRepository {
	return &GmailAccountRepository{
		GormRepository: gormpkg.NewGormRepository[domain.GmailAccount](db),
	}
}

// FindByAgentID retrieves Gmail account by agent ID
func (r *GmailAccountRepository) FindByAgentID(ctx context.Context, agentID uuid.UUID) (*domain.GmailAccount, error) {
	var gmailAccount domain.GmailAccount
	err := r.GetDB().WithContext(ctx).
		Where("agent_id = ? AND is_deleted = ?", agentID, false).
		First(&gmailAccount).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find gmail account by agent id: %w", err)
	}

	return &gmailAccount, nil
}

// FindByEmail retrieves Gmail accounts by email
func (r *GmailAccountRepository) FindByEmail(ctx context.Context, email string) ([]*domain.GmailAccount, error) {
	var gmailAccounts []domain.GmailAccount
	err := r.GetDB().WithContext(ctx).
		Where("email = ? AND is_deleted = ?", email, false).
		Find(&gmailAccounts).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find gmail accounts by email: %w", err)
	}

	result := make([]*domain.GmailAccount, len(gmailAccounts))
	for i := range gmailAccounts {
		result[i] = &gmailAccounts[i]
	}

	return result, nil
}

// FindByUserAndEmail finds Gmail account by user ID and email by joining with agents
func (r *GmailAccountRepository) FindByUserAndEmail(ctx context.Context, userID uuid.UUID, email string) (*domain.GmailAccount, error) {
	var gmailAccount domain.GmailAccount
	err := r.GetDB().WithContext(ctx).
		Joins("JOIN agents ON gmail_accounts.agent_id = agents.id").
		Where("gmail_accounts.email = ? AND agents.user_id = ? AND gmail_accounts.is_deleted = ? AND agents.is_deleted = ?",
			email, userID, false, false).
		First(&gmailAccount).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find gmail account by user and email: %w", err)
	}

	return &gmailAccount, nil
}

// FindByEmailForWatching finds Gmail accounts for email watching
func (r *GmailAccountRepository) FindByEmailForWatching(ctx context.Context, email string) ([]*domain.GmailAccount, error) {
	var gmailAccounts []domain.GmailAccount
	err := r.GetDB().WithContext(ctx).
		Where("email = ? AND is_deleted = ?", email, false).
		Find(&gmailAccounts).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find gmail accounts for watching: %w", err)
	}

	result := make([]*domain.GmailAccount, len(gmailAccounts))
	for i := range gmailAccounts {
		result[i] = &gmailAccounts[i]
	}

	return result, nil
}

// UpdateLastHistoryID updates the last history ID for Gmail account
func (r *GmailAccountRepository) UpdateLastHistoryID(ctx context.Context, id uuid.UUID, historyID string) error {
	result := r.GetDB().WithContext(ctx).
		Model(&domain.GmailAccount{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Update("last_history_id", historyID)

	if result.Error != nil {
		return fmt.Errorf("failed to update last history id: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// AtomicRefreshToken atomically refreshes OAuth token for Gmail account
func (r *GmailAccountRepository) AtomicRefreshToken(ctx context.Context, id uuid.UUID, oauthClient *googleoauth.Client) (*google_token_store.GoogleTokenStore, error) {
	var newStore *google_token_store.GoogleTokenStore

	// Use transaction to ensure atomicity
	err := r.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var gmailAccount domain.GmailAccount
		err := tx.
			Where("id = ? AND is_deleted = ?", id, false).
			First(&gmailAccount).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("gmail account not found")
			}
			return fmt.Errorf("failed to get gmail account for token refresh: %w", err)
		}

		// Get current token store
		store, err := gmailAccount.GetGoogleTokenStore()
		if err != nil {
			return fmt.Errorf("failed to get current token store: %w", err)
		}

		// Refresh token using OAuth client
		newToken, err := oauthClient.RefreshToken(ctx, store.RefreshToken)
		if err != nil {
			// Mark account as invalid grant if refresh fails
			updateErr := tx.Model(&domain.GmailAccount{}).
				Where("id = ?", id).
				Update("status", string(domain.GmailAccountStatusInvalidGrant)).Error
			if updateErr != nil {
				return fmt.Errorf("failed to refresh token and failed to update status: %w, original error: %v", updateErr, err)
			}
			return fmt.Errorf("failed to refresh token: %w", err)
		}

		// Create new token store
		newStore = &google_token_store.GoogleTokenStore{
			AccessToken:  newToken.AccessToken,
			RefreshToken: newToken.RefreshToken,
			Expiry:       newToken.Expiry,
			Scopes:       store.Scopes, // Preserve existing scopes
		}

		// Update token store in domain
		if err := gmailAccount.UpdateGoogleTokenStore(newStore); err != nil {
			return fmt.Errorf("failed to update google token store: %w", err)
		}

		// Save the updated gmail account
		if err := tx.Save(&gmailAccount).Error; err != nil {
			return fmt.Errorf("failed to save updated gmail account: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return newStore, nil
}

// AtomicStatusUpdate atomically updates Gmail account status
func (r *GmailAccountRepository) AtomicStatusUpdate(ctx context.Context, id uuid.UUID, status domain.GmailAccountStatus) error {
	result := r.GetDB().WithContext(ctx).
		Model(&domain.GmailAccount{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Update("status", string(status))

	if result.Error != nil {
		return fmt.Errorf("failed to update gmail account status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
