package gorm

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	base "github.com/rockship/cosmo-agents-go/internal/repository/base"
)

// TestModel is a simple model for testing the generic GORM repository
type TestModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key"`
	Name      string    `gorm:"size:255;not null"`
	Email     string    `gorm:"size:255;uniqueIndex"`
	Age       int
	IsActive  bool `gorm:"default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
	IsDeleted bool `gorm:"default:false"`
}

func (TestModel) TableName() string {
	return "test_models"
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&TestModel{})
	require.NoError(t, err)

	return db
}

func createTestModel(name, email string, age int) *TestModel {
	return &TestModel{
		ID:       uuid.New(),
		Name:     name,
		Email:    email,
		Age:      age,
		IsActive: true,
	}
}

func TestGormRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[TestModel](db)

	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		model := createTestModel("Test User", "test@example.com", 25)

		result, err := repo.Create(ctx, model)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Test User", result.Name)
		assert.Equal(t, "test@example.com", result.Email)
		assert.Equal(t, 25, result.Age)
		assert.True(t, result.IsActive)
	})

	t.Run("with context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "tenant_id", "test-tenant")
		model := createTestModel("Context User", "context@example.com", 30)

		result, err := repo.Create(ctx, model)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Context User", result.Name)
	})
}

func TestGormRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[TestModel](db)
	ctx := context.Background()

	// Create a test model first
	model := createTestModel("Find Me", "find@example.com", 28)
	db.Create(model)

	t.Run("success", func(t *testing.T) {
		result, err := repo.FindByID(ctx, model.ID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, model.ID, result.ID)
		assert.Equal(t, "Find Me", result.Name)
	})

	t.Run("not found", func(t *testing.T) {
		nonExistentID := uuid.New()
		result, err := repo.FindByID(ctx, nonExistentID)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("with context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "tenant_id", "test-tenant")
		result, err := repo.FindByID(ctx, model.ID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestGormRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[TestModel](db)
	ctx := context.Background()

	// Create a test model first
	model := createTestModel("Get Me", "get@example.com", 32)
	db.Create(model)

	t.Run("success", func(t *testing.T) {
		result, err := repo.GetByID(ctx, model.ID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, model.ID, result.ID)
		assert.Equal(t, "Get Me", result.Name)
	})
}

func TestGormRepository_FindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[TestModel](db)
	ctx := context.Background()

	// Create test data
	models := []*TestModel{
		createTestModel("User 1", "user1@example.com", 25),
		createTestModel("User 2", "user2@example.com", 30),
		createTestModel("User 3", "user3@example.com", 35),
		createTestModel("Another User", "another@example.com", 40),
	}

	for _, model := range models {
		db.Create(model)
	}

	t.Run("find all with default pagination", func(t *testing.T) {
		result, err := repo.FindAll(ctx, base.Filter{}, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(4), result.Total)
		assert.Len(t, result.List, 4)
	})

	t.Run("find all with custom pagination", func(t *testing.T) {
		pagination := &base.PaginationParams{
			Limit:  2,
			Offset: 1,
		}
		pagination.Validate()
		result, err := repo.FindAll(ctx, base.Filter{}, pagination)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(4), result.Total)
		assert.Len(t, result.List, 2)
		assert.Equal(t, 1, result.Offset)
		assert.Equal(t, 2, result.Limit)
	})
}

func TestGormRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[TestModel](db)
	ctx := context.Background()

	// Create a test model first
	model := createTestModel("Original Name", "original@example.com", 25)
	db.Create(model)

	t.Run("success", func(t *testing.T) {
		// First verify the record exists
		var existing TestModel
		err := db.Where("id = ?", model.ID).First(&existing).Error
		require.NoError(t, err)
		assert.Equal(t, "Original Name", existing.Name)

		updatedModel := &TestModel{
			Name:     "Updated Name",
			Email:    "updated@example.com",
			Age:      30,
			IsActive: false,
		}

		err = repo.Update(ctx, model.ID, updatedModel)
		assert.NoError(t, err)

		// Verify the update
		var result TestModel
		err = db.Where("id = ? AND is_deleted = ?", model.ID, false).First(&result).Error
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", result.Name)
		assert.Equal(t, "updated@example.com", result.Email)
		assert.Equal(t, 30, result.Age)
		assert.False(t, result.IsActive)
	})

	t.Run("non-existent record", func(t *testing.T) {
		nonExistentID := uuid.New()
		updatedModel := &TestModel{
			Name: "Should Not Update",
		}

		err := repo.Update(ctx, nonExistentID, updatedModel)

		// GORM doesn't return error for non-existent records with Updates
		assert.NoError(t, err)
	})
}

func TestGormRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[TestModel](db)
	ctx := context.Background()

	// Create a test model first
	model := createTestModel("To Delete", "delete@example.com", 25)
	db.Create(model)

	t.Run("soft delete success", func(t *testing.T) {
		err := repo.Delete(ctx, model.ID)

		assert.NoError(t, err)

		// Verify soft delete
		var result TestModel
		err = db.Where("id = ?", model.ID).First(&result).Error
		assert.NoError(t, err)
		assert.True(t, result.IsDeleted)

		// Should not be found by FindByID (which filters out deleted records)
		found, err := repo.FindByID(ctx, model.ID)
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestGormRepository_HardDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[TestModel](db)
	ctx := context.Background()

	// Create a test model first
	model := createTestModel("To Hard Delete", "harddelete@example.com", 25)
	db.Create(model)

	t.Run("hard delete success", func(t *testing.T) {
		err := repo.HardDelete(ctx, model.ID)

		assert.NoError(t, err)

		// Verify hard delete
		var result TestModel
		err = db.Where("id = ?", model.ID).First(&result).Error
		assert.Error(t, err) // Should return "record not found" error
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})
}

func TestGormRepository_Count(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[TestModel](db)
	ctx := context.Background()

	// Create test data
	models := []*TestModel{
		createTestModel("User 1", "user1@example.com", 25),
		createTestModel("User 2", "user2@example.com", 30),
		createTestModel("User 3", "user3@example.com", 35),
		createTestModel("Another User", "another@example.com", 40),
	}

	for _, model := range models {
		db.Create(model)
	}

	t.Run("count all", func(t *testing.T) {
		count, err := repo.Count(ctx, base.Filter{})

		assert.NoError(t, err)
		assert.Equal(t, int64(4), count)
	})
}

func TestGormRepository_Transaction(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[TestModel](db)
	ctx := context.Background()

	t.Run("transaction success", func(t *testing.T) {
		model1 := createTestModel("Transaction User 1", "trans1@example.com", 25)
		model2 := createTestModel("Transaction User 2", "trans2@example.com", 30)

		err := repo.Transaction(ctx, func(tx *gorm.DB) error {
			if err := tx.Create(model1).Error; err != nil {
				return err
			}
			if err := tx.Create(model2).Error; err != nil {
				return err
			}
			return nil
		})

		assert.NoError(t, err)

		// Verify both models were created
		var count int64
		db.Model(&TestModel{}).Where("email IN ?", []string{"trans1@example.com", "trans2@example.com"}).Count(&count)
		assert.Equal(t, int64(2), count)
	})

	t.Run("transaction rollback", func(t *testing.T) {
		model1 := createTestModel("Rollback User 1", "rollback1@example.com", 25)
		model2 := createTestModel("Rollback User 2", "rollback2@example.com", 30)

		err := repo.Transaction(ctx, func(tx *gorm.DB) error {
			if err := tx.Create(model1).Error; err != nil {
				return err
			}
			// Force an error
			if err := tx.Create(model2).Error; err != nil {
				return err
			}
			// Simulate an error after creation
			return assert.AnError
		})

		assert.Error(t, err)

		// Verify no models were created due to rollback
		var count int64
		db.Model(&TestModel{}).Where("email IN ?", []string{"rollback1@example.com", "rollback2@example.com"}).Count(&count)
		assert.Equal(t, int64(0), count)
	})
}

func TestGormRepository_GetDB(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[TestModel](db)

	retrievedDB := repo.GetDB()

	assert.Same(t, db, retrievedDB)
}

func TestGormRepository_WithContext(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormRepository[TestModel](db)

	// Test with context handling
	ctx := context.Background()
	model := createTestModel("Context Test", "context@example.com", 25)

	// This should work without error even with context
	result, err := repo.Create(ctx, model)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Context Test", result.Name)
}
