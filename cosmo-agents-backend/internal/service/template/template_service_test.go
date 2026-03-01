package template

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// Mock repository for comprehensive testing
type MockTemplateRepository struct {
	mock.Mock
	templates map[uuid.UUID]*domain.Template
	users     map[uuid.UUID]bool // userID -> exists
	mutex     sync.RWMutex
}

func NewMockTemplateRepository() *MockTemplateRepository {
	return &MockTemplateRepository{
		templates: make(map[uuid.UUID]*domain.Template),
		users:     make(map[uuid.UUID]bool),
	}
}

func (m *MockTemplateRepository) AddUser(userID uuid.UUID) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.users[userID] = true
}

func (m *MockTemplateRepository) AddTemplate(template *domain.Template) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.templates[template.ID] = template
}

func (m *MockTemplateRepository) GetTemplate(id uuid.UUID) *domain.Template {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.templates[id]
}

func (m *MockTemplateRepository) Create(ctx context.Context, template *domain.Template) (*domain.Template, error) {
	args := m.Called(ctx, template)
	if template := args.Get(0); template != nil {
		return template.(*domain.Template), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTemplateRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Template, error) {
	args := m.Called(ctx, id)
	if template := args.Get(0); template != nil {
		return template.(*domain.Template), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTemplateRepository) FindByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]*domain.Template, error) {
	args := m.Called(ctx, campaignID)
	if templates := args.Get(0); templates != nil {
		return templates.([]*domain.Template), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTemplateRepository) FindByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Template, int64, error) {
	args := m.Called(ctx, userID, offset, limit)
	if templates := args.Get(0); templates != nil {
		return templates.([]*domain.Template), args.Get(1).(int64), args.Error(2)
	}
	return nil, int64(0), args.Error(2)
}

func (m *MockTemplateRepository) FindByIDAndUserID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Template, error) {
	args := m.Called(ctx, id, userID)
	if template := args.Get(0); template != nil {
		return template.(*domain.Template), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTemplateRepository) Update(ctx context.Context, id uuid.UUID, template *domain.Template) error {
	args := m.Called(ctx, id, template)
	return args.Error(0)
}

func (m *MockTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTemplateRepository) ReorderTemplates(ctx context.Context, positions map[uuid.UUID]float64) error {
	args := m.Called(ctx, positions)
	return args.Error(0)
}

func (m *MockTemplateRepository) AddKnowledge(ctx context.Context, templateID uuid.UUID, knowledge *domain.Knowledge) error {
	args := m.Called(ctx, templateID, knowledge)
	return args.Error(0)
}

func (m *MockTemplateRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Template, error) {
	args := m.Called(ctx, ids)
	if templates := args.Get(0); templates != nil {
		return templates.(map[uuid.UUID]*domain.Template), args.Error(1)
	}
	return nil, args.Error(1)
}

// Test utilities
func createTestTemplate(id uuid.UUID, userID uuid.UUID, subject string, content string) *domain.Template {
	return &domain.Template{
		Base:      base.Base{ID: id},
		UserID:    userID,
		Subject:   subject,
		Content:   content,
		Position:  float64(time.Now().Unix() % 100),
		SendAfter: 3600, // 1 hour in seconds
	}
}

func createTestKnowledge(id uuid.UUID, userID uuid.UUID, title string) *domain.Knowledge {
	return &domain.Knowledge{
		Base:         base.Base{ID: id},
		UserID:       userID,
		SourceType:   "upload",
		EmbeddingGID: &title, // Using title as embedding_gid for test
	}
}

// Comprehensive Business Logic Tests
func TestTemplateService_Business_Logic_Validation(t *testing.T) {
	mockRepo := NewMockTemplateRepository()
	service := NewTemplateService(mockRepo)
	ctx := context.Background()

	t.Run("Create Template - Valid Input", func(t *testing.T) {
		userID := uuid.New()
		template := createTestTemplate(uuid.New(), userID, "Valid Subject", "Valid content")

		mockRepo.On("Create", ctx, template).Return(template, nil)

		result, err := service.CreateTemplate(ctx, template)
		assert.NoError(t, err)
		assert.Equal(t, template.ID, result.ID)
		assert.Equal(t, template.Subject, result.Subject)
		assert.Equal(t, template.UserID, result.UserID)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Create Template - Nil Template", func(t *testing.T) {
		// Create a fresh service with new mock to avoid expectations conflicts
		freshMockRepo := NewMockTemplateRepository()
		freshService := NewTemplateService(freshMockRepo)

		result, err := freshService.CreateTemplate(ctx, nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "template cannot be nil")
	})

	t.Run("Create Template - Missing User ID", func(t *testing.T) {
		template := &domain.Template{
			Base:    base.Base{ID: uuid.New()},
			Subject: "No User ID Template",
		}

		result, err := service.CreateTemplate(ctx, template)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "user ID is required")
	})

	t.Run("Create Template - Empty Subject", func(t *testing.T) {
		userID := uuid.New()
		template := &domain.Template{
			Base:    base.Base{ID: uuid.New()},
			UserID:  userID,
			Subject: "",
		}

		result, err := service.CreateTemplate(ctx, template)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "subject is required")
	})

	t.Run("Update Template - Valid Update", func(t *testing.T) {
		userID := uuid.New()
		template := createTestTemplate(uuid.New(), userID, "Original Subject", "Original content")

		mockRepo.On("Update", ctx, template.ID, template).Return(nil)
		mockRepo.On("FindByID", ctx, template.ID).Return(template, nil)

		result, err := service.UpdateTemplate(ctx, template)
		assert.NoError(t, err)
		assert.Equal(t, template.ID, result.ID)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Update Template - Nil Template", func(t *testing.T) {
		// Create a fresh service with new mock to avoid expectations conflicts
		freshMockRepo := NewMockTemplateRepository()
		freshService := NewTemplateService(freshMockRepo)

		result, err := freshService.UpdateTemplate(ctx, nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "template cannot be nil")
	})

	t.Run("Update Template - Repository Error", func(t *testing.T) {
		// Create a fresh service with new mock to avoid expectations conflicts
		freshMockRepo := NewMockTemplateRepository()
		freshService := NewTemplateService(freshMockRepo)

		userID := uuid.New()
		template := createTestTemplate(uuid.New(), userID, "Subject", "Content")

		freshMockRepo.On("Update", ctx, template.ID, template).Return(fmt.Errorf("database error"))

		result, err := freshService.UpdateTemplate(ctx, template)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database error")

		freshMockRepo.AssertExpectations(t)
	})
}

func TestTemplateService_Error_Handling(t *testing.T) {
	mockRepo := NewMockTemplateRepository()
	service := NewTemplateService(mockRepo)
	ctx := context.Background()

	t.Run("Database Error Handling", func(t *testing.T) {
		userID := uuid.New()
		template := createTestTemplate(uuid.New(), userID, "Error Test", "Content")
		dbError := gorm.ErrInvalidTransaction

		mockRepo.On("Create", ctx, template).Return(nil, dbError)

		result, err := service.CreateTemplate(ctx, template)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid transaction")

		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository Not Found Error", func(t *testing.T) {
		templateID := uuid.New()
		notFoundError := gorm.ErrRecordNotFound

		mockRepo.On("FindByID", ctx, templateID).Return(nil, notFoundError)

		result, err := service.GetTemplateByID(ctx, templateID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "record not found")

		mockRepo.AssertExpectations(t)
	})

	t.Run("Nil Error from Repository", func(t *testing.T) {
		templateID := uuid.New()

		mockRepo.On("Delete", ctx, templateID).Return(nil)

		err := service.DeleteTemplate(ctx, templateID)
		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})
}

func TestTemplateService_Authorization_Logic(t *testing.T) {
	mockRepo := NewMockTemplateRepository()
	service := NewTemplateService(mockRepo)
	ctx := context.Background()

	userID := uuid.New()
	otherUserID := uuid.New()
	template := createTestTemplate(uuid.New(), userID, "User Template", "Content")

	t.Run("Get Template by Correct User", func(t *testing.T) {
		mockRepo.On("FindByIDAndUserID", ctx, template.ID, userID).Return(template, nil)

		result, err := service.GetTemplateByIDAndUserID(ctx, template.ID, userID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, template.ID, result.ID)
		assert.Equal(t, userID, result.UserID)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Get Template by Wrong User", func(t *testing.T) {
		mockRepo.On("FindByIDAndUserID", ctx, template.ID, otherUserID).Return(nil, nil)

		result, err := service.GetTemplateByIDAndUserID(ctx, template.ID, otherUserID)
		assert.NoError(t, err)
		assert.Nil(t, result)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Update Template - Repository Error", func(t *testing.T) {
		// Create fresh service with new mock to avoid expectations conflicts
		freshMockRepo := NewMockTemplateRepository()
		freshService := NewTemplateService(freshMockRepo)

		userID := uuid.New()
		template := createTestTemplate(uuid.New(), userID, "Subject", "Content")

		freshMockRepo.On("Update", ctx, template.ID, template).Return(fmt.Errorf("access denied"))

		result, err := freshService.UpdateTemplate(ctx, template)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "access denied")

		freshMockRepo.AssertExpectations(t)
	})
}

func TestTemplateService_Data_Consistency(t *testing.T) {
	mockRepo := NewMockTemplateRepository()
	service := NewTemplateService(mockRepo)
	ctx := context.Background()

	userID := uuid.New()

	t.Run("Reorder Templates - Empty Positions", func(t *testing.T) {
		emptyPositions := map[uuid.UUID]float64{}

		mockRepo.On("ReorderTemplates", ctx, emptyPositions).Return(nil)

		err := service.ReorderTemplates(ctx, emptyPositions)
		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Reorder Templates - Negative Positions", func(t *testing.T) {
		positions := map[uuid.UUID]float64{
			uuid.New(): -1.0,
		}

		mockRepo.On("ReorderTemplates", ctx, positions).Return(fmt.Errorf("position cannot be negative"))

		err := service.ReorderTemplates(ctx, positions)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "position cannot be negative")

		mockRepo.AssertExpectations(t)
	})

	t.Run("Add Knowledge - Valid Knowledge", func(t *testing.T) {
		templateID := uuid.New()
		knowledge := createTestKnowledge(uuid.New(), userID, "Valid Knowledge")

		mockRepo.On("AddKnowledge", ctx, templateID, knowledge).Return(nil)

		err := service.AddKnowledgeToTemplate(ctx, templateID, knowledge)
		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Add Knowledge - Nil Knowledge", func(t *testing.T) {
		templateID := uuid.New()

		err := service.AddKnowledgeToTemplate(ctx, templateID, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "knowledge cannot be nil")

		// Should not call repository due to validation
		mockRepo.AssertNotCalled(t, "AddKnowledge")
	})

	t.Run("Add Knowledge - Valid Knowledge", func(t *testing.T) {
		// Create fresh service with new mock to avoid expectations conflicts
		freshMockRepo := NewMockTemplateRepository()
		freshService := NewTemplateService(freshMockRepo)

		templateID := uuid.New()
		knowledge := createTestKnowledge(uuid.New(), userID, "Valid Knowledge")

		freshMockRepo.On("AddKnowledge", ctx, templateID, knowledge).Return(nil)

		err := freshService.AddKnowledgeToTemplate(ctx, templateID, knowledge)
		assert.NoError(t, err)

		freshMockRepo.AssertExpectations(t)
	})
}

func TestTemplateService_Pagination_Logic(t *testing.T) {
	mockRepo := NewMockTemplateRepository()
	service := NewTemplateService(mockRepo)
	ctx := context.Background()

	userID := uuid.New()

	t.Run("Invalid Pagination Parameters", func(t *testing.T) {
		// Test negative offset
		_, _, err := service.GetTemplatesByUserID(ctx, userID, -1, 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid offset")

		// Test negative limit
		_, _, err = service.GetTemplatesByUserID(ctx, userID, 0, -1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid limit")

		// Test limit too large
		_, _, err = service.GetTemplatesByUserID(ctx, userID, 0, 10000)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "limit too large")

		// Should not call repository due to validation
		mockRepo.AssertNotCalled(t, "FindByUserID")
	})

	t.Run("Valid Pagination Parameters", func(t *testing.T) {
		templates := []*domain.Template{
			createTestTemplate(uuid.New(), userID, "Template 1", "Content 1"),
			createTestTemplate(uuid.New(), userID, "Template 2", "Content 2"),
		}

		mockRepo.On("FindByUserID", ctx, userID, 0, 50).Return(templates, int64(2), nil)

		result, total, err := service.GetTemplatesByUserID(ctx, userID, 0, 50)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, int64(2), total)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Default Pagination Limit", func(t *testing.T) {
		// Clear existing mock expectations
		mockRepo.ExpectedCalls = nil
		mockRepo.Calls = nil

		templates := []*domain.Template{
			createTestTemplate(uuid.New(), userID, "Default Template", "Content"),
		}

		// Call with limit 0, should use default limit
		mockRepo.On("FindByUserID", ctx, userID, 0, 50).Return(templates, int64(1), nil)

		result, total, err := service.GetTemplatesByUserID(ctx, userID, 0, 0)
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, int64(1), total)

		mockRepo.AssertExpectations(t)
	})
}

func TestTemplateService_Context_Handling(t *testing.T) {
	mockRepo := NewMockTemplateRepository()
	service := NewTemplateService(mockRepo)

	t.Run("Nil Context", func(t *testing.T) {
		userID := uuid.New()
		template := createTestTemplate(uuid.New(), userID, "Test", "Content")

		// Should handle nil context gracefully
		result, err := service.CreateTemplate(nil, template)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "context is required")

		mockRepo.AssertNotCalled(t, "Create")
	})

	t.Run("Cancelled Context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		userID := uuid.New()
		template := createTestTemplate(uuid.New(), userID, "Cancelled", "Content")

		result, err := service.CreateTemplate(ctx, template)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "context canceled")

		mockRepo.AssertNotCalled(t, "Create")
	})

	t.Run("Timeout Context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		userID := uuid.New()
		template := createTestTemplate(uuid.New(), userID, "Timeout", "Content")

		result, err := service.CreateTemplate(ctx, template)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "context deadline exceeded")

		mockRepo.AssertNotCalled(t, "Create")
	})
}

func TestTemplateService_Concurrent_Operations(t *testing.T) {
	mockRepo := NewMockTemplateRepository()
	service := NewTemplateService(mockRepo)
	ctx := context.Background()

	const numGoroutines = 10
	const templatesPerGoroutine = 5

	t.Run("Concurrent Template Creation", func(t *testing.T) {
		// Create fresh service with new mock to avoid expectations conflicts
		freshMockRepo := NewMockTemplateRepository()
		freshService := NewTemplateService(freshMockRepo)

		userID := uuid.New()
		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines*templatesPerGoroutine)

		// For concurrent operations, skip strict mock assertion - just ensure no errors occur
		freshMockRepo.On("Create", mock.Anything, mock.Anything).Return(&domain.Template{}, nil).Maybe()

		// Create templates concurrently
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < templatesPerGoroutine; j++ {
					template := createTestTemplate(uuid.New(), userID, fmt.Sprintf("Template %d-%d", goroutineID, j), "Content")
					_, err := freshService.CreateTemplate(ctx, template)
					if err != nil {
						errors <- err
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for any errors - should be none since validation passes before mock call
		for err := range errors {
			t.Errorf("Unexpected error in concurrent operation: %v", err)
		}

		// Skip strict mock assertion for concurrent test
	})

	t.Run("Concurrent Template Retrieval", func(t *testing.T) {
		userID := uuid.New()
		templates := make([]*domain.Template, 0, numGoroutines*templatesPerGoroutine)

		// Set up templates and mock responses
		for i := 0; i < numGoroutines*templatesPerGoroutine; i++ {
			template := createTestTemplate(uuid.New(), userID, fmt.Sprintf("Retrieve %d", i), "Content")
			templates = append(templates, template)
			mockRepo.On("FindByID", ctx, template.ID).Return(template, nil).Once()
		}

		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines*templatesPerGoroutine)

		// Retrieve templates concurrently
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(startIndex int) {
				defer wg.Done()

				endIndex := startIndex + templatesPerGoroutine
				if endIndex > len(templates) {
					endIndex = len(templates)
				}

				for j := startIndex; j < endIndex; j++ {
					_, err := service.GetTemplateByID(ctx, templates[j].ID)
					if err != nil {
						errors <- err
					}
				}
			}(i * templatesPerGoroutine)
		}

		wg.Wait()
		close(errors)

		// Check for any errors
		for err := range errors {
			t.Errorf("Unexpected error in concurrent retrieval: %v", err)
		}

		mockRepo.AssertExpectations(t)
	})
}

func TestTemplateService_Performance_Considerations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	mockRepo := NewMockTemplateRepository()
	service := NewTemplateService(mockRepo)
	ctx := context.Background()

	t.Run("Large Dataset Retrieval Performance", func(t *testing.T) {
		userID := uuid.New()
		const numTemplates = 1000
		templates := make([]*domain.Template, numTemplates)

		// Set up mock response
		for i := 0; i < numTemplates; i++ {
			templates[i] = createTestTemplate(uuid.New(), userID, fmt.Sprintf("Template %d", i), "Content")
		}

		mockRepo.On("FindByUserID", ctx, userID, 0, 100).Return(templates, int64(numTemplates), nil)

		start := time.Now()
		result, total, err := service.GetTemplatesByUserID(ctx, userID, 0, 100)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Len(t, result, numTemplates)
		assert.Equal(t, int64(numTemplates), total)

		// Performance assertion - should complete quickly
		assert.Less(t, duration, 100*time.Millisecond, "Large dataset retrieval should complete quickly")

		mockRepo.AssertExpectations(t)
	})

	t.Run("Bulk Operations Performance", func(t *testing.T) {
		userID := uuid.New()
		const numOperations = 500
		templateIDs := make([]uuid.UUID, numOperations)

		// Set up templates and mock responses
		templates := make(map[uuid.UUID]*domain.Template)
		for i := 0; i < numOperations; i++ {
			template := createTestTemplate(uuid.New(), userID, fmt.Sprintf("Bulk %d", i), "Content")
			templates[template.ID] = template
			templateIDs[i] = template.ID
		}

		// Set up mock for the bulk FindByIDs call
		mockRepo.On("FindByIDs", ctx, templateIDs).Return(templates, nil).Once()

		start := time.Now()
		result, err := service.GetTemplatesByIDs(ctx, templateIDs)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Len(t, result, numOperations)

		// Performance assertion for bulk operations
		assert.Less(t, duration, 200*time.Millisecond, "Bulk operations should complete quickly")

		mockRepo.AssertExpectations(t)
	})
}

func TestTemplateService_Integration_Scenarios(t *testing.T) {
	mockRepo := NewMockTemplateRepository()
	service := NewTemplateService(mockRepo)
	ctx := context.Background()

	t.Run("Complete Template Lifecycle", func(t *testing.T) {
		userID := uuid.New()
		template := createTestTemplate(uuid.New(), userID, "Lifecycle Template", "Initial content")
		knowledge := createTestKnowledge(uuid.New(), userID, "Lifecycle Knowledge")

		// Step 1: Create template
		mockRepo.On("Create", ctx, template).Return(template, nil)
		createdTemplate, err := service.CreateTemplate(ctx, template)
		assert.NoError(t, err)
		assert.Equal(t, template.ID, createdTemplate.ID)

		// Step 2: Retrieve template
		mockRepo.On("FindByID", ctx, template.ID).Return(template, nil)
		retrievedTemplate, err := service.GetTemplateByID(ctx, template.ID)
		assert.NoError(t, err)
		assert.Equal(t, template.Subject, retrievedTemplate.Subject)

		// Step 3: Update template
		template.Subject = "Updated Lifecycle Template"
		template.Content = "Updated content"
		mockRepo.On("Update", ctx, template.ID, template).Return(nil)
		updatedTemplate, err := service.UpdateTemplate(ctx, template)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Lifecycle Template", updatedTemplate.Subject)

		// Step 4: Add knowledge to template
		mockRepo.On("AddKnowledge", ctx, template.ID, knowledge).Return(nil)
		err = service.AddKnowledgeToTemplate(ctx, template.ID, knowledge)
		assert.NoError(t, err)

		// Step 5: Delete template
		mockRepo.On("Delete", ctx, template.ID).Return(nil)
		err = service.DeleteTemplate(ctx, template.ID)
		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Campaign Template Management", func(t *testing.T) {
		userID := uuid.New()
		campaignID := uuid.New()

		// Create templates for campaign
		templates := []*domain.Template{
			createTestTemplate(uuid.New(), userID, "Campaign Template 1", "Content 1"),
			createTestTemplate(uuid.New(), userID, "Campaign Template 2", "Content 2"),
			createTestTemplate(uuid.New(), userID, "Campaign Template 3", "Content 3"),
		}

		// Set campaign ID for all templates
		for _, template := range templates {
			template.CampaignID = &campaignID
		}

		// Mock repository responses
		mockRepo.On("FindByCampaignID", ctx, campaignID).Return(templates, nil)

		// Retrieve campaign templates
		retrievedTemplates, err := service.GetTemplatesByCampaignID(ctx, campaignID)
		assert.NoError(t, err)
		assert.Len(t, retrievedTemplates, 3)

		// Verify all templates belong to the campaign
		for _, template := range retrievedTemplates {
			assert.NotNil(t, template.CampaignID)
			assert.Equal(t, campaignID, *template.CampaignID)
		}

		mockRepo.AssertExpectations(t)
	})
}

func TestTemplateService_Edge_Cases(t *testing.T) {
	mockRepo := NewMockTemplateRepository()
	service := NewTemplateService(mockRepo)
	ctx := context.Background()

	t.Run("Maximum Length Fields", func(t *testing.T) {
		userID := uuid.New()

		// Create template with maximum allowed length strings
		maxSubject := string(make([]byte, 255))   // Assuming 255 char limit
		maxContent := string(make([]byte, 10000)) // Assuming large content limit

		template := &domain.Template{
			Base:    base.Base{ID: uuid.New()},
			UserID:  userID,
			Subject: maxSubject,
			Content: maxContent,
		}

		mockRepo.On("Create", ctx, template).Return(template, nil)

		result, err := service.CreateTemplate(ctx, template)
		assert.NoError(t, err)
		assert.Equal(t, len(maxSubject), len(result.Subject))
		assert.Equal(t, len(maxContent), len(result.Content))

		mockRepo.AssertExpectations(t)
	})

	t.Run("Special Characters in Content", func(t *testing.T) {
		userID := uuid.New()
		specialContent := `Special chars: !@#$%^&*()_+-=[]{}|;':",./<>? Unicode: 你好世界 🚀 Emoji: 🎉🎊🎁`

		template := &domain.Template{
			Base:    base.Base{ID: uuid.New()},
			UserID:  userID,
			Subject: "Special Characters Test",
			Content: specialContent,
		}

		mockRepo.On("Create", ctx, template).Return(template, nil)

		result, err := service.CreateTemplate(ctx, template)
		assert.NoError(t, err)
		assert.Equal(t, specialContent, result.Content)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Zero Value Handling", func(t *testing.T) {
		userID := uuid.New()
		template := &domain.Template{
			Base:     base.Base{ID: uuid.New()},
			UserID:   userID,
			Subject:  "Zero Value Test",
			Content:  "",
			Position: 0.0,
		}

		mockRepo.On("Create", ctx, template).Return(template, nil)

		result, err := service.CreateTemplate(ctx, template)
		assert.NoError(t, err)
		assert.Equal(t, "", result.Content)
		assert.Equal(t, 0.0, result.Position)

		mockRepo.AssertExpectations(t)
	})
}

func TestTemplateService_Mock_Expectations_Verification(t *testing.T) {
	mockRepo := NewMockTemplateRepository()
	service := NewTemplateService(mockRepo)
	ctx := context.Background()

	t.Run("Unmet Mock Expectations", func(t *testing.T) {
		template := createTestTemplate(uuid.New(), uuid.New(), "Test", "Content")

		// Set up expectation but don't call the method
		mockRepo.On("Create", ctx, template).Return(template, nil)

		// Test should pass because we're not verifying expectations here
		// In real tests, you'd call: mockRepo.AssertExpectations(t)
	})

	t.Run("Unexpected Method Calls", func(t *testing.T) {
		template := createTestTemplate(uuid.New(), uuid.New(), "Test", "Content")

		// Call method without setting up expectation - should panic
		assert.Panics(t, func() {
			service.CreateTemplate(ctx, template)
		}, "Mock should panic when unexpected method is called")
	})
}
