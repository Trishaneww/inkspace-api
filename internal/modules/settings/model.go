package settings

import "time"

// ── Enums ────────────────────────────────────────────────────────────────
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
	CurrentLocationID string `json:"currentLocationId"`

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
	MonthlyBookingGoal      int32  `json:"monthlyBookingGoal"`
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

	Aftercare string    `json:"aftercare"`
	FAQs      []FAQItem `json:"faqs"`

	NotifyByEmail bool `json:"notifyByEmail"`
	NotifyBySMS   bool `json:"notifyBySms"`

	Styles []string `json:"styles"`

	MinSessionPriceCents *int64   `json:"minSessionPriceCents"`
	DeclinedPlacements   []string `json:"declinedPlacements"`
	DeclinedStyles       []string `json:"declinedStyles"`
	WorkSummary          string   `json:"workSummary"`
}

type FAQItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type AvailabilityWindow struct {
	ID          string `json:"id"`
	Weekday     int32  `json:"weekday"`
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

type Location struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	Address    string  `json:"address"`
	City       string  `json:"city"`
	Province   string  `json:"province"`
	PostalCode string  `json:"postalCode"`
	Country    string  `json:"country"`
	Timezone   string  `json:"timezone"`
	IsPrimary  bool    `json:"isPrimary"`
	StartDate  *string `json:"startDate"` // YYYY-MM-DD; null for the home studio
	EndDate    *string `json:"endDate"`
}

type CreateLocationInput struct {
	Label      string  `json:"label"`
	Address    string  `json:"address"`
	City       string  `json:"city"`
	Province   string  `json:"province"`
	PostalCode string  `json:"postalCode"`
	Country    string  `json:"country"`
	Timezone   string  `json:"timezone"`
	StartDate  *string `json:"startDate"`
	EndDate    *string `json:"endDate"`
}

type UpdateLocationInput struct {
	Label      *string `json:"label"`
	Address    *string `json:"address"`
	City       *string `json:"city"`
	Province   *string `json:"province"`
	PostalCode *string `json:"postalCode"`
	Country    *string `json:"country"`
	Timezone   *string `json:"timezone"`
	StartDate  *string `json:"startDate"`
	EndDate    *string `json:"endDate"`
	ClearDates bool    `json:"clearDates"`
}

type SetCurrentLocationInput struct {
	LocationID string `json:"locationId"`
}

type SettingsResponse struct {
	Account        Account              `json:"account"`
	Locations      []Location           `json:"locations"`
	Settings       ArtistSettings       `json:"settings"`
	Availability   []AvailabilityWindow `json:"availability"`
	SessionPresets []SessionPreset      `json:"sessionPresets"`
	DaysOff        []string             `json:"daysOff"`
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
	PayoutFrequency     *string    `json:"payoutFrequency"`
	Currency            *string    `json:"currency"`
	PlatformFeePayer    *string    `json:"platformFeePayer"`
	DepositRefundPolicy *string    `json:"depositRefundPolicy"`
	AcceptingBookings   *bool      `json:"acceptingBookings"`
	MonthlyBookingGoal  *int32     `json:"monthlyBookingGoal"`
	Timezone            *string    `json:"timezone"`
	SlotIntervalMinutes *int32     `json:"slotIntervalMinutes"`
	BufferMinutes       *int32     `json:"bufferMinutes"`
	MinNoticeMinutes    *int32     `json:"minNoticeMinutes"`
	TermsText           *string    `json:"termsText"`
	TermsShowOnBooking  *bool      `json:"termsShowOnBooking"`
	TermsShowAtDeposit  *bool      `json:"termsShowAtDeposit"`
	WaiverRequired      *bool      `json:"waiverRequired"`
	Aftercare           *string    `json:"aftercare"`
	FAQs                *[]FAQItem `json:"faqs"`

	NotifyByEmail *bool `json:"notifyByEmail"`
	NotifyBySMS   *bool `json:"notifyBySms"`

	DepositFlatFeeCents *int64  `json:"depositFlatFeeCents"`
	ClearDepositFlatFee bool    `json:"clearDepositFlatFee"`
	MaxAdvanceDays      *int32  `json:"maxAdvanceDays"`
	ClearMaxAdvanceDays bool    `json:"clearMaxAdvance"`
	WaiverFileURL       *string `json:"waiverFileUrl"`
	ClearWaiverFile     bool    `json:"clearWaiverFile"`

	CancellationNoticeHours      *int32 `json:"cancellationNoticeHours"`
	ClearCancellationNoticeHours bool   `json:"clearCancellationNotice"`

	Styles *[]string `json:"styles"`

	MinSessionPriceCents *int64    `json:"minSessionPriceCents"`
	ClearMinSessionPrice bool      `json:"clearMinSessionPrice"`
	DeclinedPlacements   *[]string `json:"declinedPlacements"`
	DeclinedStyles       *[]string `json:"declinedStyles"`
	WorkSummary          *string   `json:"workSummary"`
}

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
	Day string `json:"day" binding:"required"`
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
	MonthlyBookingGoal  *int32 `json:"monthlyBookingGoal"`

	Styles []string `json:"styles"`

	MinSessionPriceCents *int64   `json:"minSessionPriceCents"`
	DeclinedPlacements   []string `json:"declinedPlacements"`
	DeclinedStyles       []string `json:"declinedStyles"`
	WorkSummary          string   `json:"workSummary"`
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

// ── Open Book ────────────────────────────────────────────────────────────────
const ThemeCustom = "custom"

var OpenBookThemes = map[string]bool{
	"inkspace":  true,
	"noir":      true,
	"sand":      true,
	"sage":      true,
	"midnight":  true,
	"navy":      true,
	ThemeCustom: true,
}

type CustomTheme struct {
	Background string `json:"background"`
	Card       string `json:"card"`
	Button     string `json:"button"`
	Text       string `json:"text"`
}

type OpenBookResponse struct {
	Slug               string       `json:"slug"`
	SchedulingMode     string       `json:"schedulingMode"`
	CustomQuestions    []string     `json:"customQuestions"`
	Theme              string       `json:"theme"`
	CustomTheme        *CustomTheme `json:"customTheme,omitempty"`
	BackgroundImageURL string       `json:"backgroundImageUrl,omitempty"`
}

type UpdateOpenBookInput struct {
	Slug                 *string      `json:"slug"`
	SchedulingMode       *string      `json:"schedulingMode"`
	CustomQuestions      *[]string    `json:"customQuestions"`
	Theme                *string      `json:"theme"`
	CustomTheme          *CustomTheme `json:"customTheme"`
	BackgroundImageKey   *string      `json:"backgroundImageKey"`
	ClearBackgroundImage bool         `json:"clearBackgroundImage"`
}
