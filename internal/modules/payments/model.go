package payments

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stripe/stripe-go/v82"

	"github.com/trishaneupnexx/inkspace-api/internal/clientaccount"
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
	Phone          string `json:"phone"`
	Password       string `json:"password"`
	MarketingOptIn bool   `json:"marketingOptIn"`
}

type ClientSession = clientaccount.Session

type CheckoutResponse struct {
	URL string `json:"url"`
}

type Earnings struct {
	IssuerName     string          `json:"issuerName"`
	Currency       string          `json:"currency"`
	AllTime        EarningsTotals  `json:"allTime"`
	ThisMonth      EarningsTotals  `json:"thisMonth"`
	RecentPayments []RecentPayment `json:"recentPayments"`
}

type EarningsTotals struct {
	CollectedCents int64 `json:"collectedCents"`
	FeeCents       int64 `json:"feeCents"`
	NetCents       int64 `json:"netCents"`
	Count          int64 `json:"count"`
}

type RecentPayment struct {
	ID                string `json:"id"`
	Reference         string `json:"reference"`
	Type              string `json:"type"`
	Status            string `json:"status"`
	Currency          string `json:"currency"`
	AmountCents       int64  `json:"amountCents"`
	PlatformFeeCents  int64  `json:"platformFeeCents"`
	ClientChargeCents int64  `json:"clientChargeCents"`
	ClientName        string `json:"clientName"`
	ClientEmail       string `json:"clientEmail"`
	RequestType       string `json:"requestType"`
	Placement         string `json:"placement"`
	PaidAt            string `json:"paidAt"`
}

func earningsTotals(collected, fee, count int64) EarningsTotals {
	return EarningsTotals{
		CollectedCents: collected,
		FeeCents:       fee,
		NetCents:       collected - fee,
		Count:          count,
	}
}

func recentPaymentFromRow(row sqlc.ListRecentPaidPaymentsForArtistRow) RecentPayment {
	return RecentPayment{
		ID:                row.ID.String(),
		Reference:         invoiceReference(row.ID),
		Type:              row.Type,
		Status:            row.Status,
		Currency:          row.Currency,
		AmountCents:       row.AmountCents,
		PlatformFeeCents:  row.PlatformFeeCents,
		ClientChargeCents: row.ClientChargeCents,
		ClientName:        row.ClientName,
		ClientEmail:       row.ClientEmail,
		RequestType:       row.RequestType,
		Placement:         row.Placement,
		PaidAt:            formatTimestamp(row.PaidAt),
	}
}

func invoiceReference(id uuid.UUID) string {
	return "INV-" + strings.ToUpper(strings.ReplaceAll(id.String(), "-", "")[:8])
}

type Payout struct {
	ID          string `json:"id"`
	Reference   string `json:"reference"`
	Currency    string `json:"currency"`
	AmountCents int64  `json:"amountCents"`
	Status      string `json:"status"`
	BankLast4   string `json:"bankLast4"`
	InitiatedAt string `json:"initiatedAt"`
	ArrivalAt   string `json:"arrivalAt"`
}

func payoutFromStripe(p *stripe.Payout) Payout {
	return Payout{
		ID:          p.ID,
		Reference:   payoutReference(p.ID),
		Currency:    strings.ToUpper(string(p.Currency)),
		AmountCents: p.Amount,
		Status:      string(p.Status),
		BankLast4:   payoutBankLast4(p),
		InitiatedAt: formatUnixSeconds(p.Created),
		ArrivalAt:   formatUnixSeconds(p.ArrivalDate),
	}
}

func payoutBankLast4(p *stripe.Payout) string {
	if p.Destination == nil {
		return ""
	}
	if p.Destination.BankAccount != nil {
		return p.Destination.BankAccount.Last4
	}
	if p.Destination.Card != nil {
		return p.Destination.Card.Last4
	}
	return ""
}

func payoutReference(id string) string {
	cleaned := strings.ToUpper(strings.ReplaceAll(strings.TrimPrefix(id, "po_"), "_", ""))
	if len(cleaned) > 8 {
		cleaned = cleaned[:8]
	}
	return "PO-" + cleaned
}

func formatUnixSeconds(seconds int64) string {
	if seconds == 0 {
		return ""
	}
	return time.Unix(seconds, 0).UTC().Format(time.RFC3339)
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
