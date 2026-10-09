package contact

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
)

// MockContactRepo simulates the repository interface for unit testing
type MockContactRepo struct {
	mock.Mock
}

func (m *MockContactRepo) Create(ctx context.Context, c *domain.Contact) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}
func (m *MockContactRepo) Update(ctx context.Context, id uuid.UUID, c *domain.Contact) error {
	args := m.Called(ctx, id, c)
	return args.Error(0)
}
func (m *MockContactRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Contact, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*domain.Contact), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockContactRepo) FindByUserIDWithPagination(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Contact, int, error) {
	args := m.Called(ctx, userID, offset, limit)
	return args.Get(0).([]*domain.Contact), args.Int(1), args.Error(2)
}
func (m *MockContactRepo) FindByEmail(ctx context.Context, userID uuid.UUID, email string) (*domain.Contact, error) {
	args := m.Called(ctx, userID, email)
	if v := args.Get(0); v != nil {
		return v.(*domain.Contact), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockContactRepo) FindBySourceID(ctx context.Context, userID uuid.UUID, source, sourceID string) (*domain.Contact, error) {
	args := m.Called(ctx, userID, source, sourceID)
	if v := args.Get(0); v != nil {
		return v.(*domain.Contact), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *MockContactRepo) Search(ctx context.Context, userID uuid.UUID, query string, offset, limit int) ([]*domain.Contact, int, error) {
	args := m.Called(ctx, userID, query, offset, limit)
	return args.Get(0).([]*domain.Contact), args.Int(1), args.Error(2)
}
func (m *MockContactRepo) SearchWithFilter(ctx context.Context, userID uuid.UUID, orgIDs []uuid.UUID, filter map[string]interface{}, pagination *baseRepo.PaginationParams) ([]*domain.Contact, int, error) {
	args := m.Called(ctx, userID, orgIDs, filter, pagination)
	return args.Get(0).([]*domain.Contact), args.Int(1), args.Error(2)
}
func (m *MockContactRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockContactRepo) UpsertMany(ctx context.Context, contacts []*domain.Contact) error {
	args := m.Called(ctx, contacts)
	return args.Error(0)
}

func TestMockContactRepo_CreateAndFind(t *testing.T) {
	repo := &MockContactRepo{}
	ctx := context.Background()
	user := uuid.New()
	contact := &domain.Contact{UserID: user, Profile: base.JSONB(`{"email":"john@example.com"}`)}

	repo.On("Create", ctx, contact).Return(nil)
	assert.NoError(t, repo.Create(ctx, contact))
	repo.AssertCalled(t, "Create", ctx, contact)

	repo.On("FindByEmail", ctx, user, "john@example.com").Return(contact, nil)
	found, err := repo.FindByEmail(ctx, user, "john@example.com")
	assert.NoError(t, err)
	assert.Equal(t, contact, found)
}

func TestMockContactRepo_SearchWithFilter(t *testing.T) {
	repo := &MockContactRepo{}
	ctx := context.Background()
	user := uuid.New()
	orgs := []uuid.UUID{uuid.New()}
	pagination := baseRepo.PaginationParams{Limit: 10, Offset: 0}
	results := []*domain.Contact{
		{UserID: user, Profile: base.JSONB(`{"email":"a@example.com"}`)},
		{UserID: user, Profile: base.JSONB(`{"email":"b@example.com"}`)},
	}

	repo.On("SearchWithFilter", ctx, user, orgs, mock.Anything, &pagination).Return(results, len(results), nil)
	list, total, err := repo.SearchWithFilter(ctx, user, orgs, map[string]interface{}{"email": "a"}, &pagination)

	assert.NoError(t, err)
	assert.Equal(t, len(results), total)
	assert.Equal(t, results, list)
	repo.AssertExpectations(t)
}

func TestNormalizeContactFilter(t *testing.T) {
	filter := map[string]interface{}{
		"name":    "Ann",
		"city":    "",
		"company": "Acme",
		"tags":    []string{"vip", "new"},
		"$or":     []interface{}{"keep"},
	}

	normalized := NormalizeContactFilter(filter)

	andBlock, ok := normalized["$and"].([]interface{})
	require.True(t, ok, "expected $and block for fuzzy filters")
	require.Len(t, andBlock, 2)

	var baseBlock map[string]interface{}
	var fuzzyBlock map[string]interface{}
	for _, item := range andBlock {
		m, ok := asFilterMap(item)
		if !ok {
			continue
		}
		if _, hasName := m["name"]; hasName {
			baseBlock = m
			continue
		}
		if _, hasOr := m["$or"]; hasOr {
			fuzzyBlock = m
		}
	}

	require.NotNil(t, baseBlock, "expected base block in $and")
	assert.Equal(t, map[string]interface{}{"$ilike": "%Ann%"}, baseBlock["name"])
	assert.Equal(t, map[string]interface{}{"$in": []interface{}{"vip", "new"}}, baseBlock["tags"])
	assert.Equal(t, []interface{}{"keep"}, baseBlock["$or"])
	_, exists := baseBlock["city"]
	assert.False(t, exists, "empty string filter should be omitted")

	require.NotNil(t, fuzzyBlock, "expected fuzzy OR block")
	orList, ok := fuzzyBlock["$or"].([]interface{})
	require.True(t, ok, "expected $or list in fuzzy block")
	require.Len(t, orList, 2)

	var companyFilter map[string]interface{}
	for _, item := range orList {
		m, ok := asFilterMap(item)
		if !ok {
			continue
		}
		if c, ok := m["company"].(map[string]interface{}); ok {
			companyFilter = c
			break
		}
	}
	require.NotNil(t, companyFilter, "expected company filter in fuzzy block")
	assert.Equal(t, "%Acme%", companyFilter["$ilike"])
}

func asFilterMap(value interface{}) (map[string]interface{}, bool) {
	switch v := value.(type) {
	case map[string]interface{}:
		return v, true
	case baseRepo.Filter:
		return map[string]interface{}(v), true
	default:
		return nil, false
	}
}
