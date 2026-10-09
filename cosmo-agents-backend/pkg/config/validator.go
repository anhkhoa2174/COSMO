package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// ValidationLevel represents the strictness level of configuration validation
type ValidationLevel int

const (
	// DevelopmentLevel is for development environments - more permissive
	DevelopmentLevel ValidationLevel = iota
	// StagingLevel is for staging environments - standard validation
	StagingLevel
	// ProductionLevel is for production environments - strictest validation
	ProductionLevel
)

// ValidationResult contains validation errors and warnings
type ValidationResult struct {
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
	Valid    bool     `json:"valid"`
}

// AddError adds an error to the validation result
func (vr *ValidationResult) AddError(message string) {
	vr.Errors = append(vr.Errors, message)
	vr.Valid = false
}

// AddWarning adds a warning to the validation result
func (vr *ValidationResult) AddWarning(message string) {
	vr.Warnings = append(vr.Warnings, message)
}

// HasErrors returns true if there are validation errors
func (vr *ValidationResult) HasErrors() bool {
	return len(vr.Errors) > 0
}

// HasWarnings returns true if there are validation warnings
func (vr *ValidationResult) HasWarnings() bool {
	return len(vr.Warnings) > 0
}

// ConfigValidator provides comprehensive configuration validation
type ConfigValidator struct {
	validationLevel ValidationLevel
	environment     string
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator(environment string) *ConfigValidator {
	validationLevel := DevelopmentLevel
	switch strings.ToLower(environment) {
	case "production":
		validationLevel = ProductionLevel
	case "staging":
		validationLevel = StagingLevel
	}

	return &ConfigValidator{
		validationLevel: validationLevel,
		environment:     environment,
	}
}

// Validate performs comprehensive configuration validation
func (cv *ConfigValidator) Validate(cfg *Config) *ValidationResult {
	result := &ValidationResult{Valid: true}

	cv.validateAuth(cfg, result)
	cv.validateDatabase(cfg, result)
	cv.validateRedis(cfg, result)
	cv.validateAI(cfg, result)
	cv.validateS3(cfg, result)
	cv.validateEmail(cfg, result)
	cv.validateOAuth(cfg, result)

	return result
}

// validateAuth validates authentication configuration
func (cv *ConfigValidator) validateAuth(cfg *Config, result *ValidationResult) {
	// JWT Secret validation
	if cfg.Auth.JWTSecret == "" {
		result.AddError("JWT_SECRET is required")
	} else if strings.EqualFold(cfg.Auth.JWTSecret, "change-me-in-production") {
		result.AddError("JWT_SECRET must be changed from default value")
	} else if cv.validationLevel >= StagingLevel {
		if len(cfg.Auth.JWTSecret) < 32 {
			result.AddError("JWT_SECRET must be at least 32 characters for security")
		}
		if cv.isWeakSecret(cfg.Auth.JWTSecret) {
			result.AddError("JWT_SECRET is too weak. Use a cryptographically secure random string")
		}
	}

	// Credentials encryption key validation for production
	if cv.validationLevel == ProductionLevel {
		if cfg.Auth.CredentialsEncKey == "" {
			result.AddError("AUTH_CREDENTIALS_ENC_KEY is required in production")
		} else if err := cv.validateCredentialsEncKey(cfg.Auth.CredentialsEncKey); err != nil {
			result.AddError(fmt.Sprintf("AUTH_CREDENTIALS_ENC_KEY validation failed: %v", err))
		}
	}

	// Access token secret validation - only required if auth is enabled
	if cfg.Auth.EnableAuth && cfg.Auth.AccessTokenSecret == "" {
		result.AddError("ACCESS_TOKEN_SECRET_KEY is required when ENABLE_AUTH=true")
	} else if cfg.Auth.AccessTokenSecret != "" && len(cfg.Auth.AccessTokenSecret) < 16 {
		result.AddError("ACCESS_TOKEN_SECRET_KEY must be at least 16 characters")
	}

	// Refresh token secret validation - only required if auth is enabled
	if cfg.Auth.EnableAuth && cfg.Auth.RefreshTokenSecret == "" {
		result.AddError("REFRESH_TOKEN_SECRET_KEY is required when ENABLE_AUTH=true")
	} else if cfg.Auth.RefreshTokenSecret != "" && len(cfg.Auth.RefreshTokenSecret) < 16 {
		result.AddError("REFRESH_TOKEN_SECRET_KEY must be at least 16 characters")
	}

	// Personal API key secret validation
	if cv.validationLevel >= StagingLevel && cfg.Auth.PersonalAPIKeySecret == "" {
		result.AddWarning("PERSONAL_API_KEY_SECRET is recommended for security in staging and production")
	}

	// Token expiration validation
	if cfg.Auth.AccessTokenExpireDays <= 0 {
		result.AddError("ACCESS_TOKEN_EXPIRE_DAYS must be positive")
	}
	if cfg.Auth.RefreshTokenExpireDays <= 0 {
		result.AddError("REFRESH_TOKEN_EXPIRE_DAYS must be positive")
	}
	if cfg.Auth.RefreshTokenExpireDays < cfg.Auth.AccessTokenExpireDays {
		result.AddError("REFRESH_TOKEN_EXPIRE_DAYS must be greater than ACCESS_TOKEN_EXPIRE_DAYS")
	}
}

// validateDatabase validates database configuration
func (cv *ConfigValidator) validateDatabase(cfg *Config, result *ValidationResult) {
	if cfg.Database.PostgresURI == "" {
		result.AddError("SQLALCHEMY_POSTGRES_URI is required")
	}

	// Connection pool validation
	if cfg.Database.MaxOpenConn <= 0 {
		result.AddError("DB_MAX_OPEN_CONN must be positive")
	}
	if cfg.Database.MaxIdleConn < 0 {
		result.AddError("DB_MAX_IDLE_CONN cannot be negative")
	}
	if cfg.Database.MaxIdleConn > cfg.Database.MaxOpenConn {
		result.AddError("DB_MAX_IDLE_CONN cannot exceed DB_MAX_OPEN_CONN")
	}
}

// validateRedis validates Redis configuration
func (cv *ConfigValidator) validateRedis(cfg *Config, result *ValidationResult) {
	if cfg.Redis.URL == "" {
		result.AddError("REDIS_URL is required")
	}

	// Password validation (Redis can work without password but it's recommended)
	if cv.validationLevel >= StagingLevel && cfg.Redis.Password == "" {
		result.AddWarning("REDIS_PASSWORD is recommended for security")
	}
}

// validateAI validates AI service configuration
func (cv *ConfigValidator) validateAI(cfg *Config, result *ValidationResult) {
	// Both checks below describe OpenAI specifically — key prefixes and model
	// families. When the client is pointed at an API-compatible host they are
	// wrong by construction: another provider's key does not begin with "sk-"
	// and its models are not named "gpt-". Applying them anyway refuses to
	// start on a configuration that would have worked.
	compatibleHost := cfg.AI.OpenAIBaseURL != ""

	if cfg.AI.OpenAIAPIKey == "" {
		result.AddWarning("OPENAI_API_KEY is not set. AI features will be disabled.")
	} else if !compatibleHost && !cv.isValidOpenAIAPIKey(cfg.AI.OpenAIAPIKey) {
		result.AddError("OPENAI_API_KEY appears to be invalid. It should start with 'sk-'")
	}

	if cfg.AI.OpenAIModel == "" {
		result.AddWarning("OPENAI_MODEL is not set. Using default model.")
	} else if !compatibleHost && !cv.isValidOpenAIModel(cfg.AI.OpenAIModel) {
		result.AddError(fmt.Sprintf("OPENAI_MODEL '%s' is not a valid OpenAI model", cfg.AI.OpenAIModel))
	}

	if compatibleHost {
		result.AddWarning(fmt.Sprintf(
			"OPENAI_BASE_URL is set to %q — AI calls go to a stand-in provider, "+
				"not OpenAI. Output will differ from the evaluated system.",
			cfg.AI.OpenAIBaseURL))
	}
}

// validateS3 validates S3 configuration
func (cv *ConfigValidator) validateS3(cfg *Config, result *ValidationResult) {
	if cfg.S3.AccessKeyID == "" {
		result.AddError("AWS_ACCESS_KEY_ID is required")
	}

	if cfg.S3.SecretAccessKey == "" {
		result.AddError("AWS_SECRET_ACCESS_KEY is required")
	}

	if cfg.S3.BucketName == "" {
		result.AddError("COSMO_BUCKET is required")
	} else if cv.validationLevel >= StagingLevel && !cv.isValidS3BucketName(cfg.S3.BucketName) {
		result.AddError("COSMO_BUCKET contains invalid characters")
	}

	if cfg.S3.Region == "" {
		result.AddError("COSMO_BUCKET_REGION is required")
	}

	// Endpoint validation (optional, for S3-compatible services)
	if cfg.S3.EndpointURL != "" {
		if !cv.isValidURL(cfg.S3.EndpointURL) {
			result.AddError("AWS_S3_ENDPOINT_URL must be a valid URL")
		}
	}
}

// validateEmail validates email configuration
func (cv *ConfigValidator) validateEmail(cfg *Config, result *ValidationResult) {
	if cfg.Email.ResendAPIKey == "" {
		result.AddWarning("RESEND_API_KEY is not set. Email features will be disabled.")
	}
}

// validateOAuth validates OAuth configuration
func (cv *ConfigValidator) validateOAuth(cfg *Config, result *ValidationResult) {
	// Google OAuth
	if cfg.OAuth.GoogleClientID == "" || cfg.OAuth.GoogleClientSecret == "" {
		result.AddWarning("Google OAuth is not configured. Google integration will be disabled.")
	} else {
		if cv.validationLevel >= StagingLevel && strings.Contains(cfg.OAuth.GoogleClientID, "localhost") {
			result.AddWarning("Google OAuth client ID appears to be for development use")
		}
	}

	// Google redirect URI validation
	if cfg.OAuth.GoogleRedirectURI != "" && !cv.isValidURL(cfg.OAuth.GoogleRedirectURI) {
		result.AddError("GOOGLE_REDIRECT_URI must be a valid URL")
	}

	// Hubspot OAuth
	if cfg.Hubspot.ClientID == "" || cfg.Hubspot.ClientSecret == "" {
		result.AddWarning("Hubspot OAuth is not configured. Hubspot integration will be disabled.")
	}

	// Hubspot redirect URI validation
	if cfg.Hubspot.RedirectURI != "" && !cv.isValidURL(cfg.Hubspot.RedirectURI) {
		result.AddError("HUBSPOT_REDIRECT_URI must be a valid URL")
	}

	// Outlook OAuth
	if cfg.Outlook.ClientID == "" || cfg.Outlook.ClientSecret == "" {
		result.AddWarning("Outlook OAuth is not configured. Outlook integration will be disabled.")
	}

	// Outlook redirect URI validation
	if cfg.Outlook.RedirectURI != "" && !cv.isValidURL(cfg.Outlook.RedirectURI) {
		result.AddError("OUTLOOK_REDIRECT_URI must be a valid URL")
	}

	// Facebook OAuth
	if cfg.Facebook.ClientID == "" || cfg.Facebook.ClientSecret == "" {
		result.AddWarning("Facebook OAuth is not configured. Facebook integration will be disabled.")
	}
}

// Helper validation functions

func (cv *ConfigValidator) validateCredentialsEncKey(key string) error {
	// Try to decode as base64 first
	if kb, err := base64.StdEncoding.DecodeString(key); err == nil {
		if len(kb) != 32 {
			return fmt.Errorf("AUTH_CREDENTIALS_ENC_KEY (base64) must decode to 32 bytes (AES-256)")
		}
	} else {
		// If not base64, check as raw string
		if len([]byte(key)) != 32 {
			return fmt.Errorf("AUTH_CREDENTIALS_ENC_KEY must be 32 bytes if not base64-encoded")
		}
	}
	return nil
}

func (cv *ConfigValidator) isWeakSecret(secret string) bool {
	// Check for common weak patterns
	weakPatterns := []string{
		"password", "secret", "123456", "admin", "test", "demo",
		"change", "default", "example", "sample", "temp",
	}

	lowerSecret := strings.ToLower(secret)
	for _, pattern := range weakPatterns {
		if strings.Contains(lowerSecret, pattern) {
			return true
		}
	}

	// Check length
	if len(secret) < 16 {
		return true
	}

	return false
}

func (cv *ConfigValidator) isValidOpenAIAPIKey(key string) bool {
	// OpenAI API keys start with "sk-"
	return strings.HasPrefix(key, "sk-") && len(key) > 40
}

func (cv *ConfigValidator) isValidOpenAIOrg(org string) bool {
	// OpenAI organization IDs start with "org-"
	return strings.HasPrefix(org, "org-") && len(org) > 20
}

func (cv *ConfigValidator) isValidOpenAIModel(model string) bool {
	// A fixed list goes stale with every model release (it predated the
	// gpt-4.1/gpt-5 families and rejected them at boot). Accept the known
	// family prefixes instead; the API remains the real authority.
	prefixes := []string{"gpt-3.5", "gpt-4", "gpt-5", "o1", "o3", "o4", "chatgpt-"}
	for _, p := range prefixes {
		if strings.HasPrefix(model, p) {
			return true
		}
	}
	return false
}

func (cv *ConfigValidator) isValidS3BucketName(bucket string) bool {
	// S3 bucket naming rules
	if len(bucket) < 3 || len(bucket) > 63 {
		return false
	}

	// Cannot start or end with hyphen
	if strings.HasPrefix(bucket, "-") || strings.HasSuffix(bucket, "-") {
		return false
	}

	// Cannot contain consecutive periods
	if strings.Contains(bucket, "..") {
		return false
	}

	// Cannot look like IP address
	if cv.isIPAddress(bucket) {
		return false
	}

	// Only lowercase letters, numbers, hyphens, and dots
	matched, _ := regexp.MatchString(`^[a-z0-9.-]+$`, bucket)
	return matched
}

func (cv *ConfigValidator) isIPAddress(str string) bool {
	// Simple check for IP address pattern
	parts := strings.Split(str, ".")
	if len(parts) != 4 {
		return false
	}

	for _, part := range parts {
		if num, err := strconv.Atoi(part); err != nil || num < 0 || num > 255 {
			return false
		}
	}

	return true
}

func (cv *ConfigValidator) isValidURL(urlStr string) bool {
	if url, err := url.Parse(urlStr); err != nil {
		return false
	} else {
		return url.Scheme == "http" || url.Scheme == "https"
	}
}

// GenerateSecureSecret generates a cryptographically secure random secret
func GenerateSecureSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// ValidateAndFixJWTSecret validates JWT secret and generates a new one if needed
func ValidateAndFixJWTSecret(secret string, environment string) (string, error) {
	if secret == "" || strings.EqualFold(secret, "change-me-in-production") {
		newSecret, err := GenerateSecureSecret(64)
		if err != nil {
			return "", fmt.Errorf("failed to generate secure JWT secret: %w", err)
		}

		if environment == "production" {
			return "", fmt.Errorf("JWT_SECRET must be set manually in production. Generated secret: %s", newSecret)
		}

		return newSecret, nil
	}

	// Validate existing secret
	validator := NewConfigValidator(environment)
	result := validator.validateJWTSecret(secret)
	if !result.Valid {
		return "", fmt.Errorf("invalid JWT secret: %s", strings.Join(result.Errors, ", "))
	}

	return secret, nil
}

// validateJWTSecret validates only the JWT secret
func (cv *ConfigValidator) validateJWTSecret(secret string) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if secret == "" {
		result.AddError("JWT_SECRET is required")
		return result
	}

	if strings.EqualFold(secret, "change-me-in-production") {
		result.AddError("JWT_SECRET must be changed from default value")
	}

	if cv.validationLevel >= StagingLevel {
		if len(secret) < 32 {
			result.AddError("JWT_SECRET must be at least 32 characters for security")
		}
		if cv.isWeakSecret(secret) {
			result.AddError("JWT_SECRET is too weak. Use a cryptographically secure random string")
		}
	}

	return result
}
