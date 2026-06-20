package dashboard

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trishaneupnexx/inkspace-api/internal/config"
)

type Module struct {
	cfg     *config.Config
	Handler *Handler
}

func New(cfg *config.Config, db *pgxpool.Pool) *Module {
	repo := NewRepository(db)
	svc := NewService(repo)
	return &Module{cfg: cfg, Handler: NewHandler(svc)}
}
