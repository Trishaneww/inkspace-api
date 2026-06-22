package bookings

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/payments"
)

type DepositRequester interface {
	CreateDepositRequest(ctx context.Context, artistID, bookingID uuid.UUID, amountCents int64, scheduledStart *time.Time) (string, error)
	SetDepositScheduledStart(ctx context.Context, bookingID uuid.UUID, scheduledStart time.Time) error
}

func (s *service) SetDepositRequester(requester DepositRequester) {
	s.depositRequester = requester
}

func (s *service) ensureClientDepositRequest(ctx context.Context, artistID, bookingID uuid.UUID, amountCents int64, scheduledStart time.Time) error {
	existing, err := s.repo.ListPaymentRequestsByBooking(ctx, bookingID)
	if err == nil {
		for _, p := range existing {
			if p.Type == "deposit" && (p.Status == "requested" || p.Status == "processing") {
				return s.depositRequester.SetDepositScheduledStart(ctx, bookingID, scheduledStart)
			}
		}
	}
	_, err = s.depositRequester.CreateDepositRequest(ctx, artistID, bookingID, amountCents, &scheduledStart)
	return err
}

func (s *service) ConfirmDepositPaid(ctx context.Context, bookingRequestID uuid.UUID, scheduledStart *time.Time) (payments.DepositConfirmation, error) {
	appt, err := s.confirmDepositAppointment(ctx, bookingRequestID, scheduledStart)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return payments.DepositConfirmation{Confirmed: false}, nil
		}
		return payments.DepositConfirmation{}, err
	}

	booking, err := s.repo.GetBookingRequest(ctx, sqlc.GetBookingRequestParams{
		ID:       bookingRequestID,
		ArtistID: appt.ArtistID,
	})
	if err != nil {
		return payments.DepositConfirmation{}, err
	}
	inquiry, err := s.enrichInquiry(ctx, booking)
	if err != nil {
		return payments.DepositConfirmation{}, err
	}
	s.syncAppointmentToCalendar(ctx, appt.ArtistID, appt, inquiry)

	when := ""
	if appt.ScheduledStart.Valid {
		settings, _ := s.repo.GetArtistSettings(ctx, appt.ArtistID)
		loc := loadLocation(settings.Timezone)
		when = appt.ScheduledStart.Time.In(loc).Format("Monday, January 2 at 3:04 PM")
	}
	artistName, artistEmail := s.artistContact(ctx, appt.ArtistID)
	s.sendBookingConfirmedEmail(booking, artistName, when, appt.DurationMinutes)

	return payments.DepositConfirmation{
		Confirmed:       true,
		ArtistName:      artistName,
		ArtistEmail:     artistEmail,
		ClientName:      booking.ClientName,
		WhenLabel:       when,
		DurationMinutes: appt.DurationMinutes,
	}, nil
}

func (s *service) confirmDepositAppointment(ctx context.Context, bookingRequestID uuid.UUID, scheduledStart *time.Time) (sqlc.Appointment, error) {
	held, err := s.repo.ConfirmHeldAppointment(ctx, bookingRequestID)
	if err == nil {
		return held, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Appointment{}, err
	}
	if scheduledStart == nil {
		return sqlc.Appointment{}, pgx.ErrNoRows
	}

	appt, err := s.repo.GetLatestAppointmentByRequest(ctx, bookingRequestID)
	if err != nil {
		return sqlc.Appointment{}, err
	}
	if appt.Status != appointmentProposed {
		return sqlc.Appointment{}, pgx.ErrNoRows
	}
	return s.repo.UpdateAppointmentSchedule(ctx, sqlc.UpdateAppointmentScheduleParams{
		ID:             appt.ID,
		ArtistID:       appt.ArtistID,
		ScheduledStart: pgtype.Timestamptz{Time: *scheduledStart, Valid: true},
	})
}

func resolveDepositCents(input *int64, settings sqlc.ArtistSetting) int64 {
	if input != nil {
		if *input < 0 {
			return 0
		}
		return *input
	}
	if settings.DepositFlatFeeCents != nil && *settings.DepositFlatFeeCents > 0 {
		return *settings.DepositFlatFeeCents
	}
	return 0
}

func (s *service) appointmentWhenLabel(ctx context.Context, artistID uuid.UUID, appt sqlc.Appointment) string {
	if !appt.ScheduledStart.Valid {
		return ""
	}
	settings, _ := s.repo.GetArtistSettings(ctx, artistID)
	loc := loadLocation(settings.Timezone)
	return appt.ScheduledStart.Time.In(loc).Format("Monday, January 2 at 3:04 PM")
}

func (s *service) artistCurrency(ctx context.Context, artistID uuid.UUID) string {
	settings, err := s.repo.GetArtistSettings(ctx, artistID)
	if err != nil || settings.Currency == "" {
		return "CAD"
	}
	return settings.Currency
}

func (s *service) artistContact(ctx context.Context, artistID uuid.UUID) (name, email string) {
	artist, err := s.repo.GetArtistByID(ctx, artistID)
	if err != nil {
		return "", ""
	}
	user, err := s.repo.GetUserByID(ctx, artist.UserID)
	if err != nil {
		return "", ""
	}
	name = strings.TrimSpace(strings.Join([]string{stringOrEmpty(user.FirstName), stringOrEmpty(user.LastName)}, " "))
	if name == "" {
		name = stringOrEmpty(user.Username)
	}
	return name, user.Email
}
