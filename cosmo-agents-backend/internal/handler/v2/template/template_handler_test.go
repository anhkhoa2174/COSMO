package template

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	knowledgeRepo "github.com/rockship/cosmo-agents-go/internal/repository/knowledge"
	template "github.com/rockship/cosmo-agents-go/internal/repository/template"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Migrate all required tables
	err = db.AutoMigrate(
		&domain.Template{},
		&domain.Knowledge{},
	)
	require.NoError(t, err)

	return db
}

// createTestTemplate creates a test template
func createTestTemplate(t *testing.T, db *gorm.DB, userID uuid.UUID) *domain.Template {
	template := &domain.Template{
		UserID:  userID,
		Subject: "Test Subject",
		Content: "Test Content",
	}

	err := db.Create(template).Error
	require.NoError(t, err)
	return template
}

// TestNewTemplateHandler tests creating a new template handler
func TestNewTemplateHandler(t *testing.T) {
	db := setupTestDB(t)
	templateRepo := template.NewTemplateRepository(db)
	knowledgeRepoInstance := knowledgeRepo.NewKnowledgeRepository(db)

	handler := NewTemplateHandler(templateRepo, knowledgeRepoInstance)
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.templateRepo)
	assert.NotNil(t, handler.knowledgeRepo)
}

// TestTemplateHandler_Get_Basic tests basic template handler setup
func TestTemplateHandler_Get_Basic(t *testing.T) {
	db := setupTestDB(t)
	templateRepo := template.NewTemplateRepository(db)
	knowledgeRepoInstance := knowledgeRepo.NewKnowledgeRepository(db)
	handler := NewTemplateHandler(templateRepo, knowledgeRepoInstance)

	// Basic test to ensure handler exists and has correct method
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.templateRepo)
	assert.NotNil(t, handler.knowledgeRepo)

	// Test that Get method exists
	assert.NotNil(t, handler.Get)
}

// TestTemplateHandler_Update_Basic tests basic update handler setup
func TestTemplateHandler_Update_Basic(t *testing.T) {
	db := setupTestDB(t)
	templateRepo := template.NewTemplateRepository(db)
	knowledgeRepoInstance := knowledgeRepo.NewKnowledgeRepository(db)
	handler := NewTemplateHandler(templateRepo, knowledgeRepoInstance)

	// Basic test to ensure handler exists
	assert.NotNil(t, handler)

	// Test that Update method exists
	assert.NotNil(t, handler.Update)
}

// TestTemplateHandler_Delete_Basic tests basic delete handler setup
func TestTemplateHandler_Delete_Basic(t *testing.T) {
	db := setupTestDB(t)
	templateRepo := template.NewTemplateRepository(db)
	knowledgeRepoInstance := knowledgeRepo.NewKnowledgeRepository(db)
	handler := NewTemplateHandler(templateRepo, knowledgeRepoInstance)

	// Basic test to ensure handler exists
	assert.NotNil(t, handler)

	// Test that Delete method exists
	assert.NotNil(t, handler.Delete)
}

// TestTemplateHandler_AddKnowledge_Basic tests basic add knowledge handler setup
func TestTemplateHandler_AddKnowledge_Basic(t *testing.T) {
	db := setupTestDB(t)
	templateRepo := template.NewTemplateRepository(db)
	knowledgeRepoInstance := knowledgeRepo.NewKnowledgeRepository(db)
	handler := NewTemplateHandler(templateRepo, knowledgeRepoInstance)

	// Basic test to ensure handler exists
	assert.NotNil(t, handler)

	// Test that AddKnowledge method exists
	assert.NotNil(t, handler.AddKnowledge)
}

// TestTemplateHandler_RouteSetup tests basic route setup
func TestTemplateHandler_RouteSetup(t *testing.T) {
	db := setupTestDB(t)
	templateRepo := template.NewTemplateRepository(db)
	knowledgeRepoInstance := knowledgeRepo.NewKnowledgeRepository(db)
	handler := NewTemplateHandler(templateRepo, knowledgeRepoInstance)

	// Create a test app with routes
	app := fiber.New()

	// Setup routes (basic test)
	app.Get("/templates/:id", handler.Get)
	app.Patch("/templates/:id", handler.Update)
	app.Delete("/templates/:id", handler.Delete)
	app.Post("/templates/:id/knowledge", handler.AddKnowledge)

	// Test that app can be created successfully
	assert.NotNil(t, app)

	// Create a test request to ensure routes are setup
	req := httptest.NewRequest("GET", "/templates/test-id", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Request should be processed (even if it returns error)
	// This just tests that the routing setup works
}
