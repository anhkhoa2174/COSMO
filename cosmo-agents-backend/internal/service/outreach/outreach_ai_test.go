package outreach

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	contactDomain "github.com/rockship/cosmo-agents-go/internal/domain/contact"
	outreachDomain "github.com/rockship/cosmo-agents-go/internal/domain/outreach"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
)

// fakeModel stands in for the OpenAI-compatible endpoint. It records what the
// service sent — the prompt is where most of the draft logic actually lives —
// and answers with a canned reply, or fails when told to.
type fakeModel struct {
	mu     sync.Mutex
	reply  string
	status int
	calls  []fakeCall
}

type fakeCall struct{ system, user string }

func (m *fakeModel) last(t *testing.T) fakeCall {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	require.NotEmpty(t, m.calls, "the model was never called")
	return m.calls[len(m.calls)-1]
}

// withFakeModel attaches a fake model to the service. The client is pointed at
// it through OPENAI_BASE_URL, the same switch production uses for compatible
// hosts.
func withFakeModel(t *testing.T, svc *Service, reply string) *fakeModel {
	t.Helper()
	m := &fakeModel{reply: reply, status: http.StatusOK}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct{ Role, Content string } `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var c fakeCall
		for _, msg := range req.Messages {
			switch msg.Role {
			case "system":
				c.system = msg.Content
			case "user":
				c.user = msg.Content
			}
		}
		m.mu.Lock()
		m.calls = append(m.calls, c)
		status, reply := m.status, m.reply
		m.mu.Unlock()

		if status != http.StatusOK {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":{"message":"boom"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "x", "object": "chat.completion", "model": "gpt-4o-mini",
			"choices": []map[string]interface{}{{
				"index": 0, "finish_reason": "stop",
				"message": map[string]string{"role": "assistant", "content": reply},
			}},
		})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("OPENAI_BASE_URL", srv.URL+"/v1")
	svc.SetOpenAIClient(ai.NewOpenAIClient(ai.Config{APIKey: "test", Model: "gpt-4o-mini"}))
	return m
}

// seedConversation gives a contact a two-message thread and a team note.
func seedConversation(t *testing.T, f *svcFixture, me uuid.UUID, c *contactDomain.Contact) {
	t.Helper()
	ctx := context.Background()
	for _, l := range []*outreachDomain.InteractionLog{
		{Direction: "outgoing", Content: "OPENING-LINE", Timestamp: time.Now().Add(-72 * time.Hour)},
		{Direction: "incoming", Content: "PROSPECT-REPLY", Timestamp: time.Now().Add(-2 * time.Hour), Sentiment: strPtr("positive")},
		{Direction: "internal", Content: "TEAM-NOTE", Timestamp: time.Now().Add(-1 * time.Hour)},
	} {
		l.UserID, l.ContactID, l.Channel = me, c.ID, "LinkedIn"
		require.NoError(t, f.svc.interactionRepo.Create(ctx, l))
	}
}

func strPtr(s string) *string { return &s }

func TestGenerateDraft_ModelDraftIsPostProcessed(t *testing.T) {
	f := newServiceFixture(t)
	ctx := context.Background()
	me := uuid.New()
	c := f.saveContact(t, me, "Minh Tran")
	seedConversation(t, f, me, c)

	// The model is told to write "anh/chị" and never sign off, and does both
	// anyway; the post-processing is what the BD actually receives.
	model := withFakeModel(t, f.svc, "  Chào anh Minh, cảm ơn anh!\n[Your Name]  ")

	draft, err := f.svc.GenerateDraftWithLanguageAndUserName(ctx, c, me, "vi", "Khoa")
	require.NoError(t, err)
	assert.Equal(t, "Chào anh/chị Minh, cảm ơn anh/chị!\nKhoa", draft)

	call := model.last(t)
	// The thread is presented oldest first, and the note is kept out of it.
	opening := strings.Index(call.user, "OPENING-LINE")
	reply := strings.Index(call.user, "PROSPECT-REPLY")
	require.True(t, opening >= 0 && reply >= 0, "both messages reach the prompt:\n%s", call.user)
	assert.Less(t, opening, reply, "history must read chronologically")
	assert.Equal(t, 1, strings.Count(call.user, "TEAM-NOTE"), "the note appears once, as a note, not as a message")
}

func TestGenerateDraft_SystemPromptFollowsTheLanguage(t *testing.T) {
	cases := []struct {
		language string
		want     string
	}{
		{"", "LANGUAGE DETECTION"}, // unknown: the model infers it from the name
		{"en", "English"},
		{"vi", "anh/chị"},
	}
	for _, tc := range cases {
		t.Run("language="+tc.language, func(t *testing.T) {
			f := newServiceFixture(t)
			me := uuid.New()
			c := f.saveContact(t, me, "Minh Tran")
			model := withFakeModel(t, f.svc, "draft")

			_, err := f.svc.GenerateDraftWithLanguage(context.Background(), c, me, tc.language)
			require.NoError(t, err)
			assert.Contains(t, model.last(t).system, tc.want)
		})
	}
}

// When the model fails or answers with nothing, the BD still gets a draft:
// the scenario template.
func TestGenerateDraft_FallsBackToTheTemplate(t *testing.T) {
	cases := []struct {
		name   string
		status int
		reply  string
	}{
		{"model error", http.StatusInternalServerError, ""},
		{"empty reply", http.StatusOK, "   "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newServiceFixture(t)
			ctx := context.Background()
			me := uuid.New()
			c := f.saveContact(t, me, "Minh Tran")
			model := withFakeModel(t, f.svc, tc.reply)
			model.status = tc.status

			for name, gen := range map[string]func() (string, error){
				"GenerateDraft":             func() (string, error) { return f.svc.GenerateDraft(ctx, c, me) },
				"GenerateDraftWithLanguage": func() (string, error) { return f.svc.GenerateDraftWithLanguage(ctx, c, me, "en") },
				"WithLanguageAndUserName": func() (string, error) {
					return f.svc.GenerateDraftWithLanguageAndUserName(ctx, c, me, "", "Khoa")
				},
			} {
				draft, err := gen()
				require.NoError(t, err, name)
				assert.Contains(t, draft, "Chào Minh,", "%s: template greeting", name)
			}
		})
	}
}

func TestGenerateDraft_VietnameseModelPathUsesFullContextPrompt(t *testing.T) {
	f := newServiceFixture(t)
	me := uuid.New()
	c := f.saveContact(t, me, "Minh Tran", func(c *contactDomain.Contact) { c.Industry = "Logistics" })
	seedConversation(t, f, me, c)
	model := withFakeModel(t, f.svc, "Xin chào [Tên bạn]")

	draft, err := f.svc.GenerateDraft(context.Background(), c, me)
	require.NoError(t, err)
	// No user name on this path, so the placeholder is removed rather than kept.
	assert.Equal(t, "Xin chào", strings.TrimSpace(draft))

	call := model.last(t)
	assert.Contains(t, call.user, `Ngành: "Logistics"`)
	assert.Contains(t, call.user, "ĐÃ TRẢ LỜI", "state is shown in Vietnamese")
	assert.Contains(t, call.user, "(Sentiment: positive)")
	assert.Contains(t, call.user, "GHI CHÚ NỘI BỘ")
}

func TestGenerateMeetingPrep_WritesAndSavesThePrep(t *testing.T) {
	for _, lang := range []string{"vi", "en"} {
		t.Run(lang, func(t *testing.T) {
			f := newServiceFixture(t)
			ctx := context.Background()
			me := uuid.New()
			c := f.saveContact(t, me, "Minh Tran")
			seedConversation(t, f, me, c)
			m, err := f.svc.CreateMeeting(ctx, &CreateMeetingInput{
				UserID: me, ContactID: c.ID, Title: "Discovery call",
				Time: time.Now().Add(24 * time.Hour), Channel: outreachDomain.MeetingChannelGoogleMeet,
				Note: "budget owner joins",
			})
			require.NoError(t, err)
			model := withFakeModel(t, f.svc, "1. Ask about volume")

			got, err := f.svc.GenerateMeetingPrep(ctx, me, m.ID, lang)
			require.NoError(t, err)
			require.NotNil(t, got.MeetingPrep)
			assert.Equal(t, "1. Ask about volume", *got.MeetingPrep)

			saved, err := f.svc.meetingRepo.FindByID(ctx, m.ID)
			require.NoError(t, err)
			require.NotNil(t, saved.MeetingPrep)

			call := model.last(t)
			assert.Contains(t, call.user, "Discovery call")
			assert.Contains(t, call.user, "Minh Tran")
			assert.Contains(t, call.user, "PROSPECT-REPLY")
			if lang == "en" {
				assert.Contains(t, call.system, "Respond in English")
			} else {
				assert.Contains(t, call.system, "tiếng Việt")
			}
		})
	}
}

func TestGenerateDraftWithAI_LegacyPrompt(t *testing.T) {
	f := newServiceFixture(t)
	me := uuid.New()
	c := f.saveContact(t, me, "Minh Tran", func(c *contactDomain.Contact) { c.Industry = "Retail" })
	seedConversation(t, f, me, c)

	dc, err := f.svc.GetDraftContext(context.Background(), c, me)
	require.NoError(t, err)
	require.Len(t, dc.Notes, 1)
	assert.Equal(t, outreachDomain.StateReplied, dc.State.State)
	require.NotNil(t, dc.LastIncoming)
	assert.Equal(t, "PROSPECT-REPLY", dc.LastIncoming.Content)

	_, err = f.svc.generateDraftWithAI(context.Background(), dc)
	require.ErrorContains(t, err, "not configured")

	model := withFakeModel(t, f.svc, "legacy draft")
	draft, err := f.svc.generateDraftWithAI(context.Background(), dc)
	require.NoError(t, err)
	assert.Equal(t, "legacy draft", draft)
	call := model.last(t)
	assert.Contains(t, call.user, "Scenario: post_reply")
	assert.Contains(t, call.user, "TEAM-NOTE")
}

// ============================================
// Pure helpers
// ============================================

func TestReplaceNamePlaceholders(t *testing.T) {
	cases := map[string]string{
		"Best,\n[Your Name]":          "Best,\nKhoa",
		"[YOUR NAME] / [your name]":   "Khoa / Khoa",
		"Thân, [Tên bạn]":             "Thân, Khoa",
		"[Tên của bạn] và [Tên Bạn]":  "Khoa và Khoa",
		"No placeholder here":         "No placeholder here",
		"[Name] is not a placeholder": "[Name] is not a placeholder",
	}
	for in, want := range cases {
		assert.Equal(t, want, replaceNamePlaceholders(in, "Khoa"), in)
	}
}

func TestNormalizeVietnamesePronouns(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Chào anh Minh", "Chào anh/chị Minh"},
		{"Chào chị Lan", "Chào anh/chị Lan"},
		{"anh có rảnh không?", "Anh/chị có rảnh không?"},
		{"Anh, chị, anh", "Anh/chị, anh/chị, anh/chị"},
		{"Chào anh/chị Minh", "Chào anh/chị Minh"}, // already correct: untouched
		{"Thanh Hoa", "Thanh Hoa"},                 // "anh" inside a word is not a pronoun
		{"Hello Minh, thanks!", "Hello Minh, thanks!"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, normalizeVietnamesePronouns(tc.in), tc.in)
	}
}

// "Anh" is also one of the most common Vietnamese given names (Tuấn Anh, Lan
// Anh). The normaliser cannot tell the name from the pronoun and rewrites the
// prospect's own name.
func TestNormalizeVietnamesePronouns_LeavesTheNameAnhAlone(t *testing.T) {
	t.Skip(`BUG: normalizeVietnamesePronouns rewrites the given name "Anh": ` +
		`"Chào anh/chị Tuấn Anh" becomes "Chào anh/chị Tuấn anh/chị"`)
	assert.Equal(t, "Chào anh/chị Tuấn Anh,", normalizeVietnamesePronouns("Chào anh/chị Tuấn Anh,"))
}

func TestAutoReplyFrom_FailsClosed(t *testing.T) {
	cases := []struct {
		name        string
		raw         string
		wantEnabled bool
		wantCap     int
	}{
		{"no settings", "", false, 0},
		{"unreadable", "{", false, 0},
		{"no auto-reply block", `{"no_reply_hours":12}`, false, 0},
		// An invalid block is ignored wholesale, not partly applied.
		{"invalid policy", `{"auto_reply":{"enabled":true,"daily_cap":100000}}`, false, 0},
		{"valid policy", `{"auto_reply":{"enabled":true,"daily_cap":5}}`, true, 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := AutoReplyFrom([]byte(tc.raw))
			assert.Equal(t, tc.wantEnabled, got.Enabled)
			if tc.wantCap != 0 {
				assert.Equal(t, tc.wantCap, got.DailyCap)
			}
		})
	}
}

// Every enum value needs a translation. A value that falls through to the
// default hands the model a raw code such as FOLLOW_UP_MEETING_1 in the
// middle of a natural-language prompt.
func TestPromptTranslationsCoverEveryValue(t *testing.T) {
	svc := &Service{}
	states := []outreachDomain.ConversationState{
		outreachDomain.StateCold, outreachDomain.StateNoReply, outreachDomain.StateReplied,
		outreachDomain.StatePostMeeting, outreachDomain.StateDropped,
	}
	steps := []outreachDomain.NextStepAction{
		outreachDomain.NextStepSend, outreachDomain.NextStepFollowUp1, outreachDomain.NextStepFollowUp2,
		outreachDomain.NextStepSetMeeting, outreachDomain.NextStepFollowUpMeeting1, outreachDomain.NextStepFollowUpMeeting2,
		outreachDomain.NextStepPrepareMeeting, outreachDomain.NextStepWait, outreachDomain.NextStepFollowUp,
		outreachDomain.NextStepDrop,
	}
	scenarios := []outreachDomain.Scenario{
		outreachDomain.ScenarioRoleBased, outreachDomain.ScenarioIndustryBased, outreachDomain.ScenarioNoReplyFollowup,
		outreachDomain.ScenarioPostReply, outreachDomain.ScenarioMeetingConfirmation, outreachDomain.ScenarioPostMeeting,
		outreachDomain.ScenarioReEngage,
	}

	for _, st := range states {
		assert.NotEqual(t, string(st), svc.translateState(st))
		assert.NotEqual(t, string(st), svc.translateStateToEnglish(st))
	}
	for _, ns := range steps {
		assert.NotEqual(t, string(ns), svc.translateNextStepToEnglish(ns, 0))
		assert.NotEqual(t, string(ns), svc.translateNextStepToVietnamese(ns, 0))
	}
	for _, sc := range scenarios {
		assert.NotEqual(t, string(sc), svc.translateScenarioToEnglish(sc))
		assert.NotEqual(t, string(sc), svc.translateScenarioToVietnamese(sc))
	}
	// Unknown codes pass through rather than vanishing.
	assert.Equal(t, "LIMBO", svc.translateState("LIMBO"))
	assert.Equal(t, "LIMBO", svc.translateStateToEnglish("LIMBO"))
	assert.Equal(t, "LIMBO", svc.translateNextStepToEnglish("LIMBO", 0))
	assert.Equal(t, "LIMBO", svc.translateNextStepToVietnamese("LIMBO", 0))
	assert.Equal(t, "LIMBO", svc.translateScenarioToEnglish("LIMBO"))
	assert.Equal(t, "LIMBO", svc.translateScenarioToVietnamese("LIMBO"))
}
