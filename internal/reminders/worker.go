package reminders

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
	"github.com/trishaneupnexx/inkspace-api/internal/email"
	"github.com/trishaneupnexx/inkspace-api/internal/messaging"
)

const sweepTimeout = 60 * time.Second
const paymentReminderAfter = 3 * 24 * time.Hour
const sessionReminderWindow = 24 * time.Hour
const fallbackTimezone = "America/Toronto"

type Worker struct {
	q           *sqlc.Queries
	sms         messaging.Sender
	email       *email.Client
	frontendURL string
	log         *slog.Logger
	interval    time.Duration
}

func New(db *pgxpool.Pool, sms messaging.Sender, emailClient *email.Client, frontendURL string, log *slog.Logger, interval time.Duration) *Worker {
	return &Worker{
		q:           sqlc.New(db),
		sms:         sms,
		email:       emailClient,
		frontendURL: frontendURL,
		log:         log,
		interval:    interval,
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

func (w *Worker) sweep(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, sweepTimeout)
	defer cancel()

	w.sweepPaymentReminders(ctx)
	w.sweepSessionReminders(ctx)
}

type paymentReminderEmailPayload struct {
	To          string `json:"to"`
	ClientName  string `json:"clientName"`
	ArtistName  string `json:"artistName"`
	AmountCents int64  `json:"amountCents"`
	Currency    string `json:"currency"`
	PayURL      string `json:"payUrl"`
}

func (w *Worker) sweepPaymentReminders(ctx context.Context) {
	cutoff := pgtype.Timestamptz{Time: time.Now().Add(-paymentReminderAfter), Valid: true}
	rows, err := w.q.ClaimDuePaymentReminders(ctx, cutoff)
	if err != nil {
		w.log.Warn("payment_reminders_claim_failed", "error", err)
		return
	}
	if len(rows) == 0 {
		return
	}

	resolve := w.artistResolver(ctx)
	for _, row := range rows {
		artist := resolve(row.ArtistID)
		w.email.Post("/api/internal/emails/payment-reminder", paymentReminderEmailPayload{
			To:          row.ClientEmail,
			ClientName:  row.ClientName,
			ArtistName:  artist.name,
			AmountCents: row.ClientChargeCents,
			Currency:    row.Currency,
			PayURL:      w.frontendURL + "/pay/" + row.PublicToken,
		})
	}
	w.log.Info("payment_reminders_sent", "count", len(rows))
}

func (w *Worker) sweepSessionReminders(ctx context.Context) {
	windowEnd := pgtype.Timestamptz{Time: time.Now().Add(sessionReminderWindow), Valid: true}
	rows, err := w.q.ClaimDueSessionReminders(ctx, windowEnd)
	if err != nil {
		w.log.Warn("session_reminders_claim_failed", "error", err)
		return
	}
	if len(rows) == 0 {
		return
	}

	resolve := w.artistResolver(ctx)
	sent := 0
	for _, row := range rows {
		artist := resolve(row.ArtistID)
		body := buildReminderSMS(row.ClientName, artist.name, row.Type, row.ScheduledStart.Time, artist.location)
		if err := w.sms.Send(ctx, row.ClientPhone, body); err != nil {
			w.log.Warn("session_reminder_sms_failed", "appointment", row.ID, "error", err)
			continue
		}
		sent++
	}
	w.log.Info("session_reminders_sent", "sent", sent, "claimed", len(rows))
}

func buildReminderSMS(clientName, artistName, apptType string, start time.Time, loc *time.Location) string {
	kind := "session"
	if apptType == "consultation" {
		kind = "consultation"
	}
	when := start.In(loc).Format("Mon Jan 2 at 3:04 PM")
	return fmt.Sprintf(
		"Hi %s, a reminder of your %s with %s coming up on %s. See you then! — Inkspace",
		firstName(clientName), kind, artistName, when,
	)
}

type artistInfo struct {
	name     string
	location *time.Location
}

func (w *Worker) artistResolver(ctx context.Context) func(uuid.UUID) artistInfo {
	cache := make(map[uuid.UUID]artistInfo)
	return func(id uuid.UUID) artistInfo {
		if info, ok := cache[id]; ok {
			return info
		}
		info := w.lookupArtist(ctx, id)
		cache[id] = info
		return info
	}
}

func (w *Worker) lookupArtist(ctx context.Context, artistID uuid.UUID) artistInfo {
	contact, err := w.q.GetArtistContact(ctx, artistID)
	if err != nil {
		return artistInfo{name: "your artist", location: fallbackLocation()}
	}

	name := strings.TrimSpace(deref(contact.FirstName) + " " + deref(contact.LastName))
	if name == "" {
		name = deref(contact.Username)
	}
	if name == "" {
		name = "your artist"
	}

	return artistInfo{name: name, location: loadLocation(deref(contact.Timezone))}
}

func loadLocation(timezone string) *time.Location {
	if timezone == "" {
		return fallbackLocation()
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return fallbackLocation()
	}
	return loc
}

func fallbackLocation() *time.Location {
	loc, err := time.LoadLocation(fallbackTimezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

func firstName(fullName string) string {
	fields := strings.Fields(fullName)
	if len(fields) == 0 {
		return "there"
	}
	return fields[0]
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
