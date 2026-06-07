package settings

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v82"

	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/crypto"
	"github.com/trishaneupnexx/inkspace-api/internal/s3client"
)

type Module struct {
	cfg     *config.Config
	Handler *Handler
	Service Service
	Repo    Repository
}

func New(cfg *config.Config, db *pgxpool.Pool, s3 *s3client.Client) *Module {
	repo := NewRepository(db)

	// Stripe uses a process-global API key. Inkspace is a single platform
	// account, so setting it once at startup is correct. Empty key => Stripe
	// integrations return ErrIntegrationConfig.
	if cfg.StripeSecretKey != "" {
		stripe.Key = cfg.StripeSecretKey
	}

	// Optional: only configured when OAUTH_TOKEN_ENCRYPTION_KEY is set. Without
	// it, OAuth integrations (e.g. Google Calendar) return a clear error.
	var cipher *crypto.Cipher
	if cfg.OAuthTokenEncryptionKey != "" {
		c, err := crypto.NewCipher(cfg.OAuthTokenEncryptionKey)
		if err != nil {
			slog.Default().Error("oauth_token_cipher_init_failed", "error", err)
		} else {
			cipher = c
		}
	}

	svc := NewService(cfg, repo, s3, cipher)
	h := NewHandler(svc)
	return &Module{cfg: cfg, Handler: h, Service: svc, Repo: repo}
}
