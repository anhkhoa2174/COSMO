package oauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// OAuthStateSigner signs and verifies OAuth state tokens using HMAC
// This is stateless and works across multiple server instances
type OAuthStateSigner struct {
	secretKey []byte
}

// statePayload contains the data embedded in the state token
type statePayload struct {
	UserID    string `json:"user_id"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	Nonce     string `json:"nonce"` // Random value to make each state unique
}

// NewOAuthStateSigner creates a new signer with the given secret key
func NewOAuthStateSigner(secretKey string) *OAuthStateSigner {
	return &OAuthStateSigner{
		secretKey: []byte(secretKey),
	}
}

// CreateState generates a signed state token for the given user ID
func (s *OAuthStateSigner) CreateState(userID uuid.UUID, ttl time.Duration) (string, error) {
	now := time.Now()
	payload := statePayload{
		UserID:    userID.String(),
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(ttl).Unix(),
		Nonce:     uuid.New().String(), // Prevent state reuse
	}

	// Encode payload as JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Base64 encode payload
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)

	// Create HMAC signature
	signature := s.sign(payloadB64)

	// Return format: payload.signature
	return fmt.Sprintf("%s.%s", payloadB64, signature), nil
}

// VerifyState verifies a signed state token and returns the user ID
func (s *OAuthStateSigner) VerifyState(state string) (uuid.UUID, error) {
	// Split state into payload and signature
	parts := strings.Split(state, ".")
	if len(parts) != 2 {
		return uuid.Nil, fmt.Errorf("invalid state format")
	}

	payloadB64 := parts[0]
	signature := parts[1]

	// Verify signature
	expectedSignature := s.sign(payloadB64)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return uuid.Nil, fmt.Errorf("invalid state signature")
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	var payload statePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return uuid.Nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Check expiration
	if time.Now().Unix() > payload.ExpiresAt {
		return uuid.Nil, fmt.Errorf("state token expired")
	}

	// Parse user ID
	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID in state: %w", err)
	}

	return userID, nil
}

// sign creates an HMAC-SHA256 signature of the data
func (s *OAuthStateSigner) sign(data string) string {
	h := hmac.New(sha256.New, s.secretKey)
	h.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
