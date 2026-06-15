package auth

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/events"
	"github.com/trishaneupnexx/inkspace-api/internal/ratelimit"
)

type Module struct {
	cfg     *config.Config
	Handler *Handler
	Service Service
	Repo    Repository
}

func New(cfg *config.Config, db *pgxpool.Pool, pub *events.Publisher, rdb *redis.Client) *Module {
	repo := NewRepository(db)

	var loginLockout *ratelimit.Lockout
	var otpLimiter ratelimit.Limiter
	if rdb != nil {
		loginLockout = ratelimit.NewLockout(rdb, cfg.LoginMaxFailures, cfg.LoginLockoutWindow)
		otpLimiter = ratelimit.NewRedisLimiter(rdb, ratelimit.Rule{
			RPS:   cfg.RateLimitOTPRPS,
			Burst: cfg.RateLimitOTPBurst,
		})
	}

	svc := NewService(cfg, repo, pub, loginLockout, otpLimiter)
	h := NewHandler(svc)
	return &Module{cfg: cfg, Handler: h, Service: svc, Repo: repo}
}
