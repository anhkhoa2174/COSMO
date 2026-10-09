package conversation

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	emailRepo "github.com/rockship/cosmo-agents-go/internal/repository/email"
	feedbackRepo "github.com/rockship/cosmo-agents-go/internal/repository/feedback"
)

// These tests drive the real handler over a real HTTP round trip against a real
// database, rather than asserting against a stub that re-implements the
// behaviour under test. The handler is built with concrete repository types, so
// substituting them is not available; an in-memory SQLite database is, and it
// is the pattern the repository tests in this project already use. A test that
// passes here has executed the SQL, not merely the branching.
//
// The case that matters most is the ownership check. Intent correction mutates
// a conversation, an email and a contact's suppression flag, so a missing
// permission check would let one user rewrite another's records.

func newTestHandler(t *testing.T) (*ConversationHandler, *gorm.DB) {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(&domain.Conversation{}, &domain.Email{},
		&domain.UserFeedback{}))
	// Contact carries a GIN index on its JSONB profile, which SQLite cannot
	// express. The table itself is created; only the index statement fails, so
	// the error is deliberately ignored rather than the whole model dropped.
	_ = db.AutoMigrate(&domain.Contact{})
	require.True(t, db.Migrator().HasTable(&domain.Contact{}))

	return NewConversationHandler(
		conversationRepo.NewConversationRepository(db),
		emailRepo.NewEmailRepository(db),
		nil,
		contactRepo.NewContactRepository(db),
		feedbackRepo.NewRepository(db),
	), db
}

// newApp mounts the route under a middleware that stands in for authentication,
// so the handler reads the caller's identity exactly as it does in production.
func newApp(h *ConversationHandler, caller uuid.UUID) *fiber.App {
	app := fiber.New()
	app.Post("/v1/conversations/:id/intent", func(c fiber.Ctx) error {
		c.Locals("user_id", caller)
		return h.CorrectIntent(c)
	})
	return app
}

func post(t *testing.T, app *fiber.App, id, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest("POST", "/v1/conversations/"+id+"/intent",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return resp.StatusCode, out
}

// seed writes a conversation the classifier has already labelled, together with
// the inbound email it looked at.
func seed(t *testing.T, db *gorm.DB, owner uuid.UUID, intent string) (*domain.Conversation, *domain.Email) {
	t.Helper()
	email := &domain.Email{
		UserID:    owner,
		FromEmail: "prospect@example.com",
		ToEmail:   "rep@cosmo.test",
		Subject:   "Re: intro",
		Intents:   []string{intent},
	}
	require.NoError(t, db.Create(email).Error)

	conv := &domain.Conversation{
		UserID:        owner,
		GmailThreadID: uuid.NewString(),
		Intents:       []string{intent},
	}
	meta := map[string]any{
		"ai_reply": map[string]any{"draft_content": "Thanks for your note."},
		"intent_detail": map[string]any{
			"intent":     intent,
			"confidence": 0.91,
			"reasoning":  `The prospect wrote "please remove me".`,
			"email_id":   email.ID.String(),
		},
	}
	require.NoError(t, conv.CMetadata.Marshal(meta))
	require.NoError(t, db.Create(conv).Error)
	return conv, email
}

func detailOf(t *testing.T, db *gorm.DB, id uuid.UUID) map[string]any {
	t.Helper()
	var conv domain.Conversation
	require.NoError(t, db.First(&conv, "id = ?", id).Error)
	var meta map[string]any
	require.NoError(t, conv.CMetadata.Unmarshal(&meta))
	d, _ := meta["intent_detail"].(map[string]any)
	return d
}

func TestCorrectIntent_RejectsAnotherUsersConversation(t *testing.T) {
	h, db := newTestHandler(t)
	owner, intruder := uuid.New(), uuid.New()
	conv, _ := seed(t, db, owner, string(domain.IntentInterested))

	code, _ := post(t, newApp(h, intruder), conv.ID.String(),
		`{"intent":"Do not contact"}`)
	assert.Equal(t, fiber.StatusForbidden, code)

	// And the label must be untouched, not merely the response refused.
	var after domain.Conversation
	require.NoError(t, db.First(&after, "id = ?", conv.ID).Error)
	assert.Equal(t, []string{string(domain.IntentInterested)}, []string(after.Intents))
}

func TestCorrectIntent_RejectsLabelOutsideTaxonomy(t *testing.T) {
	h, db := newTestHandler(t)
	owner := uuid.New()
	conv, _ := seed(t, db, owner, string(domain.IntentInterested))

	for _, bad := range []string{`{"intent":"Very interested"}`, `{"intent":""}`} {
		code, _ := post(t, newApp(h, owner), conv.ID.String(), bad)
		assert.Equal(t, fiber.StatusBadRequest, code, bad)
	}
}

func TestCorrectIntent_RelabelsAndRetiresTheClassifiersEvidence(t *testing.T) {
	h, db := newTestHandler(t)
	owner := uuid.New()
	conv, email := seed(t, db, owner, string(domain.IntentDoNotContact))

	code, body := post(t, newApp(h, owner), conv.ID.String(),
		`{"intent":"Interested","note":"terse reply, classifier misread it"}`)
	require.Equal(t, fiber.StatusOK, code)

	data, _ := body["data"].(map[string]any)
	require.NotNil(t, data)
	assert.Equal(t, string(domain.IntentInterested), data["intent"])
	assert.Equal(t, string(domain.IntentDoNotContact), data["previous_intent"])
	assert.Equal(t, true, data["changed"])
	// A draft written under the old label is reported, never silently rewritten.
	assert.Equal(t, true, data["draft_stale"])

	var after domain.Conversation
	require.NoError(t, db.First(&after, "id = ?", conv.ID).Error)
	assert.Equal(t, []string{string(domain.IntentInterested)}, []string(after.Intents))

	// The email the badge reads from must follow, or the thread shows two labels.
	var afterEmail domain.Email
	require.NoError(t, db.First(&afterEmail, "id = ?", email.ID).Error)
	assert.Equal(t, []string{string(domain.IntentInterested)}, []string(afterEmail.Intents))

	d := detailOf(t, db, conv.ID)
	assert.Equal(t, string(domain.IntentInterested), d["intent"])
	assert.Equal(t, true, d["corrected_by_user"])
	// The quoted sentences argued for the label the human just rejected. They
	// are kept for audit and withdrawn from display, never re-pointed at the
	// new label.
	assert.NotContains(t, d, "reasoning")
	assert.NotContains(t, d, "confidence")
	assert.Contains(t, d, "original_reasoning")
	assert.Equal(t, string(domain.IntentDoNotContact), d["original_intent"])

	var fb []domain.UserFeedback
	require.NoError(t, db.Find(&fb, "entity_id = ?", conv.ID).Error)
	require.Len(t, fb, 1)
	assert.Equal(t, "intent_correction", fb[0].FeedbackType)
	assert.Equal(t, owner, fb[0].UserID)
}

func TestCorrectIntent_MovesContactSuppressionBothWays(t *testing.T) {
	t.Run("lifts it when correcting away from Do Not Contact", func(t *testing.T) {
		h, db := newTestHandler(t)
		owner := uuid.New()
		conv, _ := seed(t, db, owner, string(domain.IntentDoNotContact))
		org := uuid.New()
		c := &domain.Contact{UserID: owner, OrganizationID: &org,
			ContactInformation: "prospect@example.com", DoNotContact: true}
		require.NoError(t, db.Create(c).Error)

		code, body := post(t, newApp(h, owner), conv.ID.String(),
			`{"intent":"Interested"}`)
		require.Equal(t, fiber.StatusOK, code)
		data, _ := body["data"].(map[string]any)
		assert.Equal(t, "unsuppressed", data["suppression_changed"])

		var after domain.Contact
		require.NoError(t, db.First(&after, "id = ?", c.ID).Error)
		assert.False(t, after.DoNotContact,
			"a contact suppressed by a label the human rejected must be reachable again")
	})

	t.Run("applies it when correcting to Do Not Contact", func(t *testing.T) {
		h, db := newTestHandler(t)
		owner := uuid.New()
		conv, _ := seed(t, db, owner, string(domain.IntentInterested))
		org := uuid.New()
		c := &domain.Contact{UserID: owner, OrganizationID: &org,
			ContactInformation: "prospect@example.com", DoNotContact: false}
		require.NoError(t, db.Create(c).Error)

		code, body := post(t, newApp(h, owner), conv.ID.String(),
			`{"intent":"Do not contact"}`)
		require.Equal(t, fiber.StatusOK, code)
		data, _ := body["data"].(map[string]any)
		assert.Equal(t, "suppressed", data["suppression_changed"])

		var after domain.Contact
		require.NoError(t, db.First(&after, "id = ?", c.ID).Error)
		assert.True(t, after.DoNotContact)
	})
}

func TestCorrectIntent_SameLabelChangesNothing(t *testing.T) {
	h, db := newTestHandler(t)
	owner := uuid.New()
	conv, _ := seed(t, db, owner, string(domain.IntentInterested))

	// Sent in the SCREAMING_SNAKE form the front end works in, to confirm it
	// normalises onto the stored display string rather than counting as a change.
	code, body := post(t, newApp(h, owner), conv.ID.String(), `{"intent":"INTERESTED"}`)
	require.Equal(t, fiber.StatusOK, code)
	data, _ := body["data"].(map[string]any)
	assert.Equal(t, false, data["changed"])

	// No feedback row: nothing was corrected, so there is no classifier error
	// to record.
	var n int64
	require.NoError(t, db.Model(&domain.UserFeedback{}).
		Where("entity_id = ?", conv.ID).Count(&n).Error)
	assert.Zero(t, n)

	// And the classifier's evidence survives, because it still supports the
	// label in force.
	d := detailOf(t, db, conv.ID)
	assert.Contains(t, d, "reasoning")
}
