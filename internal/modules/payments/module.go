package payments

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v82"

	"github.com/trishaneupnexx/inkspace-api/internal/config"
)

type Module struct {
	cfg     *config.Config
	Handler *Handler
	Service Service
	Repo    Repository
}

func New(cfg *config.Config, db *pgxpool.Pool) *Module {
	if cfg.StripeSecretKey != "" {
		stripe.Key = cfg.StripeSecretKey
	}
	slog.Default().Info("payments_module_ready",
		"stripe_key_configured", cfg.StripeSecretKey != "",
		"payments_webhook_secret_configured", cfg.StripePaymentsWebhookSecret != "",
	)
	repo := NewRepository(db)
	svc := NewService(cfg, repo)
	h := NewHandler(svc)
	return &Module{cfg: cfg, Handler: h, Service: svc, Repo: repo}
}
