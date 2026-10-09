package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCORS_DefaultConfig(t *testing.T) {
	app := fiber.New()
	app.Use(CORS(DefaultCORSConfig()))
	app.Get("/x", func(c fiber.Ctx) error { return c.SendString("ok") })

	t.Run("preflight", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/x", nil)
		req.Header.Set("Origin", "https://app.example.com")
		req.Header.Set("Access-Control-Request-Method", http.MethodPatch)
		req.Header.Set("Access-Control-Request-Headers", "Authorization")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Contains(t, resp.Header.Get("Access-Control-Allow-Methods"), http.MethodPatch)
		assert.Contains(t, resp.Header.Get("Access-Control-Allow-Headers"), "Authorization")
		assert.Equal(t, "3600", resp.Header.Get("Access-Control-Max-Age"))
	})

	t.Run("simple request exposes the request id header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", "https://app.example.com")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Contains(t, resp.Header.Get("Access-Control-Expose-Headers"), RequestIDHeader)
		// A wildcard origin must never be combined with credentials.
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Credentials"))
	})
}

func TestCORS_RestrictedOrigins(t *testing.T) {
	cfg := DefaultCORSConfig()
	cfg.AllowOrigins = []string{"https://app.example.com"}
	cfg.AllowCredentials = true
	app := fiber.New()
	app.Use(CORS(cfg))
	app.Get("/x", func(c fiber.Ctx) error { return c.SendString("ok") })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://app.example.com")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, "https://app.example.com", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", resp.Header.Get("Access-Control-Allow-Credentials"))

	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	resp, err = app.Test(req)
	require.NoError(t, err)
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestRateLimitConfigs(t *testing.T) {
	d := DefaultRateLimitConfig()
	assert.Equal(t, 100, d.Max)
	assert.Equal(t, time.Minute, d.Expiration)
	assert.True(t, d.ByIP)
	assert.False(t, d.ByUser)

	s := StrictRateLimitConfig()
	assert.Equal(t, 10, s.Max)
	assert.Less(t, s.Max, d.Max)
}

func rateLimitStatus(t *testing.T, app *fiber.App, user string) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if user != "" {
		req.Header.Set("X-Test-User", user)
	}
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	var body map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp.StatusCode, body
}

func TestPerIPRateLimit(t *testing.T) {
	app := fiber.New()
	app.Use(PerIPRateLimit(2, time.Minute))
	app.Get("/x", func(c fiber.Ctx) error { return c.SendString("ok") })

	for i := 0; i < 2; i++ {
		status, _ := rateLimitStatus(t, app, "")
		require.Equal(t, fiber.StatusOK, status, "request %d", i+1)
	}
	status, body := rateLimitStatus(t, app, "")
	assert.Equal(t, fiber.StatusTooManyRequests, status)
	assert.Equal(t, "error", body["status"])
	errObj, _ := body["error"].(map[string]interface{})
	assert.Equal(t, float64(fiber.StatusTooManyRequests), errObj["code"])
	assert.Equal(t, "Rate limit exceeded", errObj["message"])
}

// Per-user limiting keys on the authenticated user, so one user exhausting
// their budget must not throttle another user behind the same IP.
func TestPerUserRateLimit(t *testing.T) {
	alice, bob := uuid.New(), uuid.New()
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		switch c.Get("X-Test-User") {
		case "alice":
			c.Locals("userID", alice)
		case "bob":
			c.Locals("userID", bob)
		}
		return c.Next()
	})
	app.Use(RequestID())
	app.Use(PerUserRateLimit(1, time.Minute))
	app.Get("/x", func(c fiber.Ctx) error { return c.SendString("ok") })

	status, _ := rateLimitStatus(t, app, "alice")
	require.Equal(t, fiber.StatusOK, status)
	status, _ = rateLimitStatus(t, app, "alice")
	assert.Equal(t, fiber.StatusTooManyRequests, status, "alice is over her budget")
	status, _ = rateLimitStatus(t, app, "bob")
	assert.Equal(t, fiber.StatusOK, status, "bob has his own budget")

	// Anonymous requests on a per-user limiter fall back to the request id,
	// which is unique per request.
	for i := 0; i < 3; i++ {
		status, _ = rateLimitStatus(t, app, "")
		assert.Equal(t, fiber.StatusOK, status)
	}
}

func TestRecovery(t *testing.T) {
	app := fiber.New()
	app.Use(RequestID())
	app.Use(Recovery())
	app.Get("/boom", func(c fiber.Ctx) error { panic("kaboom") })
	app.Get("/fine", func(c fiber.Ctx) error { return c.SendString("ok") })

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/boom", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "error", body["status"])
	errObj := body["error"].(map[string]interface{})
	assert.Equal(t, "Internal server error", errObj["message"])
	assert.Equal(t, "kaboom", errObj["details"])

	resp, err = app.Test(httptest.NewRequest(http.MethodGet, "/fine", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestRequestID(t *testing.T) {
	app := fiber.New()
	app.Use(RequestID())
	app.Get("/x", func(c fiber.Ctx) error { return c.SendString(GetRequestID(c)) })

	t.Run("generated when absent", func(t *testing.T) {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/x", nil))
		require.NoError(t, err)
		id := resp.Header.Get(RequestIDHeader)
		_, parseErr := uuid.Parse(id)
		assert.NoError(t, parseErr)
		var buf [64]byte
		n, _ := resp.Body.Read(buf[:])
		assert.Equal(t, id, string(buf[:n]), "handlers see the same id as the header")
	})

	t.Run("propagated when supplied", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set(RequestIDHeader, "upstream-123")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, "upstream-123", resp.Header.Get(RequestIDHeader))
	})

	t.Run("empty outside the middleware", func(t *testing.T) {
		bare := fiber.New()
		bare.Get("/x", func(c fiber.Ctx) error { return c.SendString("[" + GetRequestID(c) + "]") })
		resp, err := bare.Test(httptest.NewRequest(http.MethodGet, "/x", nil))
		require.NoError(t, err)
		var buf [8]byte
		n, _ := resp.Body.Read(buf[:])
		assert.Equal(t, "[]", string(buf[:n]))
	})
}

// Logger wraps the handler: they must pass its status and error
// through unchanged, including for the skipped health paths.
func TestLoggerPassThrough(t *testing.T) {
	app := fiber.New()
	app.Use(RequestID())
	app.Use(Logger())
	app.Get("/health", func(c fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/created", func(c fiber.Ctx) error { return c.Status(fiber.StatusCreated).SendString("made") })
	app.Get("/err", func(c fiber.Ctx) error { return fiber.NewError(fiber.StatusConflict, "clash") })

	tests := []struct {
		path string
		want int
	}{
		{"/health", fiber.StatusOK},
		{"/created", fiber.StatusCreated},
		{"/err", fiber.StatusConflict},
		{"/missing", fiber.StatusNotFound},
	}
	for _, tt := range tests {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, tt.path, nil))
		require.NoError(t, err)
		assert.Equal(t, tt.want, resp.StatusCode, tt.path)
	}
}

func TestAuthProviderMiddleware(t *testing.T) {
	app := fiber.New()
	app.Use(AuthProviderMiddleware())
	app.Get("/x", func(c fiber.Ctx) error { return c.SendString(c.Locals("auth_provider").(string)) })

	tests := []struct {
		header string
		want   string
	}{
		{"", DefaultAuthProvider},
		{"Microsoft", "microsoft"},
		{"GOOGLE", "google"},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		if tt.header != "" {
			req.Header.Set(AuthProviderHeader, tt.header)
		}
		resp, err := app.Test(req)
		require.NoError(t, err)
		var buf [32]byte
		n, _ := resp.Body.Read(buf[:])
		assert.Equal(t, tt.want, string(buf[:n]))
	}
}
