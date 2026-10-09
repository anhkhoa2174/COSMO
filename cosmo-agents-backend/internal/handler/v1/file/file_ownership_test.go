package file

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	fileRepo "github.com/rockship/cosmo-agents-go/internal/repository/file"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
)

const testUserHeader = "X-Test-User"

// The bucket is shared by every user. These routes used to act on any key in
// it, so any signed-in user could list, download or delete anyone's files.
func TestS3RoutesOnlyReachTheCallersFiles(t *testing.T) {
	db, err := pgtest.Open(t, &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.File{}, &domain.Knowledge{}))
	repo := fileRepo.NewFileRepository(db)

	me, other := uuid.New(), uuid.New()
	mine := &domain.File{UserID: &me, Filename: "mine.pdf", S3Key: "examples/mine.pdf", Size: 10}
	theirs := &domain.File{UserID: &other, Filename: "invoice.pdf", S3Key: "examples/invoice.pdf", Size: 20}
	for _, f := range []*domain.File{mine, theirs} {
		_, err := repo.Create(t.Context(), f)
		require.NoError(t, err)
	}

	// No S3 client: every request below must be answered before S3 is touched.
	h := &FileHandler{fileRepo: repo, bucketName: "test"}
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		if id, err := uuid.Parse(c.Get(testUserHeader)); err == nil {
			c.Locals("user_id", id)
		}
		return c.Next()
	})
	app.Get("/s3/list", h.ListFromS3)
	app.Get("/s3", h.GetFromS3)
	app.Delete("/s3", h.DeleteFromS3)
	app.Get("/s3/presigned-url", h.GetPresignedURL)
	app.Post("/search", h.Search)
	// Exercises the ownership check alone; the real routes go on to S3,
	// which this test does not wire.
	app.Get("/owned", func(c fiber.Ctx) error {
		key, err := h.ownedKey(c)
		if key == "" {
			return err
		}
		return c.SendString(key)
	})

	do := func(method, path string, payload ...string) (int, []byte) {
		var rd io.Reader
		if len(payload) > 0 {
			rd = strings.NewReader(payload[0])
		}
		req := httptest.NewRequest(method, path, rd)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(testUserHeader, me.String())
		resp, err := app.Test(req)
		require.NoError(t, err)
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, body
	}

	// Knowledge-base uploads are rows in knowledges, not files, and the
	// library preview fetches them through this endpoint. Mine must pass the
	// ownership check (S3 is not wired in this test, so anything but 404 is
	// "allowed"); another user's must be 404.
	for _, k := range []struct {
		owner uuid.UUID
		key   string
	}{{me, "knowledges/mine.pdf"}, {other, "knowledges/theirs.pdf"}} {
		require.NoError(t, db.Exec(`INSERT INTO knowledges (id, user_id, source_type, type, cmetadata, is_deleted, created_at, updated_at)
			VALUES (?, ?, 'upload', 'other', ?, false, NOW(), NOW())`, uuid.New(), k.owner, `{"s3_key":"`+k.key+`"}`).Error)
	}
	t.Run("my knowledge document passes ownership", func(t *testing.T) {
		status, body := do(http.MethodGet, "/owned?key=knowledges/mine.pdf")
		assert.Equal(t, http.StatusOK, status, string(body))
	})
	t.Run("another user's knowledge document is not found", func(t *testing.T) {
		status, _ := do(http.MethodGet, "/owned?key=knowledges/theirs.pdf")
		assert.Equal(t, http.StatusNotFound, status)
	})

	t.Run("list shows only my files, with their names", func(t *testing.T) {
		status, body := do(http.MethodGet, "/s3/list")
		require.Equal(t, http.StatusOK, status, string(body))
		var out struct {
			Data []map[string]any `json:"data"`
		}
		require.NoError(t, json.Unmarshal(body, &out))
		require.Len(t, out.Data, 1)
		assert.Equal(t, "examples/mine.pdf", out.Data[0]["key"])
		assert.Equal(t, "mine.pdf", out.Data[0]["file_name"])
	})

	t.Run("search cannot reach another user's files", func(t *testing.T) {
		// Asking for the other user explicitly still returns only mine.
		status, body := do(http.MethodPost, "/search", `{"user_id":"`+other.String()+`"}`)
		require.Equal(t, http.StatusOK, status, string(body))
		var out struct {
			Data []map[string]any `json:"data"`
		}
		require.NoError(t, json.Unmarshal(body, &out))
		require.Len(t, out.Data, 1)
		assert.Equal(t, "examples/mine.pdf", out.Data[0]["s3_key"])
	})

	t.Run("search rejects a filter key that is not a column", func(t *testing.T) {
		// The key is spliced into the WHERE clause as a column name, so a
		// crafted key ran SQL of the client's choosing.
		status, _ := do(http.MethodPost, "/search", `{"(SELECT 1) = 1 OR user_id":"x"}`)
		assert.Equal(t, http.StatusBadRequest, status)
	})

	t.Run("search rejects raw SQL filters", func(t *testing.T) {
		status, _ := do(http.MethodPost, "/search", `{"filename":{"$raw":"1=1) OR (1=1"}}`)
		assert.Equal(t, http.StatusBadRequest, status)
	})

	for _, key := range []string{theirs.S3Key, "1C24TSA_00005252.pdf"} {
		for _, tc := range []struct{ method, path string }{
			{http.MethodGet, "/s3?key=" + key},
			{http.MethodDelete, "/s3?key=" + key},
			{http.MethodGet, "/s3/presigned-url?key=" + key},
		} {
			t.Run(tc.method+" "+tc.path, func(t *testing.T) {
				status, body := do(tc.method, tc.path)
				assert.Equal(t, http.StatusNotFound, status, string(body))
			})
		}
	}

	var n int64
	db.Model(&domain.File{}).Where("s3_key = ?", theirs.S3Key).Count(&n)
	assert.EqualValues(t, 1, n, "another user's file record must survive")
}
