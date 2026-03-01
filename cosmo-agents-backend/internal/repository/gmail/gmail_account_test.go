package gmail

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/agent"
)

func setupGmailTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Migrate all necessary tables
	err = db.AutoMigrate(
		&agent.Agent{},
		&domain.GmailAccount{},
	)
	require.NoError(t, err)

	return db
}

func createTestAgent(t *testing.T, db *gorm.DB, userID uuid.UUID, suffix string) *agent.Agent {
	testAgent := &agent.Agent{
		UserID: userID,
		Name:   "Test Agent " + suffix,
		Email:  fmt.Sprintf("test-agent-%s@example.com", suffix),
	}
	err := db.Create(testAgent).Error
	require.NoError(t, err)
	return testAgent
}

func createTestGmailAccount(t *testing.T, db *gorm.DB, agentID uuid.UUID, email string) *domain.GmailAccount {
	// Create a simple GmailAccount for testing
	gmailAccount := &domain.GmailAccount{
		AgentID:       agentID,
		Email:         email,
		Name:          "Test Gmail Account",
		Status:        domain.GmailAccountStatusActive,
		LastHistoryID: "history-123",
	}

	err := db.Create(gmailAccount).Error
	require.NoError(t, err)
	return gmailAccount
}

func TestNewGmailAccountRepository(t *testing.T) {
	db := setupGmailTestDB(t)
	repo := NewGmailAccountRepository(db)

	assert.NotNil(t, repo)
	assert.NotNil(t, repo.GormRepository)
	assert.Same(t, db, repo.GetDB())
}

func TestGmailAccountRepository_FindByAgentID(t *testing.T) {
	db := setupGmailTestDB(t)
	repo := NewGmailAccountRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	testAgent := createTestAgent(t, db, userID, "1")
	gmailAccount := createTestGmailAccount(t, db, testAgent.ID, "test@example.com")

	t.Run("success", func(t *testing.T) {
		result, err := repo.FindByAgentID(ctx, testAgent.ID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, gmailAccount.ID, result.ID)
		assert.Equal(t, testAgent.ID, result.AgentID)
		assert.Equal(t, "test@example.com", result.Email)
	})

	t.Run("not found", func(t *testing.T) {
		nonExistentID := uuid.New()
		result, err := repo.FindByAgentID(ctx, nonExistentID)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("deleted record", func(t *testing.T) {
		// Create a gmail account and mark it as deleted
		deletedAgent := createTestAgent(t, db, userID, "deleted")
		deletedGmail := createTestGmailAccount(t, db, deletedAgent.ID, "deleted@example.com")
		db.Model(&domain.GmailAccount{}).Where("id = ?", deletedGmail.ID).Update("is_deleted", true)

		result, err := repo.FindByAgentID(ctx, deletedAgent.ID)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestGmailAccountRepository_FindByEmail(t *testing.T) {
	db := setupGmailTestDB(t)
	repo := NewGmailAccountRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	testAgent1 := createTestAgent(t, db, userID, "1")
	testAgent2 := createTestAgent(t, db, userID, "2")

	createTestGmailAccount(t, db, testAgent1.ID, "shared1@example.com")
	createTestGmailAccount(t, db, testAgent2.ID, "shared2@example.com")

	t.Run("success - find accounts by email", func(t *testing.T) {
		result, err := repo.FindByEmail(ctx, "shared1@example.com")

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "shared1@example.com", result[0].Email)
	})

	t.Run("not found", func(t *testing.T) {
		result, err := repo.FindByEmail(ctx, "nonexistent@example.com")

		assert.NoError(t, err)
		assert.Len(t, result, 0)
	})
}

func TestGmailAccountRepository_FindByUserAndEmail(t *testing.T) {
	db := setupGmailTestDB(t)
	repo := NewGmailAccountRepository(db)
	ctx := context.Background()

	userID1 := uuid.New()
	userID2 := uuid.New()

	agent1 := createTestAgent(t, db, userID1, "1")
	agent2 := createTestAgent(t, db, userID2, "2")

	gmailAccount1 := createTestGmailAccount(t, db, agent1.ID, "user1@example.com")
	createTestGmailAccount(t, db, agent2.ID, "user2@example.com") // Different user

	t.Run("success", func(t *testing.T) {
		result, err := repo.FindByUserAndEmail(ctx, userID1, "user1@example.com")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, gmailAccount1.ID, result.ID)
		assert.Equal(t, "user1@example.com", result.Email)
	})

	t.Run("wrong user", func(t *testing.T) {
		result, err := repo.FindByUserAndEmail(ctx, userID2, "user1@example.com")

		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("wrong email", func(t *testing.T) {
		result, err := repo.FindByUserAndEmail(ctx, userID1, "wrong@example.com")

		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("deleted agent", func(t *testing.T) {
		deletedAgent := createTestAgent(t, db, userID1, "deleted")
		deletedGmail := createTestGmailAccount(t, db, deletedAgent.ID, "deleted@example.com")
		db.Model(&domain.GmailAccount{}).Where("id = ?", deletedGmail.ID).Update("is_deleted", true)

		result, err := repo.FindByUserAndEmail(ctx, userID1, "deleted@example.com")

		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("deleted gmail account", func(t *testing.T) {
		deletedAgent := createTestAgent(t, db, userID1, "deleted2")
		deletedGmail := createTestGmailAccount(t, db, deletedAgent.ID, "deleted2@example.com")
		db.Model(&domain.GmailAccount{}).Where("id = ?", deletedGmail.ID).Update("is_deleted", true)

		result, err := repo.FindByUserAndEmail(ctx, userID1, "deleted2@example.com")

		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestGmailAccountRepository_FindByEmailForWatching(t *testing.T) {
	db := setupGmailTestDB(t)
	repo := NewGmailAccountRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	testAgent1 := createTestAgent(t, db, userID, "1")
	testAgent2 := createTestAgent(t, db, userID, "2")

	createTestGmailAccount(t, db, testAgent1.ID, "watch1@example.com")
	createTestGmailAccount(t, db, testAgent2.ID, "watch2@example.com")

	t.Run("success", func(t *testing.T) {
		result, err := repo.FindByEmailForWatching(ctx, "watch1@example.com")

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "watch1@example.com", result[0].Email)
	})

	t.Run("not found", func(t *testing.T) {
		result, err := repo.FindByEmailForWatching(ctx, "notfound@example.com")

		assert.NoError(t, err)
		assert.Len(t, result, 0)
	})
}

func TestGmailAccountRepository_UpdateLastHistoryID(t *testing.T) {
	db := setupGmailTestDB(t)
	repo := NewGmailAccountRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	testAgent := createTestAgent(t, db, userID, "1")
	gmailAccount := createTestGmailAccount(t, db, testAgent.ID, "history@example.com")

	t.Run("success", func(t *testing.T) {
		newHistoryID := "history-456"
		err := repo.UpdateLastHistoryID(ctx, gmailAccount.ID, newHistoryID)

		assert.NoError(t, err)

		// Verify update
		var updated domain.GmailAccount
		err = db.Where("id = ?", gmailAccount.ID).First(&updated).Error
		assert.NoError(t, err)
		assert.Equal(t, newHistoryID, updated.LastHistoryID)
	})

	t.Run("not found", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.UpdateLastHistoryID(ctx, nonExistentID, "history-789")

		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("deleted account", func(t *testing.T) {
		deletedAgent := createTestAgent(t, db, userID, "deleted")
		deletedGmail := createTestGmailAccount(t, db, deletedAgent.ID, "deleted@example.com")
		db.Model(&domain.GmailAccount{}).Where("id = ?", deletedGmail.ID).Update("is_deleted", true)

		err := repo.UpdateLastHistoryID(ctx, deletedGmail.ID, "history-999")

		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})
}

func TestGmailAccountRepository_AtomicStatusUpdate(t *testing.T) {
	db := setupGmailTestDB(t)
	repo := NewGmailAccountRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	testAgent := createTestAgent(t, db, userID, "1")
	gmailAccount := createTestGmailAccount(t, db, testAgent.ID, "status@example.com")

	t.Run("success", func(t *testing.T) {
		err := repo.AtomicStatusUpdate(ctx, gmailAccount.ID, domain.GmailAccountStatusInactive)

		assert.NoError(t, err)

		// Verify update
		var updated domain.GmailAccount
		err = db.Where("id = ?", gmailAccount.ID).First(&updated).Error
		assert.NoError(t, err)
		assert.Equal(t, domain.GmailAccountStatusInactive, updated.Status)
	})

	t.Run("not found", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.AtomicStatusUpdate(ctx, nonExistentID, domain.GmailAccountStatusActive)

		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})
}
