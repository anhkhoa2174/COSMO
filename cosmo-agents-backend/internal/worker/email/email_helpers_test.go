package email

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

func TestBuildTemplateDataAndApply(t *testing.T) {
	agent := &domain.Agent{
		Name:      "Alice",
		Signature: "Best,\n{sender_name}",
	}
	contact := &domain.Contact{
		Name:     "Bob Smith",
		Profile:  base.JSONB(`{"email":"bob@example.com"}`),
		Company:  "ACME",
		JobTitle: "CTO",
		Country:  "US",
	}

	data := buildTemplateData(agent, contact)
	if data["contact_first_name"] != "Bob" || data["sender_name"] != "Alice" {
		t.Fatalf("unexpected template data: %+v", data)
	}

	result := applyTemplateData("Hello {contact_first_name} from {{contact_company}}", data)
	if result != "Hello Bob from ACME" {
		t.Fatalf("applyTemplateData failed, got %s", result)
	}
}

func TestPrepareEmailBodyMarkdownToHTML(t *testing.T) {
	body, isHTML := prepareEmailBody("Hello **World**", false)
	if !isHTML {
		t.Fatalf("expected markdown to be rendered as HTML")
	}
	if body != "Hello <strong>World</strong>" {
		t.Fatalf("unexpected html: %s", body)
	}
}

func TestLooksLikeMarkdownHelper(t *testing.T) {
	if !looksLikeMarkdown("This is **bold**") {
		t.Fatalf("expected markdown detection")
	}
	if looksLikeMarkdown("plain text") {
		t.Fatalf("did not expect markdown detection for plain text")
	}
}

func TestNormalizeAndSafeUUID(t *testing.T) {
	normalized := normalizeTemplateValue("  n/a  ")
	if normalized != "" {
		t.Fatalf("expected empty string for NOT_AVAILABLE, got %s", normalized)
	}
	id := uuid.New()
	if safeUUID(&id) != id.String() {
		t.Fatalf("safeUUID mismatch")
	}
	if safeUUID(nil) != "" {
		t.Fatalf("safeUUID expected empty for nil")
	}
}

func TestPrepareEmailBodyHTMLFlag(t *testing.T) {
	body, isHTML := prepareEmailBody("Line1\r\nLine2", true)
	if !isHTML {
		t.Fatalf("expected isHTML true")
	}
	if body != "Line1<br/>Line2" {
		t.Fatalf("expected HTML with <br/>, got %s", body)
	}
}

func TestMarkdownToHTMLStyles(t *testing.T) {
	text := "~~old~~ **bold** __under__ *it* _it2_"
	html := markdownToHTML(text)
	if !strings.Contains(html, "<s>old</s>") || !strings.Contains(html, "<strong>bold</strong>") || !strings.Contains(html, "<u>under</u>") {
		t.Fatalf("markdownToHTML missing conversions: %s", html)
	}
}
