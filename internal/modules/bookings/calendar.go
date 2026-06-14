package bookings

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/crypto"
	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

func (s *service) syncAppointmentToCalendar(ctx context.Context, artistID uuid.UUID, appt sqlc.Appointment, inquiry Inquiry) {
	if s.calendar == nil || !appt.ScheduledStart.Valid {
		return
	}
	settings, err := s.repo.GetArtistSettings(ctx, artistID)
	if err != nil {
		return
	}
	svc, err := s.calendar.buildService(ctx, settings)
	if err != nil {
		s.calendar.log.Error("calendar: build service", "err", err)
		return
	}
	if svc == nil {
		return // not connected
	}

	event := buildCalendarEvent(settings.Timezone, inquiry, appt)
	if appt.GoogleCalendarEventID != nil && *appt.GoogleCalendarEventID != "" {
		if _, err := svc.Events.Update("primary", *appt.GoogleCalendarEventID, event).SendUpdates("all").Do(); err != nil {
			s.calendar.log.Error("calendar: update event", "err", err, "appointment", appt.ID)
		}
		return
	}
	created, err := svc.Events.Insert("primary", event).SendUpdates("all").Do()
	if err != nil {
		s.calendar.log.Error("calendar: insert event", "err", err, "appointment", appt.ID)
		return
	}
	if err := s.repo.SetAppointmentCalendarEvent(ctx, sqlc.SetAppointmentCalendarEventParams{
		ID:                    appt.ID,
		GoogleCalendarEventID: &created.Id,
	}); err != nil {
		s.calendar.log.Error("calendar: store event id", "err", err, "appointment", appt.ID)
	}
}

func (s *service) removeAppointmentFromCalendar(ctx context.Context, artistID uuid.UUID, appt sqlc.Appointment) {
	if s.calendar == nil || appt.GoogleCalendarEventID == nil || *appt.GoogleCalendarEventID == "" {
		return
	}
	settings, err := s.repo.GetArtistSettings(ctx, artistID)
	if err != nil {
		return
	}
	svc, err := s.calendar.buildService(ctx, settings)
	if err != nil || svc == nil {
		return
	}
	if err := svc.Events.Delete("primary", *appt.GoogleCalendarEventID).SendUpdates("all").Do(); err != nil {
		s.calendar.log.Error("calendar: delete event", "err", err, "appointment", appt.ID)
	}
}

type calendarClient struct {
	cfg    *config.Config
	cipher *crypto.Cipher
	log    *slog.Logger
}

func newCalendarClient(cfg *config.Config, cipher *crypto.Cipher) *calendarClient {
	return &calendarClient{cfg: cfg, cipher: cipher, log: slog.Default()}
}

// buildService returns an authenticated Calendar service for the artist, or nil when
// the artist hasn't connected Google Calendar (or the server isn't configured
// for it). The oauth2 token source refreshes the access token transparently.
func (c *calendarClient) buildService(ctx context.Context, settings sqlc.ArtistSetting) (*calendar.Service, error) {
	if c == nil || c.cipher == nil ||
		c.cfg.GoogleClientID == "" || c.cfg.GoogleClientSecret == "" {
		return nil, nil
	}
	if settings.GoogleCalendarRefreshToken == nil ||
		*settings.GoogleCalendarRefreshToken == "" {
		return nil, nil
	}

	refresh, err := c.cipher.Decrypt(*settings.GoogleCalendarRefreshToken)
	if err != nil {
		return nil, fmt.Errorf("decrypt refresh token: %w", err)
	}
	access := ""
	if settings.GoogleCalendarAccessToken != nil {
		if dec, decErr := c.cipher.Decrypt(*settings.GoogleCalendarAccessToken); decErr == nil {
			access = dec
		}
	}

	oauthCfg := &oauth2.Config{
		ClientID:     c.cfg.GoogleClientID,
		ClientSecret: c.cfg.GoogleClientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{calendar.CalendarEventsScope},
	}
	tok := &oauth2.Token{AccessToken: access, RefreshToken: refresh}
	if settings.GoogleCalendarTokenExpiry.Valid {
		tok.Expiry = settings.GoogleCalendarTokenExpiry.Time
	}

	src := oauthCfg.TokenSource(ctx, tok)
	return calendar.NewService(ctx, option.WithTokenSource(src))
}

func buildCalendarEvent(timezone string, inquiry Inquiry, appt sqlc.Appointment) *calendar.Event {
	if timezone == "" {
		timezone = "UTC"
	}
	start := appt.ScheduledStart.Time
	end := start.Add(time.Duration(appt.DurationMinutes) * time.Minute)

	noun := "Tattoo session"
	if appt.Type == string(AppointmentConsultation) {
		noun = "Consultation"
	}

	var desc strings.Builder
	if piece := describePieceForCalendar(inquiry); piece != "" {
		desc.WriteString(piece + "\n")
	}
	desc.WriteString("Client: " + inquiry.ClientName)
	if inquiry.ClientPhone != "" {
		desc.WriteString(" · " + inquiry.ClientPhone)
	}
	desc.WriteString("\n" + inquiry.ClientEmail)
	if inquiry.Description != "" {
		desc.WriteString("\n\n" + inquiry.Description)
	}

	return &calendar.Event{
		Summary:     noun + " — " + inquiry.ClientName,
		Description: desc.String(),
		Location:    formatLocation(inquiry, appt),
		Start: &calendar.EventDateTime{
			DateTime: start.UTC().Format(time.RFC3339),
			TimeZone: timezone,
		},
		End: &calendar.EventDateTime{
			DateTime: end.UTC().Format(time.RFC3339),
			TimeZone: timezone,
		},
		Attendees: []*calendar.EventAttendee{
			{Email: inquiry.ClientEmail, DisplayName: inquiry.ClientName},
		},
	}
}

func formatLocation(inquiry Inquiry, appt sqlc.Appointment) string {
	if appt.Format != nil {
		switch *appt.Format {
		case "online":
			return "Online"
		case "phone":
			return "Phone call"
		}
	}
	if inquiry.Location != nil {
		parts := []string{}
		for _, p := range []string{
			inquiry.Location.Address,
			inquiry.Location.City,
			inquiry.Location.Country,
		} {
			if strings.TrimSpace(p) != "" {
				parts = append(parts, p)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, ", ")
		}
		return inquiry.Location.Label
	}
	return ""
}

var calendarColorLabels = map[string]string{
	"black_and_grey": "Black & grey",
	"color":          "Color",
	"either":         "Either",
}

func describePieceForCalendar(inquiry Inquiry) string {
	if inquiry.Flash != nil {
		return inquiry.Flash.Title
	}
	parts := []string{}
	if inquiry.Placement != "" {
		parts = append(parts, inquiry.Placement)
	}
	if inquiry.ApproxSizeInches != nil {
		parts = append(parts, fmt.Sprintf("%d\"", *inquiry.ApproxSizeInches))
	}
	if label, ok := calendarColorLabels[inquiry.ColorType]; ok {
		parts = append(parts, label)
	}
	return strings.Join(parts, " · ")
}
