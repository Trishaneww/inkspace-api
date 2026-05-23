package artists

import "github.com/trishaneupnexx/inkspace-api/internal/events"

type Service interface {
	// TODO: List(ctx, filters) ([]Artist, error)
	// TODO: Get(ctx, id) (*Artist, error)
	// TODO: GetByHandle(ctx, handle) (*Artist, error)
	// TODO: Create(ctx, input) (*Artist, error)
	// TODO: Update(ctx, id, input) (*Artist, error)
	// TODO: GetAvailability(ctx, artistID) (*Availability, error)
	// TODO: UpdateAvailability(ctx, artistID, input) error
}

type service struct {
	repo   Repository
	events *events.Publisher
}

func NewService(repo Repository, pub *events.Publisher) Service {
	return &service{repo: repo, events: pub}
}
