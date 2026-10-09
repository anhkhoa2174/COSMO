package contact

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	customFieldRepo "github.com/rockship/cosmo-agents-go/internal/repository/custom_field"
	operationRepo "github.com/rockship/cosmo-agents-go/internal/repository/operation"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
)

const testUserHeader = "X-Test-User"

type importFixture struct {
	db    *gorm.DB
	app   *fiber.App
	user  uuid.UUID
	orgID uuid.UUID
}

func newImportFixture(t *testing.T) *importFixture {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.User{}, &domain.Organization{}, &domain.Role{},
		&domain.Contact{}, &domain.ListContact{}, &domain.CustomField{}, &domain.Operation{}))
	// The import upsert relies on this production index (migrations).
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uq_contacts_source_id_source_user_id ON contacts(source_id, source, user_id)`).Error)

	user := &domain.User{Email: "rep@acme.test", Name: "Rep"}
	require.NoError(t, db.Create(user).Error)
	org := &domain.Organization{Name: "acme", CompanyURL: "https://acme.test"}
	require.NoError(t, db.Create(org).Error)
	require.NoError(t, db.Create(&domain.Role{UserID: user.ID, OrganizationID: org.ID, Name: domain.RoleNameAdmin, Status: domain.RoleStatusActive}).Error)
	require.NoError(t, db.Create(&domain.CustomField{UserID: user.ID, OrganizationID: &org.ID, Name: "Deal size",
		NormalizedName: "deal_size", DataType: "text", EntityType: "contact"}).Error)

	h := New(contactRepo.NewContactRepository(db), userRepo.NewUserRepository(db), roleRepo.NewRoleRepository(db),
		contactRepo.NewListContactRepository(db), customFieldRepo.NewCustomFieldRepository(db), operationRepo.NewOperationRepository(db))
	// Inline, so the operation is settled by the time the response is read.
	h.runAsync = func(f func()) { f() }

	app := fiber.New()
	// Stands in for the auth middleware, which sets user_id but, notably, no
	// organization_id: the handler used to read that and panic on nil.
	app.Use(func(c fiber.Ctx) error {
		if id, err := uuid.Parse(c.Get(testUserHeader)); err == nil {
			c.Locals("user_id", id)
		}
		return c.Next()
	})
	app.Post("/v3/contacts/import", h.ImportCSV)
	return &importFixture{db: db, app: app, user: user.ID, orgID: org.ID}
}

func (f *importFixture) upload(t *testing.T, csv string, fields map[string]string) (int, map[string]any) {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", "leads.csv")
	require.NoError(t, err)
	_, _ = part.Write([]byte(csv))
	for k, v := range fields {
		require.NoError(t, w.WriteField(k, v))
	}
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, "/v3/contacts/import", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set(testUserHeader, f.user.String())
	resp, err := f.app.Test(req)
	require.NoError(t, err)
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return resp.StatusCode, out
}

func (f *importFixture) operation(t *testing.T, resp map[string]any) domain.Operation {
	t.Helper()
	id := resp["data"].(map[string]any)["id"].(string)
	var op domain.Operation
	require.NoError(t, f.db.First(&op, "id = ?", id).Error)
	return op
}

// The page's whole flow: upload with a mapping, then the operation it polls
// settles with the rows actually imported into the caller's organisation.
func TestImportCSV_ImportsMappedColumns(t *testing.T) {
	f := newImportFixture(t)
	csv := "\ufeffGiven,Family,Mail,Org,Sector,Size,email,Notes\n" +
		"Ann,Lee,ann@x.test,Acme,Fintech,50k,decoy@x.test,hot lead\n" +
		"Bob,Tran,bob@y.test,Globex,SaaS,,decoy2@x.test,\n"
	status, resp := f.upload(t, csv, map[string]string{
		"first_name": "Given", "last_name": "Family", "email": "Mail",
		"company": "Org", "industry": "Sector", "deal_size": "Size",
		"name_import": "September leads",
	})
	require.Equal(t, http.StatusOK, status, resp)

	op := f.operation(t, resp)
	require.Equal(t, domain.OperationStatusSuccess, op.Status, string(op.Output))
	var out map[string]any
	require.NoError(t, json.Unmarshal(op.Output, &out))
	assert.EqualValues(t, 2, out["imported_rows"])

	var contacts []domain.Contact
	require.NoError(t, f.db.Order("name").Find(&contacts, "user_id = ?", f.user).Error)
	require.Len(t, contacts, 2)
	ann := contacts[0]
	assert.Equal(t, "Ann Lee", ann.Name)
	// The column the user picked, not the one that happens to be called email.
	assert.Equal(t, "ann@x.test", ann.ContactInformation)
	assert.Equal(t, "Acme", ann.Company)
	assert.Equal(t, "Fintech", ann.Industry)
	require.NotNil(t, ann.OrganizationID)
	assert.Equal(t, f.orgID, *ann.OrganizationID)

	var profile map[string]any
	require.NoError(t, json.Unmarshal(ann.Profile, &profile))
	assert.Equal(t, "50k", profile["deal_size"], "a custom-field column is kept")
	assert.NotContains(t, profile, "notes", "an unmapped column is not imported")
}

func TestImportCSV_Validation(t *testing.T) {
	f := newImportFixture(t)
	csv := "Mail,Name\nann@x.test,Ann\n"
	tests := []struct {
		name   string
		fields map[string]string
		want   int
	}{
		{"unknown field", map[string]string{"email": "Mail", "password": "Name"}, http.StatusBadRequest},
		{"no column for email", map[string]string{"name": "Name"}, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, resp := f.upload(t, csv, tt.fields)
			assert.Equal(t, tt.want, status, resp)
		})
	}
}

// A mapping naming a column the file does not have fails the operation with a
// message, instead of leaving the page polling "in progress" forever.
func TestImportCSV_BadFileFailsTheOperation(t *testing.T) {
	f := newImportFixture(t)
	status, resp := f.upload(t, "Mail\nann@x.test\n", map[string]string{"email": "E-mail address"})
	require.Equal(t, http.StatusOK, status, resp)

	op := f.operation(t, resp)
	assert.Equal(t, domain.OperationStatusFailed, op.Status)
	var out map[string]any
	require.NoError(t, json.Unmarshal(op.Output, &out))
	assert.Contains(t, out["error"], "E-mail address")

	var n int64
	f.db.Model(&domain.Contact{}).Count(&n)
	assert.Zero(t, n)
}

// Re-importing a contact updates what the file carries and keeps the rest:
// the upsert used to overwrite a real company with "N/A" when the second file
// had no company column, and dropped the new custom-field values.
func TestImportCSV_ReimportKeepsStoredValues(t *testing.T) {
	f := newImportFixture(t)
	status, resp := f.upload(t, "Mail,Org,Size\nann@x.test,Acme,50k\n",
		map[string]string{"email": "Mail", "company": "Org", "deal_size": "Size"})
	require.Equal(t, http.StatusOK, status, resp)
	require.Equal(t, domain.OperationStatusSuccess, f.operation(t, resp).Status)

	// Second file: no company column, a new deal size, a new job title.
	status, resp = f.upload(t, "Mail,Size,Role\nann@x.test,80k,CTO\n",
		map[string]string{"email": "Mail", "deal_size": "Size", "job_title": "Role"})
	require.Equal(t, http.StatusOK, status, resp)
	require.Equal(t, domain.OperationStatusSuccess, f.operation(t, resp).Status)

	var contacts []domain.Contact
	require.NoError(t, f.db.Find(&contacts, "user_id = ?", f.user).Error)
	require.Len(t, contacts, 1, "the same email is one contact")
	ann := contacts[0]
	assert.Equal(t, "Acme", ann.Company, "a column the file lacks keeps its value")
	assert.Equal(t, "CTO", ann.JobTitle)
	var profile map[string]any
	require.NoError(t, json.Unmarshal(ann.Profile, &profile))
	assert.Equal(t, "80k", profile["deal_size"], "a value the file carries is updated")
	assert.Equal(t, "ann@x.test", profile["email"])
}

func TestImportCSV_RejectsOneColumnForTwoFields(t *testing.T) {
	f := newImportFixture(t)
	status, resp := f.upload(t, "Mail\nann@x.test\n", map[string]string{"email": "Mail", "deal_size": "Mail"})
	assert.Equal(t, http.StatusBadRequest, status, resp)
}
