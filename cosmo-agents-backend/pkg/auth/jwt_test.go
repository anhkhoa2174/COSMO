package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testAccessSecret  = "test-access-secret"
	testRefreshSecret = "test-refresh-secret"
)

func newTestJWTManager(accessDuration, refreshDuration time.Duration) *JWTManager {
	return NewJWTManager(testAccessSecret, accessDuration, testRefreshSecret, refreshDuration, "HS256")
}

func TestJWTManager_GenerateToken(t *testing.T) {
	manager := newTestJWTManager(1*time.Hour, 24*time.Hour)

	userID := uuid.New()
	email := "test@example.com"
	orgID := uuid.New()

	token, err := manager.GenerateToken(userID, email, orgID)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestJWTManager_ValidateToken(t *testing.T) {
	manager := newTestJWTManager(1*time.Hour, 24*time.Hour)

	userID := uuid.New()
	email := "test@example.com"
	orgID := uuid.New()

	t.Run("valid token", func(t *testing.T) {
		token, err := manager.GenerateToken(userID, email, orgID)
		require.NoError(t, err)

		claims, err := manager.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, orgID, claims.OrganizationID)
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := manager.ValidateToken("invalid-token")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("token with wrong signature", func(t *testing.T) {
		wrongManager := NewJWTManager("wrong-secret", 1*time.Hour, testRefreshSecret, 24*time.Hour, "HS256")
		token, err := wrongManager.GenerateToken(userID, email, orgID)
		require.NoError(t, err)

		_, err = manager.ValidateToken(token)
		assert.Error(t, err)
	})

	t.Run("expired token", func(t *testing.T) {
		token, err := manager.GenerateTokenWithDuration(userID, email, orgID, -1*time.Minute)
		require.NoError(t, err)

		_, err = manager.ValidateToken(token)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrExpiredToken)
	})
}

func TestJWTManager_RefreshToken(t *testing.T) {
	manager := newTestJWTManager(1*time.Hour, 24*time.Hour)

	userID := uuid.New()
	email := "test@example.com"
	orgID := uuid.New()

	t.Run("refresh valid token", func(t *testing.T) {
		refreshToken := createTestRefreshToken(t, manager, userID, email, orgID, time.Time{})

		newAccessToken, err := manager.RefreshToken(refreshToken)
		require.NoError(t, err)
		assert.NotEmpty(t, newAccessToken)

		claims, err := manager.ValidateToken(newAccessToken)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, orgID, claims.OrganizationID)
	})

	t.Run("refresh expired token", func(t *testing.T) {
		expiredAt := time.Now().Add(-time.Minute)
		refreshToken := createTestRefreshToken(t, manager, userID, email, orgID, expiredAt)

		_, err := manager.RefreshToken(refreshToken)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrExpiredToken)
	})

	t.Run("refresh invalid token", func(t *testing.T) {
		_, err := manager.RefreshToken("invalid-token")
		assert.Error(t, err)
	})
}

func TestJWTManager_GetExpirationSeconds(t *testing.T) {
	t.Run("1 hour duration", func(t *testing.T) {
		manager := newTestJWTManager(1*time.Hour, 24*time.Hour)
		assert.Equal(t, 3600, manager.GetExpirationSeconds())
	})

	t.Run("24 hours duration", func(t *testing.T) {
		manager := newTestJWTManager(24*time.Hour, 24*time.Hour)
		assert.Equal(t, 86400, manager.GetExpirationSeconds())
	})

	t.Run("custom duration", func(t *testing.T) {
		manager := newTestJWTManager(2*time.Hour+30*time.Minute, 24*time.Hour)
		assert.Equal(t, 9000, manager.GetExpirationSeconds())
	})
}

func createTestRefreshToken(
	t *testing.T,
	manager *JWTManager,
	userID uuid.UUID,
	email string,
	orgID uuid.UUID,
	expiry time.Time,
) string {
	t.Helper()

	claims := jwt.MapClaims{
		"sub":        userID.String(),
		"email":      email,
		"user_email": email,
	}
	if orgID != uuid.Nil {
		claims["organization_id"] = orgID.String()
	}

	var tokenDetails TokenDetails
	var err error
	if !expiry.IsZero() {
		tokenDetails, err = manager.CreateRefreshToken(claims, expiry)
	} else {
		tokenDetails, err = manager.CreateRefreshToken(claims, time.Time{})
	}
	require.NoError(t, err)
	return tokenDetails.Value
}
