package bookings

import "time"

type Status string

const (
	StatusInquiry      Status = "inquiry"
	StatusConsultation Status = "consultation"
	StatusDepositHeld  Status = "deposit_held"
	StatusScheduled    Status = "scheduled"
	StatusCompleted    Status = "completed"
	StatusCancelled    Status = "cancelled"
)

type Booking struct {
	ID             string     `json:"id"`
	ClientID       string     `json:"client_id"`
	ArtistID       string     `json:"artist_id"`
	Status         Status     `json:"status"`
	Notes          string     `json:"notes"`
	EstimatedHours float64    `json:"estimated_hours"`
	DepositAmount  int64      `json:"deposit_amount_cents"`
	ScheduledFor   *time.Time `json:"scheduled_for"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type CreateBookingInput struct {
	ArtistID       string  `json:"artist_id" binding:"required"`
	Notes          string  `json:"notes"`
	EstimatedHours float64 `json:"estimated_hours"`
}

type RescheduleInput struct {
	ScheduledFor time.Time `json:"scheduled_for" binding:"required"`
}
