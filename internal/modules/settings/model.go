package settings

import "time"

// ── Enums ────────────────────────────────────────────────────────────────
// PayoutFrequency mirrors Stripe's supported automatic-payout intervals.
// Stripe has no native biweekly interval, so only weekly/monthly are offered.
type PayoutFrequency string

const (
	PayoutWeekly  PayoutFrequency = "weekly"
	PayoutMonthly PayoutFrequency = "monthly"
)

type PlatformFeePayer string

const (
	FeePayerArtist PlatformFeePayer = "artist"
	FeePayerClient PlatformFeePayer = "client"
	FeePayerSplit  PlatformFeePayer = "split"
)

// DepositRefundPolicy governs whether a client is eligible for a deposit refund.
// The chosen policy is snapshotted onto each payment request at pay time.
type DepositRefundPolicy string

const (
	RefundNonRefundable DepositRefundPolicy = "non_refundable"
	RefundWithinWindow  DepositRefundPolicy = "refundable_within_window"
	RefundAlways        DepositRefundPolicy = "always_refundable"
)

type SchedulingMode string

const (
	SchedulingArtist SchedulingMode = "artist_scheduled"
	SchedulingClient SchedulingMode = "client_scheduled"
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

	// StripeConnected = an account exists. ChargesEnabled is the real gate for
	// "can accept deposits"; an account can exist but be mid-onboarding.
	StripeConnected        bool   `json:"stripeConnected"`
	StripeChargesEnabled   bool   `json:"stripeChargesEnabled"`
	StripePayoutsEnabled   bool   `json:"stripePayoutsEnabled"`
	StripeDetailsSubmitted bool   `json:"stripeDetailsSubmitted"`
	PayoutFrequency        string `json:"payoutFrequency"`
	Currency               string `json:"currency"`

	DepositFlatFeeCents     *int64 `json:"depositFlatFeeCents"`
	PlatformFeePayer        string `json:"platformFeePayer"`
	DepositRefundPolicy     string `json:"depositRefundPolicy"`
	CancellationNoticeHours *int32 `json:"cancellationNoticeHours"`

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
	DepositRefundPolicy *string `json:"depositRefundPolicy"`
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
	DepositFlatFeeCents *int64  `json:"depositFlatFeeCents"`
	ClearDepositFlatFee bool    `json:"clearDepositFlatFee"`
	MaxAdvanceDays      *int32  `json:"maxAdvanceDays"`
	ClearMaxAdvance     bool    `json:"clearMaxAdvance"`
	WaiverFileURL       *string `json:"waiverFileUrl"`
	ClearWaiverFile     bool    `json:"clearWaiverFile"`

	CancellationNoticeHours *int32 `json:"cancellationNoticeHours"`
	ClearCancellationNotice bool   `json:"clearCancellationNotice"`
}

// StripeConnectResponse carries the Stripe-hosted onboarding URL the frontend
// redirects the artist to. Same shape whether the account is newly created or
// the artist is resuming/re-opening onboarding.
type StripeConnectResponse struct {
	URL string `json:"url"`
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

type ConnectGoogleCalendarInput struct {
	Code        string `json:"code"        binding:"required"`
	RedirectURI string `json:"redirectUri" binding:"required"`
}

// ── Onboarding ─────────────────────────────────────────────────────────────
type OnboardingInput struct {
	Username     string `json:"username"     binding:"required,min=3,max=30"`
	InstagramURL string `json:"instagramUrl" binding:"omitempty,url,max=255"`

	StudioName       string `json:"studioName"       binding:"required"`
	StudioAddress    string `json:"studioAddress"    binding:"required"`
	StudioCity       string `json:"studioCity"       binding:"required"`
	StudioProvince   string `json:"studioProvince"   binding:"required"`
	StudioPostalCode string `json:"studioPostalCode" binding:"required"`
	StudioCountry    string `json:"studioCountry"    binding:"required"`
	Timezone         string `json:"timezone"         binding:"required"`

	Availability []AvailabilityWindowInput `json:"availability"`

	DepositFlatFeeCents *int64 `json:"depositFlatFeeCents"`
	SchedulingMode      string `json:"schedulingMode" binding:"required,oneof=artist_scheduled client_scheduled"`
}

type OnboardingResponse struct {
	OnboardedAt    string `json:"onboardedAt"`
	Slug           string `json:"slug"`
	SchedulingMode string `json:"schedulingMode"`
}

type UsernameAvailabilityResponse struct {
	Username  string `json:"username"`
	Available bool   `json:"available"`
}
