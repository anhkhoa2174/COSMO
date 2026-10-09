package customfield

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
)

// createStatus posts body to Create for a user in orgID whose org already
// holds existing, and returns the status plus the repo mock.
func createStatus(t *testing.T, body string, existing []domain.CustomField) (int, *MockCustomFieldRepository) {
	t.Helper()
	userID, orgID := uuid.New(), uuid.New()
	repo := new(MockCustomFieldRepository)
	roles := new(MockRoleRepository)
	roles.On("FindByUserID", mock.Anything, userID).Return([]domain.Role{{OrganizationID: orgID, Status: "active"}}, nil)
	repo.On("FindAll", mock.Anything, mock.Anything, mock.Anything).Return(&baseRepo.PaginatedResult[domain.CustomField]{List: existing}, nil)
	repo.On("Create", mock.Anything, mock.Anything).Return(&domain.CustomField{}, nil)

	h := NewHandler(repo, new(MockUserRepository), roles)
	app := fiber.New()
	app.Post("/custom-fields", func(c fiber.Ctx) error {
		c.Locals("user_id", userID)
		return h.Create(c)
	})
	req := httptest.NewRequest("POST", "/custom-fields", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	return resp.StatusCode, repo
}

func TestCreateCustomField_ValidatesDefinition(t *testing.T) {
	cases := []struct {
		name   string
		body   string
		status int
	}{
		{"unknown data_type is a client error", `{"name":"Budget","data_type":"bogus","entity_type":"contact"}`, 400},
		{"unknown entity_type is a client error", `{"name":"Budget","data_type":"text","entity_type":"deal"}`, 400},
		{"select without options", `{"name":"Tier","data_type":"select","entity_type":"contact"}`, 400},
		{"select with blank options only", `{"name":"Tier","data_type":"select","entity_type":"contact","options":[" "]}`, 400},
		{"fallback not a number", `{"name":"Budget","data_type":"number","entity_type":"contact","fallback_value":"abc"}`, 400},
		{"sample not among options", `{"name":"Tier","data_type":"select","entity_type":"contact","options":["Gold"],"sample_data":"Bronze"}`, 400},
		{"sample not a url", `{"name":"Site","data_type":"url","entity_type":"contact","sample_data":"not a url"}`, 400},
		{"name collides with standard contact column", `{"name":"Company","data_type":"text","entity_type":"contact"}`, 400},
		{"blank sample and fallback are ignored", `{"name":"Budget","data_type":"number","entity_type":"contact","options":[],"sample_data":"","fallback_value":""}`, 201},
		{"valid select", `{"name":"Tier","data_type":"select","entity_type":"contact","options":["Gold","Silver"],"fallback_value":"Gold"}`, 201},
		{"standard column name is fine for a company field", `{"name":"Company","data_type":"text","entity_type":"company"}`, 201},
		{"website is not a contact column", `{"name":"Website","data_type":"url","entity_type":"contact"}`, 201},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, repo := createStatus(t, tc.body, nil)
			assert.Equal(t, tc.status, status)
			if tc.status != 201 {
				repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestCreateCustomField_DropsOptionsForNonSelect(t *testing.T) {
	status, repo := createStatus(t, `{"name":"Budget","data_type":"number","entity_type":"contact","options":["left","over"]}`, nil)
	assert.Equal(t, 201, status)
	repo.AssertCalled(t, "Create", mock.Anything, mock.MatchedBy(func(cf *domain.CustomField) bool {
		return len(cf.Options) == 0
	}))
}

func TestCreateCustomField_RejectsDuplicateKey(t *testing.T) {
	existing := []domain.CustomField{{
		Base: domain.Base{ID: uuid.New()}, Name: "Budget", NormalizedName: "budget",
		DataType: domain.CustomFieldDataTypeNumber, EntityType: domain.CustomFieldEntityContact,
	}}
	// Different spelling, same storage key: both would write profile.custom_fields.budget.
	status, repo := createStatus(t, `{"name":"  budget ","data_type":"text","entity_type":"contact"}`, existing)
	assert.Equal(t, 409, status)
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)

	// A renamed field keeps its old key, and its new display name is taken too.
	renamed := []domain.CustomField{{
		Base: domain.Base{ID: uuid.New()}, Name: "Annual Budget", NormalizedName: "budget",
		DataType: domain.CustomFieldDataTypeNumber, EntityType: domain.CustomFieldEntityContact,
	}}
	for _, name := range []string{"Budget", "annual budget"} {
		status, _ = createStatus(t, fmt.Sprintf(`{"name":%q,"data_type":"text","entity_type":"contact"}`, name), renamed)
		assert.Equal(t, 409, status, name)
	}
}

// updateStatus patches the field current with body and returns the status and
// the repo mock; others are the remaining fields in the same org.
func updateStatus(t *testing.T, current domain.CustomField, others []domain.CustomField, body string) (int, *MockCustomFieldRepository) {
	t.Helper()
	repo := new(MockCustomFieldRepository)
	repo.On("FindByID", mock.Anything, current.ID).Return(&current, nil)
	repo.On("FindAll", mock.Anything, mock.Anything, mock.Anything).Return(&baseRepo.PaginatedResult[domain.CustomField]{List: append(others, current)}, nil)
	repo.On("UpdateFields", mock.Anything, current.ID, mock.Anything).Return(nil)

	h := NewHandler(repo, new(MockUserRepository), new(MockRoleRepository))
	app := fiber.New()
	app.Patch("/custom-fields/:id", func(c fiber.Ctx) error {
		c.Locals("user_id", current.UserID)
		return h.Update(c)
	})
	req := httptest.NewRequest("PATCH", "/custom-fields/"+current.ID.String(), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	return resp.StatusCode, repo
}

func TestUpdateCustomField_ValidatesMergedDefinition(t *testing.T) {
	orgID := uuid.New()
	tier := domain.CustomField{
		Base: domain.Base{ID: uuid.New()}, UserID: uuid.New(), OrganizationID: &orgID,
		Name: "Tier", NormalizedName: "tier", DataType: domain.CustomFieldDataTypeSelect,
		EntityType: domain.CustomFieldEntityContact, Options: pq.StringArray{"Gold", "Silver"},
	}
	budget := domain.CustomField{
		Base: domain.Base{ID: uuid.New()}, UserID: tier.UserID, OrganizationID: &orgID,
		Name: "Budget", NormalizedName: "budget", DataType: domain.CustomFieldDataTypeNumber,
		EntityType: domain.CustomFieldEntityContact, Options: pq.StringArray{},
	}
	others := []domain.CustomField{budget}

	cases := []struct {
		name   string
		body   string
		status int
	}{
		{"unknown data_type is a client error", `{"data_type":"bogus"}`, 400},
		{"fallback outside existing options", `{"fallback_value":"Bronze"}`, 400},
		{"clearing the options of a select", `{"options":[]}`, 400},
		{"rename onto another field's key", `{"name":"budget"}`, 409},
		{"frontend edit payload with unchanged name", `{"name":"Tier","data_type":"select","entity_type":"contact","options":["Gold","Silver"],"sample_data":"","fallback_value":"","is_required":true}`, 200},
		{"rename to a free name", `{"name":"Customer Tier"}`, 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, repo := updateStatus(t, tier, others, tc.body)
			assert.Equal(t, tc.status, status)
			if tc.status != 200 {
				repo.AssertNotCalled(t, "UpdateFields", mock.Anything, mock.Anything, mock.Anything)
			}
		})
	}

	t.Run("switching to select requires options", func(t *testing.T) {
		status, _ := updateStatus(t, budget, []domain.CustomField{tier}, `{"data_type":"select"}`)
		assert.Equal(t, 400, status)
	})
}
