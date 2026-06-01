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

type CreatePortfolioItemInput struct {
	ImageURL  string   `json:"image_url" binding:"required,url"`
	Caption   string   `json:"caption"`
	Styles    []string `json:"styles"`
	Placement string   `json:"placement"`
}
