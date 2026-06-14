package bookings

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

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

	var cipher *crypto.Cipher
	if cfg.OAuthTokenEncryptionKey != "" {
		newCipher, err := crypto.NewCipher(cfg.OAuthTokenEncryptionKey)
		if err != nil {
			slog.Default().Error("oauth_token_cipher_init_failed", "error", err)
		} else {
			cipher = newCipher
		}
	}

	svc := NewService(repo, s3, cfg, cipher)
	h := NewHandler(cfg, svc)
	return &Module{cfg: cfg, Handler: h, Service: svc, Repo: repo}
}
