package campaign

import (
	"encoding/json"
	"io"
	"net"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	agentRepo "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	campaignRepo "github.com/rockship/cosmo-agents-go/internal/repository/campaign"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	draftTemplateRepo "github.com/rockship/cosmo-agents-go/internal/repository/draft_template"
	notificationRepo "github.com/rockship/cosmo-agents-go/internal/repository/notification"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	templateRepo "github.com/rockship/cosmo-agents-go/internal/repository/template"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

// These tests drive the real handlers over HTTP against a real database. The
// handler holds concrete repository types, so the database is the seam; what
// matters most here is who may read or change which campaign.

type env struct {
	t  *testing.T
	db *gorm.DB
	h  *Handler
}

func newEnv(t *testing.T, client *worker.Client) *env {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&domain.User{}, &domain.Campaign{}, &domain.Role{}, &domain.Template{},
		&domain.Notification{}, &domain.DraftTemplate{}, &domain.Conversation{},
		&domain.Agent{}, &domain.Contact{}, &domain.ListContactAssociation{},
	))
	h := NewHandler(
		campaignRepo.NewCampaignRepository(db),
		roleRepo.NewRoleRepository(db),
		notificationRepo.NewNotificationRepository(db),
		contactRepo.NewListContactRepository(db),
		templateRepo.NewTemplateRepository(db),
		draftTemplateRepo.NewDraftTemplateRepository(db),
		agentRepo.NewAgentRepository(db),
		conversationRepo.NewConversationRepository(db),
		nil,
		client,
	)
	return &env{t: t, db: db, h: h}
}

// app mounts every campaign route behind a stand-in for authentication. A nil
// caller leaves user_id unset, as for an unauthenticated request.
func (e *env) app(caller *uuid.UUID) *fiber.App {
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		if caller != nil {
			c.Locals("user_id", *caller)
		}
		return c.Next()
	})
	app.Post("/campaigns", e.h.Create)
	app.Get("/campaigns", e.h.List)
	app.Post("/campaigns/search", e.h.SearchCampaigns)
	app.Get("/campaigns/:id", e.h.GetByID)
	app.Patch("/campaigns/:id", e.h.Update)
	app.Delete("/campaigns/:id", e.h.Delete)
	app.Post("/campaigns/:id/assign", e.h.AssignMember)
	app.Patch("/campaigns/:id/client-metadata", e.h.UpdateClientMetadata)
	app.Post("/campaigns/:id/follow-up-schedule", e.h.SetFollowUpSchedule)
	app.Patch("/campaigns/:id/save-outreach", e.h.SaveOutreach)
	app.Delete("/campaigns/:id/notifications", e.h.DeleteNotifications)
	app.Post("/campaigns/:id/generate", e.h.GenerateTemplates)
	return app
}

type response struct {
	Status int
	Body   map[string]any
}

func (r response) data() map[string]any {
	d, _ := r.Body["data"].(map[string]any)
	return d
}

func (r response) message() string {
	if e, ok := r.Body["error"].(map[string]any); ok {
		m, _ := e["message"].(string)
		d, _ := e["detail"].(string)
		return m + ": " + d
	}
	return ""
}

func (e *env) do(caller *uuid.UUID, method, path, body string) response {
	e.t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.app(caller).Test(req)
	require.NoError(e.t, err)
	raw, _ := io.ReadAll(resp.Body)
	out := response{Status: resp.StatusCode}
	_ = json.Unmarshal(raw, &out.Body)
	return out
}

// member gives user a role in org.
func (e *env) member(user, org uuid.UUID, name domain.RoleName, status domain.RoleStatus) {
	e.t.Helper()
	require.NoError(e.t, e.db.Create(&domain.Role{UserID: user, OrganizationID: org, Name: name, Status: status}).Error)
}

func (e *env) campaign(owner uuid.UUID, org *uuid.UUID, mutate ...func(*domain.Campaign)) *domain.Campaign {
	e.t.Helper()
	c := &domain.Campaign{UserID: owner, OrganizationID: org, Name: "C", Playbook: "outreach", Status: domain.CampaignStatusDraft}
	for _, m := range mutate {
		m(c)
	}
	require.NoError(e.t, e.db.Create(c).Error)
	return c
}

func (e *env) reload(id uuid.UUID) domain.Campaign {
	e.t.Helper()
	var c domain.Campaign
	require.NoError(e.t, e.db.First(&c, "id = ?", id).Error)
	return c
}

// listWithContacts creates a contact list holding n live contacts.
func (e *env) listWithContacts(owner uuid.UUID, n int) uuid.UUID {
	e.t.Helper()
	listID := uuid.New()
	for i := 0; i < n; i++ {
		c := &domain.Contact{UserID: owner, SourceID: uuid.NewString(), Source: "manual"}
		require.NoError(e.t, e.db.Create(c).Error)
		require.NoError(e.t, e.db.Create(&domain.ListContactAssociation{ListContactID: listID, ContactID: c.ID}).Error)
	}
	return listID
}

func (e *env) template(owner, campaignID uuid.UUID) {
	e.t.Helper()
	require.NoError(e.t, e.db.Create(&domain.Template{UserID: owner, CampaignID: &campaignID, Type: "intro", Subject: "S", Content: "C", Position: 1}).Error)
}

// testRedis returns a queue client on a Redis database of its own and an
// inspector over it, or skips when no Redis is reachable.
func testRedis(t *testing.T) (*worker.Client, *asynq.Inspector) {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6381"
	}
	conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
	if err != nil {
		t.Skipf("no Redis at %s (set TEST_REDIS_ADDR): %v", addr, err)
	}
	_ = conn.Close()
	const db = 10
	client := worker.NewClient(worker.Config{RedisAddr: addr, RedisDB: db})
	insp := asynq.NewInspector(asynq.RedisClientOpt{Addr: addr, DB: db})
	_, _ = insp.DeleteAllPendingTasks("critical")
	t.Cleanup(func() {
		_, _ = insp.DeleteAllPendingTasks("critical")
		_ = insp.Close()
		_ = client.Close()
	})
	return client, insp
}

// newEnvlessApp serves parsePagination alone, reporting what it parsed.
func newEnvlessApp(check func(offset, limit int)) *fiber.App {
	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		check(parsePagination(c, 0, 25))
		return nil
	})
	return app
}
