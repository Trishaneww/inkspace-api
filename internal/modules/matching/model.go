package matching

import "time"

type RequestStatus string

const (
	RequestStatusOpen      RequestStatus = "open"
	RequestStatusMatched   RequestStatus = "matched"
	RequestStatusConverted RequestStatus = "converted"
	RequestStatusExpired   RequestStatus = "expired"
)

type LeadStatus string

const (
	LeadStatusPending    LeadStatus = "pending"
	LeadStatusInterested LeadStatus = "interested"
	LeadStatusPassed     LeadStatus = "passed"
)

type Request struct {
	ID          string        `json:"id"`
	ClientID    string        `json:"client_id"`
	Description string        `json:"description"`
	References  []string      `json:"references"` // image URLs
	Styles      []string      `json:"styles"`
	Placement   string        `json:"placement"`
	SizeInches  float64       `json:"size_inches"`
	BudgetMin   int64         `json:"budget_min_cents"`
	BudgetMax   int64         `json:"budget_max_cents"`
	City        string        `json:"city"`
	Country     string        `json:"country"`
	Status      RequestStatus `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
}

type Match struct {
	ID        string     `json:"id"`
	RequestID string     `json:"request_id"`
	ArtistID  string     `json:"artist_id"`
	Score     float64    `json:"score"`
	Status    LeadStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreateRequestInput struct {
	Description string   `json:"description" binding:"required"`
	References  []string `json:"references"`
	Styles      []string `json:"styles"`
	Placement   string   `json:"placement"`
	SizeInches  float64  `json:"size_inches"`
	BudgetMin   int64    `json:"budget_min_cents"`
	BudgetMax   int64    `json:"budget_max_cents"`
	City        string   `json:"city"`
	Country     string   `json:"country"`
}
