package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type paymentRequestEmailPayload struct {
	To          string `json:"to"`
	ClientName  string `json:"clientName"`
	ArtistName  string `json:"artistName"`
	Type        string `json:"type"`
	AmountCents int64  `json:"amountCents"`
	Currency    string `json:"currency"`
	PayURL      string `json:"payUrl"`
}

type paymentReceiptEmailPayload struct {
	To          string `json:"to"`
	ClientName  string `json:"clientName"`
	ArtistName  string `json:"artistName"`
	Type        string `json:"type"`
	AmountCents int64  `json:"amountCents"`
	Currency    string `json:"currency"`
	PaidAt      string `json:"paidAt"`
	Reference   string `json:"reference"`
}

func (s *service) sendPaymentRequestEmail(row sqlc.PaymentRequest, artistName string) {
	s.postInternalEmail("/api/internal/emails/payment-request", paymentRequestEmailPayload{
		To:          row.ClientEmail,
		ClientName:  row.ClientName,
		ArtistName:  artistName,
		Type:        row.Type,
		AmountCents: row.ClientChargeCents,
		Currency:    row.Currency,
		PayURL:      s.payLinkURL(row.PublicToken),
	})
}

func (s *service) sendPaymentReceiptEmail(row sqlc.PaymentRequest, artistName string) {
	paidAt := time.Now().UTC().Format(time.RFC3339)
	if row.PaidAt.Valid {
		paidAt = row.PaidAt.Time.UTC().Format(time.RFC3339)
	}
	s.postInternalEmail("/api/internal/emails/payment-receipt", paymentReceiptEmailPayload{
		To:          row.ClientEmail,
		ClientName:  row.ClientName,
		ArtistName:  artistName,
		Type:        row.Type,
		AmountCents: row.ClientChargeCents,
		Currency:    row.Currency,
		PaidAt:      paidAt,
		Reference:   row.ID.String(),
	})
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
