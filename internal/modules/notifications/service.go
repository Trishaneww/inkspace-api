package notifications

import "github.com/trishaneupnexx/inkspace-api/internal/events"

type Service interface {
	// TODO: List, MarkRead, MarkAllRead, GetPreferences, UpdatePreferences,
	// TODO: Dispatch (consume events from rabbit and fan out via email/push/in-app)
}

type service struct {
	repo   Repository
	events *events.Publisher
}

func NewService(repo Repository, pub *events.Publisher) Service {
	return &service{repo: repo, events: pub}
}
