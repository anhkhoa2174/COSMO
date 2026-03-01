package user

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Migrate all required tables
	err = db.AutoMigrate(&domain.User{})
	require.NoError(t, err)

	return db
}

// createTestUser creates a test user
func createTestUser(t *testing.T, db *gorm.DB) *domain.User {
	user := &domain.User{
		Email:    "test@example.com",
		Name:     "Test User",
		JobTitle: "Developer",
	}

	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

// TestNewUserHandler tests creating a new user handler
func TestNewUserHandler(t *testing.T) {
	db := setupTestDB(t)
	userRepo := user.NewUserRepository(db)

	handler := NewUserHandler(userRepo)
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.userRepo)
}

// TestUserHandler_WhoAmI_Success tests successful WhoAmI request
func TestUserHandler_WhoAmI_Success(t *testing.T) {
	db := setupTestDB(t)
	userRepo := user.NewUserRepository(db)
	handler := NewUserHandler(userRepo)

	// Create a test user
	_ = createTestUser(t, db)

	// Create a test app
	app := fiber.New()
	app.Get("/test", handler.WhoAmI)

	// Create a request
	req := httptest.NewRequest("GET", "/test", nil)

	// Test the handler
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// This basic test just ensures the handler doesn't panic
	// The actual testing of context would need more setup
}

// TestUserHandler_WhoAmI_Basic tests basic handler creation
func TestUserHandler_WhoAmI_Basic(t *testing.T) {
	db := setupTestDB(t)
	userRepo := user.NewUserRepository(db)
	handler := NewUserHandler(userRepo)

	// Basic test to ensure handler exists and has correct method
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.userRepo)

	// Test that WhoAmI method exists
	assert.NotNil(t, handler.WhoAmI)
}
