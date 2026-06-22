// Package holds sweeps expired deposit holds: when a client doesn't pay their
// deposit in time, the held session is cancelled, the slot is freed, the deposit
// request is expired, and both parties are notified.
package holds

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
	"github.com/trishaneupnexx/inkspace-api/internal/email"
)

const sweepTimeout = 60 * time.Second

type Worker struct {
	q        *sqlc.Queries
	email    *email.Client
	log      *slog.Logger
	interval time.Duration
}

func New(db *pgxpool.Pool, emailClient *email.Client, log *slog.Logger, interval time.Duration) *Worker {
	return &Worker{
		q:        sqlc.New(db),
		email:    emailClient,
		log:      log,
		interval: interval,
	}
}

func (w *Worker) Run(ctx context.Context) {
	w.sweep(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.sweep(ctx)
		}
	}
}

type holdExpiredEmailPayload struct {
	To         string `json:"to"`
	Audience   string `json:"audience"` // "client" | "artist"
	ClientName string `json:"clientName"`
	ArtistName string `json:"artistName"`
}

func (w *Worker) sweep(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, sweepTimeout)
	defer cancel()

	// ClaimExpiredHolds already cancels the held appointments and returns them.
	rows, err := w.q.ClaimExpiredHolds(ctx)
	if err != nil {
		w.log.Warn("expired_holds_claim_failed", "error", err)
		return
	}
	if len(rows) == 0 {
		return
	}

	for _, row := range rows {
		if err := w.q.ExpireDepositRequestsForBooking(ctx, row.BookingRequestID); err != nil {
			w.log.Warn("expire_deposit_request_failed", "booking_request_id", row.BookingRequestID, "error", err)
		}
		if err := w.q.SetBookingDepositStatus(ctx, sqlc.SetBookingDepositStatusParams{
			DepositStatus:    "not_required",
			BookingRequestID: row.BookingRequestID,
		}); err != nil {
			w.log.Warn("reset_deposit_status_failed", "booking_request_id", row.BookingRequestID, "error", err)
		}
		w.notify(ctx, row)
	}
	w.log.Info("expired_holds_swept", "count", len(rows))
}

func (w *Worker) notify(ctx context.Context, row sqlc.ClaimExpiredHoldsRow) {
	booking, err := w.q.GetBookingRequest(ctx, sqlc.GetBookingRequestParams{
		ID:       row.BookingRequestID,
		ArtistID: row.ArtistID,
	})
	if err != nil {
		return
	}
	artistName, artistAddr := w.artistContact(ctx, row.ArtistID)

	w.email.Post("/api/internal/emails/hold-expired", holdExpiredEmailPayload{
		To:         booking.ClientEmail,
		Audience:   "client",
		ClientName: booking.ClientName,
		ArtistName: artistName,
	})
	if artistAddr != "" {
		w.email.Post("/api/internal/emails/hold-expired", holdExpiredEmailPayload{
			To:         artistAddr,
			Audience:   "artist",
			ClientName: booking.ClientName,
			ArtistName: artistName,
		})
	}
}

func (w *Worker) artistContact(ctx context.Context, artistID uuid.UUID) (name, addr string) {
	artist, err := w.q.GetArtistByID(ctx, artistID)
	if err != nil {
		return "", ""
	}
	user, err := w.q.GetUserByID(ctx, artist.UserID)
	if err != nil {
		return "", ""
	}
	name = strings.TrimSpace(deref(user.FirstName) + " " + deref(user.LastName))
	if name == "" {
		name = deref(user.Username)
	}
	return name, user.Email
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
