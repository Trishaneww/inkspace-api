package notifications

import "time"

type Channel string

const (
	ChannelInApp Channel = "in_app"
	ChannelEmail Channel = "email"
	ChannelPush  Channel = "push"
)

type Kind string

const (
	KindNewLead         Kind = "new_lead"
	KindArtistInterest  Kind = "artist_interest"
	KindBookingUpdate   Kind = "booking_update"
	KindMessageReceived Kind = "message_received"
	KindDepositReceived Kind = "deposit_received"
)

type Notification struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Kind      Kind           `json:"kind"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Data      map[string]any `json:"data,omitempty"`
	ReadAt    *time.Time     `json:"read_at,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type Preferences struct {
	UserID   string              `json:"user_id"`
	Enabled  map[Kind][]Channel  `json:"enabled"`
	QuietHrs *QuietHours         `json:"quiet_hours,omitempty"`
}

type QuietHours struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Timezone  string `json:"timezone"`
}
