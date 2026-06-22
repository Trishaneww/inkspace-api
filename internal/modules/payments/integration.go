package payments

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

const depositHoldTTL = 48 * time.Hour

type DepositConfirmer interface {
	ConfirmDepositPaid(ctx context.Context, bookingRequestID uuid.UUID, scheduledStart *time.Time) (DepositConfirmation, error)
}

type DepositConfirmation struct {
	Confirmed       bool
	ArtistName      string
	ArtistEmail     string
	ClientName      string
	WhenLabel       string
	DurationMinutes int32
}

func (s *service) CreateDepositRequest(ctx context.Context, artistID, bookingID uuid.UUID, amountCents int64, scheduledStart *time.Time) (string, error) {
	if amountCents < MinChargeCents {
		return "", ErrAmountTooLow
	}

	settings, err := s.repo.GetArtistSettings(ctx, artistID)
	if err != nil {
		return "", err
	}
	if settings.StripeAccountID == nil || *settings.StripeAccountID == "" || !settings.StripeChargesEnabled {
		return "", ErrStripeNotConnected
	}

	booking, err := s.repo.GetBookingRequest(ctx, sqlc.GetBookingRequestParams{ID: bookingID, ArtistID: artistID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}

	fee := computeFee(amountCents, settings.PlatformFeePayer)
	token, err := newPublicToken()
	if err != nil {
		return "", err
	}

	start := pgtype.Timestamptz{}
	if scheduledStart != nil {
		start = pgtype.Timestamptz{Time: *scheduledStart, Valid: true}
	}

	if _, err := s.repo.CreatePaymentRequest(ctx, sqlc.CreatePaymentRequestParams{
		ArtistID:          artistID,
		BookingRequestID:  bookingID,
		Type:              typeDeposit,
		Currency:          settings.Currency,
		AmountCents:       fee.AmountCents,
		PlatformFeeCents:  fee.PlatformFeeCents,
		ClientChargeCents: fee.ClientChargeCents,
		FeePayer:          settings.PlatformFeePayer,
		ClientEmail:       booking.ClientEmail,
		ClientName:        booking.ClientName,
		Description:       booking.Description,
		PublicToken:       token,
		ExpiresAt:         pgtype.Timestamptz{Time: time.Now().Add(depositHoldTTL), Valid: true},
		ScheduledStart:    start,
	}); err != nil {
		return "", err
	}

	_ = s.repo.SetBookingDepositStatus(ctx, sqlc.SetBookingDepositStatusParams{
		DepositStatus:    "pending",
		BookingRequestID: bookingID,
	})
	return token, nil
}

func (s *service) SetDepositScheduledStart(ctx context.Context, bookingID uuid.UUID, scheduledStart time.Time) error {
	return s.repo.SetDepositScheduledStart(ctx, sqlc.SetDepositScheduledStartParams{
		BookingRequestID: bookingID,
		ScheduledStart:   pgtype.Timestamptz{Time: scheduledStart, Valid: true},
	})
}
