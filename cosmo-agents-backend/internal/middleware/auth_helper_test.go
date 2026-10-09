package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
)

type helperFixture struct {
	db     *gorm.DB
	helper *AuthHelper
}

func newHelperFixture(t *testing.T) *helperFixture {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.User{}, &domain.Organization{}, &domain.Role{}))
	return &helperFixture{
		db:     db,
		helper: NewAuthHelper(userRepo.NewUserRepository(db), roleRepo.NewRoleRepository(db)),
	}
}

func (f *helperFixture) user(t *testing.T, email string) *domain.User {
	t.Helper()
	u := &domain.User{Email: email, Name: email}
	require.NoError(t, f.db.Create(u).Error)
	return u
}

func (f *helperFixture) org(t *testing.T, name string, creator *uuid.UUID) *domain.Organization {
	t.Helper()
	o := &domain.Organization{Name: name, UserID: creator}
	require.NoError(t, f.db.Create(o).Error)
	return o
}

func (f *helperFixture) role(t *testing.T, userID, orgID uuid.UUID, name domain.RoleName) *domain.Role {
	t.Helper()
	r := &domain.Role{UserID: userID, OrganizationID: orgID, Name: name, Status: domain.RoleStatusActive}
	require.NoError(t, f.db.Create(r).Error)
	return r
}

// resolve runs GetUserAndOrganization inside a real request, with user_id set
// the way AuthMiddleware sets it.
func (f *helperFixture) resolve(t *testing.T, userID interface{}) (*domain.User, uuid.UUID, error) {
	t.Helper()
	var (
		gotUser *domain.User
		gotOrg  uuid.UUID
		gotErr  error
	)
	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		if userID != nil {
			c.Locals("user_id", userID)
		}
		gotUser, gotOrg, gotErr = f.helper.GetUserAndOrganization(c)
		return c.SendStatus(fiber.StatusNoContent)
	})
	_, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	require.NoError(t, err)
	return gotUser, gotOrg, gotErr
}

func TestAuthHelper_GetUserAndOrganization(t *testing.T) {
	f := newHelperFixture(t)

	admin := f.user(t, "admin@a.test")
	member := f.user(t, "member@a.test")
	creator := f.user(t, "creator@b.test")
	loner := f.user(t, "loner@nowhere.test")

	orgA := f.org(t, "Org A", nil)
	orgB := f.org(t, "Org B", &creator.ID)
	f.role(t, admin.ID, orgA.ID, domain.RoleNameAdmin)
	f.role(t, member.ID, orgA.ID, domain.RoleNameMember)

	tests := []struct {
		name    string
		userID  interface{}
		wantOrg uuid.UUID
		wantErr error
	}{
		{"admin resolves to the org they administer", admin.ID, orgA.ID, nil},
		{"member resolves through their role", member.ID, orgA.ID, nil},
		{"creator with no role resolves to the org they created", creator.ID, orgB.ID, nil},
		{"no user_id in context", nil, uuid.Nil, ErrUnauthorized},
		{"user_id of the wrong type", admin.ID.String(), uuid.Nil, ErrInvalidUserID},
		{"unknown user", uuid.New(), uuid.Nil, ErrUserNotFound},
		// A user who belongs to no organisation must get an auth error the
		// response helper maps to 401, not an ad-hoc error that becomes 500.
		{"user with no organisation", loner.ID, uuid.Nil, ErrNoRolesFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, org, err := f.resolve(t, tt.userID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, u)
				assert.Equal(t, uuid.Nil, org)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, u)
			assert.Equal(t, tt.userID, u.ID)
			assert.Equal(t, tt.wantOrg, org)
		})
	}
}

// A member removed from an organisation keeps a soft-deleted role row. That row
// must not keep resolving to the organisation, or removal does not revoke
// access to anything the v1 contact endpoints scope by organisation.
func TestAuthHelper_RemovedMemberNoLongerResolves(t *testing.T) {
	f := newHelperFixture(t)
	removed := f.user(t, "removed@a.test")
	orgA := f.org(t, "Org A", nil)
	r := f.role(t, removed.ID, orgA.ID, domain.RoleNameMember)
	require.NoError(t, f.db.Model(r).Update("is_deleted", true).Error)

	_, org, err := f.resolve(t, removed.ID)
	assert.ErrorIs(t, err, ErrNoRolesFound)
	assert.Equal(t, uuid.Nil, org, "removed member must not resolve to %s", orgA.ID)
}

// The admin lookup behind FindUserMainOrganization joins roles without looking
// at is_deleted, so a removed admin is the harder case of the same bug.
func TestAuthHelper_RemovedAdminNoLongerResolves(t *testing.T) {
	f := newHelperFixture(t)
	removed := f.user(t, "removed-admin@a.test")
	orgA := f.org(t, "Org A", nil)
	r := f.role(t, removed.ID, orgA.ID, domain.RoleNameAdmin)
	require.NoError(t, f.db.Model(r).Update("is_deleted", true).Error)

	_, org, err := f.resolve(t, removed.ID)
	assert.ErrorIs(t, err, ErrNoRolesFound)
	assert.Equal(t, uuid.Nil, org)

	// Membership elsewhere still resolves, to the live organisation.
	orgB := f.org(t, "Org B", nil)
	f.role(t, removed.ID, orgB.ID, domain.RoleNameMember)
	_, org, err = f.resolve(t, removed.ID)
	require.NoError(t, err)
	assert.Equal(t, orgB.ID, org)
}

func TestAuthHelper_GetUserIDAndRoles(t *testing.T) {
	f := newHelperFixture(t)
	u := f.user(t, "u@a.test")
	orgA := f.org(t, "A", nil)
	f.role(t, u.ID, orgA.ID, domain.RoleNameMember)

	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		if c.Query("anon") == "" {
			c.Locals("user_id", u.ID)
		}
		id, roles, err := f.helper.GetUserIDAndRoles(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
		}
		if id != u.ID || len(roles) != 1 || roles[0].OrganizationID != orgA.ID {
			return c.Status(fiber.StatusTeapot).SendString("unexpected")
		}
		orgID, err := f.helper.GetOrganizationID(c)
		if err != nil || orgID != orgA.ID {
			return c.Status(fiber.StatusTeapot).SendString("org mismatch")
		}
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	resp, err = app.Test(httptest.NewRequest(http.MethodGet, "/?anon=1", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAuthHelper_IsAdminInOrganization(t *testing.T) {
	f := newHelperFixture(t)
	ctx := context.Background()
	admin := f.user(t, "admin@a.test")
	member := f.user(t, "member@a.test")
	orgA := f.org(t, "A", nil)
	orgB := f.org(t, "B", nil)
	f.role(t, admin.ID, orgA.ID, domain.RoleNameAdmin)
	f.role(t, member.ID, orgA.ID, domain.RoleNameMember)
	removedAdmin := f.user(t, "removed@a.test")
	r := f.role(t, removedAdmin.ID, orgA.ID, domain.RoleNameAdmin)
	require.NoError(t, f.db.Model(r).Update("is_deleted", true).Error)

	assert.True(t, f.helper.IsAdminInOrganization(ctx, admin.ID, orgA.ID))
	assert.False(t, f.helper.IsAdminInOrganization(ctx, member.ID, orgA.ID))
	assert.False(t, f.helper.IsAdminInOrganization(ctx, admin.ID, orgB.ID), "admin of A is not admin of B")
	assert.False(t, f.helper.IsAdminInOrganization(ctx, removedAdmin.ID, orgA.ID))
	assert.False(t, f.helper.IsAdminInOrganization(ctx, uuid.New(), orgA.ID))
}

func TestAuthHelper_DatabaseErrorsPropagate(t *testing.T) {
	db, err := pgtest.Open(t, nil) // no tables at all
	require.NoError(t, err)
	h := NewAuthHelper(userRepo.NewUserRepository(db), roleRepo.NewRoleRepository(db))

	app := fiber.New()
	var gotErr error
	app.Get("/", func(c fiber.Ctx) error {
		c.Locals("user_id", uuid.New())
		_, _, gotErr = h.GetUserAndOrganization(c)
		return nil
	})
	_, err = app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	require.NoError(t, err)
	require.Error(t, gotErr)
	for _, sentinel := range []error{ErrUnauthorized, ErrUserNotFound, ErrNoRolesFound, ErrInvalidUserID} {
		assert.False(t, errors.Is(gotErr, sentinel), "a database failure must not look like %v", sentinel)
	}
}
