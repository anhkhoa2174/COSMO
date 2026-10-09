package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
	"golang.org/x/text/unicode/norm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/mapper"
	knowledgeRepo "github.com/rockship/cosmo-agents-go/internal/repository/knowledge"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	s3Service "github.com/rockship/cosmo-agents-go/internal/service/s3"
	"github.com/rockship/cosmo-agents-go/internal/skills"
	pkgworker "github.com/rockship/cosmo-agents-go/pkg/worker"
)

const knowledgeCollection = "knowledges"

const (
	knowledgeIndexingTask   = "knowledge:indexing"
	knowledgesPruningTask   = "knowledge:pruning"
	knowledgesSummarizeTask = "knowledge:summarize"
)

type knowledgeIndexingPayload struct {
	Collection   string                 `json:"collection"`
	Content      string                 `json:"content"`
	EmbeddingGID string                 `json:"embedding_gid"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type knowledgesPruningPayload struct {
	Collection    string   `json:"collection"`
	EmbeddingGIDs []string `json:"embedding_gids,omitempty"`
}

type knowledgesSummarizePayload struct {
	Content      string                 `json:"content"`
	EmbeddingGID string                 `json:"embedding_gid"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
}

var (
	supportedMimeTypes = map[string]struct{}{
		"application/pdf":  {},
		"text/plain":       {},
		"text/csv":         {},
		"application/csv":  {},
		"text/markdown":    {},
		"application/json": {},
		"text/html":        {},
		"application/xml":  {},
		"text/xml":         {},
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": {},
		// Legacy .doc stays out: it is an OLE compound file, not a ZIP of XML,
		// so it needs a different parser entirely. Users re-save as .docx.
	}

	supportedFormats = []string{
		"PDF (.pdf)",
		"Plain text (.txt)",
		"CSV (.csv)",
		"Markdown (.md)",
		"JSON (.json)",
		"HTML (.html)",
		"XML (.xml)",
		"Word (.docx)",
	}

	whitespacePattern = regexp.MustCompile(`\s+`)
)

// KnowledgeService orchestrates knowledge management (upload, indexing, pruning).
type KnowledgeService struct {
	knowledgeRepo *knowledgeRepo.KnowledgeRepository
	s3Service     *s3Service.S3Service
	workerClient  *pkgworker.Client
	bucketName    string

	// Set with WithSearch. Left unset, Search reports that retrieval is not
	// configured rather than returning an empty result, which is what it used
	// to do — an empty list is indistinguishable from "nothing matched" and
	// hid the fact that the endpoint did nothing at all.
	search chunkSearcher
}

// chunkSearcher retrieves knowledge chunks by semantic similarity. It is an
// interface so this package does not depend on the skills package, and so the
// mapping below can be tested without Redis or an embedding call.
type chunkSearcher interface {
	Search(ctx context.Context, userID uuid.UUID, query string, limit int) ([]skills.KnowledgeSearchResult, error)
}

// WithSearch enables semantic search over the knowledge base.
func (s *KnowledgeService) WithSearch(searcher chunkSearcher) *KnowledgeService {
	s.search = searcher
	return s
}

// NewKnowledgeService constructs a KnowledgeService instance.
func NewKnowledgeService(
	knowledgeRepo *knowledgeRepo.KnowledgeRepository,
	s3Service *s3Service.S3Service,
	workerClient *pkgworker.Client,
	bucketName string,
) *KnowledgeService {
	return &KnowledgeService{
		knowledgeRepo: knowledgeRepo,
		s3Service:     s3Service,
		workerClient:  workerClient,
		bucketName:    bucketName,
	}
}

// Upload processes uploaded files and mirrors Python's /v1/knowledge/upload
// behaviour. The files are recorded as type "other".
func (s *KnowledgeService) Upload(ctx context.Context, userID uuid.UUID, files []*multipart.FileHeader) ([]v1schema.UploadKnowledgeResponse, error) {
	return s.UploadOfType(ctx, userID, files, domain.KnowledgeTypeOther)
}

// UploadOfType uploads files and records every one of them as the given type,
// which retrieval uses to prefer, say, the pricing sheet for a pricing
// question.
func (s *KnowledgeService) UploadOfType(ctx context.Context, userID uuid.UUID, files []*multipart.FileHeader, knowledgeType domain.KnowledgeType) ([]v1schema.UploadKnowledgeResponse, error) {
	responses := make([]v1schema.UploadKnowledgeResponse, 0, len(files))

	if s.s3Service == nil || s.bucketName == "" {
		message := "knowledge upload not configured"
		for range files {
			responses = append(responses, v1schema.UploadKnowledgeResponse{
				Success: false,
				Result: v1schema.UploadKnowledgeResult{
					Msg: message,
				},
			})
		}
		return responses, nil
	}

	for _, fh := range files {
		response := s.handleUploadOfType(ctx, userID, fh, knowledgeType)
		responses = append(responses, response)
	}

	return responses, nil
}

// Search runs semantic search over the user's knowledge chunks.
//
// This was a stub that discarded its arguments and returned an empty slice, so
// the endpoint answered every query with no results while the handler
// dutifully validated a topK it then threw away. It now uses the same
// retrieval path that grounds generated replies, which is the point: a
// developer checking what the assistant can see should be looking at the same
// chunks the assistant does, not at a second implementation that might differ.
//
// The `filter` argument is accepted and not yet applied. Retrieval is scoped to
// the caller either way, so ignoring it narrows nothing that should have been
// narrowed; it is recorded here rather than silently dropped.
func (s *KnowledgeService) Search(ctx context.Context, userID uuid.UUID, query string, topK int, filter map[string]interface{}) ([]v1schema.KnowledgeSearchResult, error) {
	_ = filter

	if s.search == nil {
		return nil, fmt.Errorf("knowledge search is not configured on this deployment")
	}
	if strings.TrimSpace(query) == "" {
		return []v1schema.KnowledgeSearchResult{}, nil
	}
	if topK <= 0 {
		topK = 10
	}

	chunks, err := s.search.Search(ctx, userID, query, topK)
	if err != nil {
		return nil, fmt.Errorf("knowledge search failed: %w", err)
	}

	results := make([]v1schema.KnowledgeSearchResult, 0, len(chunks))
	for _, c := range chunks {
		results = append(results, v1schema.KnowledgeSearchResult{
			Document: c.ChunkText,
			// The score is a cosine distance, so it is reported under a name
			// that says which direction is better. Returning it bare as
			// "score" invites a reader to assume higher is more relevant.
			Metadata: map[string]interface{}{"cosine_distance": c.Score},
			Score:    c.Score,
		})
	}
	return results, nil
}

// EnqueuePruning schedules embedding pruning for the given GIDs.
func (s *KnowledgeService) EnqueuePruning(ctx context.Context, embeddingGIDs []string) error {
	if s.workerClient == nil || len(embeddingGIDs) == 0 {
		return nil
	}

	payload := knowledgesPruningPayload{
		Collection:    knowledgeCollection,
		EmbeddingGIDs: embeddingGIDs,
	}

	_, err := s.workerClient.EnqueueTask(ctx, knowledgesPruningTask, payload)
	return err
}

func (s *KnowledgeService) handleUpload(ctx context.Context, userID uuid.UUID, fh *multipart.FileHeader) v1schema.UploadKnowledgeResponse {
	return s.handleUploadOfType(ctx, userID, fh, domain.KnowledgeTypeOther)
}

func (s *KnowledgeService) handleUploadOfType(ctx context.Context, userID uuid.UUID, fh *multipart.FileHeader, knowledgeType domain.KnowledgeType) v1schema.UploadKnowledgeResponse {
	if knowledgeType == "" {
		knowledgeType = domain.KnowledgeTypeOther
	}
	file, err := fh.Open()
	if err != nil {
		return uploadFailure(fmt.Sprintf("failed to open file: %v", err))
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return uploadFailure(fmt.Sprintf("failed to read file: %v", err))
	}

	contentType := fh.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	if _, ok := supportedMimeTypes[contentType]; !ok {
		return uploadFailure(fmt.Sprintf("Unsupported file type: %s. Supported formats: %s",
			contentType, strings.Join(supportedFormats, ", ")))
	}

	plainText, err := extractText(data, contentType)
	if err != nil {
		return uploadFailure(err.Error())
	}
	plainText = strings.TrimSpace(plainText)
	if plainText == "" {
		return uploadFailure("Failed to process the file content")
	}

	s3Key := buildS3Key(fh.Filename)
	if err := s.uploadToS3(ctx, s3Key, data, contentType); err != nil {
		return uploadFailure(err.Error())
	}
	// Track cleanup needs explicitly to avoid err shadowing pitfalls
	uploadedToS3 := true

	embeddingGID := buildEmbeddingGID(userID, fh.Filename, data)
	metadata := buildMetadataMap(userID, fh, contentType, s3Key, embeddingGID)
	// Carried into the indexing task, where each chunk is tagged with it.
	metadata["type"] = string(knowledgeType)
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return uploadFailure(fmt.Sprintf("failed to serialize metadata: %v", err))
	}

	knowledge := &domain.Knowledge{
		UserID:       userID,
		SourceType:   domain.KnowledgeSourceUpload,
		Type:         knowledgeType,
		EmbeddingGID: &embeddingGID,
		CMetadata:    domain.JSONB(metadataBytes),
	}

	upserted, err := s.knowledgeRepo.UpsertByEmbeddingGID(ctx, knowledge)
	if err != nil {
		// best-effort: remove S3 object when DB persistence fails
		if uploadedToS3 {
			s.deleteFromS3(ctx, s3Key)
		}
		if errors.Is(err, knowledgeRepo.ErrKnowledgeOwnershipConflict) {
			return uploadFailure("knowledge already exists for a different user")
		}
		return uploadFailure(fmt.Sprintf("failed to persist knowledge: %v", err))
	}

	if err = s.enqueueIndexing(ctx, plainText, metadata, embeddingGID); err != nil {
		// best-effort: rollback S3 object and knowledge row on enqueue failure
		if uploadedToS3 {
			s.deleteFromS3(ctx, s3Key)
		}
		_ = s.knowledgeRepo.DeleteByEmbeddingGID(ctx, embeddingGID)
		return uploadFailure(fmt.Sprintf("failed to enqueue indexing: %v", err))
	}

	if err = s.enqueueSummarization(ctx, plainText, userID, metadata, embeddingGID); err != nil {
		if uploadedToS3 {
			s.deleteFromS3(ctx, s3Key)
		}
		_ = s.knowledgeRepo.DeleteByEmbeddingGID(ctx, embeddingGID)
		return uploadFailure(fmt.Sprintf("failed to enqueue summarization: %v", err))
	}

	readModel := mapper.ToKnowledgeRead(upserted)

	return v1schema.UploadKnowledgeResponse{
		Success: true,
		Result: v1schema.UploadKnowledgeResult{
			Msg:       fmt.Sprintf("Uploaded to %s", s3Key),
			Knowledge: &readModel,
		},
	}
}

func (s *KnowledgeService) uploadToS3(ctx context.Context, s3Key string, data []byte, contentType string) error {
	if s.s3Service == nil {
		return errors.New("S3 service not configured")
	}

	return s.s3Service.UploadFile(ctx, s.bucketName, s3Key, bytes.NewReader(data), contentType, map[string]string{
		"uploaded_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// deleteFromS3 attempts to remove an object from S3; best-effort cleanup.
func (s *KnowledgeService) deleteFromS3(ctx context.Context, s3Key string) {
	if s == nil || s.s3Service == nil || s.bucketName == "" {
		return
	}
	// Best-effort; ignore error to avoid masking original failures
	_ = s.s3Service.DeleteObject(ctx, s.bucketName, s3Key)
}

func (s *KnowledgeService) enqueueIndexing(ctx context.Context, content string, metadata map[string]interface{}, embeddingGID string) error {
	if s.workerClient == nil {
		return nil
	}

	payload := knowledgeIndexingPayload{
		Collection:   knowledgeCollection,
		Content:      content,
		EmbeddingGID: embeddingGID,
		Metadata:     metadata,
	}

	_, err := s.workerClient.EnqueueTask(ctx, knowledgeIndexingTask, payload)
	return err
}

func (s *KnowledgeService) enqueueSummarization(ctx context.Context, content string, userID uuid.UUID, metadata map[string]interface{}, embeddingGID string) error {
	if s.workerClient == nil {
		return nil
	}

	attributes := map[string]interface{}{
		"source_type":   string(domain.KnowledgeSourceUpload),
		"embedding_gid": embeddingGID,
		"user_id":       userID.String(),
		"cmetadata":     metadata,
	}

	payload := knowledgesSummarizePayload{
		Content:      content,
		EmbeddingGID: embeddingGID,
		Attributes:   attributes,
	}

	_, err := s.workerClient.EnqueueTask(ctx, knowledgesSummarizeTask, payload)
	return err
}

func extractText(data []byte, contentType string) (string, error) {
	switch contentType {
	case "application/pdf":
		reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return "", fmt.Errorf("failed to read pdf: %w", err)
		}
		// Read row by row: GetPlainText runs the text of consecutive lines
		// together with no separator, so "and\nexplains" was stored as
		// "andexplains" and the words never matched a query.
		var buf strings.Builder
		for i := 1; i <= reader.NumPage(); i++ {
			page := reader.Page(i)
			if page.V.IsNull() {
				continue
			}
			rows, err := page.GetTextByRow()
			if err != nil {
				return "", fmt.Errorf("failed to extract pdf text on page %d: %w", i, err)
			}
			for _, row := range rows {
				for _, word := range row.Content {
					buf.WriteString(word.S)
				}
				buf.WriteByte('\n')
			}
			buf.WriteByte('\n')
		}
		return cleanText(buf.String()), nil

	case "text/plain", "text/csv", "application/csv", "text/markdown",
		"application/json", "text/html", "application/xml", "text/xml":
		// For text-based files, return content as-is
		text := string(data)
		// Clean up non-printable characters
		return cleanText(text), nil

	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		text, err := extractDocxText(data)
		if err != nil {
			return "", fmt.Errorf("failed to extract .docx text: %w", err)
		}
		return cleanText(text), nil

	case "application/msword":
		// The legacy .doc format is a binary OLE compound file, not a ZIP of
		// XML, so the .docx parser cannot read it. Say so plainly instead of
		// failing with a confusing archive error.
		return "", fmt.Errorf("legacy .doc files are not supported; re-save the document as .docx")

	default:
		return "", fmt.Errorf("Unsupported file type, Got: %s", contentType)
	}
}

func buildEmbeddingGID(userID uuid.UUID, filename string, data []byte) string {
	stem := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	hasher := sha256.New()
	hasher.Write([]byte(userID.String()))
	hasher.Write([]byte(stem))
	hasher.Write(data)
	hashBytes := hasher.Sum(nil)
	encoded := base64.URLEncoding.EncodeToString(hashBytes)
	encoded = strings.TrimRight(encoded, "=")
	encoded = strings.ReplaceAll(encoded, "_", "-")
	if len(encoded) > 32 {
		encoded = encoded[:32]
	}
	return fmt.Sprintf("%s_%s", domain.KnowledgeSourceUpload, encoded)
}

func buildMetadataMap(userID uuid.UUID, fh *multipart.FileHeader, contentType, s3Key, embeddingGID string) map[string]interface{} {
	return map[string]interface{}{
		"gid":         embeddingGID,
		"user_id":     userID.String(),
		"source_type": string(domain.KnowledgeSourceUpload),
		"origin": map[string]interface{}{
			"filename":     filepath.Base(fh.Filename),
			"file_size":    fh.Size,
			"content_type": contentType,
		},
		"created_at": time.Now().UTC().Format(time.RFC3339),
		"s3_key":     s3Key,
	}
}

func buildS3Key(filename string) string {
	safeName := sanitizeFilename(filepath.Base(filename))
	timestamp := time.Now().UTC().Format("20060102T150405.000000000")
	unique := uuid.New().String()
	return fmt.Sprintf("knowledges/%s_%s_%s", timestamp, unique, safeName)
}

func sanitizeFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		filename = "knowledge"
	}
	filename = whitespacePattern.ReplaceAllString(filename, "_")
	filename = cleanText(filename)
	return filename
}

func uploadFailure(msg string) v1schema.UploadKnowledgeResponse {
	return v1schema.UploadKnowledgeResponse{
		Success: false,
		Result: v1schema.UploadKnowledgeResult{
			Msg: msg,
		},
	}
}

// cleanText normalises extracted text and drops characters that are not
// text. NFKC splits typographic ligatures, which PDFs use for "fi", "ff" and
// "fl", back into letters. Every printable character is kept, in any script:
// the previous pattern, [^[:print:]], is ASCII-only in Go, so it deleted every
// Vietnamese letter with a diacritic ("Giá gói" became "Gi gi") along with the
// ligatures ("first" became "rst").
func cleanText(s string) string {
	s = norm.NFKC.String(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			b.WriteRune(r)
		case r == utf8.RuneError:
			// undecodable bytes carry no text
		case unicode.IsPrint(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}
