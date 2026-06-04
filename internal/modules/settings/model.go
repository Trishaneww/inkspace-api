package settings

import "time"

// ── Enums ────────────────────────────────────────────────────────────────
type PayoutFrequency string

const (
	PayoutWeekly   PayoutFrequency = "weekly"
	PayoutBiweekly PayoutFrequency = "biweekly"
	PayoutMonthly  PayoutFrequency = "monthly"
)

type PlatformFeePayer string

const (
	FeePayerArtist PlatformFeePayer = "artist"
	FeePayerClient PlatformFeePayer = "client"
	FeePayerSplit  PlatformFeePayer = "split"
)

// ── Response shapes ──────────────────────────────────────────────────────
type Account struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	Username     string `json:"username"`
	Phone        string `json:"phone"`
	AvatarURL    string `json:"avatarUrl"`
	InstagramURL string `json:"instagramUrl"`
}

type ArtistSettings struct {
	StudioName       string `json:"studioName"`
	StudioAddress    string `json:"studioAddress"`
	StudioCity       string `json:"studioCity"`
	StudioProvince   string `json:"studioProvince"`
	StudioPostalCode string `json:"studioPostalCode"`
	StudioCountry    string `json:"studioCountry"`

	StripeConnected bool   `json:"stripeConnected"`
	PayoutFrequency string `json:"payoutFrequency"`
	Currency        string `json:"currency"`

	DepositFlatFeeCents *int64 `json:"depositFlatFeeCents"`
	PlatformFeePayer    string `json:"platformFeePayer"`

	AcceptingBookings       bool   `json:"acceptingBookings"`
	Timezone                string `json:"timezone"`
	GoogleCalendarConnected bool   `json:"googleCalendarConnected"`
	GoogleCalendarEmail     string `json:"googleCalendarEmail"`
	SlotIntervalMinutes     int32  `json:"slotIntervalMinutes"`
	BufferMinutes           int32  `json:"bufferMinutes"`
	MinNoticeMinutes        int32  `json:"minNoticeMinutes"`
	MaxAdvanceDays          *int32 `json:"maxAdvanceDays"`

	TermsText          string `json:"termsText"`
	TermsShowOnBooking bool   `json:"termsShowOnBooking"`
	TermsShowAtDeposit bool   `json:"termsShowAtDeposit"`
	WaiverFileURL      string `json:"waiverFileUrl"`
	WaiverRequired     bool   `json:"waiverRequired"`

	NotifyByEmail bool `json:"notifyByEmail"`
	NotifyBySms   bool `json:"notifyBySms"`
}

type AvailabilityWindow struct {
	ID          string `json:"id"`
	Weekday     int32  `json:"weekday"` // 0 = Sunday .. 6 = Saturday
	StartMinute int32  `json:"startMinute"`
	EndMinute   int32  `json:"endMinute"`
}

type SessionPreset struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	Description           string `json:"description"`
	ApproxDurationMinutes int32  `json:"approxDurationMinutes"`
	Position              int32  `json:"position"`
}

type BlocklistEntry struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Phone string `json:"phone"`
	Note  string `json:"note"`
}

type SettingsResponse struct {
	Account        Account              `json:"account"`
	Settings       ArtistSettings       `json:"settings"`
	Availability   []AvailabilityWindow `json:"availability"`
	SessionPresets []SessionPreset      `json:"sessionPresets"`
	DaysOff        []string             `json:"daysOff"` // YYYY-MM-DD
	Blocklist      []BlocklistEntry     `json:"blocklist"`
}

type PresignUploadResponse struct {
	URL       string    `json:"url"`
	S3Key     string    `json:"s3Key"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// ── Request payloads ─────────────────────────────────────────────────────
type UpdateProfileInput struct {
	FirstName    *string `json:"firstName"`
	LastName     *string `json:"lastName"`
	Username     *string `json:"username"`
	Phone        *string `json:"phone"`
	AvatarURL    *string `json:"avatarUrl"`
	InstagramURL *string `json:"instagramUrl"`
}

type ChangeEmailInput struct {
	NewEmail string `json:"newEmail" binding:"required,email"`
}

type ChangePasswordInput struct {
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

type PresignUploadInput struct {
	ContentType string `json:"contentType" binding:"required"`
}

type UpdateSettingsInput struct {
	StudioName       *string `json:"studioName"`
	StudioAddress    *string `json:"studioAddress"`
	StudioCity       *string `json:"studioCity"`
	StudioProvince   *string `json:"studioProvince"`
	StudioPostalCode *string `json:"studioPostalCode"`
	StudioCountry    *string `json:"studioCountry"`

	PayoutFrequency     *string `json:"payoutFrequency"`
	Currency            *string `json:"currency"`
	PlatformFeePayer    *string `json:"platformFeePayer"`
	AcceptingBookings   *bool   `json:"acceptingBookings"`
	Timezone            *string `json:"timezone"`
	SlotIntervalMinutes *int32  `json:"slotIntervalMinutes"`
	BufferMinutes       *int32  `json:"bufferMinutes"`
	MinNoticeMinutes    *int32  `json:"minNoticeMinutes"`
	TermsText           *string `json:"termsText"`
	TermsShowOnBooking  *bool   `json:"termsShowOnBooking"`
	TermsShowAtDeposit  *bool   `json:"termsShowAtDeposit"`
	WaiverRequired      *bool   `json:"waiverRequired"`
	NotifyByEmail       *bool   `json:"notifyByEmail"`
	NotifyBySms         *bool   `json:"notifyBySms"`

	// Nullable fields: send the value to set it, or set the paired clear flag
	// to unset it.
	DepositFlatFeeCents *int64 `json:"depositFlatFeeCents"`
	ClearDepositFlatFee bool   `json:"clearDepositFlatFee"`
	MaxAdvanceDays      *int32 `json:"maxAdvanceDays"`
	ClearMaxAdvance     bool   `json:"clearMaxAdvance"`
	WaiverFileURL       *string `json:"waiverFileUrl"`
	ClearWaiverFile     bool    `json:"clearWaiverFile"`
}

type AvailabilityWindowInput struct {
	Weekday     int32 `json:"weekday"     binding:"min=0,max=6"`
	StartMinute int32 `json:"startMinute" binding:"min=0,max=1439"`
	EndMinute   int32 `json:"endMinute"   binding:"min=1,max=1440"`
}

type SetAvailabilityInput struct {
	Windows []AvailabilityWindowInput `json:"windows"`
}

type CreatePresetInput struct {
	Name                  string `json:"name"                  binding:"required"`
	Description           string `json:"description"`
	ApproxDurationMinutes int32  `json:"approxDurationMinutes" binding:"required,min=1"`
	Position              int32  `json:"position"`
}

type UpdatePresetInput struct {
	Name                  *string `json:"name"`
	Description           *string `json:"description"`
	ApproxDurationMinutes *int32  `json:"approxDurationMinutes"`
	Position              *int32  `json:"position"`
}

type DayOffInput struct {
	Day string `json:"day" binding:"required"` // YYYY-MM-DD
}

type CreateBlocklistInput struct {
	Email *string `json:"email"`
	Phone *string `json:"phone"`
	Note  string  `json:"note"`
}
