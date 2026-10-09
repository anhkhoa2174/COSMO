package vectorstore

import "testing"

func TestChunkLabels_ReadsTitleAndTypeFromUploadMetadata(t *testing.T) {
	meta := map[string]interface{}{
		"type":   "pricing",
		"origin": map[string]interface{}{"filename": "pricing-2026.pdf"},
	}
	title, typ := chunkLabels(meta)
	if title != "pricing-2026.pdf" || typ != "pricing" {
		t.Fatalf("got title=%q type=%q", title, typ)
	}
}

func TestChunkLabels_MissingValuesAreEmpty(t *testing.T) {
	title, typ := chunkLabels(map[string]interface{}{})
	if title != "" || typ != "" {
		t.Fatalf("got title=%q type=%q", title, typ)
	}
}
