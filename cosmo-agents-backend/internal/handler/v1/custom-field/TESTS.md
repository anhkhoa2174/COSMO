# Custom Field Handler Tests

## Overview
Comprehensive unit tests for the custom-field module using the testify/mock framework.

## Test Coverage
**84.8% code coverage**

## Test Structure

### Test Files
- `handler_test.go` - Mock setup and Create endpoint tests
- `list_test.go` - List endpoint tests  
- `update_test.go` - Update endpoint tests
- `delete_test.go` - Delete endpoint tests

### Mocks
All repository dependencies are mocked using testify/mock:
- `MockCustomFieldRepository` - Custom field data operations
- `MockUserRepository` - User data operations
- `MockRoleRepository` - Role/organization data operations

## Running Tests

### Run all tests
```bash
go test ./internal/handler/v1/custom-field/... -v
```

### Run with coverage
```bash
go test ./internal/handler/v1/custom-field/... -cover
```

### Run specific test
```bash
go test ./internal/handler/v1/custom-field/... -run TestCreateCustomField -v
```

## Test Cases

### Create Endpoint (6 tests)
- ✅ Success - Create with organization_id from user's active role
- ✅ Success - Create without organization_id
- ✅ Fail - Unauthorized (no user_id in context)
- ✅ Fail - Invalid request body
- ✅ Fail - Validation error (missing name)
- ✅ Fail - Repository error

### List Endpoint (5 tests)
- ✅ Success - List all custom fields
- ✅ Success - Filter by entity_type
- ✅ Success - Pagination (offset & limit)
- ✅ Fail - Unauthorized
- ✅ Fail - Repository error

### Update Endpoint (3 tests)
- ✅ Success - Update name only (partial update)
- ✅ Fail - Not found (404 with gorm.ErrRecordNotFound)
- ✅ Fail - Forbidden (user doesn't own custom field)

### Delete Endpoint (6 tests)
- ✅ Success - Delete custom field (hard delete)
- ✅ Fail - Unauthorized
- ✅ Fail - Invalid UUID
- ✅ Fail - Not found (404 with gorm.ErrRecordNotFound)
- ✅ Fail - Forbidden (user doesn't own custom field)
- ✅ Fail - Delete error

## Key Testing Patterns

### 1. Interface-Based Mocking
The handler uses interfaces instead of concrete types, enabling easy mocking:
```go
type CustomFieldRepository interface {
    Create(ctx context.Context, field *domain.CustomField) (*domain.CustomField, error)
    FindByID(ctx context.Context, id uuid.UUID) (*domain.CustomField, error)
    FindAll(ctx context.Context, filter repository.Filter, pagination *repository.PaginationParams) (*repository.PaginatedResult[domain.CustomField], error)
    UpdateFields(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
    Delete(ctx context.Context, id uuid.UUID) error
}
```

### 2. Fresh Mocks Per Test
Each test creates new mock instances to avoid shared state:
```go
mockCustomFieldRepo := new(MockCustomFieldRepository)
mockUserRepo := new(MockUserRepository)
mockRoleRepo := new(MockRoleRepository)
handler := NewHandler(mockCustomFieldRepo, mockUserRepo, mockRoleRepo)
```

### 3. Matcher Functions
Complex argument matching using testify matchers:
```go
mock.MatchedBy(func(f repository.Filter) bool {
    return f["user_id"] == userID.String() && f["entity_type"] == "company"
})
```

### 4. GORM Error Handling
Tests verify proper handling of GORM errors:
```go
mockCustomFieldRepo.On("FindByID", mock.Anything, fieldID).
    Return(nil, gorm.ErrRecordNotFound).Once()
// Expect 404 response
```

## Implementation Notes

### Hard Delete vs Soft Delete
Custom fields use **hard delete** (actual database deletion) because:
- The `custom_fields` table doesn't have `is_deleted` or `deleted_at` columns
- The `CustomField` domain model doesn't include `SoftDeleteMixin`
- Migration 000007 doesn't create soft delete columns

### BeforeUpdate Hook Bypass
The UpdateFields method uses `UpdateColumns()` instead of `Updates()` to bypass GORM's BeforeUpdate hook:
```go
func (r *CustomFieldRepository) UpdateFields(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
    // UpdateColumns skips BeforeUpdate hook to avoid validation on unmodified fields
    return r.db.WithContext(ctx).
        Model(&domain.CustomField{}).
        Where("id = ?", id).
        UpdateColumns(updates).Error
}
```

### Organization ID Resolution
Create endpoint automatically populates `organization_id` from user's active role:
```go
roles, err := h.roleRepo.FindByUserID(c.Context(), userID)
for _, role := range roles {
    if role.Status == "active" {
        organizationID = &role.OrganizationID
        break
    }
}
```

## Dependencies
- `github.com/stretchr/testify` - Assertion and mocking framework
- `github.com/gofiber/fiber/v3` - HTTP framework
- `gorm.io/gorm` - ORM (for error types)
