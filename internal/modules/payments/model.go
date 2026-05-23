package payments

import "time"

type Status string

const (
	StatusPending   Status = "pending"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusRefunded  Status = "refunded"
)

type Kind string

const (
	KindDeposit Kind = "deposit"
	KindBalance Kind = "balance"
)

type Payment struct {
	ID               string    `json:"id"`
	BookingID        string    `json:"booking_id"`
	PayerID          string    `json:"payer_id"`
	PayeeID          string    `json:"payee_id"`
	Kind             Kind      `json:"kind"`
	AmountCents      int64     `json:"amount_cents"`
	Currency         string    `json:"currency"`
	Status           Status    `json:"status"`
	Provider         string    `json:"provider"`
	ProviderIntentID string    `json:"provider_intent_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CreateDepositIntentInput struct {
	BookingID   string `json:"booking_id" binding:"required"`
	AmountCents int64  `json:"amount_cents" binding:"required,gt=0"`
	Currency    string `json:"currency" binding:"required,len=3"`
}

type DepositIntentResponse struct {
	PaymentID    string `json:"payment_id"`
	ClientSecret string `json:"client_secret"`
}
