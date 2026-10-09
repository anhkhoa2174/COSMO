package service

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

// buildDocx assembles a minimal but structurally valid .docx around the given
// WordprocessingML body.
func buildDocx(t *testing.T, documentXML string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	write := func(name, content string) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	write("[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8"?><Types/>`)
	write("word/document.xml", documentXML)

	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

const docxHeader = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`

func TestExtractDocxText_ParagraphsAndRuns(t *testing.T) {
	// Word splits a styled sentence across several runs; they must join back
	// into one line, and each paragraph must land on its own line.
	doc := buildDocx(t, docxHeader+
		`<w:p><w:r><w:t>COSMO pricing</w:t></w:r></w:p>`+
		`<w:p><w:r><w:t>Growth is </w:t></w:r><w:r><w:t>$99</w:t></w:r>`+
		`<w:r><w:t> per seat.</w:t></w:r></w:p>`+
		`</w:body></w:document>`)

	text, err := extractDocxText(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "COSMO pricing\nGrowth is $99 per seat."
	if text != want {
		t.Fatalf("got %q, want %q", text, want)
	}
}

func TestExtractDocxText_IgnoresFormattingElements(t *testing.T) {
	// Styling and revision metadata carry character data of their own; none of
	// it may leak into the extracted prose.
	doc := buildDocx(t, docxHeader+
		`<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr>`+
		`<w:r><w:rPr><w:b/></w:rPr><w:t>Real text</w:t></w:r></w:p>`+
		`</w:body></w:document>`)

	text, err := extractDocxText(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Real text" {
		t.Fatalf("formatting leaked into the output: %q", text)
	}
}

func TestExtractDocxText_TabsAndBreaks(t *testing.T) {
	doc := buildDocx(t, docxHeader+
		`<w:p><w:r><w:t>A</w:t><w:tab/><w:t>B</w:t><w:br/><w:t>C</w:t></w:r></w:p>`+
		`</w:body></w:document>`)

	text, err := extractDocxText(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "A\tB\nC" {
		t.Fatalf("got %q, want %q", text, "A\tB\nC")
	}
}

func TestExtractDocxText_CollapsesEmptyParagraphs(t *testing.T) {
	doc := buildDocx(t, docxHeader+
		`<w:p><w:r><w:t>First</w:t></w:r></w:p>`+
		`<w:p/><w:p/><w:p/>`+
		`<w:p><w:r><w:t>Second</w:t></w:r></w:p>`+
		`</w:body></w:document>`)

	text, err := extractDocxText(doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "First\n\nSecond" {
		t.Fatalf("runs of empty paragraphs should collapse to one blank line, got %q", text)
	}
}

func TestExtractDocxText_RejectsNonArchive(t *testing.T) {
	if _, err := extractDocxText([]byte("this is plain text, not a zip")); err == nil {
		t.Fatal("expected an error for a file that is not a zip archive")
	}
}

func TestExtractDocxText_RejectsArchiveWithoutDocument(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("word/other.xml")
	_, _ = w.Write([]byte("<a/>"))
	_ = zw.Close()

	_, err := extractDocxText(buf.Bytes())
	if err == nil {
		t.Fatal("expected an error when word/document.xml is missing")
	}
	if !strings.Contains(err.Error(), "word/document.xml") {
		t.Fatalf("error should name the missing part, got %v", err)
	}
}

func TestExtractDocxText_RejectsEmptyDocument(t *testing.T) {
	doc := buildDocx(t, docxHeader+`<w:p/></w:body></w:document>`)

	if _, err := extractDocxText(doc); err == nil {
		t.Fatal("a document with no text should be rejected, not indexed as empty")
	}
}

func TestExtractText_DocxRoutesToParser(t *testing.T) {
	doc := buildDocx(t, docxHeader+
		`<w:p><w:r><w:t>Knowledge base entry</w:t></w:r></w:p>`+
		`</w:body></w:document>`)

	text, err := extractText(doc,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Knowledge base entry" {
		t.Fatalf("got %q", text)
	}
}

func TestExtractText_LegacyDocGivesActionableError(t *testing.T) {
	_, err := extractText([]byte{0xD0, 0xCF, 0x11, 0xE0}, "application/msword")
	if err == nil {
		t.Fatal("expected legacy .doc to be rejected")
	}
	if !strings.Contains(err.Error(), ".docx") {
		t.Fatalf("error should tell the user to re-save as .docx, got %v", err)
	}
}

func TestSupportedMimeTypes_AcceptsDocxRejectsLegacyDoc(t *testing.T) {
	// The extractor is useless if the upload gate refuses the type first.
	docx := "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	if _, ok := supportedMimeTypes[docx]; !ok {
		t.Fatal(".docx must be accepted by the upload allow-list")
	}
	if _, ok := supportedMimeTypes["application/msword"]; ok {
		t.Fatal("legacy .doc must stay out of the allow-list — there is no parser for it")
	}
}
