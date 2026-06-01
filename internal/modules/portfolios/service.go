package portfolios

import "github.com/trishaneupnexx/inkspace-api/internal/events"

type Service interface {
	// TODO: GetArtistPortfolio, CreatePortfolioItem, UpdatePortfolioItem, DeletePortfolioItem
}

type service struct {
	repo   Repository
	events *events.Publisher
}

func NewService(repo Repository, pub *events.Publisher) Service {
	return &service{repo: repo, events: pub}
}
