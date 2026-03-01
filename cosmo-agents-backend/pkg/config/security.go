package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"golang.org/x/crypto/scrypt"
)

// SecurityManager handles encryption and decryption of sensitive data
type SecurityManager struct {
	encryptionKey []byte
	environment   string
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(encKey string, environment string) (*SecurityManager, error) {
	if encKey == "" {
		return nil, fmt.Errorf("encryption key is required")
	}

	var key []byte
	var err error

	// Try to decode as base64 first
	key, err = base64.StdEncoding.DecodeString(encKey)
	if err != nil {
		// If not base64, use raw string
		key = []byte(encKey)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes")
	}

	return &SecurityManager{
		encryptionKey: key,
		environment:   environment,
	}, nil
}

// Encrypt encrypts sensitive data using AES-GCM (AEAD mode)
func (sm *SecurityManager) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(sm.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate a random nonce (12 bytes for GCM)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate - nonce should be first parameter, dst should be nil
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	// Prepend nonce to ciphertext for storage
	result := append(nonce, ciphertext...)

	// Return base64 encoded ciphertext with prepended nonce
	return base64.StdEncoding.EncodeToString(result), nil
}

// Decrypt decrypts sensitive data using AES-GCM (AEAD mode)
func (sm *SecurityManager) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(sm.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// HashPassword creates a secure hash of the password
func (sm *SecurityManager) HashPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	// Use scrypt for secure password hashing
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash, err := scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Store salt and hash together
	result := make([]byte, 64)
	copy(result, salt)
	copy(result[32:], hash)

	return base64.StdEncoding.EncodeToString(result), nil
}

// VerifyPassword verifies a password against its hash
func (sm *SecurityManager) VerifyPassword(password, hashedPassword string) bool {
	if password == "" || hashedPassword == "" {
		return false
	}

	data, err := base64.StdEncoding.DecodeString(hashedPassword)
	if err != nil {
		return false
	}

	if len(data) != 64 {
		return false
	}

	salt := data[:32]
	storedHash := data[32:]

	hash, err := scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare(storedHash, hash) == 1
}

// GenerateSecureToken generates a cryptographically secure random token
func (sm *SecurityManager) GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// GenerateAPIKey generates a secure API key
func (sm *SecurityManager) GenerateAPIKey() (string, error) {
	// Generate random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Add prefix and encode
	apiKey := "cosmo_" + base64.URLEncoding.EncodeToString(bytes)[:32]
	return apiKey, nil
}

// SanitizeString removes potentially dangerous characters using proper HTML sanitization
func (sm *SecurityManager) SanitizeString(input string) string {
	if input == "" {
		return ""
	}

	// Use bluemonday for proper HTML sanitization
	// Strict policy to remove all HTML/JS content
	p := bluemonday.StrictPolicy()

	// Sanitize the input to remove all HTML tags and potentially dangerous content
	sanitized := p.Sanitize(input)

	return strings.TrimSpace(sanitized)
}

// SanitizeFilename sanitizes a filename for security
func (sm *SecurityManager) SanitizeFilename(filename string) string {
	// Remove path separators and dangerous characters
	dangerous := []string{
		"..", "/", "\\", ":", "*", "?", "\"", "<", ">", "|",
		"\\", ":", "*", "?", "\"", "<", ">", "|",
	}

	result := filename
	for _, d := range dangerous {
		result = strings.ReplaceAll(result, d, "_")
	}

	// Remove leading/trailing dots and spaces
	result = strings.Trim(result, " ._")

	// Ensure filename is not empty
	if result == "" {
		result = "file"
	}

	return result
}

// MaskSensitiveData masks sensitive data for logging
func (sm *SecurityManager) MaskSensitiveData(data string) string {
	if data == "" {
		return ""
	}

	// If data looks like a secret, mask it
	if sm.looksLikeSecret(data) {
		return sm.maskString(data)
	}

	return data
}

// looksLikeSecret determines if a string looks like a secret key
func (sm *SecurityManager) looksLikeSecret(data string) bool {
	if len(data) < 8 {
		return false
	}

	// Common secret prefixes
	secretPrefixes := []string{
		"sk-", "pk_", "token_", "secret_", "password_", "key_",
		"auth_", "api_", "jwt_", "session_", "enc_",
	}

	lowerData := strings.ToLower(data)
	for _, prefix := range secretPrefixes {
		if strings.HasPrefix(lowerData, prefix) {
			return true
		}
	}

	// If it's long and contains no spaces, likely a secret
	if len(data) > 20 && !strings.Contains(data, " ") {
		return true
	}

	return false
}

// maskString masks a string for logging
func (sm *SecurityManager) maskString(data string) string {
	if len(data) <= 8 {
		return strings.Repeat("*", len(data))
	}

	return data[:4] + strings.Repeat("*", len(data)-8) + data[len(data)-4:]
}

// ValidateAPIKey validates an API key format
func (sm *SecurityManager) ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	if !strings.HasPrefix(apiKey, "cosmo_") {
		return fmt.Errorf("invalid API key format")
	}

	// Remove prefix and validate the remaining part
	keyPart := strings.TrimPrefix(apiKey, "cosmo_")
	if len(keyPart) < 20 {
		return fmt.Errorf("API key too short")
	}

	// Check if it's valid base64
	if _, err := base64.URLEncoding.DecodeString(keyPart); err != nil {
		return fmt.Errorf("invalid API key encoding")
	}

	return nil
}

// SecureRandomString generates a secure random string of specified length
func (sm *SecurityManager) SecureRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, length)
	rand.Read(bytes)

	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}

	return string(bytes)
}

// GenerateCSRFToken generates a CSRF token
func (sm *SecurityManager) GenerateCSRFToken() (string, error) {
	token, err := sm.GenerateSecureToken(32)
	if err != nil {
		return "", err
	}

	// Add timestamp and additional entropy
	timestamp := fmt.Sprintf("%d", os.Getpid())
	return fmt.Sprintf("%s_%s", token, timestamp), nil
}

// GetEnvironmentSecurityLevel returns security level based on environment
func (sm *SecurityManager) GetEnvironmentSecurityLevel() ValidationLevel {
	switch strings.ToLower(sm.environment) {
	case "production":
		return ProductionLevel
	case "staging":
		return StagingLevel
	default:
		return DevelopmentLevel
	}
}

// IsProduction returns true if running in production environment
func (sm *SecurityManager) IsProduction() bool {
	return sm.GetEnvironmentSecurityLevel() == ProductionLevel
}

// LogSecurityIssue logs a security issue with appropriate level
func (sm *SecurityManager) LogSecurityIssue(issue string, details interface{}) {
	if sm.IsProduction() {
		logger.Logger.Error().
			Str("issue", issue).
			Interface("details", details).
			Msg("Security issue detected")
	} else {
		logger.Logger.Warn().
			Str("issue", issue).
			Interface("details", details).
			Msg("Security issue detected")
	}
}

// CleanupTempFile securely removes a temporary file
func (sm *SecurityManager) CleanupTempFile(filepath string) error {
	if filepath == "" {
		return nil
	}

	// Overwrite the file with random data before deletion
	file, err := os.OpenFile(filepath, os.O_WRONLY, 0)
	if err == nil {
		defer file.Close()

		// Get file size
		stat, err := file.Stat()
		if err == nil && stat.Size() > 0 {
			randomData := make([]byte, stat.Size())
			rand.Read(randomData)
			file.Write(randomData)
		}
	}

	// Remove the file
	return os.Remove(filepath)
}
