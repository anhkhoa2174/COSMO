package playbook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
)

func TestCreateAndReadPlaybooks(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	svc := NewService(f.playbooks, f.rules, f.enrollments, f.approvals, f.contacts, f.segments, nil)

	if _, err := svc.CreatePlaybook(ctx, &v1schema.CreatePlaybookRequest{Name: "Empty", PlaybookType: "nurture"}); err == nil {
		t.Fatalf("CreatePlaybook with no stages succeeded, want an error")
	}

	created, err := svc.CreatePlaybook(ctx, &v1schema.CreatePlaybookRequest{
		Name:         "Outreach",
		Description:  "cold",
		PlaybookType: "outreach",
		Stages:       twoStages(),
	})
	if err != nil {
		t.Fatalf("CreatePlaybook: %v", err)
	}
	if created.PlaybookID == uuid.Nil || !created.IsActive || len(created.Config.Stages) != 2 {
		t.Fatalf("created = %+v, want an active playbook with both stages", created)
	}
	if created.Performance.ContactsEnrolled != 0 || created.Performance.ReplyRate != 0 {
		t.Errorf("performance = %+v, want zeroes", created.Performance)
	}

	got, err := svc.GetPlaybook(ctx, created.PlaybookID)
	if err != nil {
		t.Fatalf("GetPlaybook: %v", err)
	}
	if got.Name != "Outreach" || got.Description != "cold" || got.Config.Stages[1].ID != "follow" {
		t.Errorf("GetPlaybook = %+v, want the stored playbook back", got)
	}
	if _, err := svc.GetPlaybook(ctx, uuid.New()); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("GetPlaybook(unknown) err = %v, want not found", err)
	}

	// A row whose config no longer parses is skipped rather than failing the list.
	broken := f.seedPlaybook(t, twoStages())
	f.db.MustExec(`UPDATE playbooks SET config = '"oops"' WHERE playbook_id = $1`, broken.PlaybookID)
	if _, err := svc.GetPlaybook(ctx, broken.PlaybookID); err == nil {
		t.Errorf("GetPlaybook on a broken config succeeded, want an error")
	}
	list, err := svc.ListPlaybooks(ctx)
	if err != nil {
		t.Fatalf("ListPlaybooks: %v", err)
	}
	if len(list) != 1 || list[0].PlaybookID != created.PlaybookID {
		t.Fatalf("ListPlaybooks = %d items, want only the valid playbook", len(list))
	}

	if err := svc.DeletePlaybook(ctx, created.PlaybookID); err != nil {
		t.Fatalf("DeletePlaybook: %v", err)
	}
	if list, _ := svc.ListPlaybooks(ctx); len(list) != 0 {
		t.Errorf("deleted playbook still listed")
	}
	if _, err := svc.GetPlaybook(ctx, created.PlaybookID); err == nil {
		t.Errorf("deleted playbook still readable")
	}
	if err := svc.DeletePlaybook(ctx, uuid.New()); err == nil {
		t.Errorf("DeletePlaybook(unknown) succeeded, want an error")
	}
}

// fakeOpenAI answers every chat completion with content and records the
// prompt it was sent. It stands in for the provider through OPENAI_BASE_URL,
// the same switch production uses for OpenAI-compatible hosts.
func fakeOpenAI(t *testing.T, content string) (*ai.OpenAIClient, *string) {
	t.Helper()
	var prompt string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.Unmarshal(body, &req)
		if len(req.Messages) > 0 {
			prompt = req.Messages[len(req.Messages)-1].Content
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "x",
			"object":  "chat.completion",
			"choices": []map[string]any{{"index": 0, "message": map[string]any{"role": "assistant", "content": content}}},
		})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("OPENAI_BASE_URL", srv.URL)
	return ai.NewOpenAIClient(ai.Config{APIKey: "test", Model: "gpt-4o-mini"}), &prompt
}

func TestGenerateContent(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	pb := f.seedPlaybook(t, twoStages())
	contact := f.seedContact(t, `{"ai_insights":{"suspected_pain_points":[{"pain_point":"manual reporting"}],"suspected_goals":[{"goal":"close faster"}]}}`)

	client, prompt := fakeOpenAI(t, "```json\n{\"subject\":\"Hi Ada\",\"body\":\"About reporting\"}\n```")
	svc := NewService(f.playbooks, f.rules, f.enrollments, f.approvals, f.contacts, f.segments, client)

	got, err := svc.GenerateContent(ctx, pb.PlaybookID, contact.ID, "intro", "be brief")
	if err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}
	if got.Subject != "Hi Ada" || got.Body != "About reporting" {
		t.Errorf("content = %q / %q, want the fenced JSON unwrapped", got.Subject, got.Body)
	}
	if got.PersonalizationData["pain_point"] != "manual reporting" || got.PersonalizationData["company"] != "Analytical" {
		t.Errorf("personalization = %v", got.PersonalizationData)
	}
	for _, want := range []string{"Ada Lovelace", "CTO", "manual reporting", "close faster", "be brief", "nurture"} {
		if !strings.Contains(*prompt, want) {
			t.Errorf("prompt does not mention %q", want)
		}
	}

	if _, err := svc.GenerateContent(ctx, uuid.New(), contact.ID, "intro", ""); err == nil || !strings.Contains(err.Error(), "playbook not found") {
		t.Errorf("unknown playbook err = %v", err)
	}
	if _, err := svc.GenerateContent(ctx, pb.PlaybookID, uuid.New(), "intro", ""); err == nil || !strings.Contains(err.Error(), "contact not found") {
		t.Errorf("unknown contact err = %v", err)
	}

	bad, _ := fakeOpenAI(t, "Sure! Here is your email.")
	svc = NewService(f.playbooks, f.rules, f.enrollments, f.approvals, f.contacts, f.segments, bad)
	if _, err := svc.GenerateContent(ctx, pb.PlaybookID, contact.ID, "intro", ""); err == nil || !strings.Contains(err.Error(), "parse AI response") {
		t.Errorf("non-JSON reply err = %v, want a parse error", err)
	}
}

func TestProfileExtraction(t *testing.T) {
	tests := []struct {
		name      string
		profile   string
		wantPain  string
		wantGoals string
	}{
		{"no insights", `{}`, "Unknown", "Unknown"},
		{"insights of the wrong type", `{"ai_insights":"x"}`, "Unknown", "Unknown"},
		{"empty lists", `{"ai_insights":{"suspected_pain_points":[],"suspected_goals":[]}}`, "Unknown", "Unknown"},
		{"entries without the key", `{"ai_insights":{"suspected_pain_points":[{"x":1}],"suspected_goals":["plain"]}}`, "Unknown", "Unknown"},
		{"first entry wins", `{"ai_insights":{"suspected_pain_points":[{"pain_point":"a"},{"pain_point":"b"}],"suspected_goals":[{"goal":"g"}]}}`, "a", "g"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var profile map[string]interface{}
			if err := json.Unmarshal([]byte(tt.profile), &profile); err != nil {
				t.Fatal(err)
			}
			if got := extractPainPoints(profile); got != tt.wantPain {
				t.Errorf("extractPainPoints = %q, want %q", got, tt.wantPain)
			}
			if got := extractGoals(profile); got != tt.wantGoals {
				t.Errorf("extractGoals = %q, want %q", got, tt.wantGoals)
			}
		})
	}
}

func TestCleanJSONResponse(t *testing.T) {
	tests := []struct{ in, want string }{
		{`{"a":1}`, `{"a":1}`},
		{"```json\n{\"a\":1}\n```", `{"a":1}`},
		{"```\n{\"a\":1}```", `{"a":1}`},
		{" \t\r\n{\"a\":1}\n ", `{"a":1}`},
		{"", ""},
		{"   ", ""},
	}
	for _, tt := range tests {
		if got := cleanJSONResponse(tt.in); got != tt.want {
			t.Errorf("cleanJSONResponse(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
