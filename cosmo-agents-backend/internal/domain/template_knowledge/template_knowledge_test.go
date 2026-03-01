package template_knowledge

import (
	"testing"

	"github.com/google/uuid"
)

func TestTemplateKnowledgeTableName(t *testing.T) {
	if (TemplateKnowledge{}).TableName() != "template_knowledges" {
		t.Fatal("unexpected table name")
	}
}

func TestTemplateKnowledgeFields(t *testing.T) {
	tk := TemplateKnowledge{
		TemplateID:  uuid.New(),
		KnowledgeID: uuid.New(),
	}

	if tk.TemplateID == uuid.Nil || tk.KnowledgeID == uuid.Nil {
		t.Fatal("expected IDs to be set")
	}
}
