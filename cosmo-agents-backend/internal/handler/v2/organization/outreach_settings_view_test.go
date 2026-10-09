package organization

import (
	"testing"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/service/autoreply"
)

// The form keys content by the same intent strings it receives in
// auto_reply_selectable_intents, so the view must use that spelling.
func TestViewAutoReply_ReturnsContentKeyedLikeTheIntents(t *testing.T) {
	on := true
	r := autoreply.Policy{
		Enabled: &on,
		Intents: []string{"OUT_OF_OFFICE"},
		Contents: map[string]autoreply.Content{
			"OUT_OF_OFFICE": {Mode: autoreply.ModeTemplate, Body: "Back soon"},
		},
	}.Resolve()

	view := viewAutoReply(r)
	c, ok := view.Contents[string(domain.IntentOutOfOffice)]
	if !ok || c.Body != "Back soon" {
		t.Fatalf("contents = %+v", view.Contents)
	}
	if len(view.Intents) != 1 || view.Intents[0] != string(domain.IntentOutOfOffice) {
		t.Fatalf("intents = %v", view.Intents)
	}
}

func TestViewAutoReply_DisabledPolicyHasEmptyContentsNotNull(t *testing.T) {
	if v := viewAutoReply(autoreply.Disabled()); v.Contents == nil {
		t.Fatal("contents must serialise as {} so the client can index it")
	}
}
