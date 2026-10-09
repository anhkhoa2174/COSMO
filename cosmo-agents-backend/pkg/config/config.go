package config

import (
	"fmt"
	"os"

	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Logger   LoggerConfig
	Auth     AuthConfig
	OAuth    OAuthConfig
	Email    EmailConfig
	AI       AIConfig
	Hubspot  HubspotConfig
	Outlook  OutlookConfig
	Facebook FacebookConfig
	Sentry   SentryConfig
	S3       S3Config
}

type AppConfig struct {
	Name        string `mapstructure:"APP_NAME"`
	Environment string `mapstructure:"ENV"`
	Host        string `mapstructure:"APP_HOST"`
	Port        int    `mapstructure:"APP_PORT"`
	Debug       bool   `mapstructure:"DEBUG"`
}

type DatabaseConfig struct {
	PostgresURI string `mapstructure:"sqlalchemy_postgres_uri"`
	MaxOpenConn int    `mapstructure:"db_max_open_conn"`
	MaxIdleConn int    `mapstructure:"db_max_idle_conn"`
}

type RedisConfig struct {
	URL      string `mapstructure:"redis_url"`
	Password string `mapstructure:"redis_password"`
}

type LoggerConfig struct {
	Level      string `mapstructure:"LOG_LEVEL"`
	ConfigPath string `mapstructure:"LOGGING_CONFIG_PATH"`
}

type AuthConfig struct {
	JWTSecret              string `mapstructure:"JWT_SECRET"`
	JWTExpiration          int    `mapstructure:"JWT_EXPIRATION_HOURS"`         // in hours
	JWTRefreshExpiration   int    `mapstructure:"JWT_REFRESH_EXPIRATION_HOURS"` // in hours
	AccessTokenSecret      string `mapstructure:"ACCESS_TOKEN_SECRET_KEY"`
	RefreshTokenSecret     string `mapstructure:"REFRESH_TOKEN_SECRET_KEY"`
	JWTAlgorithm           string `mapstructure:"JWT_ALGORITHM"`
	AccessTokenExpireDays  int    `mapstructure:"ACCESS_TOKEN_EXPIRE_DAYS"`
	RefreshTokenExpireDays int    `mapstructure:"REFRESH_TOKEN_EXPIRE_DAYS"`
	PersonalAPIKeySecret   string `mapstructure:"PERSONAL_API_KEY_SECRET"`
	JWTIssuer              string `mapstructure:"JWT_ISSUER"`
	JWTAudience            string `mapstructure:"JWT_AUDIENCE"`
	EnableAuth             bool   `mapstructure:"ENABLE_AUTH"`
	RateLimitEnabled       bool   `mapstructure:"RATE_LIMIT_ENABLED"`
	RateLimitMax           int    `mapstructure:"RATE_LIMIT_MAX"`
	RateLimitWindow        int    `mapstructure:"RATE_LIMIT_WINDOW_MINUTES"` // in minutes
	AllowedEmailDomains    string `mapstructure:"AUTH_ALLOWED_EMAIL_DOMAINS"`
	AllowedRedirectHosts   string `mapstructure:"AUTH_ALLOWED_REDIRECT_HOSTS"`
	CookieDomain           string `mapstructure:"AUTH_COOKIE_DOMAIN"`
	CredentialsEncKey      string `mapstructure:"AUTH_CREDENTIALS_ENC_KEY"`
}

type OAuthConfig struct {
	GoogleClientID         string `mapstructure:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret     string `mapstructure:"GOOGLE_CLIENT_SECRET"`
	GoogleRedirectURI      string `mapstructure:"GOOGLE_REDIRECT_URI"`
	GooglePubSubGmailTopic string `mapstructure:"GOOGLE_PUBSUB_GMAIL_TOPIC"`
}

type EmailConfig struct {
	ResendAPIKey string `mapstructure:"RESEND_API_KEY"`
}

type AIConfig struct {
	OpenAIAPIKey string `mapstructure:"OPENAI_API_KEY"`
	OpenAIModel  string `mapstructure:"OPENAI_MODEL"`

	// OpenAIBaseURL points the OpenAI clients at a different, API-compatible
	// host. Empty means the real OpenAI, which is the default and what
	// production runs.
	//
	// It exists so the system can be pointed at Google's OpenAI-compatible
	// endpoint when the OpenAI account has no credit, which otherwise takes
	// every AI feature down at once — classification, reply drafting, daily
	// actions, and embeddings all share the one key. That is a stop-gap for a
	// demonstration, not a supported configuration: the models differ, so the
	// output differs, and any measurement taken through it is a measurement of
	// a different system.
	OpenAIBaseURL string `mapstructure:"OPENAI_BASE_URL"`

	// EmbeddingModel and EmbeddingDimensions have to move together with the
	// base URL. The vector index is built at a fixed width, so an embedding
	// provider that returns a different number of dimensions cannot be dropped
	// in without rebuilding it.
	EmbeddingModel      string `mapstructure:"EMBEDDING_MODEL"`
	EmbeddingDimensions int    `mapstructure:"EMBEDDING_DIMENSIONS"`
}

type HubspotConfig struct {
	ClientID     string `mapstructure:"HUBSPOT_CLIENT_ID"`
	ClientSecret string `mapstructure:"HUBSPOT_CLIENT_SECRET"`
	RedirectURI  string `mapstructure:"HUBSPOT_REDIRECT_URI"`
	AuthScopes   string `mapstructure:"HUBSPOT_AUTH_SCOPES"`
}

type OutlookConfig struct {
	ClientID     string `mapstructure:"OUTLOOK_CLIENT_ID"`
	ClientSecret string `mapstructure:"OUTLOOK_CLIENT_SECRET"`
	RedirectURI  string `mapstructure:"OUTLOOK_REDIRECT_URI"`
	Authority    string `mapstructure:"OUTLOOK_AUTHORITY"`
	Scopes       string `mapstructure:"OUTLOOK_SCOPES"`
}

type FacebookConfig struct {
	ClientID     string `mapstructure:"FACEBOOK_CLIENT_ID"`
	ClientSecret string `mapstructure:"FACEBOOK_CLIENT_SECRET"`
	AppID        string `mapstructure:"FACEBOOK_APP_ID"`
	AppSecret    string `mapstructure:"FACEBOOK_APP_SECRET"`
	VerifyToken  string `mapstructure:"FACEBOOK_VERIFY_TOKEN"`
}

type SentryConfig struct {
	DSN string `mapstructure:"SENTRY_DSN"`
}

type S3Config struct {
	BucketName      string `mapstructure:"COSMO_BUCKET"`
	Region          string `mapstructure:"COSMO_BUCKET_REGION"`
	AccessKeyID     string `mapstructure:"AWS_ACCESS_KEY_ID"`
	SecretAccessKey string `mapstructure:"AWS_SECRET_ACCESS_KEY"`
	EndpointURL     string `mapstructure:"AWS_S3_ENDPOINT_URL"`
}

// Load reads configuration from environment variables and .env file
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults (matching Python settings)
	v.SetDefault("APP_NAME", "cosmo-agents")
	v.SetDefault("ENV", "development")
	v.SetDefault("APP_HOST", "0.0.0.0")
	v.SetDefault("APP_PORT", 8080)
	v.SetDefault("DEBUG", false)
	v.SetDefault("DB_MAX_OPEN_CONN", 25)
	v.SetDefault("DB_MAX_IDLE_CONN", 5)
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOGGING_CONFIG_PATH", "log.server.yaml")
	v.SetDefault("OPENAI_MODEL", "gpt-4")
	// Declared so viper's AutomaticEnv can see them: an environment variable
	// with no registered key is invisible to Unmarshal and to GetString, which
	// is why an unset default reads as empty however the process was started.
	v.SetDefault("OPENAI_BASE_URL", "")
	v.SetDefault("EMBEDDING_MODEL", "")
	v.SetDefault("EMBEDDING_DIMENSIONS", 0)

	// Auth defaults
	// Do not set JWT_SECRET default to avoid insecure deployments
	v.SetDefault("JWT_EXPIRATION_HOURS", 24)
	v.SetDefault("JWT_REFRESH_EXPIRATION_HOURS", 168) // 7 days
	v.SetDefault("AUTH_ALLOWED_EMAIL_DOMAINS", "")
	v.SetDefault("AUTH_ALLOWED_REDIRECT_HOSTS", "")
	v.SetDefault("AUTH_COOKIE_DOMAIN", "")
	v.SetDefault("AUTH_CREDENTIALS_ENC_KEY", "")
	v.SetDefault("JWT_ISSUER", "")
	v.SetDefault("JWT_AUDIENCE", "")
	v.SetDefault("ACCESS_TOKEN_EXPIRE_DAYS", 1)
	v.SetDefault("REFRESH_TOKEN_EXPIRE_DAYS", 30)
	v.SetDefault("JWT_ALGORITHM", "HS256")
	v.SetDefault("ENABLE_AUTH", false) // Disabled by default for development
	v.SetDefault("RATE_LIMIT_ENABLED", true)
	v.SetDefault("RATE_LIMIT_MAX", 100)
	v.SetDefault("RATE_LIMIT_WINDOW_MINUTES", 1)
	v.SetDefault("HUBSPOT_AUTH_SCOPES", "contacts")
	v.SetDefault("OUTLOOK_SCOPES", "Contacts.Read email Mail.ReadBasic Mail.Read Mail.ReadWrite Mail.Send User.Read")

	// Allow env vars to override defaults
	v.AutomaticEnv()

	// Build config from environment variables (loaded by godotenv in main)
	cfg := &Config{
		App: AppConfig{
			Name:        v.GetString("APP_NAME"),
			Environment: v.GetString("ENV"),
			Host:        v.GetString("APP_HOST"),
			Port:        v.GetInt("APP_PORT"),
			Debug:       v.GetBool("DEBUG"),
		},
		Database: DatabaseConfig{
			PostgresURI: v.GetString("SQLALCHEMY_POSTGRES_URI"),
			MaxOpenConn: v.GetInt("DB_MAX_OPEN_CONN"),
			MaxIdleConn: v.GetInt("DB_MAX_IDLE_CONN"),
		},
		Redis: RedisConfig{
			URL:      v.GetString("REDIS_URL"),
			Password: v.GetString("REDIS_PASSWORD"),
		},
		Logger: LoggerConfig{
			Level:      v.GetString("LOG_LEVEL"),
			ConfigPath: v.GetString("LOGGING_CONFIG_PATH"),
		},
		Auth: AuthConfig{
			JWTSecret:              v.GetString("JWT_SECRET"),
			JWTExpiration:          v.GetInt("JWT_EXPIRATION_HOURS"),
			JWTRefreshExpiration:   v.GetInt("JWT_REFRESH_EXPIRATION_HOURS"),
			AccessTokenSecret:      v.GetString("ACCESS_TOKEN_SECRET_KEY"),
			RefreshTokenSecret:     v.GetString("REFRESH_TOKEN_SECRET_KEY"),
			JWTAlgorithm:           v.GetString("JWT_ALGORITHM"),
			AccessTokenExpireDays:  v.GetInt("ACCESS_TOKEN_EXPIRE_DAYS"),
			RefreshTokenExpireDays: v.GetInt("REFRESH_TOKEN_EXPIRE_DAYS"),
			PersonalAPIKeySecret:   v.GetString("PERSONAL_API_KEY_SECRET"),
			JWTIssuer:              v.GetString("JWT_ISSUER"),
			JWTAudience:            v.GetString("JWT_AUDIENCE"),
			EnableAuth:             v.GetBool("ENABLE_AUTH"),
			RateLimitEnabled:       v.GetBool("RATE_LIMIT_ENABLED"),
			RateLimitMax:           v.GetInt("RATE_LIMIT_MAX"),
			RateLimitWindow:        v.GetInt("RATE_LIMIT_WINDOW_MINUTES"),
			AllowedEmailDomains:    v.GetString("AUTH_ALLOWED_EMAIL_DOMAINS"),
			AllowedRedirectHosts:   v.GetString("AUTH_ALLOWED_REDIRECT_HOSTS"),
			CookieDomain:           v.GetString("AUTH_COOKIE_DOMAIN"),
			CredentialsEncKey:      v.GetString("AUTH_CREDENTIALS_ENC_KEY"),
		},
		OAuth: OAuthConfig{
			GoogleClientID:         v.GetString("GOOGLE_CLIENT_ID"),
			GoogleClientSecret:     v.GetString("GOOGLE_CLIENT_SECRET"),
			GoogleRedirectURI:      v.GetString("GOOGLE_REDIRECT_URI"),
			GooglePubSubGmailTopic: v.GetString("GOOGLE_PUBSUB_GMAIL_TOPIC"),
		},
		Email: EmailConfig{
			ResendAPIKey: v.GetString("RESEND_API_KEY"),
		},
		AI: AIConfig{
			OpenAIAPIKey:        v.GetString("OPENAI_API_KEY"),
			OpenAIModel:         v.GetString("OPENAI_MODEL"),
			OpenAIBaseURL:       v.GetString("OPENAI_BASE_URL"),
			EmbeddingModel:      v.GetString("EMBEDDING_MODEL"),
			EmbeddingDimensions: v.GetInt("EMBEDDING_DIMENSIONS"),
		},
		Hubspot: HubspotConfig{
			ClientID:     v.GetString("HUBSPOT_CLIENT_ID"),
			ClientSecret: v.GetString("HUBSPOT_CLIENT_SECRET"),
			RedirectURI:  v.GetString("HUBSPOT_REDIRECT_URI"),
			AuthScopes:   v.GetString("HUBSPOT_AUTH_SCOPES"),
		},
		Outlook: OutlookConfig{
			ClientID:     v.GetString("OUTLOOK_CLIENT_ID"),
			ClientSecret: v.GetString("OUTLOOK_CLIENT_SECRET"),
			RedirectURI:  v.GetString("OUTLOOK_REDIRECT_URI"),
			Authority:    v.GetString("OUTLOOK_AUTHORITY"),
			Scopes:       v.GetString("OUTLOOK_SCOPES"),
		},
		Sentry: SentryConfig{
			DSN: v.GetString("SENTRY_DSN"),
		},
		S3: S3Config{
			BucketName:      v.GetString("COSMO_BUCKET"),
			Region:          v.GetString("COSMO_BUCKET_REGION"),
			AccessKeyID:     v.GetString("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: v.GetString("AWS_SECRET_ACCESS_KEY"),
			EndpointURL:     v.GetString("AWS_S3_ENDPOINT_URL"),
		},
	}

	return cfg, nil
}

// MustLoad loads config with proper error handling
func MustLoad() *Config {
	// The logger is not configured yet at this point, so logger.Fatal writes
	// nowhere: a rejected configuration used to exit 1 having printed nothing,
	// which is the least helpful way for a program to refuse to start. The
	// reason goes to stderr directly as well.
	cfg, err := Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to load application configuration: "+err.Error())
		logger.Fatal("Failed to load application configuration: " + err.Error())
		os.Exit(1)
	}

	if err := ValidateConfig(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "Configuration validation failed: "+err.Error())
		logger.Fatal("Configuration validation failed: " + err.Error())
		os.Exit(1)
	}

	return cfg
}

// ValidateConfig validates configuration settings using comprehensive validator
func ValidateConfig(cfg *Config) error {
	validator := NewConfigValidator(cfg.App.Environment)
	result := validator.Validate(cfg)

	if !result.Valid {
		// Return the first error for compatibility with existing error handling
		if len(result.Errors) > 0 {
			return fmt.Errorf("configuration validation failed: %s", result.Errors[0])
		}
		return fmt.Errorf("configuration validation failed")
	}

	// Log warnings if any
	if len(result.Warnings) > 0 {
		for _, warning := range result.Warnings {
			logger.Logger.Warn().Msg("Configuration warning: " + warning)
		}
	}

	return nil
}

// ValidateConfigWithDetails validates configuration and returns detailed results
func ValidateConfigWithDetails(cfg *Config) *ValidationResult {
	validator := NewConfigValidator(cfg.App.Environment)
	return validator.Validate(cfg)
}
