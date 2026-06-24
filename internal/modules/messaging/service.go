package messaging

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
	"github.com/trishaneupnexx/inkspace-api/internal/events"
)

const maxMessageLength = 5000

const (
	senderArtist = "artist"
	senderClient = "client"
)

const MessageCreatedRoutingKey = "messaging.message.created"

var (
	ErrNotFound     = errors.New("conversation not found")
	ErrEmptyMessage = errors.New("message cannot be empty")
	ErrTooLong      = errors.New("message is too long")
)

type MessageCreatedEvent struct {
	ConversationID string `json:"conversationId"`
	MessageID      string `json:"messageId"`
	SenderType     string `json:"senderType"`
}

type Service interface {
	ListArtistConversations(ctx context.Context, userID uuid.UUID) ([]ConversationSummary, error)
	GetArtistConversation(ctx context.Context, userID, convID uuid.UUID) (ConversationThread, error)
	SendArtistMessage(ctx context.Context, userID, convID uuid.UUID, input SendMessageInput) (MessageView, error)
	MarkArtistRead(ctx context.Context, userID, convID uuid.UUID) error

	ListClientConversations(ctx context.Context, userID uuid.UUID) ([]ConversationSummary, error)
	GetClientConversation(ctx context.Context, userID, convID uuid.UUID) (ConversationThread, error)
	SendClientMessage(ctx context.Context, userID, convID uuid.UUID, input SendMessageInput) (MessageView, error)
	MarkClientRead(ctx context.Context, userID, convID uuid.UUID) error

	GetGuestConversation(ctx context.Context, token string) (ConversationThread, error)
	SendGuestMessage(ctx context.Context, token string, input SendMessageInput) (MessageView, error)
	MarkGuestRead(ctx context.Context, token string) error

	CreateConversationForBooking(ctx context.Context, artistID, bookingID uuid.UUID, clientName, clientEmail string) (sqlc.Conversation, error)
}

type service struct {
	repo Repository
	pub  *events.Publisher
	log  *slog.Logger
}

func NewService(repo Repository, pub *events.Publisher) Service {
	return &service{repo: repo, pub: pub, log: slog.Default()}
}

// ── Artist ──────────────────────────────────────────────────────────────────
func (s *service) ListArtistConversations(ctx context.Context, userID uuid.UUID) ([]ConversationSummary, error) {
	artist, err := s.repo.GetArtistByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListConversationsForArtist(ctx, artist.ID)
	if err != nil {
		return nil, err
	}
	out := make([]ConversationSummary, 0, len(rows))
	for _, r := range rows {
		out = append(out, ConversationSummary{
			ID:               r.ID,
			BookingRequestID: r.BookingRequestID,
			ClientName:       r.ClientName,
			ClientEmail:      r.ClientEmail,
			Status:           r.Status,
			LastMessageBody:  r.LastMessageBody,
			LastMessageAt:    timePtr(r.LastMessageAt),
			HasUnread:        r.HasUnread,
		})
	}
	return out, nil
}

func (s *service) GetArtistConversation(ctx context.Context, userID, convID uuid.UUID) (ConversationThread, error) {
	conv, err := s.artistConversation(ctx, userID, convID)
	if err != nil {
		return ConversationThread{}, err
	}
	return s.thread(ctx, conv)
}

func (s *service) SendArtistMessage(ctx context.Context, userID, convID uuid.UUID, input SendMessageInput) (MessageView, error) {
	conv, err := s.artistConversation(ctx, userID, convID)
	if err != nil {
		return MessageView{}, err
	}
	return s.send(ctx, conv, senderArtist, pgUUID(userID), input)
}

func (s *service) MarkArtistRead(ctx context.Context, userID, convID uuid.UUID) error {
	artist, err := s.repo.GetArtistByUserID(ctx, userID)
	if err != nil {
		return err
	}
	return s.repo.MarkConversationReadByArtist(ctx, sqlc.MarkConversationReadByArtistParams{ID: convID, ArtistID: artist.ID})
}

func (s *service) artistConversation(ctx context.Context, userID, convID uuid.UUID) (sqlc.Conversation, error) {
	artist, err := s.repo.GetArtistByUserID(ctx, userID)
	if err != nil {
		return sqlc.Conversation{}, err
	}
	conv, err := s.repo.GetConversationForArtist(ctx, sqlc.GetConversationForArtistParams{ID: convID, ArtistID: artist.ID})
	return conv, notFoundOr(err)
}

// ── Client (account) ──────────────────────────────────────────────────────────

func (s *service) ListClientConversations(ctx context.Context, userID uuid.UUID) ([]ConversationSummary, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListConversationsForClient(ctx, user.Email)
	if err != nil {
		return nil, err
	}
	out := make([]ConversationSummary, 0, len(rows))
	for _, r := range rows {
		out = append(out, ConversationSummary{
			ID:               r.ID,
			BookingRequestID: r.BookingRequestID,
			ClientName:       r.ClientName,
			ClientEmail:      r.ClientEmail,
			ArtistName:       r.ArtistName,
			Status:           r.Status,
			LastMessageBody:  r.LastMessageBody,
			LastMessageAt:    timePtr(r.LastMessageAt),
			HasUnread:        r.HasUnread,
		})
	}
	return out, nil
}

func (s *service) GetClientConversation(ctx context.Context, userID, convID uuid.UUID) (ConversationThread, error) {
	conv, err := s.clientConversation(ctx, userID, convID)
	if err != nil {
		return ConversationThread{}, err
	}
	return s.thread(ctx, conv)
}

func (s *service) SendClientMessage(ctx context.Context, userID, convID uuid.UUID, input SendMessageInput) (MessageView, error) {
	conv, err := s.clientConversation(ctx, userID, convID)
	if err != nil {
		return MessageView{}, err
	}
	return s.send(ctx, conv, senderClient, pgUUID(userID), input)
}

func (s *service) MarkClientRead(ctx context.Context, userID, convID uuid.UUID) error {
	if _, err := s.clientConversation(ctx, userID, convID); err != nil {
		return err
	}
	return s.repo.MarkConversationReadByClient(ctx, convID)
}

func (s *service) clientConversation(ctx context.Context, userID, convID uuid.UUID) (sqlc.Conversation, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return sqlc.Conversation{}, err
	}
	conv, err := s.repo.GetConversationForClient(ctx, sqlc.GetConversationForClientParams{ID: convID, ClientEmail: user.Email})
	return conv, notFoundOr(err)
}

// ── Guest (tokenized) ─────────────────────────────────────────────────────────

func (s *service) GetGuestConversation(ctx context.Context, token string) (ConversationThread, error) {
	conv, err := s.guestConversation(ctx, token)
	if err != nil {
		return ConversationThread{}, err
	}
	return s.thread(ctx, conv)
}

func (s *service) SendGuestMessage(ctx context.Context, token string, input SendMessageInput) (MessageView, error) {
	conv, err := s.guestConversation(ctx, token)
	if err != nil {
		return MessageView{}, err
	}
	// A guest has no account, so sender_user_id stays null.
	return s.send(ctx, conv, senderClient, pgtype.UUID{}, input)
}

func (s *service) MarkGuestRead(ctx context.Context, token string) error {
	conv, err := s.guestConversation(ctx, token)
	if err != nil {
		return err
	}
	return s.repo.MarkConversationReadByClient(ctx, conv.ID)
}

func (s *service) guestConversation(ctx context.Context, token string) (sqlc.Conversation, error) {
	conv, err := s.repo.GetConversationByToken(ctx, token)
	return conv, notFoundOr(err)
}

func (s *service) send(ctx context.Context, conv sqlc.Conversation, senderType string, senderUserID pgtype.UUID, input SendMessageInput) (MessageView, error) {
	body := strings.TrimSpace(input.Body)
	if body == "" {
		return MessageView{}, ErrEmptyMessage
	}
	if len(body) > maxMessageLength {
		return MessageView{}, ErrTooLong
	}

	row, err := s.repo.CreateMessage(ctx, sqlc.CreateMessageParams{
		ConversationID: conv.ID,
		SenderType:     senderType,
		SenderUserID:   senderUserID,
		Body:           body,
		AiDrafted:      false,
	})
	if err != nil {
		return MessageView{}, err
	}
	if err := s.repo.TouchConversationAfterSend(ctx, sqlc.TouchConversationAfterSendParams{ID: conv.ID, SenderType: senderType}); err != nil {
		return MessageView{}, err
	}

	s.publishMessageCreated(ctx, conv.ID, row.ID, senderType)
	return messageView(row), nil
}

func (s *service) thread(ctx context.Context, conv sqlc.Conversation) (ConversationThread, error) {
	rows, err := s.repo.ListMessages(ctx, conv.ID)
	if err != nil {
		return ConversationThread{}, err
	}
	msgs := make([]MessageView, 0, len(rows))
	for _, r := range rows {
		msgs = append(msgs, messageView(r))
	}
	artistName, _ := s.repo.GetArtistDisplayName(ctx, conv.ArtistID)
	clientPhone, _ := s.repo.GetBookingRequestPhone(ctx, conv.BookingRequestID)
	return ConversationThread{
		ID:               conv.ID,
		BookingRequestID: conv.BookingRequestID,
		ClientName:       conv.ClientName,
		ClientEmail:      conv.ClientEmail,
		ClientPhone:      clientPhone,
		ArtistName:       artistName,
		Status:           conv.Status,
		Messages:         msgs,
	}, nil
}

func (s *service) CreateConversationForBooking(ctx context.Context, artistID, bookingID uuid.UUID, clientName, clientEmail string) (sqlc.Conversation, error) {
	token, err := newAccessToken()
	if err != nil {
		return sqlc.Conversation{}, err
	}
	return s.repo.GetOrCreateConversation(ctx, sqlc.GetOrCreateConversationParams{
		ArtistID:          artistID,
		BookingRequestID:  bookingID,
		ClientName:        clientName,
		ClientEmail:       clientEmail,
		ClientAccessToken: token,
	})
}

func (s *service) publishMessageCreated(ctx context.Context, convID, msgID uuid.UUID, senderType string) {
	if s.pub == nil {
		return
	}
	if err := s.pub.Publish(ctx, MessageCreatedRoutingKey, MessageCreatedEvent{
		ConversationID: convID.String(),
		MessageID:      msgID.String(),
		SenderType:     senderType,
	}); err != nil {
		s.log.Warn("messaging_publish_failed", "conversation_id", convID, "error", err)
	}
}

func notFoundOr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func messageView(m sqlc.Message) MessageView {
	return MessageView{
		ID:         m.ID,
		SenderType: m.SenderType,
		Body:       m.Body,
		AIDrafted:  m.AiDrafted,
		CreatedAt:  m.CreatedAt.Time,
	}
}

func pgUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

func timePtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}

func newAccessToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
