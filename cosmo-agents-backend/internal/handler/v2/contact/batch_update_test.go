package contact

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	customFieldRepo "github.com/rockship/cosmo-agents-go/internal/repository/custom_field"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
)

// POST /v2/contacts/batch looks contacts up by email. The email column is
// gone (migration 000041), so the lookup must read the profile.
func TestBatchUpdate_HTTP_ByEmail(t *testing.T) {
	db, err := pgtest.Open(t, &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.User{}, &domain.Organization{}, &domain.Role{}, &domain.Contact{}, &domain.CustomField{}))

	user := &domain.User{Email: "rep@alpha.test", Name: "Rep"}
	require.NoError(t, db.Create(user).Error)
	org := &domain.Organization{Name: "alpha", CompanyURL: "https://alpha.test"}
	require.NoError(t, db.Create(org).Error)
	require.NoError(t, db.Create(&domain.Role{UserID: user.ID, OrganizationID: org.ID, Name: domain.RoleNameAdmin, Status: domain.RoleStatusActive}).Error)
	orgID := org.ID
	target := &domain.Contact{UserID: user.ID, OrganizationID: &orgID, Name: "Ann", Company: "Old", Source: "csv", SourceID: uuid.NewString(),
		Profile: base.JSONB(`{"email":"Ann@Example.test"}`)}
	other := &domain.Contact{UserID: user.ID, OrganizationID: &orgID, Name: "Bo", Company: "Old", Source: "csv", SourceID: uuid.NewString(),
		Profile: base.JSONB(`{"email":"bo@example.test"}`)}
	require.NoError(t, db.Create(target).Error)
	require.NoError(t, db.Create(other).Error)

	h := New(contactRepo.NewContactRepository(db), userRepo.NewUserRepository(db), roleRepo.NewRoleRepository(db),
		contactRepo.NewListContactRepository(db), customFieldRepo.NewCustomFieldRepository(db), nil, nil, nil, nil)
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user_id", user.ID)
		return c.Next()
	})
	app.Post("/v2/contacts/batch", h.BatchUpdate)

	raw, _ := json.Marshal(map[string]interface{}{
		"emails": []string{"ann@example.test"},
		"fields": map[string]interface{}{"company": "New"},
	})
	req := httptest.NewRequest(http.MethodPost, "/v2/contacts/batch", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	require.Equal(t, fiber.StatusOK, resp.StatusCode, "body: %s", body)

	var got domain.Contact
	require.NoError(t, db.First(&got, "id = ?", target.ID).Error)
	assert.Equal(t, "New", got.Company)
	var untouched domain.Contact
	require.NoError(t, db.First(&untouched, "id = ?", other.ID).Error)
	assert.Equal(t, "Old", untouched.Company, "only the named email is touched")

	// Ownership is not a batch-editable field: moving contacts to another
	// user or organisation would hand them over.
	for _, field := range []string{"user_id", "organization_id"} {
		raw, _ := json.Marshal(map[string]interface{}{
			"emails": []string{"ann@example.test"},
			"fields": map[string]interface{}{field: uuid.NewString()},
		})
		req := httptest.NewRequest(http.MethodPost, "/v2/contacts/batch", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode, field)
	}
	require.NoError(t, db.First(&got, "id = ?", target.ID).Error)
	assert.Equal(t, user.ID, got.UserID)
	require.NotNil(t, got.OrganizationID)
	assert.Equal(t, org.ID, *got.OrganizationID)
}
