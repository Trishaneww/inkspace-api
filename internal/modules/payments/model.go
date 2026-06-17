package payments

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

const (
	typeDeposit = "deposit"
	typeFinal   = "final"

	feePayerArtist = "artist"
	feePayerClient = "client"
	feePayerSplit  = "split"
)

type PaymentRequest struct {
	ID                string  `json:"id"`
	BookingRequestID  string  `json:"bookingRequestId"`
	Type              string  `json:"type"`
	Status            string  `json:"status"`
	Currency          string  `json:"currency"`
	AmountCents       int64   `json:"amountCents"`
	PlatformFeeCents  int64   `json:"platformFeeCents"`
	ClientChargeCents int64   `json:"clientChargeCents"`
	ArtistNetCents    int64   `json:"artistNetCents"`
	FeePayer          string  `json:"feePayer"`
	ClientEmail       string  `json:"clientEmail"`
	Description       string  `json:"description"`
	PayURL            string  `json:"payUrl"`
	ExpiresAt         string  `json:"expiresAt"`
	PaidAt            *string `json:"paidAt,omitempty"`
	CreatedAt         string  `json:"createdAt"`
}

type CreatePaymentRequestInput struct {
	Type        string `json:"type"`
	AmountCents *int64 `json:"amountCents"`
}

type PublicPaymentRequest struct {
	Status            string `json:"status"`
	Type              string `json:"type"`
	Currency          string `json:"currency"`
	ClientChargeCents int64  `json:"clientChargeCents"`
	ArtistName        string `json:"artistName"`
	ClientEmail       string `json:"clientEmail"`
	ClientName        string `json:"clientName"`
	Description       string `json:"description"`
	Expired           bool   `json:"expired"`

	HasAccount bool `json:"hasAccount"`
}

type CreateClientAccountInput struct {
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	Password       string `json:"password"`
	MarketingOptIn bool   `json:"marketingOptIn"`
}

type CheckoutResponse struct {
	URL string `json:"url"`
}

func paymentRequestFromRow(row sqlc.PaymentRequest, payURL string) PaymentRequest {
	out := PaymentRequest{
		ID:                row.ID.String(),
		BookingRequestID:  row.BookingRequestID.String(),
		Type:              row.Type,
		Status:            row.Status,
		Currency:          row.Currency,
		AmountCents:       row.AmountCents,
		PlatformFeeCents:  row.PlatformFeeCents,
		ClientChargeCents: row.ClientChargeCents,
		ArtistNetCents:    row.ClientChargeCents - row.PlatformFeeCents,
		FeePayer:          row.FeePayer,
		ClientEmail:       row.ClientEmail,
		Description:       row.Description,
		PayURL:            payURL,
		ExpiresAt:         formatTimestamp(row.ExpiresAt),
		CreatedAt:         formatTimestamp(row.CreatedAt),
	}
	if row.PaidAt.Valid {
		s := row.PaidAt.Time.UTC().Format(time.RFC3339)
		out.PaidAt = &s
	}
	return out
}

func formatTimestamp(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return ""
	}
	return ts.Time.UTC().Format(time.RFC3339)
}
