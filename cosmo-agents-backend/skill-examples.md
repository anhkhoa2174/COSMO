# 📚 Claude Skills - Practical Examples with Cosmo Backend

## 🎯 Example 1: Implement New Feature with TDD

### **Task:** Add WhatsApp Integration Feature

```bash
# Command to activate skill:
"Implement WhatsApp integration feature with test-driven-development and rock-test"
```

**Claude will automatically:**

1. **Phase 1: Architecture Design (cosmo-architecture-patterns)**
   - Create domain model `WhatsAppMessage`
   - Repository pattern for WhatsApp API
   - Service layer with business logic
   - Handler with REST endpoints

2. **Phase 2: Test-First Development (test-driven-development)**
```go
// File to be created: internal/domain/whatsapp/whatsapp_test.go
func TestWhatsAppMessage_Send(t *testing.T) {
    tests := []struct {
        name    string
        message WhatsAppMessage
        want    error
    }{
        {
            name: "Valid message sends successfully",
            message: WhatsAppMessage{
                To:      "+1234567890",
                Content: "Hello from Cosmo!",
            },
            want: nil,
        },
        {
            name: "Invalid phone number returns error",
            message: WhatsAppMessage{
                To:      "invalid",
                Content: "Hello",
            },
            want: ErrInvalidPhoneNumber,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := tt.message.Send()
            assert.Equal(t, tt.want, got)
        })
    }
}
```

3. **Phase 3: Implementation**
```go
// File to be created: internal/domain/whatsapp/whatsapp.go
type WhatsAppMessage struct {
    ID        string    `json:"id" gorm:"primaryKey"`
    To        string    `json:"to" gorm:"not null"`
    Content   string    `json:"content" gorm:"not null"`
    Status    string    `json:"status"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

func (wm *WhatsAppMessage) Send() error {
    // Validation
    if !isValidPhoneNumber(wm.To) {
        return ErrInvalidPhoneNumber
    }

    // API call implementation
    // ...
    return nil
}
```

## 🔍 Example 2: Code Review with rock-review

### **Task:** Review Email Service Performance

```bash
# Command to activate skill:
"Review email service performance using rock-review skill"
```

**Claude will analyze:**

1. **Security Assessment:**
   - SQL injection risks
   - Email template validation
   - Attachment security

2. **Performance Analysis:**
```go
// Claude will detect and suggest improvements for:
func (s *EmailService) SendBulkEmails(emails []Email) error {
    // ❌ BAD: N+1 queries
    for _, email := range emails {
        if err := s.db.Where("id = ?", email.TemplateID).First(&email.Template).Error; err != nil {
            return err
        }
        s.sendEmail(email)
    }

    // ✅ GOOD: Batch query
    templateIDs := extractTemplateIDs(emails)
    templates, err := s.db.Where("id IN ?", templateIDs).Find(&templates).Error
    if err != nil {
        return err
    }

    // Process emails with preloaded templates
    for _, email := range emails {
        template := templates[email.TemplateID]
        s.sendEmailWithTemplate(email, template)
    }
}
```

## 🏗️ Example 3: Architecture with cosmo-architecture-patterns

### **Task:** Design AI Content Generation Feature

```bash
# Command to activate skill:
"Design AI content generation feature using cosmo-architecture-patterns"
```

**Claude will create:**

1. **Domain Layer:**
```go
// internal/domain/ai_content/ai_content.go
type AIContent struct {
    base.Base
    Title       string     `json:"title" gorm:"not null"`
    Content     string     `json:"content" gorm:"type:text"`
    Prompt      string     `json:"prompt" gorm:"not null"`
    Model       string     `json:"model" gorm:"not null"`
    TokensUsed  int        `json:"tokens_used"`
    Status      string     `json:"status"` // draft, generating, completed, failed
    UserID      string     `json:"user_id"`
    User        User       `json:"user" gorm:"foreignKey:UserID"`
}

type AIContentRepository interface {
    Create(content *AIContent) error
    GetByID(id string) (*AIContent, error)
    Update(content *AIContent) error
    ListByUser(userID string, pagination Pagination) ([]AIContent, error)
}
```

2. **Service Layer:**
```go
// internal/service/ai_content/ai_content_service.go
type AIContentService interface {
    GenerateContent(ctx context.Context, req GenerateContentRequest) (*AIContent, error)
    RegenerateContent(ctx context.Context, id string) (*AIContent, error)
    GetUserContents(ctx context.Context, userID string, pagination Pagination) ([]AIContent, error)
}

type GenerateContentRequest struct {
    UserID  string `json:"user_id" validate:"required"`
    Title   string `json:"title" validate:"required,min=1,max=200"`
    Prompt  string `json:"prompt" validate:"required,min=10"`
    Model   string `json:"model" validate:"required,oneof=gpt-4 gpt-3.5-turbo"`
}
```

## 🚀 Example 4: Deployment with rock-deploy

### **Task:** Deploy Feature with Zero-Downtime

```bash
# Command to activate skill:
"Deploy AI content feature to staging with zero-downtime using rock-deploy"
```

**Claude will create deployment strategy:**

1. **Health Check Implementation:**
```go
// internal/handler/v1/health/health.go
type HealthChecker struct {
    db    *gorm.DB
    redis *redis.Client
    ai    ai.Client
}

func (h *HealthChecker) CheckDependencies(ctx context.Context) map[string]error {
    results := make(map[string]error)

    // Database check
    if sqlDB, err := h.db.DB(); err == nil {
        results["database"] = sqlDB.PingContext(ctx)
    }

    // Redis check
    results["redis"] = h.redis.Ping(ctx).Err()

    // AI service check
    results["ai_service"] = h.ai.HealthCheck(ctx)

    return results
}
```

2. **Migration Strategy:**
```sql
-- migrations/20240101_add_ai_content.up.sql
CREATE TABLE ai_contents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(200) NOT NULL,
    content TEXT,
    prompt TEXT NOT NULL,
    model VARCHAR(50) NOT NULL,
    tokens_used INTEGER DEFAULT 0,
    status VARCHAR(20) DEFAULT 'draft',
    user_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for performance
CREATE INDEX idx_ai_contents_user_id ON ai_contents(user_id);
CREATE INDEX idx_ai_contents_status ON ai_contents(status);
CREATE INDEX idx_ai_contents_created_at ON ai_contents(created_at);
```

## 📊 Example 5: Testing with rock-test

### **Task:** Comprehensive Tests for User Authentication

```bash
# Command to activate skill:
"Write comprehensive tests for user authentication with rock-test patterns"
```

**Claude will create:**

1. **Unit Tests:**
```go
// internal/domain/user/user_auth_test.go
func TestUser_ValidatePassword(t *testing.T) {
    // Test table approach
    tests := []struct {
        name        string
        user        User
        password    string
        expectError bool
        errorType   error
    }{
        {
            name: "Valid password matches",
            user: User{
                PasswordHash: hashPassword("correctPassword123"),
            },
            password:    "correctPassword123",
            expectError: false,
        },
        {
            name: "Invalid password returns error",
            user: User{
                PasswordHash: hashPassword("correctPassword123"),
            },
            password:    "wrongPassword",
            expectError: true,
            errorType:   ErrInvalidCredentials,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.user.ValidatePassword(tt.password)

            if tt.expectError {
                assert.Error(t, err)
                assert.ErrorIs(t, err, tt.errorType)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

2. **Integration Tests:**
```go
// tests/integration/auth_test.go
func TestAuthenticationFlow(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)

    // Create test user
    user := &domain.User{
        Email:    "test@example.com",
        Password: "password123",
    }
    err := db.Create(user).Error
    require.NoError(t, err)

    // Test login
    authService := auth.NewService(db, jwtSecret)

    loginReq := auth.LoginRequest{
        Email:    "test@example.com",
        Password: "password123",
    }

    result, err := authService.Login(context.Background(), loginReq)
    assert.NoError(t, err)
    assert.NotEmpty(t, result.Token)
    assert.Equal(t, user.ID, result.User.ID)
}
```

## 💡 Tips for Effective Usage

### **1. Combine Multiple Skills:**
```bash
"Design email campaign feature using cosmo-architecture-patterns, then implement with test-driven-development and rock-test"
```

### **2. Specific Requirements:**
```bash
"Implement WhatsApp integration with test-driven-development, focus on error handling and rate limiting"
```

### **3. Reference Existing Code:**
```bash
"Create notification service similar to existing email service, following same patterns with comprehensive testing"
```

### **4. Progressive Development:**
```bash
# Step 1: Architecture
"Design notification system architecture using cosmo-architecture-patterns"

# Step 2: Core implementation
"Implement basic notification functionality with unit tests"

# Step 3: Advanced features
"Add email templates and scheduling with integration tests"

# Step 4: Review
"Review notification system using rock-review skill"
```

## 🎯 Expected Results

When using Claude skills correctly, the team will achieve:

- **60% faster development** - Ready templates and patterns
- **80% test coverage** - Automated test generation
- **Zero production bugs** - Comprehensive testing
- **Consistent code quality** - Architecture patterns
- **3x faster reviews** - Automated quality checks