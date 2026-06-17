package settings

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

func (s *service) HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error {
	if s.cfg.StripeWebhookSecret == "" {
		return fmt.Errorf("%w: stripe webhook secret", ErrIntegrationConfig)
	}

	// IgnoreAPIVersionMismatch: the connected account / CLI may run a newer API
	// version than stripe-go is pinned to; the account fields we read are stable.
	event, err := webhook.ConstructEventWithOptions(
		payload, signature, s.cfg.StripeWebhookSecret,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true},
	)
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
