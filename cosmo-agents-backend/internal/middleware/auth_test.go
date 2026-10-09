package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	personalapikey "github.com/rockship/cosmo-agents-go/internal/repository/personal_api_key"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/rockship/cosmo-agents-go/pkg/auth"
)

const (
	testAccessSecret  = "access-secret-for-tests"
	testRefreshSecret = "refresh-secret-for-tests"
)

func newTestJWTManager() *auth.JWTManager {
	return auth.NewJWTManager(testAccessSecret, time.Hour, testRefreshSecret, 24*time.Hour, "HS256")
}

// whoami echoes what the auth middleware stored, so a test can check both that
// the request got through and which identity it got through as.
func whoami(c fiber.Ctx) error {
	out := fiber.Map{}
	if id, ok := GetUserID(c); ok {
		out["user_id"] = id.String()
	}
	if id, ok := c.Locals("user_id").(uuid.UUID); ok {
		out["user_id_snake"] = id.String()
	}
	if email, ok := GetEmail(c); ok {
		out["email"] = email
	}
	if org, ok := GetOrganizationID(c); ok {
		out["organization_id"] = org.String()
	}
	if _, ok := GetClaims(c); ok {
		out["has_claims"] = true
	}
	return c.JSON(out)
}

func newAuthApp(m fiber.Handler) *fiber.App {
	app := fiber.New()
	app.Use(m)
	app.Get("/v1/whoami", whoami)
	app.Get("/v1/auth/login", func(c fiber.Ctx) error { return c.SendString("login") })
	app.Get("/v1/auth", func(c fiber.Ctx) error { return c.SendString("authorize") })
	app.Get("/v1/pubsub/:topic", func(c fiber.Ctx) error { return c.SendString("pubsub") })
	app.Get("/v1/auth/login/extra", func(c fiber.Ctx) error { return c.SendString("not public") })
	return app
}

type errorBody struct {
	Status string `json:"status"`
	Error  struct {
		ErrorCode int         `json:"error_code"`
		Message   string      `json:"message"`
		Detail    interface{} `json:"detail"`
	} `json:"error"`
}

func doReq(t *testing.T, app *fiber.App, req *http.Request) (int, []byte) {
	t.Helper()
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, body
}

func decodeError(t *testing.T, body []byte) errorBody {
	t.Helper()
	var eb errorBody
	require.NoError(t, json.Unmarshal(body, &eb), "body: %s", body)
	return eb
}

func signHS(t *testing.T, method jwt.SigningMethod, secret string, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString([]byte(secret))
	require.NoError(t, err)
	return s
}

func baseClaims(userID uuid.UUID) jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"sub":   userID.String(),
		"email": "alice@example.com",
		"exp":   now.Add(time.Hour).Unix(),
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
	}
}

// Every way a bearer token can be wrong must end in the same 401 envelope; a
// 500 or a pass-through here would mean an unauthenticated request reaches a
// handler.
func TestAuthMiddleware_RejectsBadTokens(t *testing.T) {
	m := newTestJWTManager()
	app := newAuthApp(AuthMiddleware(m, nil, nil))
	userID := uuid.New()

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	rs256, err := jwt.NewWithClaims(jwt.SigningMethodRS256, baseClaims(userID)).SignedString(rsaKey)
	require.NoError(t, err)

	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, baseClaims(userID)).SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	expired := baseClaims(userID)
	expired["exp"] = time.Now().Add(-time.Minute).Unix()

	notYetValid := baseClaims(userID)
	notYetValid["nbf"] = time.Now().Add(time.Hour).Unix()

	noSubject := baseClaims(userID)
	delete(noSubject, "sub")

	badSubject := baseClaims(userID)
	badSubject["sub"] = "not-a-uuid"

	refresh, err := m.CreateRefreshToken(jwt.MapClaims{"sub": userID.String()}, time.Time{})
	require.NoError(t, err)

	tests := []struct {
		name        string
		header      string
		wantMessage string
	}{
		{"missing header", "", "Missing authorization header"},
		{"no scheme", signHS(t, jwt.SigningMethodHS256, testAccessSecret, baseClaims(userID)), "Invalid authorization format"},
		{"basic scheme", "Basic dXNlcjpwYXNz", "Invalid authorization format"},
		{"bearer without token", "Bearer", "Invalid authorization format"},
		{"extra segment", "Bearer a b", "Invalid authorization format"},
		{"garbage token", "Bearer not.a.jwt", "Invalid or expired token"},
		{"wrong secret", "Bearer " + signHS(t, jwt.SigningMethodHS256, "some-other-secret", baseClaims(userID)), "Invalid or expired token"},
		{"expired", "Bearer " + signHS(t, jwt.SigningMethodHS256, testAccessSecret, expired), "Invalid or expired token"},
		{"not yet valid", "Bearer " + signHS(t, jwt.SigningMethodHS256, testAccessSecret, notYetValid), "Invalid or expired token"},
		// Same secret, different HMAC: the manager pins the algorithm.
		{"HS512 instead of HS256", "Bearer " + signHS(t, jwt.SigningMethodHS512, testAccessSecret, baseClaims(userID)), "Invalid or expired token"},
		{"RS256 algorithm confusion", "Bearer " + rs256, "Invalid or expired token"},
		{"alg none", "Bearer " + none, "Invalid or expired token"},
		{"no subject claim", "Bearer " + signHS(t, jwt.SigningMethodHS256, testAccessSecret, noSubject), "Invalid or expired token"},
		{"subject not a uuid", "Bearer " + signHS(t, jwt.SigningMethodHS256, testAccessSecret, badSubject), "Invalid or expired token"},
		// Refresh tokens are signed with their own secret and must not be
		// accepted where an access token is expected.
		{"refresh token as access token", "Bearer " + refresh.Value, "Invalid or expired token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			status, body := doReq(t, app, req)
			assert.Equal(t, fiber.StatusUnauthorized, status, "body: %s", body)
			eb := decodeError(t, body)
			assert.Equal(t, "error", eb.Status)
			assert.Equal(t, fiber.StatusUnauthorized, eb.Error.ErrorCode)
			assert.Equal(t, tt.wantMessage, eb.Error.Message)
		})
	}
}

func TestAuthMiddleware_ValidTokenPopulatesLocals(t *testing.T) {
	m := newTestJWTManager()
	app := newAuthApp(AuthMiddleware(m, nil, nil))
	userID, orgID := uuid.New(), uuid.New()

	token, err := m.GenerateToken(userID, "alice@example.com", orgID)
	require.NoError(t, err)

	t.Run("header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		status, body := doReq(t, app, req)
		require.Equal(t, fiber.StatusOK, status, "body: %s", body)

		var got map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &got))
		// Handlers read both spellings of the user id key.
		assert.Equal(t, userID.String(), got["user_id"])
		assert.Equal(t, userID.String(), got["user_id_snake"])
		assert.Equal(t, "alice@example.com", got["email"])
		assert.Equal(t, orgID.String(), got["organization_id"])
		assert.Equal(t, true, got["has_claims"])
	})

	// EventSource cannot set headers, so SSE clients pass ?token=.
	t.Run("query token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/whoami?token="+token, nil)
		status, body := doReq(t, app, req)
		require.Equal(t, fiber.StatusOK, status, "body: %s", body)
		assert.Contains(t, string(body), userID.String())
	})

	// The header wins over the query string when both are present, so a
	// crafted link cannot swap the identity of a logged-in client.
	t.Run("header beats query", func(t *testing.T) {
		other, err := m.GenerateToken(uuid.New(), "mallory@example.com", uuid.Nil)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, "/v1/whoami?token="+other, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		status, body := doReq(t, app, req)
		require.Equal(t, fiber.StatusOK, status)
		assert.Contains(t, string(body), userID.String())
		assert.NotContains(t, string(body), "mallory")
	})

	t.Run("invalid query token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/whoami?token=garbage", nil)
		status, _ := doReq(t, app, req)
		assert.Equal(t, fiber.StatusUnauthorized, status)
	})
}

func TestAuthMiddleware_PublicPaths(t *testing.T) {
	app := newAuthApp(AuthMiddleware(newTestJWTManager(), nil, nil))

	tests := []struct {
		path       string
		wantStatus int
	}{
		{"/v1/auth/login", fiber.StatusOK},
		{"/v1/auth", fiber.StatusOK},
		{"/v1/pubsub/topic-1", fiber.StatusOK},
		// Public paths are exact matches: a longer path under a public one
		// is still protected.
		{"/v1/auth/login/extra", fiber.StatusUnauthorized},
		{"/v1/whoami", fiber.StatusUnauthorized},
		// Prefix matching must not be escapable with dot segments.
		{"/v1/pubsub/../whoami", fiber.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			status, body := doReq(t, app, req)
			if tt.wantStatus == fiber.StatusNotFound {
				// Whatever the router does with the dot segment, it must not
				// hand back the protected handler's output.
				assert.NotEqual(t, fiber.StatusOK, status, "body: %s", body)
				return
			}
			assert.Equal(t, tt.wantStatus, status, "body: %s", body)
		})
	}
}

func TestOptionalAuth(t *testing.T) {
	m := newTestJWTManager()
	app := newAuthApp(OptionalAuth(m, nil, nil))
	userID := uuid.New()
	token, err := m.GenerateToken(userID, "alice@example.com", uuid.Nil)
	require.NoError(t, err)

	tests := []struct {
		name         string
		header       string
		wantIdentity bool
	}{
		{"anonymous", "", false},
		{"valid", "Bearer " + token, true},
		{"invalid token passes through anonymously", "Bearer nope", false},
		{"wrong scheme passes through anonymously", "Token " + token, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			status, body := doReq(t, app, req)
			require.Equal(t, fiber.StatusOK, status)
			if tt.wantIdentity {
				assert.Contains(t, string(body), userID.String())
			} else {
				assert.JSONEq(t, `{}`, string(body))
			}
		})
	}
}

func TestRequireOrganization(t *testing.T) {
	m := newTestJWTManager()
	app := fiber.New()
	app.Use(AuthMiddleware(m, nil, nil))
	app.Get("/v1/org-only", RequireOrganization(), func(c fiber.Ctx) error { return c.SendString("ok") })

	withOrg, err := m.GenerateToken(uuid.New(), "a@example.com", uuid.New())
	require.NoError(t, err)
	withoutOrg, err := m.GenerateToken(uuid.New(), "b@example.com", uuid.Nil)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/v1/org-only", nil)
	req.Header.Set("Authorization", "Bearer "+withOrg)
	status, _ := doReq(t, app, req)
	assert.Equal(t, fiber.StatusOK, status)

	req = httptest.NewRequest(http.MethodGet, "/v1/org-only", nil)
	req.Header.Set("Authorization", "Bearer "+withoutOrg)
	status, body := doReq(t, app, req)
	assert.Equal(t, fiber.StatusForbidden, status)
	assert.Equal(t, "Organization required", decodeError(t, body).Error.Message)
}

// --- personal API keys -------------------------------------------------------

type apiKeyFixture struct {
	db     *gorm.DB
	keys   *personalapikey.PersonalApiKeyRepository
	users  *userRepo.UserRepository
	userID uuid.UUID
}

func newAPIKeyFixture(t *testing.T) *apiKeyFixture {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.User{}, &domain.PersonalApiKey{}))

	u := &domain.User{Email: "  keyholder@example.com ", Name: "Key Holder"}
	require.NoError(t, db.Create(u).Error)

	return &apiKeyFixture{
		db:     db,
		keys:   personalapikey.NewPersonalApiKeyRepository(db, "api-key-hmac-secret"),
		users:  userRepo.NewUserRepository(db),
		userID: u.ID,
	}
}

// issue stores a key for the fixture user (or for owner, when given) and
// returns the raw value a client would send.
func (f *apiKeyFixture) issue(t *testing.T, expiresAt time.Time, owner ...uuid.UUID) (string, uuid.UUID) {
	t.Helper()
	raw, hashed, err := f.keys.GenerateAPIKey()
	require.NoError(t, err)
	uid := f.userID
	if len(owner) > 0 {
		uid = owner[0]
	}
	k := &domain.PersonalApiKey{UserID: uid, Name: "ci", HashedKey: hashed, Prefix: raw[:14], ExpiresAt: expiresAt}
	_, err = f.keys.Create(t.Context(), k)
	require.NoError(t, err)
	return raw, k.ID
}

func TestAuthMiddleware_PersonalAPIKey(t *testing.T) {
	f := newAPIKeyFixture(t)
	app := newAuthApp(AuthMiddleware(newTestJWTManager(), f.keys, f.users))

	valid, validID := f.issue(t, time.Now().Add(24*time.Hour))
	expired, _ := f.issue(t, time.Now().Add(-time.Hour))
	orphan, _ := f.issue(t, time.Now().Add(24*time.Hour), uuid.New())

	t.Run("valid key authenticates as its owner", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
		req.Header.Set("Authorization", "Bearer "+valid)
		status, body := doReq(t, app, req)
		require.Equal(t, fiber.StatusOK, status, "body: %s", body)

		var got map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &got))
		assert.Equal(t, f.userID.String(), got["user_id"])
		assert.Equal(t, f.userID.String(), got["user_id_snake"])
		assert.Equal(t, "keyholder@example.com", got["email"], "email is trimmed")
		// A key carries no organisation claim.
		assert.NotContains(t, got, "organization_id")

		var stored domain.PersonalApiKey
		require.NoError(t, f.db.First(&stored, "id = ?", validID).Error)
		assert.NotNil(t, stored.LastUsedAt, "last_used_at is stamped on use")
	})

	tests := []struct {
		name        string
		token       string
		wantMessage string
	}{
		{"unknown key", personalapikey.KEY_PREFIX + "does-not-exist", "Invalid or expired token"},
		{"expired key", expired, "Personal API key expired"},
		{"key whose user is gone", orphan, "Invalid or expired token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
			req.Header.Set("Authorization", "Bearer "+tt.token)
			status, body := doReq(t, app, req)
			assert.Equal(t, fiber.StatusUnauthorized, status, "body: %s", body)
			assert.Equal(t, tt.wantMessage, decodeError(t, body).Error.Message)
		})
	}

	t.Run("key prefix with no repository configured", func(t *testing.T) {
		bare := newAuthApp(AuthMiddleware(newTestJWTManager(), nil, nil))
		req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
		req.Header.Set("Authorization", "Bearer "+valid)
		status, body := doReq(t, bare, req)
		assert.Equal(t, fiber.StatusUnauthorized, status)
		assert.Equal(t, "Personal API key authentication not configured", decodeError(t, body).Error.Message)
	})

	t.Run("repository without a secret", func(t *testing.T) {
		noSecret := newAuthApp(AuthMiddleware(newTestJWTManager(), personalapikey.NewPersonalApiKeyRepository(f.db, ""), f.users))
		req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
		req.Header.Set("Authorization", "Bearer "+valid)
		status, body := doReq(t, noSecret, req)
		assert.Equal(t, fiber.StatusUnauthorized, status)
		assert.Equal(t, "Personal API key authentication not configured", decodeError(t, body).Error.Message)
	})

	t.Run("without a user repository the key still authenticates", func(t *testing.T) {
		noUsers := newAuthApp(AuthMiddleware(newTestJWTManager(), f.keys, nil))
		req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
		req.Header.Set("Authorization", "Bearer "+valid)
		status, body := doReq(t, noUsers, req)
		require.Equal(t, fiber.StatusOK, status)
		assert.Contains(t, string(body), f.userID.String())
		assert.NotContains(t, string(body), "email")
	})

	t.Run("database failure is a 500, not a 401", func(t *testing.T) {
		broken, err := pgtest.Open(t, nil) // schema without the table
		require.NoError(t, err)
		app := newAuthApp(AuthMiddleware(newTestJWTManager(), personalapikey.NewPersonalApiKeyRepository(broken, "api-key-hmac-secret"), f.users))
		req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
		req.Header.Set("Authorization", "Bearer "+valid)
		status, body := doReq(t, app, req)
		assert.Equal(t, fiber.StatusInternalServerError, status, "body: %s", body)
		assert.Equal(t, "Failed to validate personal API key", decodeError(t, body).Error.Message)
	})
}

func TestOptionalAuth_PersonalAPIKey(t *testing.T) {
	f := newAPIKeyFixture(t)
	app := newAuthApp(OptionalAuth(newTestJWTManager(), f.keys, f.users))
	valid, _ := f.issue(t, time.Now().Add(time.Hour))
	expired, _ := f.issue(t, time.Now().Add(-time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
	req.Header.Set("Authorization", "Bearer "+valid)
	status, body := doReq(t, app, req)
	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, string(body), f.userID.String())
	assert.Contains(t, string(body), "keyholder@example.com")

	req = httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
	req.Header.Set("Authorization", "Bearer "+expired)
	status, body = doReq(t, app, req)
	require.Equal(t, fiber.StatusOK, status)
	assert.JSONEq(t, `{}`, string(body), "an expired key yields an anonymous request")
}
