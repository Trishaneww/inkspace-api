package portfolios

import "time"

type PortfolioItem struct {
	ID          string    `json:"id"`
	ArtistID    string    `json:"artist_id"`
	ImageURL    string    `json:"image_url"`
	Caption     string    `json:"caption"`
	Styles      []string  `json:"styles"`
	Placement   string    `json:"placement"`
	HealedPhoto bool      `json:"healed_photo"`
	CreatedAt   time.Time `json:"created_at"`
}

type FlashStatus string

const (
	FlashStatusAvailable FlashStatus = "available"
	FlashStatusReserved  FlashStatus = "reserved"
	FlashStatusClaimed   FlashStatus = "claimed"
	FlashStatusRetired   FlashStatus = "retired"
)

type FlashDesign struct {
	ID            string      `json:"id"`
	ArtistID      string      `json:"artist_id"`
	Title         string      `json:"title"`
	Description   string      `json:"description"`
	ImageURL      string      `json:"image_url"`
	Styles        []string    `json:"styles"`
	Placement     string      `json:"placement"`
	SizeInches    float64     `json:"size_inches"`
	PriceCents    int64       `json:"price_cents"`
	Currency      string      `json:"currency"`
	Repeatable    bool        `json:"repeatable"`
	Status        FlashStatus `json:"status"`
	ReservedBy    *string     `json:"reserved_by,omitempty"`
	ReservedUntil *time.Time  `json:"reserved_until,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
}

type CreatePortfolioItemInput struct {
	ImageURL  string   `json:"image_url" binding:"required,url"`
	Caption   string   `json:"caption"`
	Styles    []string `json:"styles"`
	Placement string   `json:"placement"`
}

type CreateFlashInput struct {
	Title       string   `json:"title" binding:"required"`
	Description string   `json:"description"`
	ImageURL    string   `json:"image_url" binding:"required,url"`
	Styles      []string `json:"styles"`
	Placement   string   `json:"placement"`
	SizeInches  float64  `json:"size_inches"`
	PriceCents  int64    `json:"price_cents" binding:"required,gt=0"`
	Currency    string   `json:"currency" binding:"required,len=3"`
	Repeatable  bool     `json:"repeatable"`
}
