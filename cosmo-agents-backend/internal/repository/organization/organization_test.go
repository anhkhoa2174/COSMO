package organization

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
)

func newTestRepo(t *testing.T) (*OrganizationRepository, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	db = db.Session(&gorm.Session{AllowGlobalUpdate: true})
	require.NoError(t, db.AutoMigrate(&domain.Organization{}, &domain.User{}, &domain.Role{}))
	return NewOrganizationRepository(db), db
}

func createUser(t *testing.T, db *gorm.DB, email string) domain.User {
	user := domain.User{
		Email:       email,
		Name:        strings.Split(email, "@")[0],
		PhoneNumber: pq.StringArray{"123"},
	}
	require.NoError(t, db.Create(&user).Error)
	return user
}

func createOrg(t *testing.T, db *gorm.DB, userID uuid.UUID, name string) domain.Organization {
	org := domain.Organization{
		UserID:                  &userID,
		Name:                    name,
		CompanyDescription:      "desc",
		ValueOffering:           "value",
		CompanyTargetingPersona: pq.StringArray{"buyer"},
	}
	require.NoError(t, db.Create(&org).Error)
	return org
}

func createRole(t *testing.T, db *gorm.DB, userID, orgID uuid.UUID, name domain.RoleName) domain.Role {
	role := domain.Role{
		UserID:         userID,
		OrganizationID: orgID,
		Name:           name,
		Status:         domain.RoleStatusActive,
	}
	require.NoError(t, db.Create(&role).Error)
	return role
}

func TestOrganizationRepository_CRUDAndFinds(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()

	user := createUser(t, db, "owner@example.com")
	otherUser := createUser(t, db, "other@example.com")

	org := createOrg(t, db, user.ID, "Org One")

	found, err := repo.FindByID(ctx, org.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, org.ID, found.ID)

	foundByUser, err := repo.FindByIDAndUserID(ctx, org.ID, user.ID)
	require.NoError(t, err)
	require.NotNil(t, foundByUser)
	assert.Equal(t, org.ID, foundByUser.ID)

	none, err := repo.FindByIDAndUserID(ctx, org.ID, otherUser.ID)
	require.NoError(t, err)
	assert.Nil(t, none)

	org.CompanyDescription = "updated"
	org.CompanyTargetingPersona = pq.StringArray{"cto", "vp"}
	require.NoError(t, repo.Update(ctx, org.ID, &org))

	updated, err := repo.FindByID(ctx, org.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "updated", updated.CompanyDescription)
	assert.ElementsMatch(t, []string{"cto", "vp"}, []string(updated.CompanyTargetingPersona))

	require.NoError(t, repo.Delete(ctx, org.ID))
	deleted, err := repo.FindByID(ctx, org.ID)
	require.NoError(t, err)
	assert.Nil(t, deleted)
}

func TestOrganizationRepository_FindByUserAndPagination(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()
	user := createUser(t, db, "user@example.com")
	other := createUser(t, db, "other@example.com")

	org1 := createOrg(t, db, user.ID, "Org One")
	org2 := createOrg(t, db, user.ID, "Org Two")
	createOrg(t, db, other.ID, "Other Org")

	require.NoError(t, db.Model(&domain.Organization{}).Where("id = ?", org2.ID).Update("updated_at", time.Now().Add(time.Minute)).Error)

	orgs, err := repo.FindByUserID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, orgs, 2)

	filter := baseRepo.Filter{
		"user_id": map[string]interface{}{string(baseRepo.OpEqual): user.ID},
	}
	pagination := &baseRepo.PaginationParams{Limit: 1, Offset: 1}
	result, err := repo.FindAll(ctx, filter, pagination)
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
	require.Len(t, result.List, 1)
	assert.Equal(t, org1.ID, result.List[0].ID)
}

func TestOrganizationRepository_FindFirstAndMain(t *testing.T) {
	t.Run("returns_admin_org", func(t *testing.T) {
		repo, db := newTestRepo(t)
		ctx := context.Background()

		user := createUser(t, db, "admin@example.com")
		org := createOrg(t, db, user.ID, "Admin Org")
		createRole(t, db, user.ID, org.ID, domain.RoleNameAdmin)

		mainOrg, err := repo.FindUserMainOrganization(ctx, user.ID)
		require.NoError(t, err)
		require.NotNil(t, mainOrg)
		assert.Equal(t, org.ID, mainOrg.ID)
	})

	t.Run("falls_back_to_first_org", func(t *testing.T) {
		repo, db := newTestRepo(t)
		ctx := context.Background()

		user := createUser(t, db, "fallback@example.com")
		org := createOrg(t, db, user.ID, "First Org")

		mainOrg, err := repo.FindUserMainOrganization(ctx, user.ID)
		require.NoError(t, err)
		require.NotNil(t, mainOrg)
		assert.Equal(t, org.ID, mainOrg.ID)
	})
}

func TestOrganizationRepository_GetManyMembersWithTotal(t *testing.T) {
	t.Run("by_explicit_org_id", func(t *testing.T) {
		repo, db := newTestRepo(t)
		ctx := context.Background()

		admin := createUser(t, db, "admin@example.com")
		member := createUser(t, db, "member@example.com")
		org := createOrg(t, db, admin.ID, "Team Org")
		org.CompanyTargetingPersona = pq.StringArray{"persona1", "persona2"}
		require.NoError(t, db.Save(&org).Error)

		createRole(t, db, admin.ID, org.ID, domain.RoleNameAdmin)
		createRole(t, db, member.ID, org.ID, domain.RoleNameMember)

		list, total, err := repo.GetManyMembersWithTotal(ctx, admin.ID, org.ID.String(), 0, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, list, 2)

		idSet := map[uuid.UUID]bool{}
		for _, item := range list {
			idSet[item.Entity.ID] = true
			assert.Equal(t, org.ID, item.Role.Organization.ID)
			assert.NotNil(t, item.Role.Organization.CompanyTargetingPersona)
		}
		assert.True(t, idSet[admin.ID])
		assert.True(t, idSet[member.ID])
	})

	t.Run("by_me_id_and_missing_role_errors", func(t *testing.T) {
		repo, _ := newTestRepo(t)
		ctx := context.Background()
		_, total, err := repo.GetManyMembersWithTotal(ctx, uuid.New(), "me", 0, 5)
		require.Error(t, err)
		assert.Equal(t, int64(0), total)
	})

	t.Run("invalid_uuid_errors", func(t *testing.T) {
		repo, db := newTestRepo(t)
		ctx := context.Background()
		admin := createUser(t, db, "admin2@example.com")
		org := createOrg(t, db, admin.ID, "Org")
		createRole(t, db, admin.ID, org.ID, domain.RoleNameAdmin)

		_, _, err := repo.GetManyMembersWithTotal(ctx, admin.ID, "not-a-uuid", 0, 1)
		require.Error(t, err)
	})
}

func TestOrganizationRepository_SoftDeleteManyByUserID(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()

	user := createUser(t, db, "deleter@example.com")
	org := createOrg(t, db, user.ID, "Org")

	memberA := createUser(t, db, "a@example.com")
	memberB := createUser(t, db, "b@example.com")
	otherOrg := createOrg(t, db, user.ID, "Other Org")

	roleA := createRole(t, db, memberA.ID, org.ID, domain.RoleNameMember)
	roleB := createRole(t, db, memberB.ID, org.ID, domain.RoleNameMember)
	_ = createRole(t, db, memberA.ID, otherOrg.ID, domain.RoleNameMember)

	err := repo.SoftDeleteManyByUserID(ctx, org.ID, []uuid.UUID{memberA.ID, memberB.ID})
	require.NoError(t, err)

	var roles []domain.Role
	require.NoError(t, db.Find(&roles).Error)

	roleMap := map[uuid.UUID]bool{}
	for _, r := range roles {
		roleMap[r.ID] = r.IsDeleted
	}

	assert.True(t, roleMap[roleA.ID])
	assert.True(t, roleMap[roleB.ID])
	for id, deleted := range roleMap {
		if id != roleA.ID && id != roleB.ID {
			assert.False(t, deleted)
		}
	}
}
