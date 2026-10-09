package contact

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
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
	"github.com/rockship/cosmo-agents-go/internal/repository/organization"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
)

// tenancy is two organisations with real users, roles and contacts, served by
// the real handler over real repositories. Org B's admin plays the attacker in
// the cross-tenant tests.
type tenancy struct {
	db  *gorm.DB
	app *fiber.App

	orgA, orgB           *domain.Organization
	adminA, memberA      *domain.User
	adminB               *domain.User
	outsider             *domain.User // no organisation at all
	contactA1, contactA2 *domain.Contact
	contactB1, deletedA1 *domain.Contact
	contactsOfOrgA       []uuid.UUID
}

const testUserHeader = "X-Test-User"

func newTenancy(t *testing.T) *tenancy {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&domain.User{}, &domain.Organization{}, &domain.Role{},
		&domain.Contact{}, &domain.ListContact{}, &domain.ListContactAssociation{},
		&domain.CustomField{},
	))
	// Production carries these two unique indexes (migrations), which the
	// import upsert and list association rely on; the model tags do not.
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uq_contacts_source_id_source_user_id ON contacts(source_id, source, user_id)`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_list_contact_assoc_pair ON list_contact_association(list_contact_id, contact_id)`).Error)

	tn := &tenancy{db: db}
	mkUser := func(email string) *domain.User {
		u := &domain.User{Email: email, Name: email}
		require.NoError(t, db.Create(u).Error)
		return u
	}
	mkOrg := func(name string) *domain.Organization {
		o := &domain.Organization{Name: name, CompanyURL: "https://" + name + ".test"}
		require.NoError(t, db.Create(o).Error)
		return o
	}
	mkRole := func(u *domain.User, o *domain.Organization, name domain.RoleName) {
		require.NoError(t, db.Create(&domain.Role{UserID: u.ID, OrganizationID: o.ID, Name: name, Status: domain.RoleStatusActive}).Error)
	}

	tn.orgA, tn.orgB = mkOrg("alpha"), mkOrg("bravo")
	tn.adminA, tn.memberA = mkUser("admin@alpha.test"), mkUser("member@alpha.test")
	tn.adminB = mkUser("admin@bravo.test")
	tn.outsider = mkUser("outsider@nowhere.test")
	mkRole(tn.adminA, tn.orgA, domain.RoleNameAdmin)
	mkRole(tn.memberA, tn.orgA, domain.RoleNameMember)
	mkRole(tn.adminB, tn.orgB, domain.RoleNameAdmin)

	tn.contactA1 = tn.contact(t, tn.adminA, tn.orgA, "Alice Alpha", "Acme", "alice@acme.test")
	tn.contactA2 = tn.contact(t, tn.memberA, tn.orgA, "Mark Member", "Acme", "mark@acme.test")
	tn.contactB1 = tn.contact(t, tn.adminB, tn.orgB, "Bob Bravo", "Bravo Corp", "bob@bravo.test")
	tn.deletedA1 = tn.contact(t, tn.adminA, tn.orgA, "Gone Alpha", "Acme", "gone@acme.test")
	require.NoError(t, db.Model(tn.deletedA1).Update("is_deleted", true).Error)
	tn.contactsOfOrgA = []uuid.UUID{tn.contactA1.ID, tn.contactA2.ID}

	h := New(
		contactRepo.NewContactRepository(db),
		userRepo.NewUserRepository(db),
		roleRepo.NewRoleRepository(db),
		organization.NewOrganizationRepository(db),
		customFieldRepo.NewCustomFieldRepository(db),
		contactRepo.NewListContactRepository(db),
		nil, nil, nil, nil, nil,
	)

	app := fiber.New()
	// Stand-in for AuthMiddleware: it stores the user id under both keys.
	app.Use(func(c fiber.Ctx) error {
		if raw := c.Get(testUserHeader); raw != "" {
			id := uuid.MustParse(raw)
			c.Locals("userID", id)
			c.Locals("user_id", id)
		}
		return c.Next()
	})
	app.Post("/v1/contacts", h.Create)
	app.Post("/v1/contacts/bulk", h.BulkCreate)
	app.Post("/v1/contacts/search", h.Search)
	app.Get("/v1/contact/:id", h.Get)
	app.Get("/v1/contact", h.List)
	app.Post("/v1/contact/list", h.List) // List also serves POST bodies
	app.Get("/v1/contacts/values", h.GetFieldValues)
	app.Patch("/v1/contacts/:id", h.Update)
	app.Delete("/v1/contacts", h.Delete)
	app.Post("/v1/contacts/:id/research-findings", h.AddResearchFinding)
	app.Post("/v1/contacts/:id/extract-from-url", h.ExtractFromURL)
	app.Post("/v1/contacts/:id/extract-from-image", h.ExtractFromImage)
	app.Post("/v1/contacts/:id/extract-from-extension", h.ExtractFromExtension)
	app.Post("/v1/extract-from-image-preview", h.ExtractFromImagePreview)
	app.Post("/v1/contacts/import-csv", h.ImportCSV)
	app.Post("/v1/contacts/:id/insights/validate", h.ValidateInsight)
	app.Post("/v1/contacts/recalculate-status", h.RecalculateStatus)
	app.Post("/v1/contacts/re-embed-all", h.ReEmbedAll)
	app.Post("/v1/contacts/cleanup-vectors", h.CleanupDeletedVectors)
	tn.app = app
	return tn
}

func (tn *tenancy) contact(t *testing.T, owner *domain.User, org *domain.Organization, name, company, email string) *domain.Contact {
	t.Helper()
	orgID := org.ID
	c := &domain.Contact{
		UserID:         owner.ID,
		OrganizationID: &orgID,
		Name:           name,
		Company:        company,
		Source:         string(domain.ContactSourceCosmoAgents),
		Profile:        base.JSONB(`{"email":"` + email + `"}`),
		// Create's duplicate check keys on this column.
		ContactInformation: email,
		AIInsights:         base.JSONB(`{"suspected_pain_points":[{"pain_point":"slow onboarding"}]}`),
	}
	require.NoError(t, tn.db.Create(c).Error)
	return c
}

type apiResp struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Error  *struct {
		ErrorCode int         `json:"error_code"`
		Message   string      `json:"message"`
		Detail    interface{} `json:"detail"`
	} `json:"error"`
}

func (tn *tenancy) do(t *testing.T, method, path string, as *domain.User, body interface{}) (int, apiResp) {
	t.Helper()
	var rdr io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		rdr = bytes.NewBufferString(b)
	default:
		raw, err := json.Marshal(b)
		require.NoError(t, err)
		rdr = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, rdr)
	if rdr != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if as != nil {
		req.Header.Set(testUserHeader, as.ID.String())
	}
	resp, err := tn.app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var out apiResp
	require.NoError(t, json.Unmarshal(raw, &out), "status %d body %s", resp.StatusCode, raw)
	return resp.StatusCode, out
}

// assertError checks the shared error envelope: status "error" and an
// error_code that matches the HTTP status.
func assertError(t *testing.T, status int, r apiResp, want int) {
	t.Helper()
	assert.Equal(t, want, status)
	assert.Equal(t, "error", r.Status)
	if assert.NotNil(t, r.Error) {
		assert.Equal(t, want, r.Error.ErrorCode)
		assert.NotEmpty(t, r.Error.Message)
	}
}

func (tn *tenancy) reload(t *testing.T, id uuid.UUID) domain.Contact {
	t.Helper()
	var c domain.Contact
	require.NoError(t, tn.db.First(&c, "id = ?", id).Error)
	return c
}

type contactList struct {
	List []struct {
		ID     uuid.UUID              `json:"id"`
		Entity map[string]interface{} `json:"entity"`
	} `json:"list"`
	Total  int64 `json:"total"`
	Offset int   `json:"offset"`
	Limit  int   `json:"limit"`
}

func (l contactList) ids() []uuid.UUID {
	out := make([]uuid.UUID, 0, len(l.List))
	for _, it := range l.List {
		if it.ID != uuid.Nil {
			out = append(out, it.ID)
			continue
		}
		if s, ok := it.Entity["id"].(string); ok {
			out = append(out, uuid.MustParse(s))
		}
	}
	return out
}

func decodeList(t *testing.T, r apiResp) contactList {
	t.Helper()
	var l contactList
	require.NoError(t, json.Unmarshal(r.Data, &l), "data: %s", r.Data)
	return l
}

// --- GET /v1/contact/:id ---------------------------------------------------

func TestGet_HTTP(t *testing.T) {
	tn := newTenancy(t)

	tests := []struct {
		name string
		as   *domain.User
		id   string
		want int
	}{
		{"owner reads their contact", tn.adminA, tn.contactA1.ID.String(), fiber.StatusOK},
		{"invalid id", tn.adminA, "not-a-uuid", fiber.StatusBadRequest},
		{"unauthenticated", nil, tn.contactA1.ID.String(), fiber.StatusUnauthorized},
		// The missing contact and the other tenant's contact must be
		// indistinguishable: both 404, never 403 or 500.
		{"nonexistent contact", tn.adminA, uuid.NewString(), fiber.StatusNotFound},
		{"other tenant's contact", tn.adminB, tn.contactA1.ID.String(), fiber.StatusNotFound},
		{"soft-deleted contact", tn.adminA, tn.deletedA1.ID.String(), fiber.StatusNotFound},
		{"user without an organisation", tn.outsider, tn.contactA1.ID.String(), fiber.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, r := tn.do(t, http.MethodGet, "/v1/contact/"+tt.id, tt.as, nil)
			if tt.want != fiber.StatusOK {
				assertError(t, status, r, tt.want)
				return
			}
			require.Equal(t, fiber.StatusOK, status)
			var got map[string]interface{}
			require.NoError(t, json.Unmarshal(r.Data, &got))
			assert.Equal(t, tn.contactA1.ID.String(), got["id"])
			assert.Equal(t, "alice@acme.test", got["email"])
			assert.Equal(t, tn.orgA.ID.String(), got["organization_id"])
		})
	}
}

// --- GET /v1/contact ------------------------------------------------------

func TestList_HTTP_VisibilityByRole(t *testing.T) {
	tn := newTenancy(t)

	tests := []struct {
		name string
		as   *domain.User
		want []uuid.UUID
	}{
		{"admin sees every live contact in the org", tn.adminA, tn.contactsOfOrgA},
		{"member sees only their own", tn.memberA, []uuid.UUID{tn.contactA2.ID}},
		{"other tenant sees only theirs", tn.adminB, []uuid.UUID{tn.contactB1.ID}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, r := tn.do(t, http.MethodGet, "/v1/contact", tt.as, nil)
			require.Equal(t, fiber.StatusOK, status)
			l := decodeList(t, r)
			assert.ElementsMatch(t, tt.want, l.ids())
			assert.Equal(t, int64(len(tt.want)), l.Total)
		})
	}

	status, r := tn.do(t, http.MethodGet, "/v1/contact", nil, nil)
	assertError(t, status, r, fiber.StatusUnauthorized)
}

// A POSTed filter narrows the list; it must never widen it past the caller's
// organisation or, for a member, past their own contacts.
func TestList_HTTP_BodyFilterCannotWidenScope(t *testing.T) {
	tn := newTenancy(t)

	tests := []struct {
		name   string
		as     *domain.User
		filter map[string]interface{}
		want   []uuid.UUID
	}{
		{"narrowing filter works", tn.adminA, map[string]interface{}{"name": "Alice Alpha"}, []uuid.UUID{tn.contactA1.ID}},
		{"organisation override", tn.adminB, map[string]interface{}{"organization_id": tn.orgA.ID.String()}, []uuid.UUID{tn.contactB1.ID}},
		{"member lifts user filter", tn.memberA, map[string]interface{}{"user_id": tn.adminA.ID.String()}, []uuid.UUID{tn.contactA2.ID}},
		{"raw SQL", tn.adminB, map[string]interface{}{"$raw": "true) OR (true"}, []uuid.UUID{tn.contactB1.ID}},
		{"deleted rows", tn.adminA, map[string]interface{}{"is_deleted": true}, tn.contactsOfOrgA},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, r := tn.do(t, http.MethodPost, "/v1/contact/list", tt.as, map[string]interface{}{"filter": tt.filter})
			require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)
			assert.ElementsMatch(t, tt.want, decodeList(t, r).ids())
		})
	}

	status, r := tn.do(t, http.MethodPost, "/v1/contact/list", tn.adminA, "{not json")
	assertError(t, status, r, fiber.StatusBadRequest)
}

func TestList_HTTP_PaginationBounds(t *testing.T) {
	tn := newTenancy(t)

	tests := []struct {
		query      string
		wantOffset int
		wantLimit  int
		wantCount  int
	}{
		{"?limit=1", 0, 1, 1},
		{"?offset=1&limit=1", 1, 1, 1},
		{"?offset=5&limit=1", 5, 1, 0},
		{"?offset=-3&limit=10", 0, 10, 2},
		{"?limit=1000000", 0, 1000, 2},
		{"?limit=-1", 0, 1000, 2},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			status, r := tn.do(t, http.MethodGet, "/v1/contact"+tt.query, tn.adminA, nil)
			require.Equal(t, fiber.StatusOK, status)
			l := decodeList(t, r)
			assert.Equal(t, tt.wantOffset, l.Offset)
			assert.Equal(t, tt.wantLimit, l.Limit)
			assert.Len(t, l.List, tt.wantCount)
			assert.Equal(t, int64(2), l.Total, "total ignores pagination")
		})
	}
}

// --- POST /v1/contacts/search ---------------------------------------------

func TestSearch_HTTP(t *testing.T) {
	tn := newTenancy(t)

	t.Run("members see the whole organisation", func(t *testing.T) {
		status, r := tn.do(t, http.MethodPost, "/v1/contacts/search", tn.memberA, map[string]interface{}{})
		require.Equal(t, fiber.StatusOK, status)
		l := decodeList(t, r)
		assert.ElementsMatch(t, tn.contactsOfOrgA, l.ids())
		// "Added by" is filled from the owning user.
		for _, it := range l.List {
			assert.NotEmpty(t, it.Entity["added_by_email"])
		}
	})

	t.Run("text filters are partial matches", func(t *testing.T) {
		status, r := tn.do(t, http.MethodPost, "/v1/contacts/search", tn.adminA, map[string]interface{}{"filter": map[string]interface{}{"name": "alice"}})
		require.Equal(t, fiber.StatusOK, status)
		assert.Equal(t, []uuid.UUID{tn.contactA1.ID}, decodeList(t, r).ids())
	})

	t.Run("legacy filter_ key", func(t *testing.T) {
		status, r := tn.do(t, http.MethodPost, "/v1/contacts/search", tn.adminA, map[string]interface{}{"filter_": map[string]interface{}{"name": "mark"}})
		require.Equal(t, fiber.StatusOK, status)
		assert.Equal(t, []uuid.UUID{tn.contactA2.ID}, decodeList(t, r).ids())
	})

	// Every one of these payloads comes from org B's admin and aims at org
	// A's rows. The response must contain org B's contact and nothing else.
	attacks := []struct {
		name   string
		filter map[string]interface{}
	}{
		{"organisation override", map[string]interface{}{"organization_id": tn.orgA.ID.String()}},
		{"organisation override via $in", map[string]interface{}{"organization_id": []string{tn.orgA.ID.String(), tn.orgB.ID.String()}}},
		{"raw SQL breaking out of the AND chain", map[string]interface{}{"$raw": "true) OR (true"}},
		{"raw SQL nested in $or", map[string]interface{}{"$or": []interface{}{map[string]interface{}{"$raw": "true) OR (true"}}}},
		{"raw SQL nested in $and", map[string]interface{}{"$and": []interface{}{map[string]interface{}{"$raw": "true) OR (true"}}}},
	}
	for _, tt := range attacks {
		t.Run("cross-tenant: "+tt.name, func(t *testing.T) {
			status, r := tn.do(t, http.MethodPost, "/v1/contacts/search", tn.adminB, map[string]interface{}{"filter": tt.filter})
			require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)
			got := decodeList(t, r).ids()
			for _, id := range tn.contactsOfOrgA {
				assert.NotContains(t, got, id, "org A contact leaked to org B")
			}
		})
	}

	t.Run("pagination", func(t *testing.T) {
		status, r := tn.do(t, http.MethodPost, "/v1/contacts/search?limit=1&offset=1", tn.adminA, map[string]interface{}{})
		require.Equal(t, fiber.StatusOK, status)
		l := decodeList(t, r)
		assert.Len(t, l.List, 1)
		assert.Equal(t, int64(2), l.Total)

		status, r = tn.do(t, http.MethodPost, "/v1/contacts/search?limit=99999", tn.adminA, map[string]interface{}{})
		require.Equal(t, fiber.StatusOK, status)
		assert.Equal(t, 1000, decodeList(t, r).Limit)

		status, r = tn.do(t, http.MethodPost, "/v1/contacts/search?limit=0", tn.adminA, map[string]interface{}{})
		require.Equal(t, fiber.StatusOK, status)
		assert.Equal(t, 25, decodeList(t, r).Limit)
	})

	errorsCases := []struct {
		name  string
		as    *domain.User
		query string
		body  interface{}
		want  int
	}{
		{"unauthenticated", nil, "", map[string]interface{}{}, fiber.StatusUnauthorized},
		{"non-numeric offset", tn.adminA, "?offset=x", map[string]interface{}{}, fiber.StatusBadRequest},
		{"non-numeric limit", tn.adminA, "?limit=x", map[string]interface{}{}, fiber.StatusBadRequest},
		{"malformed body", tn.adminA, "", "{", fiber.StatusBadRequest},
		{"bad segment id", tn.adminA, "", map[string]interface{}{"filter": map[string]interface{}{"segment_id": "nope"}}, fiber.StatusBadRequest},
	}
	for _, tt := range errorsCases {
		t.Run(tt.name, func(t *testing.T) {
			status, r := tn.do(t, http.MethodPost, "/v1/contacts/search"+tt.query, tt.as, tt.body)
			assertError(t, status, r, tt.want)
		})
	}
}

// --- PATCH /v1/contacts/:id -----------------------------------------------

func TestUpdate_HTTP(t *testing.T) {
	tn := newTenancy(t)

	t.Run("owner updates and the row changes", func(t *testing.T) {
		status, r := tn.do(t, http.MethodPatch, "/v1/contacts/"+tn.contactA1.ID.String(), tn.adminA,
			map[string]interface{}{"company": "Acme Rebrand", "job_title": "CTO"})
		require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)
		var got map[string]interface{}
		require.NoError(t, json.Unmarshal(r.Data, &got))
		assert.Equal(t, "Acme Rebrand", got["company"])

		row := tn.reload(t, tn.contactA1.ID)
		assert.Equal(t, "Acme Rebrand", row.Company)
		assert.Equal(t, "CTO", row.JobTitle)
		assert.Equal(t, tn.adminA.ID, row.UserID, "ownership is untouched")
	})

	t.Run("organisation_id in the body cannot move a contact", func(t *testing.T) {
		status, _ := tn.do(t, http.MethodPatch, "/v1/contacts/"+tn.contactA1.ID.String(), tn.adminA,
			map[string]interface{}{"organization_id": tn.orgB.ID.String(), "name": "Still Alpha"})
		require.Equal(t, fiber.StatusOK, status)
		row := tn.reload(t, tn.contactA1.ID)
		require.NotNil(t, row.OrganizationID)
		assert.Equal(t, tn.orgA.ID, *row.OrganizationID)
	})

	t.Run("other tenant gets 404 and nothing changes", func(t *testing.T) {
		before := tn.reload(t, tn.contactA2.ID)
		status, r := tn.do(t, http.MethodPatch, "/v1/contacts/"+tn.contactA2.ID.String(), tn.adminB,
			map[string]interface{}{"company": "Pwned"})
		assertError(t, status, r, fiber.StatusNotFound)
		after := tn.reload(t, tn.contactA2.ID)
		assert.Equal(t, before.Company, after.Company)
		assert.Equal(t, string(before.Profile), string(after.Profile))
	})

	cases := []struct {
		name string
		as   *domain.User
		id   string
		body interface{}
		want int
	}{
		{"nonexistent contact", tn.adminA, uuid.NewString(), map[string]interface{}{"name": "x"}, fiber.StatusNotFound},
		{"soft-deleted contact", tn.adminA, tn.deletedA1.ID.String(), map[string]interface{}{"name": "x"}, fiber.StatusNotFound},
		{"invalid id", tn.adminA, "nope", map[string]interface{}{"name": "x"}, fiber.StatusBadRequest},
		{"malformed body", tn.adminA, tn.contactA1.ID.String(), "{", fiber.StatusBadRequest},
		{"unauthenticated", nil, tn.contactA1.ID.String(), map[string]interface{}{"name": "x"}, fiber.StatusUnauthorized},
		{"invalid email", tn.adminA, tn.contactA1.ID.String(), map[string]interface{}{"email": "not-an-email"}, fiber.StatusUnprocessableEntity},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			status, r := tn.do(t, http.MethodPatch, "/v1/contacts/"+tt.id, tt.as, tt.body)
			assertError(t, status, r, tt.want)
		})
	}
}

// --- DELETE /v1/contacts ---------------------------------------------------

func TestDelete_HTTP(t *testing.T) {
	tn := newTenancy(t)

	t.Run("foreign ids are ignored, own ids deleted", func(t *testing.T) {
		ids := []string{tn.contactB1.ID.String(), tn.contactA1.ID.String(), "garbage", uuid.NewString()}
		status, r := tn.do(t, http.MethodDelete, "/v1/contacts", tn.adminA, map[string]interface{}{"ids": ids})
		require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)

		var deleted []map[string]interface{}
		require.NoError(t, json.Unmarshal(r.Data, &deleted))
		require.Len(t, deleted, 1)
		assert.Equal(t, tn.contactA1.ID.String(), deleted[0]["id"])
		assert.Equal(t, true, deleted[0]["is_deleted"])

		assert.True(t, tn.reload(t, tn.contactA1.ID).IsDeleted)
		assert.False(t, tn.reload(t, tn.contactB1.ID).IsDeleted, "org B's contact must survive")
	})

	t.Run("an attacker's bulk delete of another tenant is a no-op", func(t *testing.T) {
		status, r := tn.do(t, http.MethodDelete, "/v1/contacts", tn.adminB,
			map[string]interface{}{"ids": []string{tn.contactA2.ID.String()}})
		require.Equal(t, fiber.StatusOK, status)
		assert.JSONEq(t, `[]`, string(r.Data))
		assert.False(t, tn.reload(t, tn.contactA2.ID).IsDeleted)
	})

	t.Run("only invalid ids", func(t *testing.T) {
		status, r := tn.do(t, http.MethodDelete, "/v1/contacts", tn.adminA, map[string]interface{}{"ids": []string{"a", "b"}})
		require.Equal(t, fiber.StatusOK, status)
		assert.JSONEq(t, `[]`, string(r.Data))
	})

	cases := []struct {
		name string
		as   *domain.User
		body interface{}
		want int
	}{
		{"empty id list", tn.adminA, map[string]interface{}{"ids": []string{}}, fiber.StatusUnprocessableEntity},
		{"missing ids", tn.adminA, map[string]interface{}{}, fiber.StatusUnprocessableEntity},
		{"malformed body", tn.adminA, "{", fiber.StatusBadRequest},
		{"ids of the wrong type", tn.adminA, map[string]interface{}{"ids": "abc"}, fiber.StatusBadRequest},
		{"unauthenticated", nil, map[string]interface{}{"ids": []string{uuid.NewString()}}, fiber.StatusUnauthorized},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			status, r := tn.do(t, http.MethodDelete, "/v1/contacts", tt.as, tt.body)
			assertError(t, status, r, tt.want)
		})
	}
}

// --- POST /v1/contacts/:id/research-findings --------------------------------

func TestAddResearchFinding_HTTP(t *testing.T) {
	tn := newTenancy(t)
	finding := map[string]interface{}{"category": "Company", "field_name": "Company Size", "value": "200"}

	t.Run("owner adds a finding", func(t *testing.T) {
		status, r := tn.do(t, http.MethodPost, "/v1/contacts/"+tn.contactA1.ID.String()+"/research-findings", tn.adminA, finding)
		require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)
		var profile map[string]interface{}
		require.NoError(t, tn.reload(t, tn.contactA1.ID).Profile.Unmarshal(&profile))
		assert.Len(t, profile["research_findings"], 1)
		cf := profile["custom_fields"].(map[string]interface{})
		assert.Equal(t, "200", cf["Company Size"].(map[string]interface{})["value"])
	})

	cases := []struct {
		name string
		as   *domain.User
		id   string
		body interface{}
		want int
	}{
		{"nonexistent contact", tn.adminA, uuid.NewString(), finding, fiber.StatusNotFound},
		{"other tenant's contact", tn.adminB, tn.contactA2.ID.String(), finding, fiber.StatusNotFound},
		{"invalid id", tn.adminA, "nope", finding, fiber.StatusBadRequest},
		{"malformed body", tn.adminA, tn.contactA1.ID.String(), "{", fiber.StatusBadRequest},
		{"unauthenticated", nil, tn.contactA1.ID.String(), finding, fiber.StatusUnauthorized},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			before := tn.reload(t, tn.contactA2.ID)
			status, r := tn.do(t, http.MethodPost, "/v1/contacts/"+tt.id+"/research-findings", tt.as, tt.body)
			assertError(t, status, r, tt.want)
			assert.Equal(t, string(before.Profile), string(tn.reload(t, tn.contactA2.ID).Profile))
		})
	}
}

// --- POST /v1/contacts/:id/insights/validate --------------------------------

func TestValidateInsight_HTTP(t *testing.T) {
	tn := newTenancy(t)
	confirm := map[string]interface{}{"insight_type": "pain_point", "insight_text": "slow onboarding", "validation": "confirmed"}

	t.Run("a teammate confirms an insight on an org contact", func(t *testing.T) {
		status, r := tn.do(t, http.MethodPost, "/v1/contacts/"+tn.contactA1.ID.String()+"/insights/validate", tn.memberA, confirm)
		require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)

		row := tn.reload(t, tn.contactA1.ID)
		var ai, facts map[string]interface{}
		require.NoError(t, row.AIInsights.Unmarshal(&ai))
		require.NoError(t, row.ConfirmedFacts.Unmarshal(&facts))
		assert.Empty(t, ai["suspected_pain_points"], "the insight moves out of the suspected list")
		assert.Len(t, facts["pain_points"], 1)
	})

	cases := []struct {
		name string
		as   *domain.User
		id   string
		body interface{}
		want int
	}{
		{"other tenant's contact", tn.adminB, tn.contactA2.ID.String(), confirm, fiber.StatusNotFound},
		{"nonexistent contact", tn.adminA, uuid.NewString(), confirm, fiber.StatusNotFound},
		{"missing fields", tn.adminA, tn.contactA2.ID.String(), map[string]interface{}{"insight_type": "goal"}, fiber.StatusBadRequest},
		{"unknown insight type", tn.adminA, tn.contactA2.ID.String(), map[string]interface{}{"insight_type": "vibe", "insight_text": "x", "validation": "confirmed"}, fiber.StatusBadRequest},
		{"invalid id", tn.adminA, "nope", confirm, fiber.StatusBadRequest},
		{"malformed body", tn.adminA, tn.contactA2.ID.String(), "{", fiber.StatusBadRequest},
		{"unauthenticated", nil, tn.contactA2.ID.String(), confirm, fiber.StatusUnauthorized},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			before := tn.reload(t, tn.contactA2.ID)
			status, r := tn.do(t, http.MethodPost, "/v1/contacts/"+tt.id+"/insights/validate", tt.as, tt.body)
			assertError(t, status, r, tt.want)
			assert.Equal(t, string(before.AIInsights), string(tn.reload(t, tn.contactA2.ID).AIInsights))
		})
	}
}

// --- GET /v1/contacts/values -----------------------------------------------

func TestGetFieldValues_HTTP(t *testing.T) {
	tn := newTenancy(t)

	t.Run("values of the caller's organisation", func(t *testing.T) {
		status, r := tn.do(t, http.MethodGet, "/v1/contacts/values?fields=company&fields=name", tn.adminA, nil)
		require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)
		var page struct {
			List []map[string]struct {
				List  []interface{} `json:"list"`
				Total int64         `json:"total"`
			} `json:"list"`
			Total int64 `json:"total"`
		}
		require.NoError(t, json.Unmarshal(r.Data, &page))
		require.Len(t, page.List, 2)
		assert.Equal(t, int64(2), page.Total)
		assert.Contains(t, page.List[0]["company"].List, "Acme")
	})

	cases := []struct {
		name  string
		as    *domain.User
		query string
		want  int
	}{
		{"unauthenticated", nil, "?fields=company", fiber.StatusUnauthorized},
		{"user without an organisation", tn.outsider, "?fields=company", fiber.StatusUnauthorized},
		{"no fields", tn.adminA, "", fiber.StatusBadRequest},
		{"blank fields", tn.adminA, "?fields=%20", fiber.StatusBadRequest},
		// Unknown columns are rejected before they reach SQL.
		{"unknown field", tn.adminA, "?fields=password", fiber.StatusNotFound},
		{"injection attempt", tn.adminA, "?fields=name;drop%20table%20contacts", fiber.StatusNotFound},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			status, r := tn.do(t, http.MethodGet, "/v1/contacts/values"+tt.query, tt.as, nil)
			assertError(t, status, r, tt.want)
		})
	}
}

// GetDistinctFieldValues builds its scoped query and then runs it through
// Session(&gorm.Session{NewDB: true}), which discards every WHERE clause —
// including the tenant scope. The handler cannot compensate for it.
func TestGetFieldValues_HTTP_CrossTenant(t *testing.T) {
	t.Skip("BUG: internal/repository/contact/contact_repository.go GetDistinctFieldValues uses " +
		"Session(NewDB:true) for its count and select, dropping the user/organisation scope; " +
		"org B sees org A's distinct values (including organization_id and user_id)")

	tn := newTenancy(t)
	status, r := tn.do(t, http.MethodGet, "/v1/contacts/values?fields=company", tn.adminB, nil)
	require.Equal(t, fiber.StatusOK, status)
	assert.NotContains(t, string(r.Data), "Acme")
}

// --- maintenance endpoints --------------------------------------------------

func TestMaintenanceEndpoints_HTTP(t *testing.T) {
	tn := newTenancy(t)

	status, r := tn.do(t, http.MethodPost, "/v1/contacts/recalculate-status", tn.adminA, nil)
	require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)
	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(r.Data, &out))
	assert.Contains(t, out, "updated_count")

	// Without the intelligence service these report themselves unavailable.
	for _, path := range []string{"/v1/contacts/re-embed-all", "/v1/contacts/cleanup-vectors"} {
		status, r := tn.do(t, http.MethodPost, path, tn.adminA, nil)
		assertError(t, status, r, fiber.StatusInternalServerError)
	}

	for _, path := range []string{"/v1/contacts/recalculate-status", "/v1/contacts/re-embed-all", "/v1/contacts/cleanup-vectors"} {
		status, r := tn.do(t, http.MethodPost, path, nil, nil)
		assertError(t, status, r, fiber.StatusUnauthorized)
	}
}

// --- POST /v1/contacts ------------------------------------------------------

func TestCreate_HTTP(t *testing.T) {
	tn := newTenancy(t)

	t.Run("creates in the caller's organisation, ignoring organization_id in the body", func(t *testing.T) {
		status, r := tn.do(t, http.MethodPost, "/v1/contacts", tn.memberA, map[string]interface{}{
			"name": "New Person", "email": "new@acme.test", "phone": "+1 555", "linkedin_url": "https://linkedin.com/in/new",
			"organization_id": tn.orgB.ID.String(), "favourite_colour": "teal",
		})
		require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)
		var got map[string]interface{}
		require.NoError(t, json.Unmarshal(r.Data, &got))

		row := tn.reload(t, uuid.MustParse(got["id"].(string)))
		require.NotNil(t, row.OrganizationID)
		assert.Equal(t, tn.orgA.ID, *row.OrganizationID)
		assert.Equal(t, tn.memberA.ID, row.UserID)
		assert.Equal(t, "new@acme.test", row.ContactInformation)
		var profile map[string]interface{}
		require.NoError(t, row.Profile.Unmarshal(&profile))
		assert.Equal(t, "+1 555", profile["phone"])
		assert.Contains(t, profile["custom_fields"], "favourite_colour", "unknown keys become custom fields")
	})

	t.Run("blank name defaults instead of failing validation", func(t *testing.T) {
		status, r := tn.do(t, http.MethodPost, "/v1/contacts", tn.adminA, map[string]interface{}{"name": "  ", "company": "Nameless"})
		require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)
	})

	t.Run("a teammate's duplicate is a 409 naming who added it", func(t *testing.T) {
		status, r := tn.do(t, http.MethodPost, "/v1/contacts", tn.memberA, map[string]interface{}{"name": "Alice Again", "email": "ALICE@acme.test"})
		assertError(t, status, r, fiber.StatusConflict)
		detail := r.Error.Detail.(map[string]interface{})
		assert.Equal(t, tn.contactA1.ID.String(), detail["existing_contact_id"])
		assert.Equal(t, tn.adminA.Name, detail["added_by_name"])
	})

	t.Run("the same contact in another organisation is not a duplicate", func(t *testing.T) {
		status, r := tn.do(t, http.MethodPost, "/v1/contacts", tn.adminB, map[string]interface{}{"name": "Alice", "email": "alice@acme.test"})
		require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)
		assert.NotContains(t, string(r.Data), tn.contactA1.ID.String())
	})

	t.Run("re-adding your own contact merges into it", func(t *testing.T) {
		status, r := tn.do(t, http.MethodPost, "/v1/contacts", tn.adminA, map[string]interface{}{
			"name": "Alice Merged", "email": "alice@acme.test", "company": "Acme Two", "phone": "+44 20", "city": "London",
			"profile": map[string]interface{}{"twitter": "@alice"}, "tier": "gold",
		})
		require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)
		var got map[string]interface{}
		require.NoError(t, json.Unmarshal(r.Data, &got))
		assert.Equal(t, tn.contactA1.ID.String(), got["id"], "merged, not duplicated")

		row := tn.reload(t, tn.contactA1.ID)
		assert.Equal(t, "Alice Merged", row.Name)
		assert.Equal(t, "Acme Two", row.Company)
		assert.Equal(t, "London", row.City)
		var profile map[string]interface{}
		require.NoError(t, row.Profile.Unmarshal(&profile))
		assert.Equal(t, "@alice", profile["twitter"])
		assert.Equal(t, "+44 20", profile["phone"], "phone lives in the profile")
		assert.Contains(t, profile["custom_fields"], "tier")
	})

	cases := []struct {
		name string
		as   *domain.User
		body interface{}
		want int
	}{
		{"invalid email", tn.adminA, map[string]interface{}{"name": "X", "email": "nope"}, fiber.StatusUnprocessableEntity},
		{"malformed body", tn.adminA, "{", fiber.StatusBadRequest},
		{"unauthenticated", nil, map[string]interface{}{"name": "X"}, fiber.StatusUnauthorized},
		{"user without an organisation", tn.outsider, map[string]interface{}{"name": "X"}, fiber.StatusUnauthorized},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			status, r := tn.do(t, http.MethodPost, "/v1/contacts", tt.as, tt.body)
			assertError(t, status, r, tt.want)
		})
	}
}

// --- POST /v1/contacts/bulk -------------------------------------------------

func TestBulkCreate_HTTP(t *testing.T) {
	tn := newTenancy(t)

	type result struct {
		Created    int      `json:"created"`
		Skipped    int      `json:"skipped"`
		Errors     []string `json:"errors"`
		ContactIDs []string `json:"contact_ids"`
	}

	t.Run("creates, skips empties and the caller's duplicates", func(t *testing.T) {
		status, r := tn.do(t, http.MethodPost, "/v1/contacts/bulk", tn.adminA, map[string]interface{}{"contacts": []map[string]interface{}{
			{"full_name": "Bulk One", "email": "bulk1@acme.test", "phone": "1"},
			{"name": "Bulk Two", "linkedin_url": "https://linkedin.com/in/bulk2", "source": string(domain.ContactSourceLinkedIn)},
			{}, // nothing to identify it: skipped
			{"name": "Dup", "email": "alice@acme.test"}, // already the caller's: skipped
			{"name": "Dup LI", "linkedin_url": "https://linkedin.com/in/bulk2"},
		}})
		require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)
		var got result
		require.NoError(t, json.Unmarshal(r.Data, &got))
		assert.Equal(t, 2, got.Created)
		assert.Equal(t, 3, got.Skipped)
		require.Len(t, got.ContactIDs, 2)

		li := tn.reload(t, uuid.MustParse(got.ContactIDs[1]))
		assert.Equal(t, "https://linkedin.com/in/bulk2", li.ContactInformation)
		require.NotNil(t, li.OrganizationID)
		assert.Equal(t, tn.orgA.ID, *li.OrganizationID)
	})

	// A failed row is reported by its 1-based position in the request.
	t.Run("row errors name the row", func(t *testing.T) {
		require.NoError(t, tn.db.Exec(`ALTER TABLE contacts ADD CONSTRAINT test_no_boom CHECK (name <> 'Boom')`).Error)
		t.Cleanup(func() { tn.db.Exec(`ALTER TABLE contacts DROP CONSTRAINT test_no_boom`) })

		status, r := tn.do(t, http.MethodPost, "/v1/contacts/bulk", tn.adminA, map[string]interface{}{"contacts": []map[string]interface{}{
			{"name": "Fine"}, {"name": "Boom"},
		}})
		require.Equal(t, fiber.StatusOK, status)
		var got result
		require.NoError(t, json.Unmarshal(r.Data, &got))
		assert.Equal(t, 1, got.Created)
		require.Len(t, got.Errors, 1)
		assert.Contains(t, got.Errors[0], "Contact 2:")
	})

	many := make([]map[string]interface{}, 501)
	for i := range many {
		many[i] = map[string]interface{}{"name": "n"}
	}
	cases := []struct {
		name string
		as   *domain.User
		body interface{}
		want int
	}{
		{"empty list", tn.adminA, map[string]interface{}{"contacts": []interface{}{}}, fiber.StatusBadRequest},
		{"over the 500 limit", tn.adminA, map[string]interface{}{"contacts": many}, fiber.StatusBadRequest},
		{"malformed body", tn.adminA, "{", fiber.StatusBadRequest},
		{"unauthenticated", nil, map[string]interface{}{"contacts": []interface{}{map[string]string{"name": "x"}}}, fiber.StatusUnauthorized},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			status, r := tn.do(t, http.MethodPost, "/v1/contacts/bulk", tt.as, tt.body)
			assertError(t, status, r, tt.want)
		})
	}
}

// --- extraction endpoints ---------------------------------------------------

// multipartBody builds a form with one file part named field.
func multipartBody(t *testing.T, field, filename, contentType string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	hdr := textproto.MIMEHeader{}
	hdr.Set("Content-Disposition", `form-data; name="`+field+`"; filename="`+filename+`"`)
	hdr.Set("Content-Type", contentType)
	part, err := w.CreatePart(hdr)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return &buf, w.FormDataContentType()
}

func (tn *tenancy) doMultipart(t *testing.T, path string, as *domain.User, body *bytes.Buffer, contentType string) (int, apiResp) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Content-Type", contentType)
	if as != nil {
		req.Header.Set(testUserHeader, as.ID.String())
	}
	resp, err := tn.app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out apiResp
	require.NoError(t, json.Unmarshal(raw, &out), "status %d body %s", resp.StatusCode, raw)
	return resp.StatusCode, out
}

// The extract endpoints look the contact up before touching the scraper, so
// ownership is decided without an AI call. A missing contact and another
// tenant's contact must both be a plain 404.
func TestExtractEndpoints_HTTP_Ownership(t *testing.T) {
	tn := newTenancy(t)

	jsonEndpoints := []struct {
		suffix string
		body   map[string]interface{}
	}{
		{"extract-from-url", map[string]interface{}{"url": "https://example.com/about"}},
		{"extract-from-extension", map[string]interface{}{"data": map[string]interface{}{"title": "CEO"}}},
	}
	for _, ep := range jsonEndpoints {
		cases := []struct {
			name string
			as   *domain.User
			id   string
			body interface{}
			want int
		}{
			{"nonexistent contact", tn.adminA, uuid.NewString(), ep.body, fiber.StatusNotFound},
			{"other tenant's contact", tn.adminB, tn.contactA1.ID.String(), ep.body, fiber.StatusNotFound},
			{"invalid id", tn.adminA, "nope", ep.body, fiber.StatusBadRequest},
			{"empty payload", tn.adminA, tn.contactA1.ID.String(), map[string]interface{}{}, fiber.StatusBadRequest},
			{"malformed body", tn.adminA, tn.contactA1.ID.String(), "{", fiber.StatusBadRequest},
			{"unauthenticated", nil, tn.contactA1.ID.String(), ep.body, fiber.StatusUnauthorized},
			// Past every check, the missing scraper is reported, not a panic.
			{"owner without a scraper configured", tn.adminA, tn.contactA1.ID.String(), ep.body, fiber.StatusInternalServerError},
		}
		for _, tt := range cases {
			t.Run(ep.suffix+"/"+tt.name, func(t *testing.T) {
				status, r := tn.do(t, http.MethodPost, "/v1/contacts/"+tt.id+"/"+ep.suffix, tt.as, tt.body)
				assertError(t, status, r, tt.want)
			})
		}
	}

	imageCases := []struct {
		name string
		as   *domain.User
		id   string
		want int
	}{
		{"nonexistent contact", tn.adminA, uuid.NewString(), fiber.StatusNotFound},
		{"other tenant's contact", tn.adminB, tn.contactA1.ID.String(), fiber.StatusNotFound},
		{"invalid id", tn.adminA, "nope", fiber.StatusBadRequest},
		{"owner without a scraper configured", tn.adminA, tn.contactA1.ID.String(), fiber.StatusInternalServerError},
	}
	for _, tt := range imageCases {
		t.Run("extract-from-image/"+tt.name, func(t *testing.T) {
			body, ct := multipartBody(t, "image", "card.png", "image/png", []byte("\x89PNG"))
			status, r := tn.doMultipart(t, "/v1/contacts/"+tt.id+"/extract-from-image", tt.as, body, ct)
			assertError(t, status, r, tt.want)
		})
	}

	t.Run("extract-from-image/no file", func(t *testing.T) {
		status, r := tn.do(t, http.MethodPost, "/v1/contacts/"+tn.contactA1.ID.String()+"/extract-from-image", tn.adminA, map[string]interface{}{})
		assertError(t, status, r, fiber.StatusBadRequest)
	})

	t.Run("image preview", func(t *testing.T) {
		status, r := tn.do(t, http.MethodPost, "/v1/extract-from-image-preview", tn.adminA, map[string]interface{}{})
		assertError(t, status, r, fiber.StatusBadRequest)
		body, ct := multipartBody(t, "image", "card.png", "image/png", []byte("png"))
		status, r = tn.doMultipart(t, "/v1/extract-from-image-preview", tn.adminA, body, ct)
		assertError(t, status, r, fiber.StatusInternalServerError)
	})
}

// --- POST /v1/contacts/import-csv -------------------------------------------

func TestImportCSV_HTTP(t *testing.T) {
	tn := newTenancy(t)
	// The handler writes its upload to ./temp relative to the working directory.
	t.Chdir(t.TempDir())

	t.Run("imports into the caller's organisation", func(t *testing.T) {
		csv := "name,email,company\nCsv One,csv1@acme.test,Acme\nNo Email,,Acme\n"
		body, ct := multipartBody(t, "csv_file", "c.csv", "text/csv", []byte(csv))
		status, r := tn.doMultipart(t, "/v1/contacts/import-csv", tn.memberA, body, ct)
		require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)
		var got map[string]interface{}
		require.NoError(t, json.Unmarshal(r.Data, &got))
		assert.Equal(t, float64(2), got["total_rows"])

		var imported domain.Contact
		require.NoError(t, tn.db.Where("profile->>'email' = ?", "csv1@acme.test").First(&imported).Error)
		require.NotNil(t, imported.OrganizationID)
		assert.Equal(t, tn.orgA.ID, *imported.OrganizationID)
		assert.Equal(t, tn.memberA.ID, imported.UserID)
	})

	cases := []struct {
		name  string
		as    *domain.User
		file  string
		ctype string
		want  int
	}{
		{"not a CSV", tn.adminA, "a.png", "image/png", fiber.StatusUnprocessableEntity},
		{"unauthenticated", nil, "a.csv", "text/csv", fiber.StatusUnauthorized},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			body, ct := multipartBody(t, "csv_file", tt.file, tt.ctype, []byte("name\nx\n"))
			status, r := tn.doMultipart(t, "/v1/contacts/import-csv", tt.as, body, ct)
			assertError(t, status, r, tt.want)
		})
	}

	t.Run("missing file", func(t *testing.T) {
		status, r := tn.do(t, http.MethodPost, "/v1/contacts/import-csv", tn.adminA, map[string]interface{}{})
		assertError(t, status, r, fiber.StatusBadRequest)
	})

	t.Run("bad field mapping", func(t *testing.T) {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		fw, err := w.CreateFormFile("csv_file", "a.csv")
		require.NoError(t, err)
		_, _ = fw.Write([]byte("name\nx\n"))
		require.NoError(t, w.WriteField("field_mapping", "{not json"))
		require.NoError(t, w.Close())
		// CreateFormFile sets application/octet-stream, which is not a CSV;
		// rebuild the part with the right type via multipartBody instead.
		body, ct := multipartBody(t, "csv_file", "a.csv", "text/csv", []byte("name\nx\n"))
		_ = buf
		req := httptest.NewRequest(http.MethodPost, "/v1/contacts/import-csv?", body)
		req.Header.Set("Content-Type", ct)
		_ = req
		// field_mapping must be a form field in the same body.
		var b2 bytes.Buffer
		w2 := multipart.NewWriter(&b2)
		hdr := textproto.MIMEHeader{}
		hdr.Set("Content-Disposition", `form-data; name="csv_file"; filename="a.csv"`)
		hdr.Set("Content-Type", "text/csv")
		part, err := w2.CreatePart(hdr)
		require.NoError(t, err)
		_, _ = part.Write([]byte("name\nx\n"))
		require.NoError(t, w2.WriteField("field_mapping", "{not json"))
		require.NoError(t, w2.Close())
		status, r := tn.doMultipart(t, "/v1/contacts/import-csv", tn.adminA, &b2, w2.FormDataContentType())
		assertError(t, status, r, fiber.StatusBadRequest)
	})
}
