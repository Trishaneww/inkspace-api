package bookings

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/events"
)

type Module struct {
	cfg     *config.Config
	Handler *Handler
	Service Service
	Repo    Repository
}

func New(cfg *config.Config, db *pgxpool.Pool, pub *events.Publisher) *Module {
	repo := NewRepository(db)
	svc := NewService(repo, pub)
	h := NewHandler(svc)
	return &Module{cfg: cfg, Handler: h, Service: svc, Repo: repo}
}
