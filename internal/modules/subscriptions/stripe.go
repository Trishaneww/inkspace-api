package subscriptions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stripe/stripe-go/v82"
	billingportalsession "github.com/stripe/stripe-go/v82/billingportal/session"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/customer"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

func (s *service) ensureCustomer(ctx context.Context, artist sqlc.Artist) (string, error) {
	if artist.StripeCustomerID != nil && *artist.StripeCustomerID != "" {
		return *artist.StripeCustomerID, nil
	}

	user, err := s.repo.GetUserByID(ctx, artist.UserID)
	if err != nil {
		return "", err
	}

	params := &stripe.CustomerParams{
		Email:    stripe.String(user.Email),
		Metadata: map[string]string{"artist_id": artist.ID.String()},
	}
	if name := fullName(user); name != "" {
		params.Name = stripe.String(name)
	}
	params.Context = ctx

	cust, err := customer.New(params)
	if err != nil {
		s.log.Warn("stripe_customer_create_failed", "artist_id", artist.ID, "error", err)
		return "", ErrStripeAPI
	}

	if err := s.repo.SetArtistStripeCustomerID(ctx, sqlc.SetArtistStripeCustomerIDParams{
		ID:               artist.ID,
		StripeCustomerID: &cust.ID,
	}); err != nil {
		return "", err
	}
	return cust.ID, nil
}

func (s *service) createCheckoutSession(ctx context.Context, artistID uuid.UUID, customerID string) (*stripe.CheckoutSession, error) {
	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		Customer:          stripe.String(customerID),
		SuccessURL:        stripe.String(s.returnURL("success")),
		CancelURL:         stripe.String(s.returnURL("cancel")),
		ClientReferenceID: stripe.String(artistID.String()),
		LineItems: []*stripe.CheckoutSessionLineItemParams{{
			Price:    stripe.String(s.cfg.StripePremiumPriceID),
			Quantity: stripe.Int64(1),
		}},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{"artist_id": artistID.String()},
		},
	}
	params.Context = ctx

	sess, err := session.New(params)
	if err != nil {
		s.log.Warn("subscription_checkout_create_failed", "artist_id", artistID, "error", err)
		return nil, ErrStripeAPI
	}
	return sess, nil
}

func (s *service) createPortalSession(ctx context.Context, customerID string) (*stripe.BillingPortalSession, error) {
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(s.cfg.FrontendURL + "/dashboard/artist/settings?tab=billing"),
	}
	params.Context = ctx

	sess, err := billingportalsession.New(params)
	if err != nil {
		s.log.Warn("subscription_portal_create_failed", "stripe_customer_id", customerID, "error", err)
		return nil, ErrStripeAPI
	}
	return sess, nil
}

func (s *service) returnURL(status string) string {
	return fmt.Sprintf("%s/dashboard/artist/settings?tab=billing&subscription=%s", s.cfg.FrontendURL, status)
}

func periodEnd(sub *stripe.Subscription) pgtype.Timestamptz {
	if sub.Items == nil || len(sub.Items.Data) == 0 || sub.Items.Data[0].CurrentPeriodEnd == 0 {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{
		Time:  time.Unix(sub.Items.Data[0].CurrentPeriodEnd, 0).UTC(),
		Valid: true,
	}
}

func fullName(u sqlc.User) string {
	return strings.TrimSpace(strings.Join([]string{deref(u.FirstName), deref(u.LastName)}, " "))
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
