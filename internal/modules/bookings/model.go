package bookings

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type RequestType string

const (
	RequestTypeFlash  RequestType = "flash"
	RequestTypeCustom RequestType = "custom"
)

type Inquiry struct {
	ID                     string          `json:"id"`
	Type                   RequestType     `json:"type"`
	FlashID                *string         `json:"flashId,omitempty"`
	Description            string          `json:"description"`
	ReferenceImageKeys     []string        `json:"referenceImageKeys"`
	ReferenceImageURLs     []string        `json:"referenceImageUrls"`
	Placement              string          `json:"placement"`
	ApproxSizeInches       *int32          `json:"approxSizeInches,omitempty"`
	Styles                 []string        `json:"styles"`
	ClientAvailability     json.RawMessage `json:"clientAvailability"`
	CustomAnswers          json.RawMessage `json:"customAnswers"`
	ClientName             string          `json:"clientName"`
	ClientEmail            string          `json:"clientEmail"`
	ClientPhone            string          `json:"clientPhone,omitempty"`
	Status                 string          `json:"status"`
	DepositStatus          string          `json:"depositStatus"`
	WaiverStatus           string          `json:"waiverStatus"`
	SessionDurationMinutes *int32          `json:"sessionDurationMinutes,omitempty"`
	CreatedAt              string          `json:"createdAt"`
	DecidedAt              *string         `json:"decidedAt,omitempty"`
}

type BookingStats struct {
	NewInquiries    int64 `json:"newInquiries"`
	AwaitingDeposit int64 `json:"awaitingDeposit"`
	BookedThisMonth int64 `json:"bookedThisMonth"`
}

type InquiryListResponse struct {
	Inquiries []Inquiry    `json:"inquiries"`
	Stats     BookingStats `json:"stats"`
}

// ── Request payloads ─────────────────────────────────────────────────────────
type AcceptInput struct {
	SessionDurationMinutes *int32 `json:"sessionDurationMinutes"`
}

// ── Conversions ──────────────────────────────────────────────────────────────
func inquiryFromRow(row sqlc.BookingRequest) Inquiry {
	out := Inquiry{
		ID:                     row.ID.String(),
		Type:                   RequestType(row.Type),
		Description:            row.Description,
		ReferenceImageKeys:     row.ReferenceImageKeys,
		Placement:              row.Placement,
		ApproxSizeInches:       row.ApproxSizeInches,
		Styles:                 row.Styles,
		ClientAvailability:     normalizeJSON(row.ClientAvailability),
		CustomAnswers:          normalizeJSON(row.CustomAnswers),
		ClientName:             row.ClientName,
		ClientEmail:            row.ClientEmail,
		Status:                 row.Status,
		DepositStatus:          row.DepositStatus,
		WaiverStatus:           row.WaiverStatus,
		SessionDurationMinutes: row.SessionDurationMinutes,
		CreatedAt:              formatTimestamp(row.CreatedAt),
	}
	if out.ReferenceImageKeys == nil {
		out.ReferenceImageKeys = []string{}
	}
	out.ReferenceImageURLs = []string{}
	if out.Styles == nil {
		out.Styles = []string{}
	}
	if row.ClientPhone != nil {
		out.ClientPhone = *row.ClientPhone
	}
	if row.FlashID.Valid {
		s := uuid.UUID(row.FlashID.Bytes).String()
		out.FlashID = &s
	}
	if row.DecidedAt.Valid {
		s := row.DecidedAt.Time.UTC().Format(time.RFC3339)
		out.DecidedAt = &s
	}
	return out
}

func statsFromRow(row sqlc.GetBookingStatsRow) BookingStats {
	return BookingStats{
		NewInquiries:    row.NewInquiries,
		AwaitingDeposit: row.AwaitingDeposit,
		BookedThisMonth: row.BookedThisMonth,
	}
}

func normalizeJSON(b []byte) json.RawMessage {
	if len(b) == 0 {
		return json.RawMessage("[]")
	}
	return json.RawMessage(b)
}

func formatTimestamp(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return ""
	}
	return ts.Time.UTC().Format(time.RFC3339)
}
