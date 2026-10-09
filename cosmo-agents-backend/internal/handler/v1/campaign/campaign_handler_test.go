package campaign

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	internalworker "github.com/rockship/cosmo-agents-go/internal/worker/dto"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

func TestRoutesRejectMissingIdentityAndBadIDs(t *testing.T) {
	e := newEnv(t, nil)
	id := uuid.New().String()
	anyone := uuid.New()

	unauthenticated := []struct{ method, path string }{
		{"POST", "/campaigns"},
		{"GET", "/campaigns"},
		{"POST", "/campaigns/search"},
		{"GET", "/campaigns/" + id},
		{"PATCH", "/campaigns/" + id},
		{"DELETE", "/campaigns/" + id},
		{"POST", "/campaigns/" + id + "/assign"},
		{"PATCH", "/campaigns/" + id + "/client-metadata"},
		{"POST", "/campaigns/" + id + "/follow-up-schedule"},
		{"PATCH", "/campaigns/" + id + "/save-outreach"},
		{"DELETE", "/campaigns/" + id + "/notifications"},
	}
	for _, r := range unauthenticated {
		t.Run("no identity "+r.method+" "+r.path, func(t *testing.T) {
			assert.Equal(t, http.StatusUnauthorized, e.do(nil, r.method, r.path, "{}").Status)
		})
	}

	for _, r := range []struct{ method, suffix string }{
		{"GET", ""}, {"PATCH", ""}, {"DELETE", ""}, {"POST", "/assign"},
		{"PATCH", "/client-metadata"}, {"POST", "/follow-up-schedule"},
		{"PATCH", "/save-outreach"}, {"DELETE", "/notifications"},
	} {
		t.Run("bad id "+r.method+r.suffix, func(t *testing.T) {
			assert.Equal(t, http.StatusBadRequest, e.do(&anyone, r.method, "/campaigns/not-a-uuid"+r.suffix, "{}").Status)
		})
	}
}

func TestCreate(t *testing.T) {
	e := newEnv(t, nil)
	user, org, foreignOrg := uuid.New(), uuid.New(), uuid.New()
	e.member(user, org, domain.RoleNameAdmin, domain.RoleStatusActive)
	orphan := uuid.New() // belongs to no organization

	tests := []struct {
		name       string
		caller     uuid.UUID
		body       string
		wantStatus int
	}{
		{"malformed body", user, `{`, http.StatusBadRequest},
		{"missing playbook", user, `{"name":"x"}`, http.StatusBadRequest},
		{"unknown status", user, `{"playbook":"p","status":"exploded"}`, http.StatusBadRequest},
		{"caller without an organization", orphan, `{"playbook":"p"}`, http.StatusBadRequest},
		// Naming another organization would plant the campaign in its members'
		// lists, where they can open and edit it.
		{"organization the caller is not in", user, fmt.Sprintf(`{"playbook":"p","organization_id":%q}`, foreignOrg), http.StatusForbidden},
		{"own organization named explicitly", user, fmt.Sprintf(`{"playbook":"p","organization_id":%q}`, org), http.StatusCreated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := e.do(&tt.caller, "POST", "/campaigns", tt.body)
			assert.Equal(t, tt.wantStatus, r.Status, r.message())
		})
	}

	var planted int64
	e.db.Model(&domain.Campaign{}).Where("organization_id = ?", foreignOrg).Count(&planted)
	assert.Zero(t, planted, "campaign created in a foreign organization")

	t.Run("defaults", func(t *testing.T) {
		r := e.do(&user, "POST", "/campaigns", `{"playbook":"cold-outreach_v2"}`)
		require.Equal(t, http.StatusCreated, r.Status, r.message())
		id, err := uuid.Parse(r.data()["id"].(string))
		require.NoError(t, err)
		c := e.reload(id)
		assert.Equal(t, "Cold Outreach V2", c.Name, "name derived from the playbook")
		assert.Equal(t, domain.CampaignStatusDraft, c.Status)
		require.NotNil(t, c.OrganizationID)
		assert.Equal(t, org, *c.OrganizationID, "primary organization used")
		assert.Equal(t, user, c.UserID)
	})
}

func TestGetByIDAccess(t *testing.T) {
	e := newEnv(t, nil)
	owner, colleague, outsider, roleless := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	org := uuid.New()
	e.member(owner, org, domain.RoleNameAdmin, domain.RoleStatusActive)
	e.member(colleague, org, domain.RoleNameMember, domain.RoleStatusActive)
	e.member(outsider, uuid.New(), domain.RoleNameAdmin, domain.RoleStatusActive)

	c := e.campaign(owner, &org)
	e.template(owner, c.ID)
	require.NoError(t, e.db.Create(&domain.Notification{UserID: colleague, CampaignID: c.ID}).Error)
	private := e.campaign(owner, nil)

	tests := []struct {
		name   string
		caller uuid.UUID
		id     uuid.UUID
		want   int
	}{
		{"owner", owner, c.ID, http.StatusOK},
		{"member of the campaign's organization", colleague, c.ID, http.StatusOK},
		// Not 403: a stranger must not learn the campaign exists.
		{"outsider", outsider, c.ID, http.StatusNotFound},
		{"colleague on a campaign with no organization", colleague, private.ID, http.StatusNotFound},
		{"caller with no roles", roleless, c.ID, http.StatusUnauthorized},
		{"unknown campaign", owner, uuid.New(), http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, e.do(&tt.caller, "GET", "/campaigns/"+tt.id.String(), "").Status)
		})
	}

	r := e.do(&colleague, "GET", "/campaigns/"+c.ID.String(), "")
	d := r.data()
	assert.Len(t, d["templates"], 1)
	assert.Len(t, d["notifications"], 1)
	assert.NotNil(t, d["cmetadata"])
}

func TestDelete(t *testing.T) {
	e := newEnv(t, nil)
	owner, colleague := uuid.New(), uuid.New()
	org := uuid.New()
	c := e.campaign(owner, &org)
	e.member(owner, org, domain.RoleNameAdmin, domain.RoleStatusActive)
	e.member(colleague, org, domain.RoleNameMember, domain.RoleStatusActive)

	assert.Equal(t, http.StatusForbidden, e.do(&colleague, "DELETE", "/campaigns/"+c.ID.String(), "").Status, "only the owner deletes")
	assert.False(t, e.reload(c.ID).IsDeleted)
	assert.Equal(t, http.StatusNotFound, e.do(&owner, "DELETE", "/campaigns/"+uuid.NewString(), "").Status)

	assert.Equal(t, http.StatusOK, e.do(&owner, "DELETE", "/campaigns/"+c.ID.String(), "").Status)
	assert.True(t, e.reload(c.ID).IsDeleted)
	assert.Equal(t, http.StatusNotFound, e.do(&owner, "GET", "/campaigns/"+c.ID.String(), "").Status)
}

func TestUpdateValidation(t *testing.T) {
	e := newEnv(t, nil)
	owner, outsider, orphan := uuid.New(), uuid.New(), uuid.New()
	org := uuid.New()
	e.member(owner, org, domain.RoleNameAdmin, domain.RoleStatusActive)
	e.member(outsider, uuid.New(), domain.RoleNameAdmin, domain.RoleStatusActive)
	agent := uuid.New()
	emptyList := uuid.New()
	fullList := e.listWithContacts(owner, 2)

	noTemplates := e.campaign(owner, &org, func(c *domain.Campaign) { c.ListContactID = &fullList; c.AgentID = &agent })
	c := e.campaign(owner, &org)
	path := "/campaigns/" + c.ID.String()

	tests := []struct {
		name   string
		caller uuid.UUID
		path   string
		body   string
		want   int
		detail string
	}{
		{"caller without organization", orphan, path, `{}`, http.StatusUnauthorized, ""},
		{"outsider", outsider, path, `{"name":"x"}`, http.StatusForbidden, ""},
		{"unknown campaign", owner, "/campaigns/" + uuid.NewString(), `{}`, http.StatusNotFound, ""},
		{"malformed body", owner, path, `{`, http.StatusBadRequest, ""},
		{"invalid status", owner, path, `{"status":"exploded"}`, http.StatusBadRequest, ""},
		{"schedule in the past", owner, path, `{"schedule":"2001-01-01T00:00:00Z"}`, http.StatusBadRequest, "schedule in the past"},
		{"activate without a list", owner, path, `{"status":"active"}`, http.StatusBadRequest, "list contact is required"},
		{"activate an empty list", owner, path, fmt.Sprintf(`{"status":"active","list_contact_id":%q}`, emptyList), http.StatusBadRequest, "list contact is empty"},
		{"activate without an agent", owner, path, fmt.Sprintf(`{"status":"scheduled","list_contact_id":%q}`, fullList), http.StatusBadRequest, "agent is required"},
		{"activate without templates", owner, "/campaigns/" + noTemplates.ID.String(), `{"status":"active"}`, http.StatusBadRequest, "no templates"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := e.do(&tt.caller, "PATCH", tt.path, tt.body)
			assert.Equal(t, tt.want, r.Status, r.message())
			assert.Contains(t, r.message(), tt.detail)
		})
	}
	// None of the rejected updates reached the row.
	got := e.reload(c.ID)
	assert.Equal(t, domain.CampaignStatusDraft, got.Status)
	assert.Equal(t, "C", got.Name)
	assert.Nil(t, got.ListContactID)
}

func TestUpdateFields(t *testing.T) {
	e := newEnv(t, nil)
	owner, colleague := uuid.New(), uuid.New()
	org := uuid.New()
	e.member(owner, org, domain.RoleNameAdmin, domain.RoleStatusActive)
	e.member(colleague, org, domain.RoleNameAdmin, domain.RoleStatusActive)
	c := e.campaign(owner, nil, func(c *domain.Campaign) {
		c.CMetadata.Client = map[string]interface{}{"keep": "me", "nested": map[string]interface{}{"a": 1.0}}
	})
	future := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)

	body := fmt.Sprintf(`{"name":"Renamed","status":"paused","schedule":%q,"cmetadata":{"client":{"new":"v","nested":{"b":2}}}}`, future.Format(time.RFC3339))
	r := e.do(&owner, "PATCH", "/campaigns/"+c.ID.String(), body)
	require.Equal(t, http.StatusOK, r.Status, r.message())

	got := e.reload(c.ID)
	assert.Equal(t, "Renamed", got.Name)
	assert.Equal(t, domain.CampaignStatusPaused, got.Status)
	require.NotNil(t, got.Schedule)
	assert.True(t, got.Schedule.Equal(future))
	// Metadata is merged, not replaced.
	assert.Equal(t, "me", got.CMetadata.Client["keep"])
	assert.Equal(t, "v", got.CMetadata.Client["new"])
	assert.Equal(t, map[string]interface{}{"a": 1.0, "b": 2.0}, got.CMetadata.Client["nested"])

	// The campaign had no organization, so the owner's is not someone else's
	// licence to edit it.
	assert.Equal(t, http.StatusForbidden, e.do(&colleague, "PATCH", "/campaigns/"+c.ID.String(), `{"name":"x"}`).Status)
}

// activatable returns a campaign that passes every activation check.
func activatable(e *env, owner uuid.UUID, org uuid.UUID) (*domain.Campaign, uuid.UUID) {
	agent := uuid.New()
	list := e.listWithContacts(owner, 1)
	c := e.campaign(owner, &org, func(c *domain.Campaign) { c.ListContactID = &list; c.AgentID = &agent })
	e.template(owner, c.ID)
	return c, agent
}

func TestUpdateActivationEnqueuesExecution(t *testing.T) {
	client, insp := testRedis(t)
	e := newEnv(t, client)
	owner, org := uuid.New(), uuid.New()
	e.member(owner, org, domain.RoleNameAdmin, domain.RoleStatusActive)
	c, agent := activatable(e, owner, org)

	r := e.do(&owner, "PATCH", "/campaigns/"+c.ID.String(), `{"status":"active"}`)
	require.Equal(t, http.StatusOK, r.Status, r.message())
	assert.Equal(t, domain.CampaignStatusActive, e.reload(c.ID).Status)

	tasks, err := insp.ListPendingTasks(worker.QueueCritical)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, worker.TypeExecuteCampaign, tasks[0].Type)
	var p internalworker.ExecuteCampaignPayload
	require.NoError(t, json.Unmarshal(tasks[0].Payload, &p))
	assert.Equal(t, c.ID, p.CampaignID)
	assert.Equal(t, owner, p.UserID)
	assert.Equal(t, agent, p.AgentID)

	// Re-sending active is not a transition and does not execute twice.
	require.Equal(t, http.StatusOK, e.do(&owner, "PATCH", "/campaigns/"+c.ID.String(), `{"status":"active"}`).Status)
	tasks, _ = insp.ListPendingTasks(worker.QueueCritical)
	assert.Len(t, tasks, 1)
}

// When the execution cannot be queued the request fails, and the campaign must
// not be left "active": a retry of the same PATCH is then no transition, so
// nothing would ever run it.
func TestUpdateActivationFailureLeavesStatus(t *testing.T) {
	tests := []struct {
		name   string
		client *worker.Client
	}{
		{"no worker client", nil},
		// Nothing listens on port 1, so every enqueue fails.
		{"queue unreachable", worker.NewClient(worker.Config{RedisAddr: "127.0.0.1:1"})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t, tt.client)
			owner, org := uuid.New(), uuid.New()
			e.member(owner, org, domain.RoleNameAdmin, domain.RoleStatusActive)
			c, _ := activatable(e, owner, org)

			r := e.do(&owner, "PATCH", "/campaigns/"+c.ID.String(), `{"status":"active"}`)
			assert.Equal(t, http.StatusInternalServerError, r.Status)
			assert.Equal(t, domain.CampaignStatusDraft, e.reload(c.ID).Status)
		})
	}
}

func TestListAndSearch(t *testing.T) {
	e := newEnv(t, nil)
	user, colleague, orphan := uuid.New(), uuid.New(), uuid.New()
	org := uuid.New()
	e.member(user, org, domain.RoleNameAdmin, domain.RoleStatusActive)
	e.member(colleague, org, domain.RoleNameMember, domain.RoleStatusActive)

	mine := e.campaign(user, &org, func(c *domain.Campaign) { c.Status = domain.CampaignStatusActive })
	theirs := e.campaign(colleague, &org)
	e.campaign(uuid.New(), nil) // a stranger's
	deleted := e.campaign(user, &org)
	e.db.Model(deleted).Update("is_deleted", true)

	ids := func(r response) []string {
		var out []string
		list, _ := r.data()["list"].([]any)
		for _, item := range list {
			out = append(out, item.(map[string]any)["entity"].(map[string]any)["id"].(string))
		}
		return out
	}

	r := e.do(&user, "GET", "/campaigns", "")
	require.Equal(t, http.StatusOK, r.Status, r.message())
	assert.ElementsMatch(t, []string{mine.ID.String(), theirs.ID.String()}, ids(r))
	assert.EqualValues(t, 2, r.data()["total"])

	r = e.do(&user, "GET", "/campaigns?status=active", "")
	assert.Equal(t, []string{mine.ID.String()}, ids(r))
	r = e.do(&user, "GET", "/campaigns?organization_id="+org.String(), "")
	assert.Len(t, ids(r), 2)

	assert.Equal(t, http.StatusForbidden, e.do(&user, "GET", "/campaigns?organization_id="+uuid.NewString(), "").Status)
	assert.Equal(t, http.StatusBadRequest, e.do(&user, "GET", "/campaigns?organization_id=nope", "").Status)
	assert.Equal(t, http.StatusUnauthorized, e.do(&orphan, "GET", "/campaigns", "").Status)

	// Search only ever covers the caller's own campaigns.
	r = e.do(&user, "POST", "/campaigns/search", "")
	require.Equal(t, http.StatusOK, r.Status, r.message())
	assert.Equal(t, []string{mine.ID.String()}, ids(r))
	r = e.do(&user, "POST", "/campaigns/search", `{"filter":{"status":"draft"}}`)
	assert.Empty(t, ids(r))
	assert.Equal(t, http.StatusBadRequest, e.do(&user, "POST", "/campaigns/search", `{"filter":`).Status)
}

// The search filter is handed to the generic filter builder, whose "$raw" key
// is spliced into the WHERE clause verbatim. From the request body that is SQL
// injection: here it answers a yes/no question about another table.
func TestSearchRejectsRawSQL(t *testing.T) {
	e := newEnv(t, nil)
	user := uuid.New()
	e.campaign(user, nil)
	e.campaign(uuid.New(), nil) // someone else's
	require.NoError(t, e.db.Create(&domain.User{Base: domain.Base{ID: uuid.New()}, Email: "victim@secret.test"}).Error)

	for _, body := range []string{
		`{"filter":{"$raw":"EXISTS (SELECT 1 FROM users WHERE email LIKE 'victim%')"}}`,
		// Escapes the user_id scope: returned every user's campaigns.
		`{"filter":{"$raw":"1=1) OR (1=1"}}`,
		`{"filter":{"$or":[{"$raw":"1=1"}]}}`,
		`{"filter":{"$and":[{"status":"draft"},{"$or":[{"$raw":"1=1"}]}]}}`,
	} {
		r := e.do(&user, "POST", "/campaigns/search", body)
		assert.Equal(t, http.StatusBadRequest, r.Status, body)
	}
	// Plain logical operators over real columns still work.
	r := e.do(&user, "POST", "/campaigns/search", `{"filter":{"$or":[{"status":"draft"},{"status":"active"}]}}`)
	require.Equal(t, http.StatusOK, r.Status, r.message())
	assert.EqualValues(t, 1, r.data()["total"])
}

func TestAssignMember(t *testing.T) {
	e := newEnv(t, nil)
	user, orphan := uuid.New(), uuid.New()
	org := uuid.New()
	e.member(user, org, domain.RoleNameAdmin, domain.RoleStatusActive)
	c := e.campaign(user, &org)
	foreign := e.campaign(uuid.New(), func() *uuid.UUID { o := uuid.New(); return &o }())
	path := "/campaigns/" + c.ID.String() + "/assign"
	ok := `{"config":[{"who":"Let AI reply","intent_type":"Interested","payload":{}},{"who":"Assign to a person","intent_type":"Referral","payload":{"user":"x"}}]}`

	tests := []struct {
		name   string
		caller uuid.UUID
		path   string
		body   string
		want   int
	}{
		{"caller without organization", orphan, path, ok, http.StatusForbidden},
		{"malformed body", user, path, `{`, http.StatusBadRequest},
		{"empty config", user, path, `{"config":[]}`, http.StatusBadRequest},
		{"unknown handler", user, path, `{"config":[{"who":"Robot","intent_type":"Interested","payload":{}}]}`, http.StatusBadRequest},
		{"unknown intent", user, path, `{"config":[{"who":"Let AI reply","intent_type":"Angry","payload":{}}]}`, http.StatusBadRequest},
		{"duplicate intent", user, path, `{"config":[{"who":"Let AI reply","intent_type":"Interested","payload":{}},{"who":"Draft an email","intent_type":" interested ","payload":{}}]}`, http.StatusBadRequest},
		{"unknown campaign", user, "/campaigns/" + uuid.NewString() + "/assign", ok, http.StatusNotFound},
		{"another organization's campaign", user, "/campaigns/" + foreign.ID.String() + "/assign", ok, http.StatusForbidden},
		{"valid", user, path, ok, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := e.do(&tt.caller, "POST", tt.path, tt.body)
			assert.Equal(t, tt.want, r.Status, r.message())
		})
	}

	cfg := e.reload(c.ID).CMetadata.Config
	require.Len(t, cfg, 2)
	assert.Equal(t, domain.HandlerAI, cfg[0].Who)
	assert.Equal(t, domain.HandlerHuman, cfg[1].Who)
	assert.Equal(t, domain.IntentType("Referral"), cfg[1].IntentType)
	assert.Empty(t, e.reload(foreign.ID).CMetadata.Config)
}

func TestMetadataEndpointsAccess(t *testing.T) {
	e := newEnv(t, nil)
	owner, colleague, pending, outsider := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	org := uuid.New()
	e.member(owner, org, domain.RoleNameAdmin, domain.RoleStatusActive)
	e.member(colleague, org, domain.RoleNameMember, domain.RoleStatusActive)
	e.member(pending, org, domain.RoleNameMember, domain.RoleStatusPending)
	c := e.campaign(owner, &org)
	private := e.campaign(owner, nil)
	id := c.ID.String()

	tests := []struct {
		name   string
		caller uuid.UUID
		method string
		path   string
		body   string
		want   int
	}{
		{"client metadata: colleague", colleague, "PATCH", "/campaigns/" + id + "/client-metadata", `{"client":{"tone":"warm"}}`, http.StatusOK},
		{"client metadata: outsider", outsider, "PATCH", "/campaigns/" + id + "/client-metadata", `{"client":{"tone":"cold"}}`, http.StatusForbidden},
		{"client metadata: campaign without organization", colleague, "PATCH", "/campaigns/" + private.ID.String() + "/client-metadata", `{"client":{}}`, http.StatusForbidden},
		{"client metadata: malformed", owner, "PATCH", "/campaigns/" + id + "/client-metadata", `{`, http.StatusBadRequest},
		{"client metadata: unknown", owner, "PATCH", "/campaigns/" + uuid.NewString() + "/client-metadata", `{"client":{}}`, http.StatusNotFound},

		{"follow-up: owner", owner, "POST", "/campaigns/" + id + "/follow-up-schedule", `{"follow_up_1_schedule":3,"follow_up_2_schedule":7}`, http.StatusOK},
		{"follow-up: colleague is not the owner", colleague, "POST", "/campaigns/" + id + "/follow-up-schedule", `{"follow_up_1_schedule":1}`, http.StatusNotFound},
		{"follow-up: malformed", owner, "POST", "/campaigns/" + id + "/follow-up-schedule", `{`, http.StatusBadRequest},
		{"follow-up: unknown", owner, "POST", "/campaigns/" + uuid.NewString() + "/follow-up-schedule", `{}`, http.StatusNotFound},

		{"outreach: colleague", colleague, "PATCH", "/campaigns/" + id + "/save-outreach", `{"sequence":[{"type":"intro","subject":"S","content":"B"},{"type":"follow"}]}`, http.StatusOK},
		// An invitation not yet accepted grants nothing.
		{"outreach: pending member", pending, "PATCH", "/campaigns/" + id + "/save-outreach", `{"sequence":[{"type":"x"}]}`, http.StatusForbidden},
		{"outreach: campaign without organization", colleague, "PATCH", "/campaigns/" + private.ID.String() + "/save-outreach", `{"sequence":[{"type":"x"}]}`, http.StatusForbidden},
		{"outreach: empty sequence", owner, "PATCH", "/campaigns/" + id + "/save-outreach", `{"sequence":[]}`, http.StatusBadRequest},
		{"outreach: malformed", owner, "PATCH", "/campaigns/" + id + "/save-outreach", `{`, http.StatusBadRequest},
		{"outreach: unknown", owner, "PATCH", "/campaigns/" + uuid.NewString() + "/save-outreach", `{"sequence":[{"type":"x"}]}`, http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := e.do(&tt.caller, tt.method, tt.path, tt.body)
			assert.Equal(t, tt.want, r.Status, r.message())
		})
	}

	got := e.reload(c.ID).CMetadata
	assert.Equal(t, "warm", got.Client["tone"], "the outsider's write must not land")
	assert.EqualValues(t, 3, got.Client["follow_up_1_schedule"])
	assert.EqualValues(t, 7, got.Client["follow_up_2_schedule"])
	require.Len(t, got.Sequence, 2)
	assert.Equal(t, "S", got.Sequence[0].Subject)
	assert.Equal(t, "", got.Sequence[1].Subject)
}

func TestDeleteNotifications(t *testing.T) {
	e := newEnv(t, nil)
	owner, colleague := uuid.New(), uuid.New()
	c := e.campaign(owner, nil)
	n := &domain.Notification{UserID: colleague, CampaignID: c.ID}
	require.NoError(t, e.db.Create(n).Error)
	path := "/campaigns/" + c.ID.String() + "/notifications"
	body := fmt.Sprintf(`{"ids":[%q]}`, n.ID)

	assert.Equal(t, http.StatusBadRequest, e.do(&owner, "DELETE", path, `{"ids":[]}`).Status)
	assert.Equal(t, http.StatusBadRequest, e.do(&owner, "DELETE", path, `{`).Status)
	assert.Equal(t, http.StatusNotFound, e.do(&colleague, "DELETE", path, body).Status)
	var count int64
	e.db.Model(&domain.Notification{}).Where("id = ?", n.ID).Count(&count)
	assert.EqualValues(t, 1, count)

	assert.Equal(t, http.StatusOK, e.do(&owner, "DELETE", path, body).Status)
	e.db.Model(&domain.Notification{}).Where("id = ? AND is_deleted = ?", n.ID, false).Count(&count)
	assert.Zero(t, count)
}

func TestGenerateTemplatesWithoutAIService(t *testing.T) {
	e := newEnv(t, nil)
	user := uuid.New()
	path := "/campaigns/" + uuid.NewString() + "/generate"
	assert.Equal(t, http.StatusBadRequest, e.do(&user, "POST", path, `{`).Status)
	assert.Equal(t, http.StatusBadRequest, e.do(&user, "POST", path, `{}`).Status)
	assert.Equal(t, http.StatusServiceUnavailable, e.do(&user, "POST", path, `{"client_data":{"first_name":"Ada"}}`).Status)
}

func TestSyncNotifications(t *testing.T) {
	e := newEnv(t, nil)
	owner, colleague, stranger := uuid.New(), uuid.New(), uuid.New()
	org := uuid.New()
	e.member(owner, org, domain.RoleNameAdmin, domain.RoleStatusActive)
	e.member(colleague, org, domain.RoleNameMember, domain.RoleStatusActive)
	c := e.campaign(owner, &org)
	ctx := t.Context()

	_, err := e.h.syncNotifications(ctx, c.ID, nil, owner, nil)
	assert.Error(t, err, "campaign without organization")

	_, err = e.h.syncNotifications(ctx, c.ID, &org, owner, []uuid.UUID{stranger})
	assert.ErrorContains(t, err, "not found in organization")

	created, err := e.h.syncNotifications(ctx, c.ID, &org, owner, []uuid.UUID{colleague})
	require.NoError(t, err)
	var subscribers []uuid.UUID
	for _, n := range created {
		subscribers = append(subscribers, n.UserID)
	}
	assert.ElementsMatch(t, []uuid.UUID{owner, colleague}, subscribers, "the owner is always subscribed")
}

func TestHelpers(t *testing.T) {
	t.Run("parsePagination", func(t *testing.T) {
		tests := []struct {
			query              string
			wantOffset, wantLi int
		}{
			{"", 0, 25},
			{"offset=50&limit=10", 50, 10},
			{"limit=0", 0, 25},
			{"limit=-5&offset=-1", 0, 25},
			{"page_size=40", 0, 40},
			{"pageSize=30", 0, 30},
			{"per_page=20", 0, 20},
			{"page=3&limit=10", 20, 10},
			{"page_index=2&limit=10", 20, 10},
			{"pageIndex=1&limit=5", 5, 5},
			// offset below limit is read as a page index
			{"offset=2&limit=10", 20, 10},
			{"offset=junk&limit=junk", 0, 25},
		}
		for _, tt := range tests {
			app := newEnvlessApp(func(offset, limit int) {
				assert.Equal(t, tt.wantOffset, offset, tt.query)
				assert.Equal(t, tt.wantLi, limit, tt.query)
			})
			req, _ := http.NewRequest("GET", "/?"+tt.query, nil)
			_, err := app.Test(req)
			require.NoError(t, err)
		}
	})
	t.Run("deriveCampaignName", func(t *testing.T) {
		assert.Equal(t, "Re Engagement Q3", deriveCampaignName(" re-engagement_q3 "))
		assert.Equal(t, "", deriveCampaignName(""))
	})
	t.Run("normalizeHandler", func(t *testing.T) {
		assert.Equal(t, domain.HandlerAI, normalizeHandler(" let ai reply"))
		assert.Equal(t, domain.HandlerHuman, normalizeHandler("HUMAN"))
		assert.Equal(t, domain.HandlerDraft, normalizeHandler("Draft an email"))
		assert.Equal(t, domain.Handler("other"), normalizeHandler("other"))
	})
	t.Run("campaignMetadataToMap", func(t *testing.T) {
		assert.Contains(t, campaignMetadataToMap(nil), "config")
		assert.Equal(t, map[string]interface{}{"x": 1}, campaignMetadataToMap(map[string]interface{}{"x": 1}))
		assert.Contains(t, campaignMetadataToMap(42), "config")
		assert.Contains(t, campaignMetadataToMap(domain.CampaignMetadata{}), "sequence")
		assert.NotNil(t, ensureEmptyMap(nil))
		assert.Nil(t, stringPtrOrNil(""))
	})
	t.Run("campaign metadata round trip", func(t *testing.T) {
		assert.Equal(t, domain.CampaignMetadata{}, campaignMetadataFromMap(nil))
		m := metadataToMap(domain.CampaignMetadata{Client: map[string]interface{}{"a": "b"}})
		assert.Equal(t, "b", campaignMetadataFromMap(m).Client["a"])
		assert.Equal(t, map[string]interface{}{"a": 1}, mergeMaps(nil, map[string]interface{}{"a": 1}))
	})
	t.Run("replacePlaceholders", func(t *testing.T) {
		got := replacePlaceholders("Hi {{first_name}} of {{company_name}}, {{sender_name}}", map[string]interface{}{"first_name": "Ada", "company_name": 7})
		assert.Equal(t, "Hi Ada of Acme Corp, Sales Team", got)
		assert.True(t, strings.HasPrefix(buildContextFromClientData(map[string]interface{}{"a": 1}), "Client Data:"))
		assert.Equal(t, "", buildContextFromClientData(map[string]interface{}{"f": func() {}}))
	})
	t.Run("createCampaignDetailResponse nil", func(t *testing.T) {
		assert.Equal(t, uuid.Nil, (&Handler{}).createCampaignDetailResponse(nil, nil, nil, nil).ID)
	})
	t.Run("error helpers without a context", func(t *testing.T) {
		assert.Error(t, badRequest(nil, "m", nil))
		assert.Error(t, internalError(nil, "m", fmt.Errorf("x")))
	})
}

// The email worker sends through whatever agent a campaign names, with that
// mailbox's Gmail token. Naming another tenant's agent used to be accepted,
// so a user could mail from someone else's inbox.
func TestCampaignAgentMustBeUsableByCaller(t *testing.T) {
	e := newEnv(t, nil)
	user, org, otherOrg := uuid.New(), uuid.New(), uuid.New()
	e.member(user, org, domain.RoleNameMember, domain.RoleStatusActive)

	agent := func(owner uuid.UUID, agentOrg *uuid.UUID) uuid.UUID {
		a := &domain.Agent{UserID: owner, OrganizationID: agentOrg, Email: uuid.NewString() + "@mail.test"}
		require.NoError(t, e.db.Create(a).Error)
		return a.ID
	}
	own := agent(user, nil)
	teams := agent(uuid.New(), &org)
	strangers := agent(uuid.New(), &otherOrg)

	for _, tc := range []struct {
		name  string
		agent uuid.UUID
		want  int
	}{
		{"my own mailbox", own, http.StatusCreated},
		{"a mailbox of my organisation", teams, http.StatusCreated},
		{"another tenant's mailbox", strangers, http.StatusForbidden},
		{"an agent that does not exist", uuid.New(), http.StatusForbidden},
	} {
		t.Run("create with "+tc.name, func(t *testing.T) {
			r := e.do(&user, "POST", "/campaigns", fmt.Sprintf(`{"playbook":"p","agent_id":%q}`, tc.agent))
			assert.Equal(t, tc.want, r.Status, r.message())
		})
	}

	r := e.do(&user, "POST", "/campaigns", `{"playbook":"p"}`)
	require.Equal(t, http.StatusCreated, r.Status, r.message())
	id := r.data()["id"].(string)
	r = e.do(&user, "PATCH", "/campaigns/"+id, fmt.Sprintf(`{"agent_id":%q}`, strangers))
	assert.Equal(t, http.StatusForbidden, r.Status, "update must not switch to another tenant's mailbox: %s", r.message())
}
