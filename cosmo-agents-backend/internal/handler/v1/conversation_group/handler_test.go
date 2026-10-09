package conversation_group

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	agentRepo "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
)

const userHeader = "X-Test-User"

type fixture struct {
	app *fiber.App
	db  *gorm.DB
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Conversation{}, &domain.Agent{}, &domain.Role{},
		&domain.ConversationGroup{}, &domain.ConversationGroupMember{}))
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX uq_conversation_groups_user_name
		ON conversation_groups (user_id, lower(btrim(name)))`).Error)

	h := NewHandler(conversationRepo.NewGroupRepository(db), conversationRepo.NewConversationRepository(db),
		agentRepo.NewAgentRepository(db), roleRepo.NewRoleRepository(db))
	app := fiber.New()
	// Stands in for the auth middleware, which puts the caller in locals.
	app.Use(func(c fiber.Ctx) error {
		if id, err := uuid.Parse(c.Get(userHeader)); err == nil {
			c.Locals("user_id", id)
		}
		return c.Next()
	})
	app.Get("/groups", h.List)
	app.Post("/groups", h.Create)
	app.Patch("/groups/:id", h.Update)
	app.Delete("/groups/:id", h.Delete)
	app.Post("/groups/:id/conversations", h.AddConversation)
	app.Delete("/groups/:id/conversations/:conversation_id", h.RemoveConversation)
	return &fixture{app: app, db: db}
}

func (f *fixture) do(t *testing.T, user uuid.UUID, method, path, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if user != uuid.Nil {
		req.Header.Set(userHeader, user.String())
	}
	resp, err := f.app.Test(req)
	require.NoError(t, err)
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return resp.StatusCode, out
}

func (f *fixture) createGroup(t *testing.T, user uuid.UUID, name string) string {
	t.Helper()
	status, body := f.do(t, user, http.MethodPost, "/groups", `{"name":"`+name+`","color":"blue"}`)
	require.Equal(t, http.StatusCreated, status, body)
	return body["data"].(map[string]any)["id"].(string)
}

func (f *fixture) conversationOf(t *testing.T, owner uuid.UUID) uuid.UUID {
	t.Helper()
	conv := &domain.Conversation{UserID: owner, GmailThreadID: uuid.NewString()}
	require.NoError(t, f.db.Create(conv).Error)
	return conv.ID
}

func TestCreateValidation(t *testing.T) {
	f := newFixture(t)
	me := uuid.New()
	tests := []struct {
		name string
		body string
		want int
	}{
		{"valid", `{"name":"  Acme   Corp ","color":"Teal"}`, http.StatusCreated},
		{"missing name", `{"color":"teal"}`, http.StatusBadRequest},
		{"blank name", `{"name":"   "}`, http.StatusBadRequest},
		{"name too long", `{"name":"` + strings.Repeat("x", 51) + `"}`, http.StatusBadRequest},
		{"unknown colour", `{"name":"Beta","color":"#ff0000"}`, http.StatusBadRequest},
		{"same name, different case", `{"name":"acme corp"}`, http.StatusConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := f.do(t, me, http.MethodPost, "/groups", tt.body)
			assert.Equal(t, tt.want, status, body)
		})
	}
	// Whitespace is collapsed and the colour lower-cased on the way in.
	_, body := f.do(t, me, http.MethodGet, "/groups", "")
	groups := body["data"].([]any)
	require.Len(t, groups, 1)
	assert.Equal(t, "Acme Corp", groups[0].(map[string]any)["name"])
	assert.Equal(t, "teal", groups[0].(map[string]any)["color"])

	status, _ := f.do(t, uuid.Nil, http.MethodGet, "/groups", "")
	assert.Equal(t, http.StatusUnauthorized, status)
}

// Groups are personal: another user's group is not found, whatever is asked
// of it, and the owner's list never includes it.
func TestAnotherUsersGroupIsNotFound(t *testing.T) {
	f := newFixture(t)
	me, other := uuid.New(), uuid.New()
	theirs := f.createGroup(t, other, "Secret")
	conv := f.conversationOf(t, me)

	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPatch, "/groups/" + theirs, `{"name":"Mine now"}`},
		{http.MethodDelete, "/groups/" + theirs, ""},
		{http.MethodPost, "/groups/" + theirs + "/conversations", `{"conversation_id":"` + conv.String() + `"}`},
		{http.MethodDelete, "/groups/" + theirs + "/conversations/" + conv.String(), ""},
	} {
		status, body := f.do(t, me, tc.method, tc.path, tc.body)
		assert.Equal(t, http.StatusNotFound, status, "%s %s: %v", tc.method, tc.path, body)
	}

	_, body := f.do(t, me, http.MethodGet, "/groups", "")
	assert.Empty(t, body["data"])
	var members int64
	f.db.Table("conversation_group_members").Count(&members)
	assert.Zero(t, members, "nothing may be filed in another user's group")
}

func TestAddConversationChecksVisibility(t *testing.T) {
	f := newFixture(t)
	me, stranger, colleague := uuid.New(), uuid.New(), uuid.New()
	group := f.createGroup(t, me, "Pipeline")
	org := uuid.New()

	// A conversation on an agent in my organisation is visible to me.
	agent := &domain.Agent{UserID: colleague, OrganizationID: &org, Email: "team@acme.com"}
	require.NoError(t, f.db.Create(agent).Error)
	require.NoError(t, f.db.Create(&domain.Role{UserID: me, OrganizationID: org, Name: domain.RoleName("member"), Status: domain.RoleStatusActive}).Error)
	teamConv := &domain.Conversation{UserID: colleague, AgentID: &agent.ID, GmailThreadID: uuid.NewString()}
	require.NoError(t, f.db.Create(teamConv).Error)

	tests := []struct {
		name string
		conv string
		want int
	}{
		{"my own conversation", f.conversationOf(t, me).String(), http.StatusOK},
		{"my organisation's conversation", teamConv.ID.String(), http.StatusOK},
		{"a stranger's conversation", f.conversationOf(t, stranger).String(), http.StatusNotFound},
		{"a conversation that does not exist", uuid.NewString(), http.StatusNotFound},
		{"not a uuid", "abc", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := f.do(t, me, http.MethodPost, "/groups/"+group+"/conversations", `{"conversation_id":"`+tt.conv+`"}`)
			assert.Equal(t, tt.want, status, body)
		})
	}

	_, body := f.do(t, me, http.MethodGet, "/groups", "")
	assert.EqualValues(t, 2, body["data"].([]any)[0].(map[string]any)["conversation_count"])
}

func TestRenameAndDelete(t *testing.T) {
	f := newFixture(t)
	me := uuid.New()
	a := f.createGroup(t, me, "Alpha")
	f.createGroup(t, me, "Beta")

	status, body := f.do(t, me, http.MethodPatch, "/groups/"+a, `{"color":"red"}`)
	require.Equal(t, http.StatusOK, status, body)
	assert.Equal(t, "Alpha", body["data"].(map[string]any)["name"], "a field left out keeps its value")
	assert.Equal(t, "red", body["data"].(map[string]any)["color"])

	status, _ = f.do(t, me, http.MethodPatch, "/groups/"+a, `{"name":"BETA"}`)
	assert.Equal(t, http.StatusConflict, status)

	status, _ = f.do(t, me, http.MethodDelete, "/groups/"+a, "")
	assert.Equal(t, http.StatusOK, status)
	status, _ = f.do(t, me, http.MethodDelete, "/groups/"+a, "")
	assert.Equal(t, http.StatusNotFound, status)
}
