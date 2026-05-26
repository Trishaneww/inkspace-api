package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv   string
	AppPort  string
	LogLevel string

	DatabaseURL string
	RabbitMQURL string

	// Comma-separated list of origins allowed by the CORS middleware.
	CORSAllowedOrigins string

	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	// Phone verification (OTP) tunables.
	PhoneCodeTTL     time.Duration
	PhoneCodeLength  int
	PhoneMaxAttempts int

	// Vonage SMS — unset means "log to stdout instead of sending".
	VonageAPIKey    string
	VonageAPISecret string
	VonageFromName  string

	// OAuth — public client IDs are also exposed in the frontend; the
	// secrets stay here. When unset, OAuth endpoints return 501.
	GoogleClientID        string
	GoogleClientSecret    string
	MicrosoftClientID     string
	MicrosoftClientSecret string
	// "common" (default) supports both personal and work/school accounts.
	// Set to a specific tenant ID for single-tenant apps.
	MicrosoftTenantID string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		AppPort:     getEnv("APP_PORT", "8080"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RabbitMQURL: os.Getenv("RABBITMQ_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),

		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),

		VonageAPIKey:    os.Getenv("VONAGE_API_KEY"),
		VonageAPISecret: os.Getenv("VONAGE_API_SECRET"),
		VonageFromName:  getEnv("VONAGE_FROM_NAME", "Inkspace"),

		GoogleClientID:        os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:    os.Getenv("GOOGLE_CLIENT_SECRET"),
		MicrosoftClientID:     os.Getenv("MICROSOFT_CLIENT_ID"),
		MicrosoftClientSecret: os.Getenv("MICROSOFT_CLIENT_SECRET"),
		MicrosoftTenantID:     getEnv("MICROSOFT_TENANT_ID", "common"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.RabbitMQURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	accessTTL, err := time.ParseDuration(getEnv("JWT_ACCESS_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TTL: %w", err)
	}
	refreshTTL, err := time.ParseDuration(getEnv("JWT_REFRESH_TTL", "720h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TTL: %w", err)
	}
	cfg.JWTAccessTTL = accessTTL
	cfg.JWTRefreshTTL = refreshTTL

	phoneTTL, err := time.ParseDuration(getEnv("PHONE_CODE_TTL", "10m"))
	if err != nil {
		return nil, fmt.Errorf("invalid PHONE_CODE_TTL: %w", err)
	}
	cfg.PhoneCodeTTL = phoneTTL
	cfg.PhoneCodeLength = 6
	cfg.PhoneMaxAttempts = 5

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
