package service

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	knowledgeRepo "github.com/rockship/cosmo-agents-go/internal/repository/knowledge"
	s3Service "github.com/rockship/cosmo-agents-go/internal/service/s3"
)

type recordTransport struct {
	calls []string
}

func (rt *recordTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.calls = append(rt.calls, req.Method)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("ok")),
		Request:    req,
	}, nil
}

func newKnowledgeServiceWithDeps(t *testing.T) (*KnowledgeService, *knowledgeRepo.KnowledgeRepository, *recordTransport) {
	t.Helper()
	rt := &recordTransport{}
	client := s3.New(s3.Options{
		Region:           "us-east-1",
		Credentials:      credentials.NewStaticCredentialsProvider("key", "secret", ""),
		HTTPClient:       &http.Client{Transport: rt},
		EndpointResolver: s3.EndpointResolverFromURL("http://example.com"),
	})
	s3Svc := &s3Service.S3Service{}
	field := reflect.ValueOf(s3Svc).Elem().FieldByName("client")
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(client))
	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Knowledge{}))

	repo := knowledgeRepo.NewKnowledgeRepository(db)
	svc := NewKnowledgeService(repo, s3Svc, nil, "bucket")
	return svc, repo, rt
}

func TestKnowledgeServiceUploadStorageDisabled(t *testing.T) {
	svc := NewKnowledgeService(nil, nil, nil, "")

	files := []*multipart.FileHeader{{}}

	responses, err := svc.Upload(context.Background(), uuid.New(), files)
	assert.NoError(t, err)
	if assert.Len(t, responses, 1) {
		assert.False(t, responses[0].Success)
		assert.Equal(t, "knowledge upload not configured", responses[0].Result.Msg)
	}
}

func newFileHeader(t *testing.T, name, contentType, content string) *multipart.FileHeader {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", name)
	require.NoError(t, err)
	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	_, fh, err := req.FormFile("file")
	require.NoError(t, err)
	fh.Header.Set("Content-Type", contentType)
	return fh
}

func TestKnowledgeService_HandleUploadFailures(t *testing.T) {
	userID := uuid.New()
	svc := NewKnowledgeService(nil, nil, nil, "bucket")

	unsupported := newFileHeader(t, "file.bin", "application/octet-stream", "data")
	resp := svc.handleUpload(context.Background(), userID, unsupported)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Result.Msg, "Unsupported file type")

	plain := newFileHeader(t, "note.txt", "text/plain", "hello world")
	resp = svc.handleUpload(context.Background(), userID, plain)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Result.Msg, "S3 service not configured")
}

func TestKnowledgeService_HandleUploadSuccessAndCleanup(t *testing.T) {
	svc, repo, rt := newKnowledgeServiceWithDeps(t)
	userID := uuid.New()

	textFile := newFileHeader(t, "doc.txt", "text/plain", "hello world")
	resp := svc.handleUpload(context.Background(), userID, textFile)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, rt.calls)
	saved, err := repo.GetByEmbeddingGID(context.Background(), *resp.Result.Knowledge.EmbeddingGID)
	require.NoError(t, err)
	assert.NotNil(t, saved)

	// ownership conflict using fresh service with pre-existing knowledge for another user
	conflictSvc, conflictRepo, conflictRT := newKnowledgeServiceWithDeps(t)
	otherUser := uuid.New()
	existing := &domain.Knowledge{
		Base:         base.Base{ID: uuid.New()},
		UserID:       otherUser,
		EmbeddingGID: resp.Result.Knowledge.EmbeddingGID,
		SourceType:   domain.KnowledgeSourceUpload,
		CMetadata:    base.JSONB([]byte(`{}`)),
	}
	require.NoError(t, conflictRepo.GetDB().Create(existing).Error)
	conflictRT.calls = nil
	resp = conflictSvc.handleUpload(context.Background(), userID, textFile)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Result.Msg, "already exists")
	assert.Contains(t, conflictRT.calls, http.MethodDelete)
}

func TestExtractTextAndHelpers(t *testing.T) {
	txt, err := extractText([]byte("line1\nline2"), "text/plain")
	require.NoError(t, err)
	assert.Contains(t, txt, "line1")

	csv, err := extractText([]byte("a,b\nc,d"), "text/csv")
	require.NoError(t, err)
	assert.Contains(t, csv, "a,b")

	_, err = extractText([]byte("???"), "application/unknown")
	assert.Error(t, err)

	metadata := buildMetadataMap(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), newFileHeader(t, "doc.md", "text/markdown", "x"), "text/markdown", "s3key", "gid")
	assert.Equal(t, "gid", metadata["gid"])
	assert.Equal(t, "s3key", metadata["s3_key"])

	assert.NotEmpty(t, sanitizeFilename(" my file .txt "))
	assert.NotEqual(t, buildS3Key("a.txt"), buildS3Key("a.txt"))

	svc := NewKnowledgeService(nil, nil, nil, "")
	assert.NoError(t, svc.EnqueuePruning(context.Background(), nil))
	assert.NoError(t, svc.EnqueuePruning(context.Background(), []string{"gid1"}))
}

func TestBuildEmbeddingGIDDeterministic(t *testing.T) {
	data := []byte("hello world")
	userID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	id1 := buildEmbeddingGID(userID, "example.pdf", data)
	id2 := buildEmbeddingGID(userID, "example.pdf", data)

	assert.Equal(t, id1, id2)
	assert.Contains(t, id1, string(domain.KnowledgeSourceUpload))
}

func TestBuildEmbeddingGIDDifferentUsers(t *testing.T) {
	data := []byte("hello world")
	userOne := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	userTwo := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	id1 := buildEmbeddingGID(userOne, "example.pdf", data)
	id2 := buildEmbeddingGID(userTwo, "example.pdf", data)

	assert.NotEqual(t, id1, id2)
}

func TestSanitizeFilename(t *testing.T) {
	name := " My File \n.pdf "
	sanitized := sanitizeFilename(name)

	assert.NotEmpty(t, sanitized)
	assert.NotContains(t, sanitized, " ")
	assert.NotContains(t, sanitized, "\n")
}

func TestBuildS3KeyUnique(t *testing.T) {
	key1 := buildS3Key("example.pdf")
	key2 := buildS3Key("example.pdf")

	assert.NotEqual(t, key1, key2)
	assert.True(t, strings.HasPrefix(key1, "knowledges/"))
	assert.True(t, strings.HasSuffix(key1, "_example.pdf"))
}
