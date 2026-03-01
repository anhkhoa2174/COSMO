package file

import (
	"testing"
)

func TestFileTableName(t *testing.T) {
	if (File{}).TableName() != "files" {
		t.Fatalf("unexpected table name")
	}
}

func TestFileFields(t *testing.T) {
	file := File{Filename: "doc.pdf", MimeType: "application/pdf", S3Key: "files/doc.pdf", Size: 2048}
	if file.Filename != "doc.pdf" {
		t.Fatalf("expected filename doc.pdf")
	}
	if file.Size != 2048 {
		t.Fatalf("expected size 2048")
	}
}
