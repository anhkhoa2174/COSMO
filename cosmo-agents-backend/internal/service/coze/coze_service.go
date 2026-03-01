package coze

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// CozeService provides minimal operations to obtain Coze access tokens
type CozeService struct {
	AppID         string
	PublicKeyID   string
	PrivateKeyPEM []byte
	BaseURL       string
	HTTPClient    *http.Client
	privateKey    *rsa.PrivateKey
}

// NewCozeService loads configuration from environment variables and returns a service instance
// Required env vars: COZE_APP_ID, COZE_PUBLIC_KEY_ID, COZE_PRIVATE_KEY_PATH
// Optional: COZE_BASE_URL (default: https://api.coze.com)
func NewCozeService() (*CozeService, error) {
	appID := os.Getenv("COZE_APP_ID")
	kid := os.Getenv("COZE_PUBLIC_KEY_ID")
	keyPath := os.Getenv("COZE_PRIVATE_KEY_PATH")
	if appID == "" || kid == "" || keyPath == "" {
		return nil, errors.New("Coze configuration missing: require COZE_APP_ID, COZE_PUBLIC_KEY_ID, COZE_PRIVATE_KEY_PATH")
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return nil, errors.New("COZE private key file not found")
	} else if err != nil {
		return nil, fmt.Errorf("cannot access COZE private key: %w", err)
	}
	pemBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, errors.New("failed to read COZE private key file")
	}
	key, err := parseRSAPrivateKey(pemBytes)
	if err != nil {
		return nil, errors.New("invalid COZE private key format")
	}
	baseURL := os.Getenv("COZE_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.coze.com"
	}
	return &CozeService{
		AppID:         appID,
		PublicKeyID:   kid,
		PrivateKeyPEM: pemBytes,
		BaseURL:       baseURL,
		HTTPClient:    &http.Client{Timeout: 30 * time.Second},
		privateKey:    key,
	}, nil
}

// parseRSAPrivateKey parses RSA private key from PEM (PKCS1 or PKCS8)
func parseRSAPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("invalid PEM block in COZE private key")
	}
	// Try PKCS1 first
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	// Try PKCS8
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	rsaKey, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("COZE private key is not RSA")
	}
	return rsaKey, nil
}

// generateJWT builds a RS256-signed JWT with Coze headers/claims
func (s *CozeService) generateJWT(sessionName *string, durationSeconds int) (string, error) {
	if durationSeconds <= 0 || durationSeconds > 86399 {
		durationSeconds = 86399
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": s.AppID,
		"aud": "api.coze.com",
		"iat": now.Unix(),
		"exp": now.Add(time.Duration(durationSeconds) * time.Second).Unix(),
		"jti": uuid.NewString(),
	}
	if sessionName != nil && *sessionName != "" {
		claims["session_name"] = *sessionName
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.PublicKeyID

	return token.SignedString(s.privateKey)
}

// CozeAccessTokenResponse mirrors Coze token response
type CozeAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// GetAccessToken exchanges a signed JWT for a Coze access token
func (s *CozeService) GetAccessToken(ctx context.Context, sessionName *string, durationSeconds int) (*CozeAccessTokenResponse, error) {
	jwtStr, err := s.generateJWT(sessionName, durationSeconds)
	if err != nil {
		return nil, err
	}
	url := s.BaseURL + "/api/permission/oauth2/token"
	payload := map[string]interface{}{
		"duration_seconds": durationSeconds,
		"grant_type":       "urn:ietf:params:oauth:grant-type:jwt-bearer",
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwtStr)

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Log full response server-side, return sanitized error to client
		logger.Logger.Error().
			Int("status", resp.StatusCode).
			Str("response", string(respBody)).
			Msg("Coze token request failed")
		return nil, fmt.Errorf("coze token request failed with status %d", resp.StatusCode)
	}
	var out CozeAccessTokenResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("invalid coze token response: %w", err)
	}
	return &out, nil
}
