package flashes

import "time"

type FlashStatus string

const (
	FlashStatusDraft     FlashStatus = "draft"
	FlashStatusAvailable FlashStatus = "available"
	FlashStatusClaimed   FlashStatus = "claimed"
	FlashStatusArchived  FlashStatus = "archived"
)

type ColorType string

const (
	ColorBlackAndGrey ColorType = "black_and_grey"
	ColorColor        ColorType = "color"
	ColorBoth         ColorType = "both"
)

type PricingMode string

const (
	PricingPerSize PricingMode = "per_size"
	PricingFlat    PricingMode = "flat"
)

type SizeCode string

const (
	SizeXSmall SizeCode = "x_small"
	SizeSmall  SizeCode = "small"
	SizeMedium SizeCode = "medium"
	SizeLarge  SizeCode = "large"
	SizeXLarge SizeCode = "x_large"
)

type Flash struct {
	ID                  string        `json:"id"`
	ArtistID            string        `json:"artist_id"`
	Status              FlashStatus   `json:"status"`
	Title               string        `json:"title"`
	Description         *string       `json:"description,omitempty"`
	S3Key               *string       `json:"s3_key,omitempty"`
	ReferenceS3Key      *string       `json:"reference_s3_key,omitempty"`
	ImageURL            *string       `json:"image_url,omitempty"`           // presigned GET URL for s3_key
	ReferenceImageURL   *string       `json:"reference_image_url,omitempty"` // presigned GET URL for reference_s3_key
	ColorType           ColorType     `json:"color_type"`
	Styles              []string      `json:"styles"`
	Placements          []string      `json:"placements"`
	PricingMode         PricingMode   `json:"pricing_mode"`
	FlatPriceCents      *int64        `json:"flat_price_cents,omitempty"`
	FlatDurationMinutes *int32        `json:"flat_duration_minutes,omitempty"`
	DepositCents        *int64        `json:"deposit_cents,omitempty"`
	Currency            string        `json:"currency"`
	Repeatable          bool          `json:"repeatable"`
	ClaimedAt           *time.Time    `json:"claimed_at,omitempty"`
	ClaimedByBookingID  *string       `json:"claimed_by_booking_id,omitempty"`
	ArchivedAt          *time.Time    `json:"archived_at,omitempty"`
	PublishedAt         *time.Time    `json:"published_at,omitempty"`
	ViewCount           int64         `json:"view_count"`
	SaveCount           int64         `json:"save_count"`
	PricingTiers        []PricingTier `json:"pricing_tiers"`
	CreatedAt           time.Time     `json:"created_at"`
	UpdatedAt           time.Time     `json:"updated_at"`
}

type FlashListResult struct {
	Items     []Flash `json:"items"`
	Total     int64   `json:"total"`
	Available int64   `json:"available"`
	Limit     int32   `json:"limit"`
	Offset    int32   `json:"offset"`
}

type ListFilter struct {
	Status *FlashStatus
	Limit  int32
	Offset int32
}

type PricingTier struct {
	SizeCode        SizeCode `json:"size_code"`
	DurationMinutes int32    `json:"duration_minutes"`
	PriceCents      int64    `json:"price_cents"`
}

type PresignUploadInput struct {
	ContentType string `json:"content_type" binding:"required"`
}

type PresignUploadResponse struct {
	URL       string    `json:"url"`
	S3Key     string    `json:"s3_key"`
	ExpiresAt time.Time `json:"expires_at"`
}

type CreateFlashInput struct {
	Title               string        `json:"title" binding:"required"`
	Description         *string       `json:"description,omitempty"`
	S3Key               *string       `json:"s3_key,omitempty"`
	ReferenceS3Key      *string       `json:"reference_s3_key,omitempty"`
	ColorType           ColorType     `json:"color_type" binding:"required"`
	Styles              []string      `json:"styles"`
	Placements          []string      `json:"placements"`
	PricingMode         PricingMode   `json:"pricing_mode" binding:"required"`
	FlatPriceCents      *int64        `json:"flat_price_cents,omitempty"`
	FlatDurationMinutes *int32        `json:"flat_duration_minutes,omitempty"`
	DepositCents        *int64        `json:"deposit_cents,omitempty"`
	Currency            string        `json:"currency" binding:"required,len=3"`
	Repeatable          bool          `json:"repeatable"`
	PricingTiers        []PricingTier `json:"pricing_tiers,omitempty"`
	Publish             bool          `json:"publish"` // true = create as 'available', false = 'draft'
}

type UpdateFlashInput struct {
	Title          *string `json:"title,omitempty"`
	Description    *string `json:"description,omitempty"`
	S3Key          *string `json:"s3_key,omitempty"`
	ReferenceS3Key *string `json:"reference_s3_key,omitempty"`

	ClearReferenceImage bool           `json:"clear_reference_image,omitempty"`
	ColorType           *ColorType     `json:"color_type,omitempty"`
	Styles              *[]string      `json:"styles,omitempty"`
	Placements          *[]string      `json:"placements,omitempty"`
	PricingMode         *PricingMode   `json:"pricing_mode,omitempty"`
	FlatPriceCents      *int64         `json:"flat_price_cents,omitempty"`
	FlatDurationMinutes *int32         `json:"flat_duration_minutes,omitempty"`
	DepositCents        *int64         `json:"deposit_cents,omitempty"`
	Currency            *string        `json:"currency,omitempty"`
	Repeatable          *bool          `json:"repeatable,omitempty"`
	PricingTiers        *[]PricingTier `json:"pricing_tiers,omitempty"`
}
