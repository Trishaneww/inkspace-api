package artists

import "time"

type Artist struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Handle      string    `json:"handle"`
	DisplayName string    `json:"display_name"`
	Bio         string    `json:"bio"`
	Styles      []string  `json:"styles"`
	City        string    `json:"city"`
	Country     string    `json:"country"`
	BookingLink string    `json:"booking_link"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Availability struct {
	ArtistID      string   `json:"artist_id"`
	AcceptingNew  bool     `json:"accepting_new"`
	WeeklyWindows []Window `json:"weekly_windows"`
	WaitlistDays  int      `json:"waitlist_days"`
}

type Window struct {
	Weekday   int    `json:"weekday"` // 0=Sun..6=Sat
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type CreateArtistInput struct {
	Handle      string   `json:"handle" binding:"required"`
	DisplayName string   `json:"display_name" binding:"required"`
	Bio         string   `json:"bio"`
	Styles      []string `json:"styles"`
	City        string   `json:"city"`
	Country     string   `json:"country"`
}
