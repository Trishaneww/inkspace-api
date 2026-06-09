package settings

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/account"
	"github.com/stripe/stripe-go/v82/accountlink"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

// ConnectStripe returns a Stripe-hosted onboarding URL for the artist. On the
// first call it creates an Express connected account and stores its id; on
// subsequent calls it reuses the existing account so the artist can resume or
// re-open onboarding. Inkspace stays merchant-of-record (destination charges),
// so the only thing we persist is the acct_… id — all KYC/bank data lives in
// Stripe.
func (s *service) ConnectStripe(ctx context.Context, userID uuid.UUID) (StripeConnectResponse, error) {
	if s.cfg.StripeSecretKey == "" {
		return StripeConnectResponse{}, fmt.Errorf("%w: stripe", ErrIntegrationConfig)
	}

	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return StripeConnectResponse{}, err
	}
	if err := s.repo.EnsureArtistSettings(ctx, artist.ID); err != nil {
		return StripeConnectResponse{}, err
	}
	current, err := s.repo.GetArtistSettings(ctx, artist.ID)
	if err != nil {
		return StripeConnectResponse{}, err
	}

	accountID := ""
	if current.StripeAccountID != nil {
		accountID = *current.StripeAccountID
	}

	if accountID == "" {
		acct, err := account.New(&stripe.AccountParams{
			Type: stripe.String(string(stripe.AccountTypeExpress)),
			Capabilities: &stripe.AccountCapabilitiesParams{
				CardPayments: &stripe.AccountCapabilitiesCardPaymentsParams{Requested: stripe.Bool(true)},
				Transfers:    &stripe.AccountCapabilitiesTransfersParams{Requested: stripe.Bool(true)},
			},
			Settings: &stripe.AccountSettingsParams{
				Payouts: &stripe.AccountSettingsPayoutsParams{
					Schedule: payoutScheduleParams(current.PayoutFrequency),
				},
			},
		})
		if err != nil {
			s.log.Warn("stripe_account_create_failed", "artist_id", artist.ID, "error", err)
			return StripeConnectResponse{}, ErrStripeAPI
		}
		accountID = acct.ID
		if _, err := s.repo.SetStripeAccount(ctx, sqlc.SetStripeAccountParams{
			ArtistID:        artist.ID,
			StripeAccountID: accountID,
		}); err != nil {
			return StripeConnectResponse{}, err
		}
	}

	link, err := accountlink.New(&stripe.AccountLinkParams{
		Account:    stripe.String(accountID),
		RefreshURL: stripe.String(s.buildStripeOnboardingURL("refresh")),
		ReturnURL:  stripe.String(s.buildStripeOnboardingURL("return")),
		Type:       stripe.String("account_onboarding"),
	})
	if err != nil {
		s.log.Warn("stripe_account_link_failed", "artist_id", artist.ID, "error", err)
		return StripeConnectResponse{}, ErrStripeAPI
	}
	return StripeConnectResponse{URL: link.URL}, nil
}

// RefreshStripeStatus pulls the connected account's current capability flags
// from Stripe and mirrors them locally. Called when the artist returns from
// onboarding (the account.updated webhook is the authoritative path, but the
// return-trip refresh gives instant feedback without waiting for the event).
func (s *service) RefreshStripeStatus(ctx context.Context, userID uuid.UUID) (ArtistSettings, error) {
	if s.cfg.StripeSecretKey == "" {
		return ArtistSettings{}, fmt.Errorf("%w: stripe", ErrIntegrationConfig)
	}

	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return ArtistSettings{}, err
	}
	current, err := s.repo.GetArtistSettings(ctx, artist.ID)
	if err != nil {
		return ArtistSettings{}, err
	}
	if current.StripeAccountID == nil || *current.StripeAccountID == "" {
		// Nothing connected yet — return current settings unchanged.
		return s.settingsResponse(ctx, current), nil
	}

	acct, err := account.GetByID(*current.StripeAccountID, nil)
	if err != nil {
		s.log.Warn("stripe_account_fetch_failed", "artist_id", artist.ID, "error", err)
		return ArtistSettings{}, ErrStripeAPI
	}

	row, err := s.repo.UpdateStripeAccountStatus(ctx, sqlc.UpdateStripeAccountStatusParams{
		ArtistID:         artist.ID,
		ChargesEnabled:   acct.ChargesEnabled,
		PayoutsEnabled:   acct.PayoutsEnabled,
		DetailsSubmitted: acct.DetailsSubmitted,
	})
	if err != nil {
		return ArtistSettings{}, err
	}
	return s.settingsResponse(ctx, row), nil
}

// DisconnectStripe unlinks the artist's connected account. It best-effort
// deletes the Express account we created (Stripe rejects deletion if the
// account still holds a balance) and clears the local link regardless, so the
// artist can reconnect fresh.
func (s *service) DisconnectStripe(ctx context.Context, userID uuid.UUID) (ArtistSettings, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return ArtistSettings{}, err
	}
	current, err := s.repo.GetArtistSettings(ctx, artist.ID)
	if err != nil {
		return ArtistSettings{}, err
	}

	if s.cfg.StripeSecretKey != "" && current.StripeAccountID != nil && *current.StripeAccountID != "" {
		if _, err := account.Del(*current.StripeAccountID, nil); err != nil {
			s.log.Warn("stripe_account_delete_failed", "artist_id", artist.ID, "error", err)
		}
	}

	row, err := s.repo.ClearStripeAccount(ctx, artist.ID)
	if err != nil {
		return ArtistSettings{}, err
	}
	return s.settingsResponse(ctx, row), nil
}

// handleStripeAccountUpdated mirrors an account.updated webhook onto the owning
// artist's settings. Unknown accounts (not ours, or already disconnected) are a
// no-op so stray events don't error the webhook.
func (s *service) handleStripeAccountUpdated(ctx context.Context, accountID string, chargesEnabled, payoutsEnabled, detailsSubmitted bool) error {
	if accountID == "" {
		return nil
	}
	row, err := s.repo.GetArtistSettingsByStripeAccount(ctx, &accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	_, err = s.repo.UpdateStripeAccountStatus(ctx, sqlc.UpdateStripeAccountStatusParams{
		ArtistID:         row.ArtistID,
		ChargesEnabled:   chargesEnabled,
		PayoutsEnabled:   payoutsEnabled,
		DetailsSubmitted: detailsSubmitted,
	})
	return err
}

// syncStripePayoutSchedule pushes the artist's payout frequency to their
// connected account so the live Stripe schedule matches the saved setting.
// Called from UpdateSettings when the frequency changes on a connected account.
func (s *service) syncStripePayoutSchedule(ctx context.Context, accountID, frequency string) error {
	if s.cfg.StripeSecretKey == "" {
		return fmt.Errorf("%w: stripe", ErrIntegrationConfig)
	}
	params := &stripe.AccountParams{
		Settings: &stripe.AccountSettingsParams{
			Payouts: &stripe.AccountSettingsPayoutsParams{
				Schedule: payoutScheduleParams(frequency),
			},
		},
	}
	params.Context = ctx
	if _, err := account.Update(accountID, params); err != nil {
		s.log.Warn("stripe_payout_schedule_sync_failed", "account_id", accountID, "error", err)
		return ErrStripeAPI
	}
	return nil
}

// payoutScheduleParams maps the artist's payout frequency to a Stripe payout
// schedule. Stripe requires an anchor whenever the interval is weekly or
// monthly, so we default to a Monday weekly anchor / 1st-of-month monthly
// anchor. Unknown values fall back to weekly.
func payoutScheduleParams(frequency string) *stripe.AccountSettingsPayoutsScheduleParams {
	if frequency == string(PayoutMonthly) {
		return &stripe.AccountSettingsPayoutsScheduleParams{
			Interval:      stripe.String(string(PayoutMonthly)),
			MonthlyAnchor: stripe.Int64(1),
		}
	}
	return &stripe.AccountSettingsPayoutsScheduleParams{
		Interval:     stripe.String(string(PayoutWeekly)),
		WeeklyAnchor: stripe.String("monday"),
	}
}

// buildStripeOnboardingURL builds the return/refresh links Stripe sends the artist
// back to after the hosted onboarding flow. Lands on the Payments & Payouts tab,
// which reads the stripe status param to refresh the connection state ("return")
// or re-trigger onboarding ("refresh", used when an account link expires).
func (s *service) buildStripeOnboardingURL(status string) string {
	return fmt.Sprintf("%s/dashboard/artist/settings?tab=payments&stripe=%s", s.cfg.FrontendURL, status)
}
