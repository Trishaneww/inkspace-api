package subscriptions

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/subscription"
	"github.com/stripe/stripe-go/v82/webhook"

	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
	subentitlement "github.com/trishaneupnexx/inkspace-api/internal/subscription"
)

var (
	ErrNotConfigured     = errors.New("subscriptions are not configured")
	ErrAlreadySubscribed = errors.New("artist already has a live subscription")
	ErrNoSubscription    = errors.New("artist has no subscription to manage")
	ErrStripeAPI         = errors.New("stripe api error")
)

type Service interface {
	GetSubscription(ctx context.Context, userID uuid.UUID) (SubscriptionStatus, error)
	CreateCheckout(ctx context.Context, userID uuid.UUID) (CheckoutResponse, error)
	CreatePortalSession(ctx context.Context, userID uuid.UUID) (PortalResponse, error)
	ProcessWebhook(ctx context.Context, payload []byte, signature string) error
}

type service struct {
	cfg  *config.Config
	repo Repository
	log  *slog.Logger
}

func NewService(cfg *config.Config, repo Repository) Service {
	return &service{cfg: cfg, repo: repo, log: slog.Default()}
}

func (s *service) CreateCheckout(ctx context.Context, userID uuid.UUID) (CheckoutResponse, error) {
	if s.cfg.StripeSecretKey == "" || s.cfg.StripePremiumPriceID == "" {
		return CheckoutResponse{}, ErrNotConfigured
	}

	artist, err := s.repo.GetArtistByUserID(ctx, userID)
	if err != nil {
		return CheckoutResponse{}, err
	}
	if isLiveSubscription(artist.SubscriptionStatus) {
		return CheckoutResponse{}, ErrAlreadySubscribed
	}

	customerID, err := s.ensureCustomer(ctx, artist)
	if err != nil {
		return CheckoutResponse{}, err
	}

	sess, err := s.createCheckoutSession(ctx, artist.ID, customerID)
	if err != nil {
		return CheckoutResponse{}, err
	}
	return CheckoutResponse{URL: sess.URL}, nil
}

func (s *service) GetSubscription(ctx context.Context, userID uuid.UUID) (SubscriptionStatus, error) {
	artist, err := s.repo.GetArtistByUserID(ctx, userID)
	if err != nil {
		return SubscriptionStatus{}, err
	}

	var periodEnd *time.Time
	if artist.SubscriptionCurrentPeriodEnd.Valid {
		t := artist.SubscriptionCurrentPeriodEnd.Time
		periodEnd = &t
	}

	return SubscriptionStatus{
		Status:            artist.SubscriptionStatus,
		IsPremium:         subentitlement.IsPremium(artist.SubscriptionStatus),
		CancelAtPeriodEnd: artist.SubscriptionCancelAtPeriodEnd,
		CurrentPeriodEnd:  periodEnd,
	}, nil
}

func (s *service) CreatePortalSession(ctx context.Context, userID uuid.UUID) (PortalResponse, error) {
	if s.cfg.StripeSecretKey == "" {
		return PortalResponse{}, ErrNotConfigured
	}

	artist, err := s.repo.GetArtistByUserID(ctx, userID)
	if err != nil {
		return PortalResponse{}, err
	}
	if artist.StripeCustomerID == nil || *artist.StripeCustomerID == "" {
		return PortalResponse{}, ErrNoSubscription
	}

	sess, err := s.createPortalSession(ctx, *artist.StripeCustomerID)
	if err != nil {
		return PortalResponse{}, err
	}
	return PortalResponse{URL: sess.URL}, nil
}

func (s *service) ProcessWebhook(ctx context.Context, payload []byte, signature string) error {
	if s.cfg.StripeSubscriptionWebhookSecret == "" {
		s.log.Warn("subscription_webhook_secret_not_configured")
		return ErrStripeAPI
	}

	event, err := webhook.ConstructEventWithOptions(
		payload, signature, s.cfg.StripeSubscriptionWebhookSecret,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true},
	)
	if err != nil {
		s.log.Warn("subscription_webhook_signature_invalid", "error", err)
		return ErrStripeAPI
	}

	inserted, err := s.repo.InsertStripeEvent(ctx, sqlc.InsertStripeEventParams{ID: event.ID, Type: string(event.Type)})
	if err != nil {
		return err
	}
	if inserted == 0 {
		return nil // already processed
	}

	switch event.Type {
	case "checkout.session.completed":
		return s.handleCheckoutCompleted(ctx, event)
	case "customer.subscription.created",
		"customer.subscription.updated",
		"customer.subscription.deleted":
		return s.handleSubscriptionEvent(ctx, event)
	case "invoice.payment_failed":
		s.log.Info("subscription_invoice_payment_failed", "event_id", event.ID)
		return nil
	case "invoice.payment_succeeded":
		return nil
	default:
		s.log.Debug("subscription_webhook_unhandled", "type", event.Type)
		return nil
	}
}

func (s *service) handleCheckoutCompleted(ctx context.Context, event stripe.Event) error {
	var sess stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
		return err
	}
	if sess.Mode != stripe.CheckoutSessionModeSubscription || sess.Subscription == nil {
		return nil
	}
	sub, err := subscription.Get(sess.Subscription.ID, nil)
	if err != nil {
		s.log.Warn("subscription_fetch_failed", "subscription_id", sess.Subscription.ID, "error", err)
		return ErrStripeAPI
	}
	return s.applySubscription(ctx, sub)
}

func (s *service) handleSubscriptionEvent(ctx context.Context, event stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return err
	}
	return s.applySubscription(ctx, &sub)
}

func (s *service) applySubscription(ctx context.Context, sub *stripe.Subscription) error {
	if sub.Customer == nil || sub.Customer.ID == "" {
		return nil
	}
	customerID := sub.Customer.ID

	artist, err := s.repo.GetArtistByStripeCustomerID(ctx, &customerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.log.Warn("subscription_webhook_artist_not_found", "stripe_customer_id", customerID)
			return nil
		}
		return err
	}

	subID := sub.ID
	return s.repo.UpdateArtistSubscription(ctx, sqlc.UpdateArtistSubscriptionParams{
		ID:                            artist.ID,
		StripeSubscriptionID:          &subID,
		SubscriptionStatus:            string(sub.Status),
		SubscriptionCurrentPeriodEnd:  periodEnd(sub),
		SubscriptionCancelAtPeriodEnd: sub.CancelAtPeriodEnd,
	})
}

func isLiveSubscription(status string) bool {
	switch status {
	case subentitlement.StatusTrialing,
		subentitlement.StatusActive,
		subentitlement.StatusPastDue,
		subentitlement.StatusUnpaid:
		return true
	default:
		return false
	}
}
