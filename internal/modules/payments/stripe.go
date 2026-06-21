package payments

import (
	"context"
	"fmt"
	"strings"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/payout"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

func (s *service) listStripePayouts(ctx context.Context, artistAccountID string) ([]*stripe.Payout, error) {
	params := &stripe.PayoutListParams{}
	params.Context = ctx
	params.Limit = stripe.Int64(50)
	params.SetStripeAccount(artistAccountID)
	params.AddExpand("data.destination")

	iter := payout.List(params)
	payouts := make([]*stripe.Payout, 0)
	for iter.Next() {
		payouts = append(payouts, iter.Payout())
	}
	if err := iter.Err(); err != nil {
		s.log.Warn("stripe_payouts_list_failed", "account_id", artistAccountID, "error", err)
		return nil, ErrStripeAPI
	}
	return payouts, nil
}

func (s *service) createCheckoutSession(ctx context.Context, pr sqlc.PaymentRequest, artistAccountID string) (*stripe.CheckoutSession, error) {
	productName := "Tattoo deposit"
	if pr.Type == typeFinal {
		productName = "Tattoo session payment"
	}
	description := strings.TrimSpace(pr.Description)
	if description == "" {
		description = productName
	}

	params := &stripe.CheckoutSessionParams{
		Mode:          stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:    stripe.String(s.payPageURL(pr.PublicToken, "success")),
		CancelURL:     stripe.String(s.payPageURL(pr.PublicToken, "cancel")),
		CustomerEmail: stripe.String(pr.ClientEmail),
		LineItems: []*stripe.CheckoutSessionLineItemParams{{
			Quantity: stripe.Int64(1),
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency:   stripe.String(strings.ToLower(pr.Currency)),
				UnitAmount: stripe.Int64(pr.ClientChargeCents),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name:        stripe.String(productName),
					Description: stripe.String(description),
				},
			},
		}},
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			ApplicationFeeAmount: stripe.Int64(pr.PlatformFeeCents),
			TransferData: &stripe.CheckoutSessionPaymentIntentDataTransferDataParams{
				Destination: stripe.String(artistAccountID),
			},
		},
		Metadata: map[string]string{
			"payment_request_id": pr.ID.String(),
			"booking_request_id": pr.BookingRequestID.String(),
			"type":               pr.Type,
		},
	}
	params.Context = ctx

	sess, err := session.New(params)
	if err != nil {
		s.log.Warn("stripe_checkout_create_failed", "payment_request_id", pr.ID, "error", err)
		return nil, ErrStripeAPI
	}
	return sess, nil
}

func (s *service) payPageURL(token, status string) string {
	return fmt.Sprintf("%s/pay/%s?status=%s", s.cfg.FrontendURL, token, status)
}

func paymentIntentID(sess *stripe.CheckoutSession) *string {
	if sess == nil || sess.PaymentIntent == nil {
		return nil
	}
	id := sess.PaymentIntent.ID
	if id == "" {
		return nil
	}
	return &id
}
