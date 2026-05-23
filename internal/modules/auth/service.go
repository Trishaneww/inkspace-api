package auth

import (
	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/events"
)

type Service interface {
	// TODO: Register(ctx, input) (*User, *TokenPair, error)
	// TODO: Login(ctx, email, password) (*TokenPair, error)
	// TODO: Refresh(ctx, refreshToken) (*TokenPair, error)
	// TODO: Logout(ctx, userID) error
	// TODO: Me(ctx, userID) (*User, error)
}

type service struct {
	cfg    *config.Config
	repo   Repository
	events *events.Publisher
}

func NewService(cfg *config.Config, repo Repository, pub *events.Publisher) Service {
	return &service{cfg: cfg, repo: repo, events: pub}
}
