package daily_action

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	contact "github.com/rockship/cosmo-agents-go/internal/domain/contact"
	domain "github.com/rockship/cosmo-agents-go/internal/domain/daily_action"
	outreach "github.com/rockship/cosmo-agents-go/internal/domain/outreach"
	outreachSvc "github.com/rockship/cosmo-agents-go/internal/service/outreach"
)

func TestMapActionTypeAndCategory(t *testing.T) {
	sc := func(status string) *outreachSvc.SuggestContact {
		return &outreachSvc.SuggestContact{Contact: &contact.Contact{Status: status}}
	}
	st := func(conv, step string) *outreach.OutreachState {
		return &outreach.OutreachState{ConversationState: conv, NextStep: step}
	}
	tests := []struct {
		name     string
		sc       *outreachSvc.SuggestContact
		state    *outreach.OutreachState
		wantType domain.ActionType
		wantCat  domain.CategoryID
	}{
		// A pending contact with no conversation yet cannot be reached, so the
		// only useful action is to enrich it.
		{"no state pending", sc("pending"), nil, domain.ActionTypeEnrich, domain.CategoryEnrichment},
		{"no state ready", sc("ready"), nil, domain.ActionTypeOutreach, domain.CategoryNewOutreach},
		{"replied", sc("ready"), st("REPLIED", "WAIT"), domain.ActionTypeRespond, domain.CategoryReplied},
		{"no reply fu1", sc("ready"), st("NO_REPLY", "FOLLOW_UP_1"), domain.ActionTypeFollowup, domain.CategoryFollowup},
		{"no reply drop", sc("ready"), st("NO_REPLY", "DROP"), domain.ActionTypeFollowup, domain.CategoryFollowup},
		{"no reply wait", sc("ready"), st("NO_REPLY", "WAIT"), domain.ActionTypeFollowup, domain.CategoryFollowup},
		{"cold send", sc("ready"), st("COLD", "SEND"), domain.ActionTypeOutreach, domain.CategoryNewOutreach},
		{"cold other", sc("ready"), st("COLD", "WAIT"), domain.ActionTypeOutreach, domain.CategoryNewOutreach},
		{"post meeting", sc("ready"), st("POST_MEETING", "FOLLOW_UP"), domain.ActionTypeFollowup, domain.CategoryFollowup},
		{"unknown state pending", sc("pending"), st("WEIRD", ""), domain.ActionTypeEnrich, domain.CategoryEnrichment},
		{"unknown state ready", sc("ready"), st("WEIRD", ""), domain.ActionTypeOutreach, domain.CategoryNewOutreach},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotType, gotCat := mapActionTypeAndCategory(tc.sc, tc.state)
			if gotType != tc.wantType || gotCat != tc.wantCat {
				t.Fatalf("got %s/%s, want %s/%s", gotType, gotCat, tc.wantType, tc.wantCat)
			}
		})
	}
}

func TestMapEngineAction(t *testing.T) {
	with := func(action string) *contact.Contact { return &contact.Contact{NextAction: &action} }
	tests := []struct {
		name     string
		c        *contact.Contact
		wantType domain.ActionType
		wantCat  domain.CategoryID
		wantOK   bool
	}{
		{"no decision", &contact.Contact{}, "", "", false},
		{"answer reply", with("ANSWER_REPLY"), domain.ActionTypeRespond, domain.CategoryReplied, true},
		{"propose meeting", with("PROPOSE_MEETING"), domain.ActionTypeRespond, domain.CategoryReplied, true},
		{"escalate", with("ESCALATE"), domain.ActionTypeRespond, domain.CategoryReplied, true},
		{"follow-up", with("SEND_FOLLOW_UP"), domain.ActionTypeFollowup, domain.CategoryFollowup, true},
		{"nurture", with("NURTURE"), domain.ActionTypeFollowup, domain.CategoryFollowup, true},
		{"meeting follow-up", with("MEETING_FOLLOW_UP"), domain.ActionTypeFollowup, domain.CategoryFollowup, true},
		{"intro", with("SEND_INTRO"), domain.ActionTypeOutreach, domain.CategoryNewOutreach, true},
		{"fix data", with("FIX_DATA"), domain.ActionTypeEnrich, domain.CategoryEnrichment, true},
		{"wait falls back", with("WAIT"), "", "", false},
		{"unknown falls back", with("SOMETHING_ELSE"), "", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotType, gotCat, ok := mapEngineAction(tc.c)
			if gotType != tc.wantType || gotCat != tc.wantCat || ok != tc.wantOK {
				t.Errorf("mapEngineAction = (%q, %q, %v), want (%q, %q, %v)",
					gotType, gotCat, ok, tc.wantType, tc.wantCat, tc.wantOK)
			}
		})
	}
}

func TestBuildReasoning(t *testing.T) {
	sc := &outreachSvc.SuggestContact{Contact: &contact.Contact{Name: "An", Company: "Acme"}}
	tests := []struct {
		typ   domain.ActionType
		state *outreach.OutreachState
		want  string
	}{
		{domain.ActionTypeRespond, nil, "An replied — review and respond"},
		{domain.ActionTypeFollowup, &outreach.OutreachState{DaysSinceLastInteraction: 6}, "Follow up with An (6 days since last interaction)"},
		{domain.ActionTypeFollowup, nil, "Follow up with An (0 days since last interaction)"},
		{domain.ActionTypeOutreach, nil, "Initial outreach to An at Acme"},
		{domain.ActionTypeMeetingPrep, nil, "Prepare for upcoming meeting with An"},
		// Name and company are present; job_title, source and contact info are not.
		{domain.ActionTypeEnrich, nil, "Enrich profile for An — 3 fields missing"},
		{"other", nil, "Action for An"},
	}
	for _, tc := range tests {
		if got := buildReasoning(tc.typ, sc, tc.state); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.typ, got, tc.want)
		}
	}
}

func TestBuildOutreachData_FollowupNumberAndDraftReuse(t *testing.T) {
	draft := "Hi An, following up"
	lastTouch := time.Now().Add(-48 * time.Hour)
	s := &Service{}

	tests := []struct {
		name      string
		state     *outreach.OutreachState
		wantFU    *int
		wantFinal bool
		wantDraft string
	}{
		{"fu1", &outreach.OutreachState{NextStep: "FOLLOW_UP_1", MaxFollowups: 2, MessageDraft: &draft, UpdatedAt: time.Now()}, intp(1), false, draft},
		{"fu2 is final", &outreach.OutreachState{NextStep: "FOLLOW_UP_2", FollowupCount: 1, MaxFollowups: 2, MessageDraft: &draft, UpdatedAt: time.Now()}, intp(2), true, draft},
		{"generic fu counts on", &outreach.OutreachState{NextStep: "FOLLOW_UP", FollowupCount: 2, MaxFollowups: 4, MessageDraft: &draft, UpdatedAt: time.Now()}, intp(3), false, draft},
		{"wait reports last sent", &outreach.OutreachState{NextStep: "WAIT", FollowupCount: 1, MaxFollowups: 3, MessageDraft: &draft, UpdatedAt: time.Now()}, intp(1), false, draft},
		{"send has none", &outreach.OutreachState{NextStep: "SEND", MaxFollowups: 2, MessageDraft: &draft, UpdatedAt: time.Now()}, nil, false, draft},
		// A draft written after the last touch is still about the current
		// conversation and is reused instead of paying for a new one.
		{"fresh draft reused", &outreach.OutreachState{NextStep: "WAIT", MaxFollowups: 2, MessageDraft: &draft,
			LastInteractionAt: &lastTouch, UpdatedAt: time.Now()}, nil, false, draft},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sc := &outreachSvc.SuggestContact{Contact: &contact.Contact{Scenario: "role_based", ContextLevel: "LOW"}}
			data := s.buildOutreachData(context.Background(), uuid.New(), uuid.New(), sc, tc.state, "Rep", "en")
			if (data.FollowupNumber == nil) != (tc.wantFU == nil) ||
				(tc.wantFU != nil && *data.FollowupNumber != *tc.wantFU) {
				t.Errorf("followup number = %v, want %v", deref(data.FollowupNumber), deref(tc.wantFU))
			}
			if data.IsFinalFollowup != tc.wantFinal {
				t.Errorf("final = %v, want %v", data.IsFinalFollowup, tc.wantFinal)
			}
			if data.DraftMessage != tc.wantDraft {
				t.Errorf("draft = %q, want %q", data.DraftMessage, tc.wantDraft)
			}
			if data.OutreachState == nil || data.OutreachState.NextStep != tc.state.NextStep {
				t.Errorf("outreach state snapshot missing: %+v", data.OutreachState)
			}
		})
	}

	// A suggestion that already carries a draft wins over the stored one.
	sc := &outreachSvc.SuggestContact{Contact: &contact.Contact{}, MessageDraft: "from suggest"}
	data := s.buildOutreachData(context.Background(), uuid.New(), uuid.New(), sc,
		&outreach.OutreachState{MessageDraft: &draft, MaxFollowups: 2, UpdatedAt: time.Now(), LastInteractionAt: &lastTouch}, "Rep", "en")
	if data.DraftMessage != "from suggest" || data.LastSentDate == nil || *data.LastSentDate != lastTouch.Format("2006-01-02") {
		t.Fatalf("got draft %q last sent %v", data.DraftMessage, data.LastSentDate)
	}
}

func intp(i int) *int { return &i }

func deref(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func TestBuildRespondAndMeetingData(t *testing.T) {
	at := time.Date(2026, 9, 1, 8, 30, 0, 0, time.UTC)
	r := buildRespondData(&outreachSvc.SuggestContact{Contact: &contact.Contact{ContactChannel: "Email"}},
		&outreach.OutreachState{LastInteractionAt: &at})
	if r.ReplyChannel != "Email" || r.ReplyTimestamp != "2026-09-01T08:30:00Z" || r.RecommendedAction != "reply" {
		t.Fatalf("respond data = %+v", r)
	}
	if r := buildRespondData(&outreachSvc.SuggestContact{Contact: &contact.Contact{}}, nil); r.ReplyTimestamp != "" {
		t.Fatalf("no state should leave timestamp empty, got %q", r.ReplyTimestamp)
	}

	title := "Demo"
	m := &outreach.Meeting{ID: uuid.New(), Time: time.Now().Add(3 * time.Hour), DurationMinutes: 45, Channel: "Zoom", Title: &title}
	md := buildMeetingData(m)
	if md.MeetingTitle != "Demo" || md.MeetingDurationMinutes != 45 || md.HoursUntilMeeting < 2.9 || md.HoursUntilMeeting > 3.1 {
		t.Fatalf("meeting data = %+v", md)
	}
	if md := buildMeetingData(&outreach.Meeting{Time: time.Now()}); md.MeetingTitle != "" {
		t.Fatal("untitled meeting should have an empty title")
	}
}

func TestBuildEnrichmentData(t *testing.T) {
	tests := []struct {
		name       string
		c          *contact.Contact
		impact     string
		numSources int
	}{
		{"complete", completeContact("LEAD"), "low", 0},
		{"one missing", &contact.Contact{Name: "An", Company: "Acme", JobTitle: "CTO", Source: "csv"}, "low", 1},
		{"two missing", &contact.Contact{Name: "An", Company: "Acme", Source: "csv"}, "medium", 2},
		// name and source have no suggested source; company, title and
		// contact info do.
		{"empty", &contact.Contact{}, "high", 3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := buildEnrichmentData(tc.c)
			if d.QualityImpact != tc.impact || len(d.SuggestedSources) != tc.numSources {
				t.Fatalf("impact=%s sources=%d, want %s/%d", d.QualityImpact, len(d.SuggestedSources), tc.impact, tc.numSources)
			}
		})
	}
}

func actionOf(typ domain.ActionType, cat domain.CategoryID, prio int, name string) domain.DailyAction {
	snap, _ := json.Marshal(domain.ContactSnapshot{ID: uuid.New(), Name: name})
	return domain.DailyAction{Type: typ, CategoryID: cat, Priority: prio, ContactSnapshot: base.JSONB(snap), Reasoning: "why " + name}
}

func TestGenerateAgentBriefing(t *testing.T) {
	actions := []domain.DailyAction{
		actionOf(domain.ActionTypeRespond, domain.CategoryReplied, 40, "R1"),
		actionOf(domain.ActionTypeFollowup, domain.CategoryFollowup, 25, "F-urgent"),
		// A follow-up below the urgency cut is not memorable enough to cite.
		actionOf(domain.ActionTypeFollowup, domain.CategoryFollowup, 60, "F-lazy"),
		actionOf(domain.ActionTypeOutreach, domain.CategoryNewOutreach, 70, "O1"),
		actionOf(domain.ActionTypeRespond, domain.CategoryReplied, 10, "R2"),
		actionOf(domain.ActionTypeRespond, domain.CategoryReplied, 11, "R3"),
	}

	b := GenerateAgentBriefing(actions, "en")

	// Category badges come out in fixed order and skip empty categories.
	var cats []string
	for _, c := range b.CategoryCounts {
		cats = append(cats, c.Category+":"+c.Label)
	}
	if got := strings.Join(cats, ","); got != "replied:Replied,followup:Follow-up,new_outreach:New Outreach" {
		t.Fatalf("category counts = %s", got)
	}
	if b.CategoryCounts[0].Count != 3 || b.CategoryCounts[1].Count != 2 {
		t.Fatalf("counts wrong: %+v", b.CategoryCounts)
	}

	// Memory references are capped at three and never include the lazy one.
	if len(b.MemoryReferences) != 3 {
		t.Fatalf("want 3 refs, got %d", len(b.MemoryReferences))
	}
	for _, r := range b.MemoryReferences {
		if r.ContactName == "F-lazy" {
			t.Fatal("low-urgency follow-up must not be cited")
		}
	}
	if !strings.Contains(b.Greeting, "6 actions") {
		t.Fatalf("greeting = %q", b.Greeting)
	}
	if !strings.HasPrefix(b.StrategicReasoning, "3 contacts have replied") {
		t.Fatalf("reasoning = %q", b.StrategicReasoning)
	}

	vi := GenerateAgentBriefing(actions, "vi")
	if vi.CategoryCounts[0].Label != "Đã trả lời" || !strings.Contains(vi.Greeting, "6 hành động") {
		t.Fatalf("vietnamese briefing = %+v", vi)
	}
}

func TestBuildStrategicReasoning(t *testing.T) {
	tests := []struct {
		lang   string
		counts map[domain.CategoryID]int
		prefix string
	}{
		{"en", map[domain.CategoryID]int{domain.CategoryReplied: 1}, "1 contacts have replied"},
		{"en", map[domain.CategoryID]int{domain.CategoryFollowup: 2, domain.CategoryNewOutreach: 1}, "Focus on following up with 2"},
		{"en", map[domain.CategoryID]int{domain.CategoryNewOutreach: 4}, "Focus on reaching out to 4"},
		{"vi", map[domain.CategoryID]int{domain.CategoryReplied: 1}, "Có 1 liên hệ"},
		{"vi", map[domain.CategoryID]int{domain.CategoryFollowup: 2}, "Hôm nay tập trung theo dõi 2"},
		{"vi", map[domain.CategoryID]int{}, "Tập trung tiếp cận 0"},
	}
	for _, tc := range tests {
		if got := buildStrategicReasoning(nil, tc.counts, tc.lang); !strings.HasPrefix(got, tc.prefix) {
			t.Errorf("%s %v: got %q, want prefix %q", tc.lang, tc.counts, got, tc.prefix)
		}
	}
}

func TestParseRerankResponse_Rejections(t *testing.T) {
	actions := testActions(2)
	tests := []struct {
		name string
		raw  string
		ok   bool
	}{
		{"valid", `[{"id":"a1"},{"id":"a0"}]`, true},
		{"prose around json", `Sure! Here you go: [{"id":"a0"},{"id":"a1"}] hope that helps`, true},
		{"not json", `I think a1 first`, false},
		{"object not array", `{"id":"a0"}`, false},
		// A hallucinated or real UUID id must not be accepted in place of the
		// positional ids the prompt handed out.
		{"unknown id", `[{"id":"a0"},{"id":"a7"}]`, false},
		{"repeated id", `[{"id":"a0"},{"id":"a0"}]`, false},
		{"padded", `[{"id":"a0"},{"id":"a1"},{"id":"a2"}]`, false},
		{"empty", ``, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseRerankResponse(tc.raw, actions)
			if (err == nil) != tc.ok {
				t.Fatalf("err = %v, want ok=%v", err, tc.ok)
			}
		})
	}
}

func TestRerankActions_BatchSizeBounds(t *testing.T) {
	// One action has nothing to compare against; more than the cap would be
	// an oversized prompt. Neither should reach the model.
	for _, n := range []int{0, 1, maxRerankCandidates + 1} {
		client := &stubChatClient{reply: `[]`}
		s := (&Service{}).WithAIPrioritizer(client)
		s.rerankActions(context.Background(), testActions(n))
		if client.calls != 0 {
			t.Errorf("n=%d: model called %d times", n, client.calls)
		}
	}
}

// Contact names here are overwhelmingly Vietnamese, where most letters are
// two or three bytes. Truncating by bytes cut a character in half and put
// invalid UTF-8 into the prompt (rendered by %q as "\xe1\xbb" noise).
func TestTruncate_KeepsRunesWhole(t *testing.T) {
	name := strings.Repeat("Nguyễn Thị Phương Thảo ", 5)
	for n := 1; n < len(name); n++ {
		if got := truncate(name, n); !utf8.ValidString(got) {
			t.Fatalf("truncate(_, %d) produced invalid UTF-8: %q", n, got)
		}
	}
	got := truncate(name, 60)
	if !strings.HasSuffix(got, "…") || utf8.RuneCountInString(got) != 61 {
		t.Fatalf("want 60 runes plus ellipsis, got %d: %q", utf8.RuneCountInString(got), got)
	}
	if truncate("short", 60) != "short" {
		t.Fatal("short strings must pass through untouched")
	}

	// 21 runes but 31 bytes: the byte cut at 60 lands inside "ễ".
	snap, _ := json.Marshal(map[string]string{"name": strings.Repeat("Nguyễn ", 9)})
	actions := []domain.DailyAction{{ContactSnapshot: base.JSONB(snap)}, {}}
	prompt, _ := buildRerankPrompt(actions)
	if strings.Contains(prompt, `\x`) {
		t.Fatalf("prompt carries escaped broken bytes: %s", prompt)
	}
}

func TestContainsAny(t *testing.T) {
	if !containsAny("how many left", "total", "how many") || containsAny("hello", "a b", "xyz") {
		t.Fatal("containsAny mismatch")
	}
}
