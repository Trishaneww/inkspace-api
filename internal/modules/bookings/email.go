package bookings

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type bookingRequestEmailPayload struct {
	To              string `json:"to"`
	ClientName      string `json:"clientName"`
	ArtistName      string `json:"artistName"`
	DurationMinutes int32  `json:"durationMinutes"`
	BookingURL      string `json:"bookingUrl"`
}

func (s *service) sendScheduleInvite(ctx context.Context, artistID, inquiryID uuid.UUID, durationMinutes int32) {
	token, err := newScheduleToken()
	if err != nil {
		s.log.Warn("schedule_token_generation_failed", "error", err)
		return
	}
	row, err := s.repo.SetScheduleToken(ctx, sqlc.SetScheduleTokenParams{
		ID:            inquiryID,
		ArtistID:      artistID,
		ScheduleToken: token,
	})
	if err != nil {
		s.log.Warn("schedule_token_set_failed", "inquiry", inquiryID, "error", err)
		return
	}
	s.sendBookingRequestEmail(row, s.artistDisplayName(ctx, artistID), durationMinutes)
}

type bookingConfirmedEmailPayload struct {
	To              string `json:"to"`
	ClientName      string `json:"clientName"`
	ArtistName      string `json:"artistName"`
	WhenLabel       string `json:"whenLabel"`
	DurationMinutes int32  `json:"durationMinutes"`
}

func (s *service) sendBookingConfirmedEmail(row sqlc.BookingRequest, artistName, whenLabel string, durationMinutes int32) {
	s.postInternalEmail("/api/internal/emails/booking-confirmed", bookingConfirmedEmailPayload{
		To:              row.ClientEmail,
		ClientName:      row.ClientName,
		ArtistName:      artistName,
		WhenLabel:       whenLabel,
		DurationMinutes: durationMinutes,
	})
}

func (s *service) sendBookingRequestEmail(row sqlc.BookingRequest, artistName string, durationMinutes int32) {
	if row.ScheduleToken == nil || *row.ScheduleToken == "" {
		return
	}
	s.postInternalEmail("/api/internal/emails/booking-request", bookingRequestEmailPayload{
		To:              row.ClientEmail,
		ClientName:      row.ClientName,
		ArtistName:      artistName,
		DurationMinutes: durationMinutes,
		BookingURL:      s.scheduleLinkURL(*row.ScheduleToken),
	})
}

type depositRequestEmailPayload struct {
	To          string `json:"to"`
	ClientName  string `json:"clientName"`
	ArtistName  string `json:"artistName"`
	AmountCents int64  `json:"amountCents"`
	Currency    string `json:"currency"`
	WhenLabel   string `json:"whenLabel"`
	PayURL      string `json:"payUrl"`
}

func (s *service) sendDepositRequestEmail(inquiry Inquiry, artistName, token string, depositCents int64, currency, whenLabel string) {
	s.postInternalEmail("/api/internal/emails/deposit-request", depositRequestEmailPayload{
		To:          inquiry.ClientEmail,
		ClientName:  inquiry.ClientName,
		ArtistName:  artistName,
		AmountCents: depositCents,
		Currency:    currency,
		WhenLabel:   whenLabel,
		PayURL:      s.payLinkURL(token),
	})
}

func (s *service) payLinkURL(token string) string {
	return s.cfg.FrontendURL + "/pay/" + token
}

func (s *service) scheduleLinkURL(token string) string {
	return s.cfg.FrontendURL + "/book/" + token
}

func (s *service) postInternalEmail(path string, payload any) {
	if s.cfg.InternalEmailSecret == "" {
		return
	}
	body, err := json.Marshal(payload)
	if err != nil {
		s.log.Warn("internal_email_marshal_failed", "path", path, "error", err)
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.FrontendURL+path, bytes.NewReader(body))
		if err != nil {
			s.log.Warn("internal_email_request_failed", "path", path, "error", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-internal-secret", s.cfg.InternalEmailSecret)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			s.log.Warn("internal_email_send_failed", "path", path, "error", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusMultipleChoices {
			s.log.Warn("internal_email_rejected", "path", path, "status", resp.StatusCode)
		}
	}()
}
