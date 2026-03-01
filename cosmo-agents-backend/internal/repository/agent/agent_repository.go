package agent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	googleoauth "github.com/rockship/cosmo-agents-go/pkg/oauth2/google"
)

// OAuthClientInterface defines the interface for OAuth client operations
type OAuthClientInterface interface {
	RefreshToken(ctx context.Context, refreshToken string) (*googleoauth.Token, error)
}

// AgentRepository handles agent data operations.
type AgentRepository struct {
	*gormpkg.GormRepository[domain.Agent]
}

// NewAgentRepository creates a new agent gormpkg.
func NewAgentRepository(db *gorm.DB) *AgentRepository {
	return &AgentRepository{
		GormRepository: gormpkg.NewGormRepository[domain.Agent](db),
	}
}

// FindByIDs returns agents mapped by ID.
func (r *AgentRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Agent, error) {
	result := make(map[uuid.UUID]*domain.Agent)
	if len(ids) == 0 {
		return result, nil
	}

	var agents []domain.Agent
	if err := r.GetDB().WithContext(ctx).
		Where("id IN ?", ids).
		Find(&agents).Error; err != nil {
		return nil, err
	}

	for i := range agents {
		agent := agents[i]
		result[agent.ID] = &agent
	}

	return result, nil
}

// GetByEmail finds an agent by email.
func (r *AgentRepository) GetByEmail(ctx context.Context, email string) (*domain.Agent, error) {
	var agent domain.Agent

	err := r.GetDB().WithContext(ctx).
		Where("email = ?", email).
		First(&agent).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &agent, nil
}

// GetByUserID finds all agents for a user.
func (r *AgentRepository) GetByUserID(ctx context.Context, userID string) ([]domain.Agent, error) {
	var agents []domain.Agent

	err := r.GetDB().WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&agents).Error

	if err != nil {
		return nil, err
	}

	return agents, nil
}

// GetActiveAgents finds all active agents.
func (r *AgentRepository) GetActiveAgents(ctx context.Context) ([]domain.Agent, error) {
	var agents []domain.Agent

	err := r.GetDB().WithContext(ctx).
		Where("status = ?", "active").
		Find(&agents).Error

	if err != nil {
		return nil, err
	}

	return agents, nil
}

// FindByUserAndEmail fetches an agent by user ID and email.
func (r *AgentRepository) FindByUserAndEmail(ctx context.Context, userID uuid.UUID, email string) (*domain.Agent, error) {
	var agent domain.Agent
	err := r.GetDB().WithContext(ctx).
		Where("user_id = ? AND email = ?", userID, email).
		First(&agent).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

// FindByEmail finds all agents by email address
// Used by Gmail notification worker to process notifications
func (r *AgentRepository) FindByEmail(ctx context.Context, email string) ([]domain.Agent, error) {
	var agents []domain.Agent
	err := r.GetDB().WithContext(ctx).
		Where("email = ?", email).
		Find(&agents).Error

	if err != nil {
		return nil, err
	}
	return agents, nil
}

// ResetDailyEmailCounts resets the daily email count for all agents.
func (r *AgentRepository) ResetDailyEmailCounts(ctx context.Context) error {
	return r.GetDB().WithContext(ctx).
		Model(&domain.Agent{}).
		Update("emails_sent_today", 0).Error
}

// IncrementEmailCount increments the email sent count for an agent.
func (r *AgentRepository) IncrementEmailCount(ctx context.Context, agentID string, count int) error {
	return r.GetDB().WithContext(ctx).
		Model(&domain.Agent{}).
		Where("id = ?", agentID).
		UpdateColumn("emails_sent_today", gorm.Expr("emails_sent_today + ?", count)).Error
}

// UpdateLastHistoryID updates only the last_history_id column to avoid clobbering other fields.
func (r *AgentRepository) UpdateLastHistoryID(ctx context.Context, agentID uuid.UUID, historyID string) error {
	return r.GetDB().WithContext(ctx).
		Model(&domain.Agent{}).
		Where("id = ?", agentID).
		Update("last_history_id", historyID).Error
}

// AtomicStatusUpdate atomically updates the agent status using database-level locking to prevent race conditions.
func (r *AgentRepository) AtomicStatusUpdate(ctx context.Context, agentID uuid.UUID, status domain.AgentStatus) error {
	// Use FOR UPDATE to lock the row during the update
	tx := r.GetDB().WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback() // Always safe - no-op after Commit()

	var agent domain.Agent
	if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&agent, "id = ?", agentID).Error; err != nil {
		return err // Rollback via defer
	}

	if err := tx.Model(&agent).Update("status", status).Error; err != nil {
		return err // Rollback via defer
	}

	return tx.Commit().Error
}

// AtomicUpdateGoogleCredentials atomically updates Gmail credentials using database-level locking to prevent race conditions.
func (r *AgentRepository) AtomicUpdateGoogleCredentials(ctx context.Context, agentID uuid.UUID, accessToken, refreshToken string, expiry time.Time) error {
	// Use FOR UPDATE to lock the row during the update
	tx := r.GetDB().WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback() // Always safe - no-op after Commit()

	var agent domain.Agent
	if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&agent, "id = ?", agentID).Error; err != nil {
		return err // Rollback via defer
	}

	// Get existing token store to preserve scopes
	existingStore, err := agent.GetGoogleTokenStore()
	if err != nil {
		return err // Rollback via defer
	}

	// Create new token store with updated credentials, preserving existing scopes
	tokenStore := &domain.GoogleTokenStore{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       expiry,
		Scopes:       existingStore.Scopes, // Preserve existing scopes
	}

	// Update credentials atomically
	if err := agent.UpdateGoogleTokenStore(tokenStore); err != nil {
		return err // Rollback via defer
	}

	if err := tx.Model(&agent).
		Updates(map[string]interface{}{
			"credentials": agent.Credentials,
			"cmetadata":   agent.CMetadata,
		}).Error; err != nil {
		return err // Rollback via defer
	}

	return tx.Commit().Error
}

// AtomicRefreshToken atomically checks token validity and refreshes if needed using database locking
// This prevents race conditions where multiple goroutines might refresh the same token simultaneously
func (r *AgentRepository) AtomicRefreshToken(ctx context.Context, agentID uuid.UUID, oauthClient OAuthClientInterface) (*domain.GoogleTokenStore, error) {
	// Use FOR UPDATE to lock the row during the check-and-update
	tx := r.GetDB().WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer tx.Rollback() // Always safe - no-op after Commit()

	var agent domain.Agent
	if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&agent, "id = ?", agentID).Error; err != nil {
		return nil, err // Rollback via defer
	}

	// Get current token store
	store, err := agent.GetGoogleTokenStore()
	if err != nil {
		return nil, err // Rollback via defer
	}

	// Check if token is still valid (no refresh token or not expired)
	if len(store.RefreshToken) == 0 {
		// Commit transaction since we're done with the locked row
		if err := tx.Commit().Error; err != nil {
			return nil, err
		}
		return store, nil // No need to refresh, commit and return
	}

	if store.Expiry.After(time.Now().Add(1 * time.Minute)) {
		// Commit transaction since we're done with the locked row
		if err := tx.Commit().Error; err != nil {
			return nil, err
		}
		return store, nil // Token still valid, commit and return
	}

	// CRITICAL FIX: Release database lock BEFORE making HTTP call to OAuth provider
	// This prevents database connection starvation and deadlocks during slow OAuth responses
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Now make the HTTP call OUTSIDE of the transaction (optimistic locking pattern)
	refreshed, err := oauthClient.RefreshToken(ctx, store.RefreshToken)
	if err != nil {
		return nil, err // No transaction to rollback
	}

	// Create new token store with updated credentials, preserving existing scopes and refresh token
	newTokenStore := &domain.GoogleTokenStore{
		AccessToken:  refreshed.AccessToken,
		RefreshToken: refreshed.RefreshToken,
		Expiry:       refreshed.Expiry,
		Scopes:       store.Scopes, // Preserve existing scopes
	}

	// Preserve existing refresh token if OAuth provider doesn't return a new one
	if refreshed.RefreshToken == "" {
		newTokenStore.RefreshToken = store.RefreshToken
	}

	// CRITICAL: Start NEW transaction to update the database with refreshed token
	// Use optimistic locking - if token was updated by another goroutine, this will fail gracefully
	updateTx := r.GetDB().WithContext(ctx).Begin()
	if updateTx.Error != nil {
		return nil, updateTx.Error
	}
	defer updateTx.Rollback() // Always safe - no-op after Commit()

	// Re-fetch the agent with FOR UPDATE to ensure we have latest version
	var latestAgent domain.Agent
	if err := updateTx.Set("gorm:query_option", "FOR UPDATE").First(&latestAgent, "id = ?", agentID).Error; err != nil {
		return nil, err // Rollback via defer
	}

	// Double-check that the token still needs refreshing (optimistic lock check)
	latestStore, err := latestAgent.GetGoogleTokenStore()
	if err != nil {
		return nil, err // Rollback via defer
	}

	// If token was already refreshed by another goroutine, return the latest version
	if latestStore.Expiry.After(store.Expiry) {
		// Another goroutine already refreshed it, commit and return the latest
		if err := updateTx.Commit().Error; err != nil {
			return nil, err
		}
		return latestStore, nil
	}

	// Update credentials atomically within the new transaction
	if err := latestAgent.UpdateGoogleTokenStore(newTokenStore); err != nil {
		return nil, err // Rollback via defer
	}

	if err := updateTx.Model(&latestAgent).
		Updates(map[string]interface{}{
			"credentials": latestAgent.Credentials,
			"cmetadata":   latestAgent.CMetadata,
		}).Error; err != nil {
		return nil, err // Rollback via defer
	}

	// Commit the update transaction
	if err := updateTx.Commit().Error; err != nil {
		return nil, err
	}

	return newTokenStore, nil
}

// UpdateAgentStatusBasedOnTokenExpiry checks and updates agent status based on token expiry
// This method ensures proactive detection of expired tokens to prevent service interruption
func (r *AgentRepository) UpdateAgentStatusBasedOnTokenExpiry(ctx context.Context, agentID uuid.UUID) error {
	// CRITICAL FIX: Use transaction with FOR UPDATE lock to prevent race conditions
	tx := r.GetDB().WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback() // Always safe - no-op after Commit()

	// Get agent with row-level lock to prevent concurrent modifications
	var agent domain.Agent
	if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&agent, "id = ?", agentID).Error; err != nil {
		return err
	}

	// Check token status within the same transaction
	newStatus, err := agent.CheckTokenStatus()
	if err != nil {
		return fmt.Errorf("failed to check token status: %w", err)
	}

	// If status hasn't changed, no need to update
	if newStatus == agent.Status {
		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
		return nil
	}

	// Update status within the same transaction
	if err := tx.Model(&agent).Update("status", newStatus).Error; err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit status update: %w", err)
	}

	// Log the status change (after successful commit)
	logger.Logger.Info().
		Str("agent_id", agentID.String()).
		Str("old_status", string(agent.Status)).
		Str("new_status", string(newStatus)).
		Msg("Agent status updated due to token expiry check")

	return nil
}
