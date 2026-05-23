package bookings

import "github.com/trishaneupnexx/inkspace-api/internal/events"

type Service interface {
	// TODO: List, Get, Create, Update, Confirm, Cancel, Reschedule
}

type service struct {
	repo   Repository
	events *events.Publisher
}

func NewService(repo Repository, pub *events.Publisher) Service {
	return &service{repo: repo, events: pub}
}
