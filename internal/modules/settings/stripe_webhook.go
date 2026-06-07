package settings

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

// HandleStripeWebhook verifies a Stripe webhook's signature and dispatches the
// event. Verification uses STRIPE_WEBHOOK_SECRET so we trust only events Stripe
// actually sent. Unhandled event types are acknowledged (return nil) so Stripe
// stops retrying them.
//
// For now only account.updated is handled (Connect onboarding status). Payment
// events (payment_intent.succeeded, charge.refunded, charge.dispute.created)
// land here once the payment-request feature is built, alongside a stripe_events
// table for idempotency + audit.
func (s *service) HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error {
	if s.cfg.StripeWebhookSecret == "" {
		return fmt.Errorf("%w: stripe webhook secret", ErrIntegrationConfig)
	}

	event, err := webhook.ConstructEvent(payload, signature, s.cfg.StripeWebhookSecret)
	if err != nil {
		s.log.Warn("stripe_webhook_signature_invalid", "error", err)
		return ErrStripeAPI
	}

	switch event.Type {
	case "account.updated":
		var acct stripe.Account
		if err := json.Unmarshal(event.Data.Raw, &acct); err != nil {
			return fmt.Errorf("decode account.updated: %w", err)
		}
		return s.handleStripeAccountUpdated(ctx, acct.ID, acct.ChargesEnabled, acct.PayoutsEnabled, acct.DetailsSubmitted)
	default:
		s.log.Debug("stripe_webhook_unhandled", "type", event.Type)
		return nil
	}
}
