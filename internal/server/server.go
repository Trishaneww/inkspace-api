package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/events"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
	"github.com/trishaneupnexx/inkspace-api/internal/ratelimit"
	"github.com/trishaneupnexx/inkspace-api/internal/s3client"
)

type Server struct {
	cfg    *config.Config
	log    *slog.Logger
	http   *http.Server
	engine *gin.Engine
}

func New(cfg *config.Config, log *slog.Logger, db *pgxpool.Pool, pub *events.Publisher, s3 *s3client.Client, rdb *redis.Client) *Server {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	// Only trust configured proxies for the client IP, so X-Forwarded-For can't
	// be forged to evade per-IP rate limiting. Empty list = trust none.
	if err := engine.SetTrustedProxies(parseTrustedProxies(cfg.TrustedProxies)); err != nil {
		log.Error("set_trusted_proxies_failed", "error", err)
	}

	engine.Use(
		middleware.Recover(log),
		middleware.Logger(log),
		middleware.CORS(cfg.CORSAllowedOrigins),
		middleware.BodyLimit(cfg.MaxRequestBodyBytes),
	)

	// Distributed rate limiting (shared across instances via Redis): a loose
	// global per-IP bucket on everything, plus a strict bucket on /auth/* to
	// blunt credential stuffing, signup spam, and OTP/SMS abuse.
	if rdb != nil {
		global := ratelimit.NewRedisLimiter(rdb, ratelimit.Rule{
			RPS:   cfg.RateLimitGlobalRPS,
			Burst: cfg.RateLimitGlobalBurst,
		})
		auth := ratelimit.NewRedisLimiter(rdb, ratelimit.Rule{
			RPS:   cfg.RateLimitAuthRPS,
			Burst: cfg.RateLimitAuthBurst,
		})
		engine.Use(middleware.RateLimit(global, "rl:global", log))
		engine.Use(middleware.FilterByPath("/v1/auth/", middleware.RateLimit(auth, "rl:auth", log)))
	} else {
		log.Warn("rate_limiting_disabled", "reason", "REDIS_URL not set")
	}

	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	registerRoutes(engine, cfg, db, pub, s3, rdb)

	return &Server{
		cfg:    cfg,
		log:    log,
		engine: engine,
		http: &http.Server{
			Addr:              fmt.Sprintf(":%s", cfg.AppPort),
			Handler:           engine,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	s.log.Info("server_starting", "port", s.cfg.AppPort)
	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

// parseTrustedProxies splits the comma-separated config value. Returns nil when
// empty, which tells gin to trust no proxies (use the socket peer as ClientIP).
func parseTrustedProxies(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
