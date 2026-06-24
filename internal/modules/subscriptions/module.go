package subscriptions

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v82"

	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/auth"
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
	slog.Default().Info("subscriptions_module_ready",
		"stripe_key_configured", cfg.StripeSecretKey != "",
		"premium_price_configured", cfg.StripePremiumPriceID != "",
		"subscription_webhook_secret_configured", cfg.StripeSubscriptionWebhookSecret != "",
	)
	repo := NewRepository(db)
	svc := NewService(cfg, repo)
	h := NewHandler(svc)
	return &Module{cfg: cfg, Handler: h, Service: svc, Repo: repo}
}

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/webhooks/stripe/subscription", m.Handler.Webhook)

	artist := rg.Group("/current-user/subscription")
	artist.Use(middleware.RequireAuth(m.cfg.JWTSecret))
	artist.Use(middleware.RequireRole(string(auth.RoleArtist)))
	artist.GET("", m.Handler.GetSubscription)
	artist.POST("/checkout", m.Handler.CreateCheckout)
	artist.POST("/portal", m.Handler.CreatePortalSession)
}
