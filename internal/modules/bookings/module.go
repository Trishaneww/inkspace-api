package bookings

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trishaneupnexx/inkspace-api/internal/config"
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
	svc := NewService(repo, s3)
	h := NewHandler(cfg, svc)
	return &Module{cfg: cfg, Handler: h, Service: svc, Repo: repo}
}
