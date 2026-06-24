package messaging

import (
	"time"

	"github.com/google/uuid"
)

type ConversationSummary struct {
	ID               uuid.UUID  `json:"id"`
	BookingRequestID uuid.UUID  `json:"bookingRequestId"`
	ClientName       string     `json:"clientName"`
	ClientEmail      string     `json:"clientEmail"`
	ArtistName       string     `json:"artistName"`
	Status           string     `json:"status"`
	LastMessageBody  string     `json:"lastMessageBody"`
	LastMessageAt    *time.Time `json:"lastMessageAt"`
	HasUnread        bool       `json:"hasUnread"`
}

type MessageView struct {
	ID         uuid.UUID `json:"id"`
	SenderType string    `json:"senderType"`
	Body       string    `json:"body"`
	AIDrafted  bool      `json:"aiDrafted"`
	CreatedAt  time.Time `json:"createdAt"`
}

type ConversationThread struct {
	ID               uuid.UUID     `json:"id"`
	BookingRequestID uuid.UUID     `json:"bookingRequestId"`
	ClientName       string        `json:"clientName"`
	ClientEmail      string        `json:"clientEmail"`
	ClientPhone      string        `json:"clientPhone"`
	ArtistName       string        `json:"artistName"`
	Status           string        `json:"status"`
	Messages         []MessageView `json:"messages"`
}

type SendMessageInput struct {
	Body string `json:"body"`
}
