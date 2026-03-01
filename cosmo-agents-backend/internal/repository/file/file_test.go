package file

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	base "github.com/rockship/cosmo-agents-go/internal/repository/base"
)

func setupFileTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&domain.File{})
	require.NoError(t, err)

	return db
}

func createTestFile(userID uuid.UUID, fileName, s3Key string) *domain.File {
	return &domain.File{
		UserID:   &userID,
		Filename: fileName,
		S3Key:    s3Key,
		MimeType: "image/jpeg",
		Size:     1024,
	}
}

func TestNewFileRepository(t *testing.T) {
	db := setupFileTestDB(t)
	repo := NewFileRepository(db)

	assert.NotNil(t, repo)
	assert.Same(t, db, repo.db)
}

func TestFileRepository_Create(t *testing.T) {
	db := setupFileTestDB(t)
	repo := NewFileRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	file := createTestFile(userID, "test.jpg", "uploads/test.jpg")

	t.Run("success", func(t *testing.T) {
		result, err := repo.Create(ctx, file)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, file.ID, result.ID)
		assert.Equal(t, "test.jpg", result.Filename)
		assert.Equal(t, "uploads/test.jpg", result.S3Key)
		assert.Equal(t, userID, *result.UserID)

		// Verify it's in the database
		var found domain.File
		err = db.Where("id = ?", file.ID).First(&found).Error
		assert.NoError(t, err)
		assert.Equal(t, file.Filename, found.Filename)
	})

	t.Run("with context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "tenant_id", "test-tenant")
		newFile := createTestFile(userID, "context-test.jpg", "uploads/context-test.jpg")

		result, err := repo.Create(ctx, newFile)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "context-test.jpg", result.Filename)
	})
}

func TestFileRepository_FindByID(t *testing.T) {
	db := setupFileTestDB(t)
	repo := NewFileRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	file := createTestFile(userID, "findbyid.jpg", "uploads/findbyid.jpg")
	db.Create(file)

	t.Run("success", func(t *testing.T) {
		result, err := repo.FindByID(ctx, file.ID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, file.ID, result.ID)
		assert.Equal(t, "findbyid.jpg", result.Filename)
		assert.False(t, result.IsDeleted)
	})

	t.Run("not found", func(t *testing.T) {
		nonExistentID := uuid.New()
		result, err := repo.FindByID(ctx, nonExistentID)

		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("deleted file", func(t *testing.T) {
		deletedFile := createTestFile(userID, "deleted.jpg", "uploads/deleted.jpg")
		db.Create(deletedFile)
		db.Model(&domain.File{}).Where("id = ?", deletedFile.ID).Update("is_deleted", true)

		result, err := repo.FindByID(ctx, deletedFile.ID)

		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		assert.Nil(t, result)
	})
}

func TestFileRepository_FindByS3Key(t *testing.T) {
	db := setupFileTestDB(t)
	repo := NewFileRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	file := createTestFile(userID, "s3key.jpg", "uploads/unique-key.jpg")
	db.Create(file)

	t.Run("success", func(t *testing.T) {
		result, err := repo.FindByS3Key(ctx, "uploads/unique-key.jpg")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, file.ID, result.ID)
		assert.Equal(t, "uploads/unique-key.jpg", result.S3Key)
	})

	t.Run("not found", func(t *testing.T) {
		result, err := repo.FindByS3Key(ctx, "uploads/nonexistent.jpg")

		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("deleted file", func(t *testing.T) {
		deletedFile := createTestFile(userID, "deleted-s3.jpg", "uploads/deleted-key.jpg")
		db.Create(deletedFile)
		db.Model(&domain.File{}).Where("id = ?", deletedFile.ID).Update("is_deleted", true)

		result, err := repo.FindByS3Key(ctx, "uploads/deleted-key.jpg")

		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		assert.Nil(t, result)
	})
}

func TestFileRepository_FindByUserID(t *testing.T) {
	db := setupFileTestDB(t)
	repo := NewFileRepository(db)
	ctx := context.Background()

	userID1 := uuid.New()
	userID2 := uuid.New()

	// Create files for user 1
	files1 := []*domain.File{
		createTestFile(userID1, "user1-1.jpg", "uploads/user1-1.jpg"),
		createTestFile(userID1, "user1-2.jpg", "uploads/user1-2.jpg"),
		createTestFile(userID1, "user1-3.jpg", "uploads/user1-3.jpg"),
	}

	// Create files for user 2
	files2 := []*domain.File{
		createTestFile(userID2, "user2-1.jpg", "uploads/user2-1.jpg"),
		createTestFile(userID2, "user2-2.jpg", "uploads/user2-2.jpg"),
	}

	// Create and save all files
	for _, file := range append(files1, files2...) {
		db.Create(file)
	}

	t.Run("find all files for user 1", func(t *testing.T) {
		result, err := repo.FindByUserID(ctx, userID1, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(3), result.Total)
		assert.Len(t, result.List, 3)

		for _, file := range result.List {
			assert.Equal(t, userID1, *file.UserID)
		}
	})

	t.Run("find all files for user 2", func(t *testing.T) {
		result, err := repo.FindByUserID(ctx, userID2, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(2), result.Total)
		assert.Len(t, result.List, 2)

		for _, file := range result.List {
			assert.Equal(t, userID2, *file.UserID)
		}
	})

	t.Run("with pagination", func(t *testing.T) {
		pagination := &base.PaginationParams{
			Limit:  2,
			Offset: 1,
		}
		result, err := repo.FindByUserID(ctx, userID1, pagination)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(3), result.Total) // Total count should be 3
		assert.Len(t, result.List, 2)           // But we only get 2 due to pagination
		assert.Equal(t, 1, result.Offset)
		assert.Equal(t, 2, result.Limit)
	})

	t.Run("user with no files", func(t *testing.T) {
		emptyUserID := uuid.New()
		result, err := repo.FindByUserID(ctx, emptyUserID, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(0), result.Total)
		assert.Len(t, result.List, 0)
	})

	t.Run("exclude deleted files", func(t *testing.T) {
		// Create a deleted file for user 1
		deletedFile := createTestFile(userID1, "deleted.jpg", "uploads/deleted.jpg")
		db.Create(deletedFile)
		db.Model(&domain.File{}).Where("id = ?", deletedFile.ID).Update("is_deleted", true)

		result, err := repo.FindByUserID(ctx, userID1, nil)

		assert.NoError(t, err)
		assert.Equal(t, int64(3), result.Total) // Should still be 3, not 4
		assert.Len(t, result.List, 3)
	})
}

func TestFileRepository_FindAll(t *testing.T) {
	db := setupFileTestDB(t)
	repo := NewFileRepository(db)
	ctx := context.Background()

	userID1 := uuid.New()
	userID2 := uuid.New()

	// Create test files with different properties
	files := []*domain.File{
		{UserID: &userID1, Filename: "image1.jpg", S3Key: "uploads/image1.jpg", MimeType: "image/jpeg", Size: 1024},
		{UserID: &userID2, Filename: "image2.png", S3Key: "uploads/image2.png", MimeType: "image/png", Size: 2048},
		{UserID: &userID1, Filename: "document.pdf", S3Key: "uploads/document.pdf", MimeType: "application/pdf", Size: 3072},
		{UserID: &userID2, Filename: "video.mp4", S3Key: "uploads/video.mp4", MimeType: "video/mp4", Size: 4096},
	}

	// Create deleted file separately
	deletedFile := &domain.File{
		UserID:   &userID1,
		Filename: "deleted.jpg",
		S3Key:    "uploads/deleted.jpg",
		MimeType: "image/jpeg",
		Size:     512,
	}
	db.Create(deletedFile)
	deletedFile.IsDeleted = true
	db.Save(deletedFile)

	for _, file := range files {
		db.Create(file)
	}

	t.Run("find all without filters", func(t *testing.T) {
		result, err := repo.FindAll(ctx, base.Filter{}, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(4), result.Total) // Excluding deleted file
		assert.Len(t, result.List, 4)
	})

	t.Run("find all with mime type filter", func(t *testing.T) {
		filter := base.Filter{
			"mime_type": "image/jpeg",
		}
		result, err := repo.FindAll(ctx, filter, nil)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), result.Total) // Only image1.jpg (deleted.jpg is excluded)
		assert.Len(t, result.List, 1)
		assert.Equal(t, "image1.jpg", result.List[0].Filename)
	})

	t.Run("find all with pagination", func(t *testing.T) {
		pagination := &base.PaginationParams{
			Limit:  2,
			Offset: 1,
		}
		result, err := repo.FindAll(ctx, base.Filter{}, pagination)

		assert.NoError(t, err)
		assert.Equal(t, int64(4), result.Total)
		assert.Len(t, result.List, 2)
		assert.Equal(t, 1, result.Offset)
		assert.Equal(t, 2, result.Limit)
	})

	t.Run("find all with multiple filters", func(t *testing.T) {
		filter := base.Filter{
			"user_id": &userID1,
		}
		result, err := repo.FindAll(ctx, filter, nil)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), result.Total) // image1.jpg and document.pdf (deleted.jpg is excluded)
		assert.Len(t, result.List, 2)
	})
}

func TestFileRepository_Update(t *testing.T) {
	db := setupFileTestDB(t)
	repo := NewFileRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	file := createTestFile(userID, "update.jpg", "uploads/update.jpg")
	db.Create(file)

	t.Run("success", func(t *testing.T) {
		// Modify the file
		file.Filename = "updated-name.jpg"
		file.Size = 2048
		file.MimeType = "image/png"

		err := repo.Update(ctx, file)

		assert.NoError(t, err)

		// Verify the update
		var updated domain.File
		err = db.Where("id = ?", file.ID).First(&updated).Error
		assert.NoError(t, err)
		assert.Equal(t, "updated-name.jpg", updated.Filename)
		assert.Equal(t, int64(2048), updated.Size)
		assert.Equal(t, "image/png", updated.MimeType)
	})

	t.Run("non-existent file", func(t *testing.T) {
		nonExistentFile := createTestFile(userID, "nonexistent.jpg", "uploads/nonexistent.jpg")

		err := repo.Update(ctx, nonExistentFile)

		// GORM Save doesn't return error for new records, it creates them
		assert.NoError(t, err)

		// Verify it was created
		var found domain.File
		err = db.Where("id = ?", nonExistentFile.ID).First(&found).Error
		assert.NoError(t, err)
		assert.Equal(t, "nonexistent.jpg", found.Filename)
	})
}

func TestFileRepository_Delete(t *testing.T) {
	db := setupFileTestDB(t)
	repo := NewFileRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	file := createTestFile(userID, "delete.jpg", "uploads/delete.jpg")
	db.Create(file)

	t.Run("soft delete success", func(t *testing.T) {
		err := repo.Delete(ctx, file.ID)

		assert.NoError(t, err)

		// Verify soft delete
		var deleted domain.File
		err = db.Where("id = ?", file.ID).First(&deleted).Error
		assert.NoError(t, err)
		assert.True(t, deleted.IsDeleted)

		// Should not be found by FindByID (which filters out deleted records)
		found, err := repo.FindByID(ctx, file.ID)
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		assert.Nil(t, found)
	})

	t.Run("delete non-existent file", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.Delete(ctx, nonExistentID)

		// GORM doesn't return error when no rows are affected by Update
		assert.NoError(t, err)
	})

	t.Run("delete already deleted file", func(t *testing.T) {
		alreadyDeletedFile := createTestFile(userID, "already-deleted.jpg", "uploads/already-deleted.jpg")
		db.Create(alreadyDeletedFile)
		db.Model(&domain.File{}).Where("id = ?", alreadyDeletedFile.ID).Update("is_deleted", true)

		err := repo.Delete(ctx, alreadyDeletedFile.ID)

		assert.NoError(t, err)

		// Verify it's still marked as deleted
		var deleted domain.File
		err = db.Where("id = ?", alreadyDeletedFile.ID).First(&deleted).Error
		assert.NoError(t, err)
		assert.True(t, deleted.IsDeleted)
	})
}
