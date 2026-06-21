package payments

import (
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

type depositPaidEmailPayload struct {
	To              string `json:"to"`
	ArtistName      string `json:"artistName"`
	ClientName      string `json:"clientName"`
	AmountCents     int64  `json:"amountCents"`
	Currency        string `json:"currency"`
	WhenLabel       string `json:"whenLabel"`
	DurationMinutes int32  `json:"durationMinutes"`
}

func (s *service) sendDepositPaidEmail(row sqlc.PaymentRequest, confirmation DepositConfirmation) {
	if confirmation.ArtistEmail == "" {
		return
	}
	s.postInternalEmail("/api/internal/emails/deposit-paid", depositPaidEmailPayload{
		To:              confirmation.ArtistEmail,
		ArtistName:      confirmation.ArtistName,
		ClientName:      confirmation.ClientName,
		AmountCents:     row.AmountCents,
		Currency:        row.Currency,
		WhenLabel:       confirmation.WhenLabel,
		DurationMinutes: confirmation.DurationMinutes,
	})
}

func (s *service) postInternalEmail(path string, payload any) {
	s.email.Post(path, payload)
}
