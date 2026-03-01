package organization

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/role"
	orgRepo "github.com/rockship/cosmo-agents-go/internal/repository/organization"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
)

func newService(t *testing.T) (*Service, *orgRepo.OrganizationRepository, *userRepo.UserRepository, *roleRepo.RoleRepository, *gorm.DB, uuid.UUID) {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.User{}, &domain.Organization{}, &domain.Role{}))

	uRepo := userRepo.NewUserRepository(db)
	oRepo := orgRepo.NewOrganizationRepository(db)
	rRepo := roleRepo.NewRoleRepository(db)

	userID := uuid.New()
	_, err = uRepo.Create(context.Background(), &domain.User{
		Base:  domain.Base{ID: userID},
		Email: "owner@example.com",
		Name:  "Owner",
	})
	require.NoError(t, err)

	svc := NewService(oRepo, uRepo, rRepo, db)
	return svc, oRepo, uRepo, rRepo, db, userID
}

func TestCreateAndGetOrganization(t *testing.T) {
	svc, oRepo, _, _, _, userID := newService(t)
	ctx := context.Background()

	org, err := svc.CreateOrganization(ctx, CreateOrganizationRequest{
		UserID:                  userID,
		Name:                    "Org",
		CompanyURL:              "https://example.com",
		CompanyDescription:      "desc",
		CompanyTargetingPersona: []string{"a", "b"},
		ValueOffering:           "value",
	})
	require.NoError(t, err)
	require.NotNil(t, org)
	assert.Equal(t, "Org", org.Name)

	found, err := svc.GetOrganizationByID(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, org.ID, found.ID)

	_, err = svc.CreateOrganization(ctx, CreateOrganizationRequest{UserID: uuid.New(), Name: "X"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")

	missing, err := svc.GetOrganizationByID(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, missing)

	list, err := svc.GetOrganizationsByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	paged, total, err := svc.ListOrganizations(ctx, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, paged, 1)

	// delete then ensure filtered out
	require.NoError(t, svc.DeleteOrganization(ctx, org.ID))
	none, err := oRepo.FindByID(ctx, org.ID)
	require.NoError(t, err)
	assert.Nil(t, none)
	require.NoError(t, svc.DeleteOrganization(ctx, uuid.New()))
}

func TestUpdateOrganization(t *testing.T) {
	svc, _, _, _, _, userID := newService(t)
	ctx := context.Background()

	org, err := svc.CreateOrganization(ctx, CreateOrganizationRequest{
		UserID: userID,
		Name:   "Old",
	})
	require.NoError(t, err)

	updated, err := svc.UpdateOrganization(ctx, org.ID, UpdateOrganizationRequest{
		Name:                    "New",
		CompanyDescription:      "desc",
		CompanyTargetingPersona: []string{"x"},
	})
	require.NoError(t, err)
	assert.Equal(t, "New", updated.Name)
	assert.Equal(t, "desc", updated.CompanyDescription)
	assert.ElementsMatch(t, []string{"x"}, []string(updated.CompanyTargetingPersona))

}

func TestRolesAddRemoveAndUsers(t *testing.T) {
	svc, _, uRepo, rRepo, db, ownerID := newService(t)
	ctx := context.Background()

	org, err := svc.CreateOrganization(ctx, CreateOrganizationRequest{UserID: ownerID, Name: "Org"})
	require.NoError(t, err)

	memberID := uuid.New()
	_, err = uRepo.Create(ctx, &domain.User{
		Base:  domain.Base{ID: memberID},
		Email: "member@example.com",
		Name:  "Member",
	})
	require.NoError(t, err)

	roleCreated, err := svc.AddUserToOrganization(ctx, org.ID, memberID, domain.RoleNameMember)
	require.NoError(t, err)
	assert.Equal(t, domain.RoleStatusPending, roleCreated.Status)

	_, err = svc.AddUserToOrganization(ctx, org.ID, memberID, domain.RoleNameMember)
	assert.Error(t, err)

	users, err := svc.GetOrganizationUsers(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, memberID, users[0].ID)

	require.NoError(t, svc.RemoveUserFromOrganization(ctx, org.ID, memberID))
	roleAfter, err := rRepo.FindByUserAndOrganization(ctx, memberID, org.ID)
	require.NoError(t, err)
	assert.Nil(t, roleAfter)

	userOrgs, err := svc.GetUserOrganizations(ctx, memberID)
	require.NoError(t, err)
	assert.Len(t, userOrgs, 0)

	// create another org and role to test GetUserOrganizations returns it
	org2 := &domain.Organization{
		Base:   domain.Base{ID: uuid.New()},
		UserID: &ownerID,
		Name:   "Second",
	}
	require.NoError(t, db.Create(org2).Error)
	_, err = rRepo.Create(ctx, &domain.Role{
		UserID:         memberID,
		OrganizationID: org2.ID,
		Name:           role.RoleNameMember,
		Status:         role.RoleStatusActive,
	})
	require.NoError(t, err)

	userOrgs, err = svc.GetUserOrganizations(ctx, memberID)
	require.NoError(t, err)
	assert.Len(t, userOrgs, 1)
	assert.Equal(t, org2.ID, userOrgs[0].ID)
}
