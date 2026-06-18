package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv   string
	AppPort  string
	LogLevel string

	DatabaseURL string
	RabbitMQURL string
	RedisURL    string

	CORSAllowedOrigins string
	FrontendURL        string

	TrustedProxies string

	RateLimitGlobalRPS   float64
	RateLimitGlobalBurst int
	RateLimitAuthRPS     float64
	RateLimitAuthBurst   int

	MaxRequestBodyBytes int64

	LoginMaxFailures   int
	LoginLockoutWindow time.Duration
	RateLimitOTPRPS    float64
	RateLimitOTPBurst  int

	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	PhoneCodeTTL     time.Duration
	PhoneCodeLength  int
	PhoneMaxAttempts int

	VonageAPIKey    string
	VonageAPISecret string
	VonageFromName  string

	GoogleClientID          string
	GoogleClientSecret      string
	MicrosoftClientID       string
	MicrosoftClientSecret   string
	MicrosoftTenantID       string
	OAuthTokenEncryptionKey string

	AWSRegion          string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	AWSS3Bucket        string

	StripeSecretKey             string
	StripeWebhookSecret         string
	StripePaymentsWebhookSecret string

	InternalEmailSecret string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		AppPort:     getEnv("APP_PORT", "8080"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RabbitMQURL: os.Getenv("RABBITMQ_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),

		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:3000"),

		TrustedProxies: os.Getenv("TRUSTED_PROXIES"),

		RateLimitGlobalRPS:   getEnvFloat("RATE_LIMIT_GLOBAL_RPS", 40),
		RateLimitGlobalBurst: getEnvInt("RATE_LIMIT_GLOBAL_BURST", 80),
		RateLimitAuthRPS:     getEnvFloat("RATE_LIMIT_AUTH_RPS", 0.5),
		RateLimitAuthBurst:   getEnvInt("RATE_LIMIT_AUTH_BURST", 10),
		MaxRequestBodyBytes:  getEnvInt64("MAX_REQUEST_BODY_BYTES", 1<<20),

		LoginMaxFailures:  getEnvInt("LOGIN_MAX_FAILURES", 5),
		RateLimitOTPRPS:   getEnvFloat("RATE_LIMIT_OTP_RPS", 0.0083),
		RateLimitOTPBurst: getEnvInt("RATE_LIMIT_OTP_BURST", 4),

		VonageAPIKey:    os.Getenv("VONAGE_API_KEY"),
		VonageAPISecret: os.Getenv("VONAGE_API_SECRET"),
		VonageFromName:  getEnv("VONAGE_FROM_NAME", "Inkspace"),

		GoogleClientID:        os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:    os.Getenv("GOOGLE_CLIENT_SECRET"),
		MicrosoftClientID:     os.Getenv("MICROSOFT_CLIENT_ID"),
		MicrosoftClientSecret: os.Getenv("MICROSOFT_CLIENT_SECRET"),
		MicrosoftTenantID:     getEnv("MICROSOFT_TENANT_ID", "common"),

		OAuthTokenEncryptionKey: os.Getenv("OAUTH_TOKEN_ENCRYPTION_KEY"),

		AWSRegion:          os.Getenv("AWS_REGION"),
		AWSAccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		AWSS3Bucket:        os.Getenv("AWS_S3_BUCKET"),

		StripeSecretKey:             os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret:         os.Getenv("STRIPE_WEBHOOK_SECRET"),
		StripePaymentsWebhookSecret: os.Getenv("STRIPE_PAYMENTS_WEBHOOK_SECRET"),

		InternalEmailSecret: os.Getenv("INTERNAL_EMAIL_SECRET"),
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
	if cfg.AppEnv == "production" && cfg.RedisURL == "" {
		return nil, fmt.Errorf("REDIS_URL is required in production (rate limiting)")
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

	lockoutWindow, err := time.ParseDuration(getEnv("LOGIN_LOCKOUT_WINDOW", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid LOGIN_LOCKOUT_WINDOW: %w", err)
	}
	cfg.LoginLockoutWindow = lockoutWindow

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return fallback
}
