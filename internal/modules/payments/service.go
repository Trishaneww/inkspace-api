package payments

import "github.com/trishaneupnexx/inkspace-api/internal/events"

type Service interface {
	// TODO: List, Get, CreateDepositIntent, Refund, HandleStripeWebhook
}

type service struct {
	repo   Repository
	events *events.Publisher
}

func NewService(repo Repository, pub *events.Publisher) Service {
	return &service{repo: repo, events: pub}
}
