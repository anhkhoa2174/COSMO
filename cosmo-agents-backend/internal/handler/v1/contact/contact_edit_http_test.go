package contact

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func profileOf(t *testing.T, raw []byte) map[string]interface{} {
	t.Helper()
	var p map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &p))
	return p
}

// The edit dialogs (all-prospects and the daily-actions panel) post email and
// phone with every save. Both columns are gone, so the handler must route
// them to the profile instead of the UPDATE statement.
func TestUpdate_HTTP_EmailAndPhone(t *testing.T) {
	tn := newTenancy(t)
	path := "/v1/contacts/" + tn.contactA1.ID.String()

	status, r := tn.do(t, http.MethodPatch, path, tn.adminA,
		map[string]interface{}{"email": "alice.new@acme.test", "phone": "+84 90 123 4567", "company": "Acme 2"})
	require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)

	row := tn.reload(t, tn.contactA1.ID)
	p := profileOf(t, row.Profile)
	assert.Equal(t, "alice.new@acme.test", p["email"])
	assert.Equal(t, "+84 90 123 4567", p["phone"])
	assert.Equal(t, "Acme 2", row.Company)
	// contact_information held the old email, so it follows the new one.
	assert.Equal(t, "alice.new@acme.test", row.ContactInformation)

	// A duplicate check on the new address now finds this contact.
	status, r = tn.do(t, http.MethodPost, "/v1/contacts", tn.adminA,
		map[string]interface{}{"name": "Alice Again", "email": "alice.new@acme.test"})
	require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)
	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(r.Data, &got))
	assert.Equal(t, tn.contactA1.ID.String(), got["id"], "merged into the edited contact")
}

func TestUpdate_HTTP_ContactInformationSync(t *testing.T) {
	tn := newTenancy(t)

	t.Run("a hand-set contact_information is kept", func(t *testing.T) {
		require.NoError(t, tn.db.Model(tn.contactA2).Update("contact_information", "https://linkedin.com/in/mark").Error)
		status, r := tn.do(t, http.MethodPatch, "/v1/contacts/"+tn.contactA2.ID.String(), tn.adminA,
			map[string]interface{}{"email": "mark.new@acme.test"})
		require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)
		assert.Equal(t, "https://linkedin.com/in/mark", tn.reload(t, tn.contactA2.ID).ContactInformation)
	})

	t.Run("linkedin_url is stored in the profile, not as a custom field", func(t *testing.T) {
		status, r := tn.do(t, http.MethodPatch, "/v1/contacts/"+tn.contactA2.ID.String(), tn.adminA,
			map[string]interface{}{"linkedin_url": "https://linkedin.com/in/mark-2"})
		require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)
		p := profileOf(t, tn.reload(t, tn.contactA2.ID).Profile)
		assert.Equal(t, "https://linkedin.com/in/mark-2", p["linkedin_url"])
		if cf, ok := p["custom_fields"].(map[string]interface{}); ok {
			assert.NotContains(t, cf, "linkedin_url")
		}
	})
}

// The create form sends its one contact field as contact_information; the
// duplicate check must see it, as it does an email.
func TestCreate_HTTP_DuplicateByContactInformation(t *testing.T) {
	tn := newTenancy(t)
	body := map[string]interface{}{
		"name": "Dup Person", "company": "DupCo", "job_title": "CEO",
		"source": "Email", "contact_information": "dup@example.test",
	}
	status, r := tn.do(t, http.MethodPost, "/v1/contacts", tn.adminA, body)
	require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)
	var first map[string]interface{}
	require.NoError(t, json.Unmarshal(r.Data, &first))

	body["company"] = "DupCo Renamed"
	status, r = tn.do(t, http.MethodPost, "/v1/contacts", tn.adminA, body)
	require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)
	var second map[string]interface{}
	require.NoError(t, json.Unmarshal(r.Data, &second))
	assert.Equal(t, first["id"], second["id"], "resubmitting merges instead of adding a row")

	var n int64
	require.NoError(t, tn.db.Table("contacts").Where("contact_information = ?", "dup@example.test").Count(&n).Error)
	assert.Equal(t, int64(1), n)

	// Another member of the org gets the conflict, not a second copy.
	status, r = tn.do(t, http.MethodPost, "/v1/contacts", tn.memberA, body)
	assertError(t, status, r, fiber.StatusConflict)
}

// The prospects list spreads the profile over the columns, so the merge path
// must keep the two in step or the list shows the pre-merge values.
func TestCreate_HTTP_MergeRefreshesListedValues(t *testing.T) {
	tn := newTenancy(t)
	status, r := tn.do(t, http.MethodPost, "/v1/contacts", tn.adminA,
		map[string]interface{}{"name": "Merge Me", "email": "merge@example.test", "company": "OldCo", "job_title": "CTO", "industry": "SaaS"})
	require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)

	status, r = tn.do(t, http.MethodPost, "/v1/contacts", tn.adminA,
		map[string]interface{}{"name": "Merge Me", "email": "MERGE@example.test", "company": "NewCo", "job_title": "CEO", "industry": "Fintech"})
	require.Equal(t, fiber.StatusOK, status, "error: %+v", r.Error)

	status, r = tn.do(t, http.MethodPost, "/v1/contacts/search", tn.adminA,
		map[string]interface{}{"filter": map[string]interface{}{"name": "Merge Me"}})
	require.Equal(t, fiber.StatusOK, status)
	l := decodeList(t, r)
	require.Len(t, l.List, 1)
	e := l.List[0].Entity
	assert.Equal(t, "NewCo", e["company"])
	assert.Equal(t, "CEO", e["job_title"])
	assert.Equal(t, "Fintech", e["industry"])
}
