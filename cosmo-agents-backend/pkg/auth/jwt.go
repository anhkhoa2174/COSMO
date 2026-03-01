package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token has expired")
	ErrInvalidSignature = errors.New("invalid token signature")
)

// TokenType distinguishes access/refresh/id tokens.
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
	TokenTypeID      TokenType = "id"

	defaultIDTokenDuration      = time.Hour
	defaultShortLivedDuration   = 30 * time.Minute
	defaultRefreshTokenDuration = 7 * 24 * time.Hour
)

// TokenDetails captures a signed JWT together with its expiry and raw claims.
type TokenDetails struct {
	Value  string
	Expiry time.Time
	Claims jwt.MapClaims
}

// Claims represents the subset of JWT claims the application cares about.
type Claims struct {
	UserID         uuid.UUID
	Email          string
	OrganizationID uuid.UUID
	Raw            jwt.MapClaims
}

// JWTManager handles JWT operations (generation and validation).
type JWTManager struct {
	accessSecret    []byte
	refreshSecret   []byte
	accessDuration  time.Duration
	refreshDuration time.Duration
	signingMethod   *jwt.SigningMethodHMAC
	issuer          string
	audience        string
}

// NewJWTManager creates a new JWT manager configured for access/refresh token handling.
func NewJWTManager(accessSecret string, accessDuration time.Duration, refreshSecret string, refreshDuration time.Duration, algorithm string) *JWTManager {
	// Fallbacks for legacy single-secret configs.
	if refreshSecret == "" {
		refreshSecret = accessSecret
	}
	if accessSecret == "" {
		accessSecret = refreshSecret
	}
	if algorithm == "" {
		algorithm = jwt.SigningMethodHS256.Alg()
	}

	method := jwt.GetSigningMethod(algorithm)
	hmacMethod, ok := method.(*jwt.SigningMethodHMAC)
	if !ok || hmacMethod == nil {
		hmacMethod = jwt.SigningMethodHS256
	}

	if accessDuration <= 0 {
		accessDuration = defaultShortLivedDuration
	}
	if refreshDuration <= 0 {
		refreshDuration = defaultRefreshTokenDuration
	}

	return &JWTManager{
		accessSecret:    []byte(accessSecret),
		refreshSecret:   []byte(refreshSecret),
		accessDuration:  accessDuration,
		refreshDuration: refreshDuration,
		signingMethod:   hmacMethod,
	}
}

// SetIssuer configures the JWT issuer claim value.
func (m *JWTManager) SetIssuer(issuer string) { m.issuer = issuer }

// SetAudience configures the JWT audience claim value.
func (m *JWTManager) SetAudience(aud string) { m.audience = aud }

// GenerateToken generates a new access token with default claims used by legacy handlers.
func (m *JWTManager) GenerateToken(userID uuid.UUID, email string, organizationID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"sub":        userID.String(),
		"user_id":    userID.String(),
		"userID":     userID.String(), // backward compatibility
		"email":      email,
		"user_email": email,
	}
	if organizationID != uuid.Nil {
		claims["organization_id"] = organizationID.String()
	}
	token, err := m.CreateAccessToken(claims, time.Time{})
	if err != nil {
		return "", err
	}
	return token.Value, nil
}

// GenerateTokenWithDuration generates an access token using a custom duration.
func (m *JWTManager) GenerateTokenWithDuration(userID uuid.UUID, email string, organizationID uuid.UUID, duration time.Duration) (string, error) {
	expiry := time.Now().UTC().Add(duration)
	claims := jwt.MapClaims{
		"sub":        userID.String(),
		"user_id":    userID.String(),
		"userID":     userID.String(),
		"email":      email,
		"user_email": email,
	}
	if organizationID != uuid.Nil {
		claims["organization_id"] = organizationID.String()
	}
	token, err := m.CreateAccessToken(claims, expiry)
	if err != nil {
		return "", err
	}
	return token.Value, nil
}

// CreateAccessToken builds and signs an access token.
func (m *JWTManager) CreateAccessToken(claims jwt.MapClaims, expiry time.Time) (TokenDetails, error) {
	return m.createToken(TokenTypeAccess, claims, expiry)
}

// CreateRefreshToken builds and signs a refresh token.
func (m *JWTManager) CreateRefreshToken(claims jwt.MapClaims, expiry time.Time) (TokenDetails, error) {
	return m.createToken(TokenTypeRefresh, claims, expiry)
}

// CreateIDToken builds and signs an ID token (shares the access token secret).
func (m *JWTManager) CreateIDToken(claims jwt.MapClaims, expiry time.Time) (TokenDetails, error) {
	return m.createToken(TokenTypeID, claims, expiry)
}

// ValidateToken validates an access token and returns the parsed claims.
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	claims, _, err := m.ParseAccessToken(tokenString)
	return claims, err
}

// ParseAccessToken parses and validates an access token.
func (m *JWTManager) ParseAccessToken(tokenString string) (*Claims, jwt.MapClaims, error) {
	return m.parseToken(tokenString, m.accessSecret)
}

// ParseRefreshToken parses and validates a refresh token.
func (m *JWTManager) ParseRefreshToken(tokenString string) (*Claims, jwt.MapClaims, error) {
	return m.parseToken(tokenString, m.refreshSecret)
}

// RefreshToken refreshes an access token from a refresh token (legacy helper returning only access token).
func (m *JWTManager) RefreshToken(refreshToken string) (string, error) {
	_, refreshClaims, err := m.ParseRefreshToken(refreshToken)
	if err != nil {
		return "", err
	}

	// Remove expiry-related claims to avoid clashing.
	delete(refreshClaims, "exp")
	delete(refreshClaims, "iat")
	delete(refreshClaims, "nbf")

	token, err := m.CreateAccessToken(refreshClaims, time.Time{})
	if err != nil {
		return "", err
	}
	return token.Value, nil
}

// GetExpirationSeconds returns the configured access token expiration duration in seconds.
func (m *JWTManager) GetExpirationSeconds() int {
	return int(m.accessDuration.Seconds())
}

func (m *JWTManager) createToken(tokenType TokenType, claims jwt.MapClaims, explicitExpiry time.Time) (TokenDetails, error) {
	now := time.Now().UTC()

	if claims == nil {
		claims = jwt.MapClaims{}
	} else {
		claims = cloneMapClaims(claims)
	}

	var expiry time.Time
	if !explicitExpiry.IsZero() {
		expiry = explicitExpiry.UTC()
	} else {
		switch tokenType {
		case TokenTypeRefresh:
			expiry = now.Add(m.refreshDuration)
		case TokenTypeID:
			expiry = now.Add(defaultIDTokenDuration)
		default:
			expiry = now.Add(m.accessDuration)
		}
	}

	claims["exp"] = expiry.Unix()
	claims["iat"] = now.Unix()
	claims["nbf"] = now.Unix()
	if m.issuer != "" {
		claims["iss"] = m.issuer
	}
	if m.audience != "" {
		claims["aud"] = m.audience
	}

	var secret []byte
	switch tokenType {
	case TokenTypeRefresh:
		secret = m.refreshSecret
	default:
		secret = m.accessSecret
	}

	token := jwt.NewWithClaims(m.signingMethod, claims)
	value, err := token.SignedString(secret)
	if err != nil {
		return TokenDetails{}, err
	}

	return TokenDetails{
		Value:  value,
		Expiry: expiry,
		Claims: claims,
	}, nil
}

func (m *JWTManager) parseToken(tokenString string, secret []byte) (*Claims, jwt.MapClaims, error) {
	if tokenString == "" {
		return nil, nil, ErrInvalidToken
	}

	mapClaims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		mapClaims,
		func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != m.signingMethod.Alg() {
				return nil, ErrInvalidSignature
			}
			return secret, nil
		},
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, nil, ErrExpiredToken
		}
		return nil, nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, nil, ErrInvalidToken
	}

	claims, err := mapClaimsToClaims(mapClaims)
	if err != nil {
		return nil, nil, err
	}

	return claims, mapClaims, nil
}

func mapClaimsToClaims(mapClaims jwt.MapClaims) (*Claims, error) {
	if mapClaims == nil {
		return nil, ErrInvalidToken
	}

	result := &Claims{Raw: mapClaims}

	// UserID: check multiple keys for compatibility with Python tokens.
	if sub, ok := mapClaims["sub"].(string); ok {
		if uid, err := uuid.Parse(sub); err == nil {
			result.UserID = uid
		}
	}
	if result.UserID == uuid.Nil {
		if idStr, ok := mapClaims["user_id"].(string); ok {
			if uid, err := uuid.Parse(idStr); err == nil {
				result.UserID = uid
			}
		}
	}
	if result.UserID == uuid.Nil {
		if idStr, ok := mapClaims["userID"].(string); ok {
			if uid, err := uuid.Parse(idStr); err == nil {
				result.UserID = uid
			}
		}
	}
	if result.UserID == uuid.Nil {
		return nil, ErrInvalidToken
	}

	// Email can be provided under different keys.
	if email, ok := mapClaims["email"].(string); ok && email != "" {
		result.Email = email
	} else if email, ok := mapClaims["user_email"].(string); ok && email != "" {
		result.Email = email
	}

	// Organization ID is optional.
	if org, ok := mapClaims["organization_id"].(string); ok {
		if oid, err := uuid.Parse(org); err == nil {
			result.OrganizationID = oid
		}
	}

	return result, nil
}

func cloneMapClaims(src jwt.MapClaims) jwt.MapClaims {
	if src == nil {
		return jwt.MapClaims{}
	}
	dst := jwt.MapClaims{}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
