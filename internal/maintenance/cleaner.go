package maintenance

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

const sweepTimeout = 30 * time.Second
type Cleaner struct {
	q        *sqlc.Queries
	log      *slog.Logger
	interval time.Duration
}

func NewCleaner(db *pgxpool.Pool, log *slog.Logger, interval time.Duration) *Cleaner {
	return &Cleaner{q: sqlc.New(db), log: log, interval: interval}
}

func (c *Cleaner) Run(ctx context.Context) {
	c.sweep(ctx)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.sweep(ctx)
		}
	}
}

func (c *Cleaner) sweep(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, sweepTimeout)
	defer cancel()

	if n, err := c.q.DeleteExpiredRefreshTokens(ctx); err != nil {
		c.log.Warn("cleanup_refresh_tokens_failed", "error", err)
	} else if n > 0 {
		c.log.Info("cleanup_refresh_tokens", "deleted", n)
	}

	if n, err := c.q.DeleteExpiredPhoneVerifications(ctx); err != nil {
		c.log.Warn("cleanup_phone_verifications_failed", "error", err)
	} else if n > 0 {
		c.log.Info("cleanup_phone_verifications", "deleted", n)
	}
}
