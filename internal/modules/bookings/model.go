package bookings

import (
	"encoding/json"
	"strings"
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

// AppointmentType distinguishes the two scheduled outcomes that ride the same
// scheduling rails: an optional pre-booking consultation and the tattoo session.
type AppointmentType string

const (
	AppointmentConsultation AppointmentType = "consultation"
	AppointmentSession      AppointmentType = "session"
)

// Scheduling-mode and appointment string values, kept verbatim with the DB
// CHECK constraints (open_books.scheduling_mode, appointments.*).
const (
	schedulingArtist = "artist_scheduled"
	schedulingClient = "client_scheduled"

	appointmentProposed        = "proposed"
	appointmentAwaitingDeposit = "awaiting_deposit"
	appointmentScheduled       = "scheduled"

	originArtistSet    = "artist_set"
	originClientBooked = "client_booked"

	// Defaults applied when a consultation is requested without an explicit
	// length or format (consultations are configured per request, not in settings).
	defaultConsultationDurationMinutes int32 = 30
	defaultConsultationFormat                = "in_person"
)

type Inquiry struct {
	ID                     string           `json:"id"`
	Type                   RequestType      `json:"type"`
	FlashID                *string          `json:"flashId,omitempty"`
	Description            string           `json:"description"`
	ReferenceImageKeys     []string         `json:"referenceImageKeys"`
	ReferenceImageURLs     []string         `json:"referenceImageUrls"`
	Placement              string           `json:"placement"`
	ApproxSizeInches       *int32           `json:"approxSizeInches,omitempty"`
	ColorType              string           `json:"colorType"`
	LocationID             string           `json:"locationId,omitempty"`
	Location               *InquiryLocation `json:"location"`
	Flash                  *InquiryFlash    `json:"flash,omitempty"`
	Styles                 []string         `json:"styles"`
	ClientAvailability     json.RawMessage  `json:"clientAvailability"`
	CustomAnswers          json.RawMessage  `json:"customAnswers"`
	ClientName             string           `json:"clientName"`
	ClientEmail            string           `json:"clientEmail"`
	ClientPhone            string           `json:"clientPhone,omitempty"`
	Status                 string           `json:"status"`
	DepositStatus          string           `json:"depositStatus"`
	DepositAmountCents     *int64           `json:"depositAmountCents,omitempty"`
	WaiverStatus           string           `json:"waiverStatus"`
	SessionDurationMinutes *int32           `json:"sessionDurationMinutes,omitempty"`
	CreatedAt              string           `json:"createdAt"`
	DecidedAt              *string          `json:"decidedAt,omitempty"`

	// Scheduling context the inbox needs to drive the accept / consultation
	// actions: how this artist schedules, their weekly hours (so the time picker
	// can surface "your hours" first), and the appointment produced once the
	// request is accepted or a consult is requested.
	SchedulingMode     string               `json:"schedulingMode"`
	ArtistAvailability []AvailabilityWindow `json:"artistAvailability"`
	Appointment        *Appointment         `json:"appointment,omitempty"`
	LiveAppointments   []Appointment        `json:"liveAppointments"`

	Payments []InquiryPayment `json:"payments"`
}

type InquiryPayment struct {
	ID                string  `json:"id"`
	Type              string  `json:"type"`
	Status            string  `json:"status"`
	Currency          string  `json:"currency"`
	AmountCents       int64   `json:"amountCents"`
	ClientChargeCents int64   `json:"clientChargeCents"`
	PublicToken       string  `json:"publicToken"`
	ScheduledStart    *string `json:"scheduledStart,omitempty"`
	CreatedAt         string  `json:"createdAt"`
	PaidAt            *string `json:"paidAt,omitempty"`
}

// AvailabilityWindow is one of the artist's recurring weekly working spans,
// surfaced so the scheduling UI can group "your hours" ahead of off-hours.
type AvailabilityWindow struct {
	Weekday     int32 `json:"weekday"`
	StartMinute int32 `json:"startMinute"`
	EndMinute   int32 `json:"endMinute"`
}

// Appointment is the scheduled outcome of an inquiry — a consultation or the
// session itself, placed by the artist or awaiting the client's self-booking.
type Appointment struct {
	ID               string  `json:"id"`
	Type             string  `json:"type"`
	Status           string  `json:"status"`
	ScheduledStart   *string `json:"scheduledStart,omitempty"`
	DurationMinutes  int32   `json:"durationMinutes"`
	Format           *string `json:"format,omitempty"`
	SchedulingOrigin string  `json:"schedulingOrigin"`
}

type CalendarEvent struct {
	ID               string  `json:"id"`
	BookingRequestID string  `json:"bookingRequestId"`
	Type             string  `json:"type"`
	Status           string  `json:"status"`
	ClientName       string  `json:"clientName"`
	LocationLabel    string  `json:"locationLabel"`
	LocationAddress  string  `json:"locationAddress"`
	ScheduledStart   string  `json:"scheduledStart"`
	DurationMinutes  int32   `json:"durationMinutes"`
	Format           *string `json:"format,omitempty"`
}

// InquiryFlash is the claimed flash's display info, shown on a flash inquiry.
type InquiryFlash struct {
	Title     string   `json:"title"`
	ImageURLs []string `json:"imageUrls"`
	SizeCode  string   `json:"sizeCode,omitempty"`
}

type InquiryLocation struct {
	Label     string  `json:"label"`
	Address   string  `json:"address"`
	City      string  `json:"city"`
	Country   string  `json:"country"`
	IsPrimary bool    `json:"isPrimary"`
	StartDate *string `json:"startDate,omitempty"`
	EndDate   *string `json:"endDate,omitempty"`
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

type ClientInquiry struct {
	Inquiry
	ArtistName string `json:"artistName"`
	ArtistSlug string `json:"artistSlug"`
}

type ClientInquiryListResponse struct {
	Inquiries []ClientInquiry `json:"inquiries"`
}

// ── Request payloads ─────────────────────────────────────────────────────────

type ScheduleBookingInput struct {
	ScheduledStart time.Time `json:"scheduledStart"`
}

type SlotOption struct {
	Start string `json:"start"` // RFC3339 (UTC) absolute start
	Label string `json:"label"` // e.g. "2:00 PM", in the artist's timezone
}

type SlotList struct {
	Slots           []SlotOption `json:"slots"`
	DurationMinutes int32        `json:"durationMinutes"`
}

type PublicBookingRequest struct {
	ArtistName      string `json:"artistName"`
	ClientEmail     string `json:"clientEmail"`
	ClientName      string `json:"clientName"`
	Status          string `json:"status"`
	DurationMinutes int32  `json:"durationMinutes"`
	HasAccount      bool   `json:"hasAccount"`
}

type CreateClientAccountInput struct {
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	Phone          string `json:"phone"`
	Password       string `json:"password"`
	MarketingOptIn bool   `json:"marketingOptIn"`
}

type AcceptInput struct {
	SessionDurationMinutes *int32     `json:"sessionDurationMinutes"`
	ScheduledStart         *time.Time `json:"scheduledStart"`
	ClientScheduled        *bool      `json:"clientScheduled"`
	// Deposit for this session. nil falls back to the artist's default; an
	// explicit 0 means "no deposit". Ignored for consultations.
	DepositAmountCents *int64 `json:"depositAmountCents"`
}

type RequestConsultationInput struct {
	ScheduledStart  *time.Time `json:"scheduledStart"`
	DurationMinutes *int32     `json:"durationMinutes"`
	Format          *string    `json:"format"`
}

type RescheduleInput struct {
	ScheduledStart  *time.Time `json:"scheduledStart"`
	DurationMinutes *int32     `json:"durationMinutes"`
	Format          *string    `json:"format"`

	AppointmentID *string `json:"appointmentId"`
}

type CreateAppointmentInput struct {
	Type            string     `json:"type"`
	Format          *string    `json:"format"`
	ClientName      string     `json:"clientName"`
	ClientEmail     string     `json:"clientEmail"`
	ClientPhone     string     `json:"clientPhone"`
	ScheduledStart  *time.Time `json:"scheduledStart"`
	DurationMinutes int32      `json:"durationMinutes"`
	LocationID      *string    `json:"locationId"`

	Placement        string `json:"placement"`
	ApproxSizeInches *int32 `json:"approxSizeInches"`
	ColorType        string `json:"colorType"`
	Description      string `json:"description"`

	// Deposit for a session (ignored for consultations). nil → artist default,
	// explicit 0 → no deposit.
	DepositAmountCents *int64 `json:"depositAmountCents"`
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
		ColorType:              row.ColorType,
		Styles:                 row.Styles,
		ClientAvailability:     normalizeJSON(row.ClientAvailability),
		CustomAnswers:          normalizeJSON(row.CustomAnswers),
		ClientName:             row.ClientName,
		ClientEmail:            row.ClientEmail,
		Status:                 row.Status,
		DepositStatus:          row.DepositStatus,
		DepositAmountCents:     row.DepositAmountCents,
		WaiverStatus:           row.WaiverStatus,
		SessionDurationMinutes: row.SessionDurationMinutes,
		CreatedAt:              formatTimestamp(row.CreatedAt),
	}
	if out.ReferenceImageKeys == nil {
		out.ReferenceImageKeys = []string{}
	}
	out.ReferenceImageURLs = []string{}
	out.ArtistAvailability = []AvailabilityWindow{}
	out.LiveAppointments = []Appointment{}
	out.Payments = []InquiryPayment{}
	if out.Styles == nil {
		out.Styles = []string{}
	}
	out.ClientPhone = row.ClientPhone
	if row.FlashID.Valid {
		s := uuid.UUID(row.FlashID.Bytes).String()
		out.FlashID = &s
	}
	if row.LocationID.Valid {
		out.LocationID = uuid.UUID(row.LocationID.Bytes).String()
	}
	if row.DecidedAt.Valid {
		s := row.DecidedAt.Time.UTC().Format(time.RFC3339)
		out.DecidedAt = &s
	}
	return out
}

func inquiryLocationFromRow(l sqlc.ArtistLocation) *InquiryLocation {
	out := &InquiryLocation{
		Label:     l.Label,
		Address:   l.Address,
		City:      l.City,
		Country:   l.Country,
		IsPrimary: l.IsPrimary,
	}
	if l.StartDate.Valid {
		s := l.StartDate.Time.Format("2006-01-02")
		out.StartDate = &s
	}
	if l.EndDate.Valid {
		s := l.EndDate.Time.Format("2006-01-02")
		out.EndDate = &s
	}
	return out
}

func availabilityWindowsFromRows(rows []sqlc.ArtistAvailabilityWindow) []AvailabilityWindow {
	out := make([]AvailabilityWindow, 0, len(rows))
	for _, r := range rows {
		out = append(out, AvailabilityWindow{
			Weekday:     r.Weekday,
			StartMinute: r.StartMinute,
			EndMinute:   r.EndMinute,
		})
	}
	return out
}

func appointmentFromRow(a sqlc.Appointment) *Appointment {
	out := &Appointment{
		ID:               a.ID.String(),
		Type:             a.Type,
		Status:           a.Status,
		DurationMinutes:  a.DurationMinutes,
		Format:           a.Format,
		SchedulingOrigin: a.SchedulingOrigin,
	}
	if a.ScheduledStart.Valid {
		s := a.ScheduledStart.Time.UTC().Format(time.RFC3339)
		out.ScheduledStart = &s
	}
	return out
}

func paymentFromRow(p sqlc.PaymentRequest) InquiryPayment {
	out := InquiryPayment{
		ID:                p.ID.String(),
		Type:              p.Type,
		Status:            p.Status,
		Currency:          p.Currency,
		AmountCents:       p.AmountCents,
		ClientChargeCents: p.ClientChargeCents,
		PublicToken:       p.PublicToken,
		CreatedAt:         formatTimestamp(p.CreatedAt),
	}
	if p.ScheduledStart.Valid {
		s := p.ScheduledStart.Time.UTC().Format(time.RFC3339)
		out.ScheduledStart = &s
	}
	if p.PaidAt.Valid {
		s := p.PaidAt.Time.UTC().Format(time.RFC3339)
		out.PaidAt = &s
	}
	return out
}

func calendarEventFromRow(r sqlc.ListAppointmentsByArtistInRangeRow) CalendarEvent {
	return CalendarEvent{
		ID:               r.ID.String(),
		BookingRequestID: r.BookingRequestID.String(),
		Type:             r.Type,
		Status:           r.Status,
		ClientName:       r.ClientName,
		LocationLabel:    r.LocationLabel,
		LocationAddress:  joinNonEmpty(", ", r.LocationAddress, r.LocationCity),
		ScheduledStart:   r.ScheduledStart.Time.UTC().Format(time.RFC3339),
		DurationMinutes:  r.DurationMinutes,
		Format:           r.Format,
	}
}

// joinNonEmpty joins the non-blank parts with sep, e.g. a street and city into
// a single address line.
func joinNonEmpty(sep string, parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
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
