package helper

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserWithRelationsRepository_GetUserWithOrganizations(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserWithRelationsRepository(db)
	ctx := context.Background()

	user := &domain.User{
		Email: "relations-org@example.com",
		Name:  "Org User",
	}
	require.NoError(t, db.Create(user).Error)

	orgs := []domain.Organization{
		{UserID: &user.ID, Name: "Org 1"},
		{UserID: &user.ID, Name: "Org 2"},
	}
	require.NoError(t, db.Create(&orgs).Error)

	result, err := repo.GetUserWithOrganizations(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, user.ID, result.ID)
	assert.Len(t, result.Organizations, 2)

	notFound, err := repo.GetUserWithOrganizations(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestUserWithRelationsRepository_GetUserWithRoles(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserWithRelationsRepository(db)
	ctx := context.Background()

	user := &domain.User{
		Email: "relations-role@example.com",
		Name:  "Role User",
	}
	require.NoError(t, db.Create(user).Error)

	org := &domain.Organization{
		UserID: &user.ID,
		Name:   "Role Org",
	}
	require.NoError(t, db.Create(org).Error)

	role := &domain.Role{
		UserID:         user.ID,
		OrganizationID: org.ID,
		Name:           domain.RoleNameAdmin,
	}
	require.NoError(t, db.Create(role).Error)

	result, err := repo.GetUserWithRoles(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, user.ID, result.ID)
	require.Len(t, result.Roles, 1)
	assert.Equal(t, org.ID, result.Roles[0].OrganizationID)

	notFound, err := repo.GetUserWithRoles(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestUserWithRelationsRepository_GetUserWithFullRelations(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserWithRelationsRepository(db)
	ctx := context.Background()

	user := &domain.User{
		Email: "relations-full@example.com",
		Name:  "Full User",
	}
	require.NoError(t, db.Create(user).Error)

	org := &domain.Organization{
		UserID: &user.ID,
		Name:   "Full Org",
	}
	require.NoError(t, db.Create(org).Error)

	role := &domain.Role{
		UserID:         user.ID,
		OrganizationID: org.ID,
		Name:           domain.RoleNameMember,
	}
	require.NoError(t, db.Create(role).Error)

	campaign := &domain.Campaign{
		UserID: user.ID,
		Name:   "Full Campaign",
		Status: domain.CampaignStatusActive,
	}
	require.NoError(t, db.Create(campaign).Error)

	notification := &domain.Notification{
		UserID:     user.ID,
		CampaignID: campaign.ID,
	}
	require.NoError(t, db.Create(notification).Error)

	agent := &domain.Agent{
		UserID: user.ID,
		Name:   "Full Agent",
		Email:  "agent@example.com",
		Status: domain.AgentStatusActive,
	}
	require.NoError(t, db.Create(agent).Error)

	result, err := repo.GetUserWithFullRelations(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, user.ID, result.ID)
	require.Len(t, result.Organizations, 1)
	require.Len(t, result.Roles, 1)
	require.Len(t, result.Notifications, 1)
	require.Len(t, result.Agents, 1)
	assert.Equal(t, notification.CampaignID, result.Notifications[0].CampaignID)

	notFound, err := repo.GetUserWithFullRelations(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestUserWithRelationsRepository_GetUsersWithOrganizations(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserWithRelationsRepository(db)
	ctx := context.Background()

	empty, err := repo.GetUsersWithOrganizations(ctx, []uuid.UUID{})
	require.NoError(t, err)
	assert.Empty(t, empty)

	first := &domain.User{Email: "multi-org-1@example.com", Name: "First"}
	second := &domain.User{Email: "multi-org-2@example.com", Name: "Second"}
	require.NoError(t, db.Create(first).Error)
	require.NoError(t, db.Create(second).Error)

	orgs := []domain.Organization{
		{UserID: &first.ID, Name: "Org A"},
		{UserID: &second.ID, Name: "Org B"},
	}
	require.NoError(t, db.Create(&orgs).Error)

	results, err := repo.GetUsersWithOrganizations(ctx, []uuid.UUID{first.ID, second.ID})
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestUserWithRelationsRepository_GetUsersWithRoles(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserWithRelationsRepository(db)
	ctx := context.Background()

	empty, err := repo.GetUsersWithRoles(ctx, []uuid.UUID{})
	require.NoError(t, err)
	assert.Empty(t, empty)

	first := &domain.User{Email: "multi-role-1@example.com", Name: "Role First"}
	second := &domain.User{Email: "multi-role-2@example.com", Name: "Role Second"}
	require.NoError(t, db.Create(first).Error)
	require.NoError(t, db.Create(second).Error)

	org := &domain.Organization{Name: "Role Org"}
	require.NoError(t, db.Create(org).Error)

	roles := []domain.Role{
		{UserID: first.ID, OrganizationID: org.ID, Name: domain.RoleNameAdmin},
		{UserID: second.ID, OrganizationID: org.ID, Name: domain.RoleNameMember},
	}
	require.NoError(t, db.Create(&roles).Error)

	results, err := repo.GetUsersWithRoles(ctx, []uuid.UUID{first.ID, second.ID})
	require.NoError(t, err)
	assert.Len(t, results, 2)
}
