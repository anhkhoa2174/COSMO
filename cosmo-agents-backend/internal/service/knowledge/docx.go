package service

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// maxDocxUncompressed bounds how much text one .docx may expand to. A zip can
// declare a small compressed size and inflate to gigabytes ("zip bomb"), so the
// reader is capped rather than trusted.
const maxDocxUncompressed = 32 << 20 // 32 MB

// extractDocxText pulls the readable text out of a .docx file.
//
// A .docx is a ZIP archive whose main body lives in word/document.xml as
// WordprocessingML. Rather than pull in a full Office library for one format,
// the parser streams that XML and keeps only what carries text:
//
//	<w:t>   a run of literal text
//	<w:p>   a paragraph — becomes a newline
//	<w:tab> a tab, <w:br> a line break
//
// Everything else (styling, revision marks, numbering) is skipped, which is
// exactly what the RAG pipeline wants: prose, not formatting.
func extractDocxText(data []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("not a valid .docx archive: %w", err)
	}

	var document *zip.File
	for _, f := range reader.File {
		if f.Name == "word/document.xml" {
			document = f
			break
		}
	}
	if document == nil {
		return "", fmt.Errorf("`.docx` archive has no word/document.xml; the file may be a .doc renamed to .docx")
	}

	rc, err := document.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open word/document.xml: %w", err)
	}
	defer rc.Close()

	text, err := parseWordprocessingML(io.LimitReader(rc, maxDocxUncompressed))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("no readable text found in the document")
	}
	return text, nil
}

// parseWordprocessingML walks the document body and rebuilds the plain text.
func parseWordprocessingML(r io.Reader) (string, error) {
	decoder := xml.NewDecoder(r)

	var sb strings.Builder
	// inText is set while inside a <w:t>, so only literal runs are collected —
	// character data belonging to other elements is ignored.
	inText := false

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("malformed document.xml: %w", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "t":
				inText = true
			case "tab":
				sb.WriteString("\t")
			case "br", "cr":
				sb.WriteString("\n")
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				inText = false
			case "p":
				// One paragraph per line keeps the chunker's paragraph-first
				// split boundaries meaningful.
				sb.WriteString("\n")
			}
		case xml.CharData:
			if inText {
				sb.Write(t)
			}
		}
	}

	return collapseBlankLines(sb.String()), nil
}

// collapseBlankLines trims trailing spaces and squeezes runs of empty lines
// down to one, so empty Word paragraphs do not become chunk-sized gaps.
func collapseBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	blank := false

	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" {
			if blank {
				continue
			}
			blank = true
		} else {
			blank = false
		}
		out = append(out, line)
	}

	return strings.TrimSpace(strings.Join(out, "\n"))
}
