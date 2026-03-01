package workflow

import (
	"testing"

	"github.com/rockship/cosmo-agents-go/internal/domain/base"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

func TestWorkflowBeforeCreateDefaults(t *testing.T) {
	wf := Workflow{}

	if err := wf.BeforeCreate(nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if wf.ID == uuid.Nil {
		t.Fatal("expected ID to be set")
	}
	if wf.Nodes == nil || len(wf.Nodes) != 0 {
		t.Fatalf("expected empty nodes, got %v", wf.Nodes)
	}
	if string(wf.Edges) != "[]" {
		t.Fatalf("expected edges default '[]', got %s", string(wf.Edges))
	}
	if string(wf.State) != "{}" {
		t.Fatalf("expected state default '{}', got %s", string(wf.State))
	}
	if string(wf.CMetadata) != "{}" {
		t.Fatalf("expected cmetadata default '{}', got %s", string(wf.CMetadata))
	}
}

func TestWorkflowPreservesProvidedValues(t *testing.T) {
	nodes := pq.StringArray{"node-1"}
	edges := base.JSONB([]byte(`[{"from":"node-1","to":"node-2"}]`))
	state := base.JSONB([]byte(`{"status":"active"}`))
	meta := base.JSONB([]byte(`{"version":1}`))

	wf := Workflow{
		UserID:    uuid.New(),
		Nodes:     nodes,
		Edges:     edges,
		State:     state,
		CMetadata: meta,
	}

	if err := wf.BeforeCreate(nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(wf.Nodes) != len(nodes) {
		t.Fatalf("expected nodes length %d, got %d", len(nodes), len(wf.Nodes))
	}
	if string(wf.Edges) != string(edges) {
		t.Fatalf("expected edges to remain %s, got %s", string(edges), string(wf.Edges))
	}
	if string(wf.State) != string(state) {
		t.Fatalf("expected state to remain %s, got %s", string(state), string(wf.State))
	}
	if string(wf.CMetadata) != string(meta) {
		t.Fatalf("expected cmetadata to remain %s, got %s", string(meta), string(wf.CMetadata))
	}
}
