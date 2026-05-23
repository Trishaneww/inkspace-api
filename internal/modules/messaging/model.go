package messaging

import "time"

type Conversation struct {
	ID            string    `json:"id"`
	ClientID      string    `json:"client_id"`
	ArtistID      string    `json:"artist_id"`
	BookingID     *string   `json:"booking_id,omitempty"`
	LastMessageAt time.Time `json:"last_message_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	Body           string    `json:"body"`
	Attachments    []string  `json:"attachments,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateConversationInput struct {
	ArtistID  string  `json:"artist_id" binding:"required"`
	BookingID *string `json:"booking_id"`
}

type SendMessageInput struct {
	Body        string   `json:"body" binding:"required"`
	Attachments []string `json:"attachments"`
}
