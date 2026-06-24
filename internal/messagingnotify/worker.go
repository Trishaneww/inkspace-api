package messagingnotify

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
	"github.com/trishaneupnexx/inkspace-api/internal/email"
	"github.com/trishaneupnexx/inkspace-api/internal/events"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/messaging"
)

const queueName = "messaging.notifications"

type worker struct {
	q           *sqlc.Queries
	email       *email.Client
	frontendURL string
	log         *slog.Logger
}

func Run(ctx context.Context, rabbitURL string, db *pgxpool.Pool, emailClient *email.Client, frontendURL string, log *slog.Logger) {
	consumer, err := events.NewConsumer(rabbitURL, queueName, []string{messaging.MessageCreatedRoutingKey})
	if err != nil {
		log.Warn("messaging_notify_consumer_init_failed", "error", err)
		return
	}
	defer consumer.Close()

	w := &worker{q: sqlc.New(db), email: emailClient, frontendURL: frontendURL, log: log}
	if err := consumer.Run(ctx, w.handle); err != nil && ctx.Err() == nil {
		log.Warn("messaging_notify_consumer_stopped", "error", err)
	}
}

type newMessageEmailPayload struct {
	To            string `json:"to"`
	Audience      string `json:"audience"`
	RecipientName string `json:"recipientName"`
	SenderName    string `json:"senderName"`
	ArtistName    string `json:"artistName"`
	URL           string `json:"url"`
}

func (w *worker) handle(ctx context.Context, routingKey string, body []byte) error {
	if routingKey != messaging.MessageCreatedRoutingKey {
		return nil
	}

	var ev messaging.MessageCreatedEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		w.log.Warn("messaging_notify_bad_payload", "error", err)
		return nil
	}
	convID, err := uuid.Parse(ev.ConversationID)
	if err != nil {
		return nil
	}

	conv, err := w.q.GetConversationByID(ctx, convID)
	if err != nil {
		w.log.Warn("messaging_notify_conversation_missing", "conversation_id", convID, "error", err)
		return nil
	}

	if !w.isFirstUnread(ctx, conv, ev) {
		return nil
	}

	artistName, artistEmail := w.artistContact(ctx, conv.ArtistID)

	switch ev.SenderType {
	case "artist":
		w.email.Post("/api/internal/emails/new-message", newMessageEmailPayload{
			To:            conv.ClientEmail,
			Audience:      "client",
			RecipientName: conv.ClientName,
			SenderName:    artistName,
			ArtistName:    artistName,
			URL:           w.clientMessageURL(ctx, conv),
		})
	case "client":
		if artistEmail == "" {
			return nil
		}
		w.email.Post("/api/internal/emails/new-message", newMessageEmailPayload{
			To:            artistEmail,
			Audience:      "artist",
			RecipientName: artistName,
			SenderName:    conv.ClientName,
			ArtistName:    artistName,
			URL:           w.frontendURL + "/dashboard/artist/messages?c=" + conv.ID.String(),
		})
	}
	return nil
}

func (w *worker) isFirstUnread(ctx context.Context, conv sqlc.Conversation, ev messaging.MessageCreatedEvent) bool {
	msgID, err := uuid.Parse(ev.MessageID)
	if err != nil {
		return true
	}

	recipientLastRead := conv.ClientLastReadAt
	if ev.SenderType == "client" {
		recipientLastRead = conv.ArtistLastReadAt
	}

	prior, err := w.q.CountPriorUnreadFromSender(ctx, sqlc.CountPriorUnreadFromSenderParams{
		ConversationID:    conv.ID,
		SenderType:        ev.SenderType,
		RecipientLastRead: recipientLastRead,
		MessageID:         msgID,
	})
	if err != nil {
		w.log.Warn("messaging_notify_unread_count_failed", "conversation_id", conv.ID, "error", err)
		return true
	}
	return prior == 0
}

func (w *worker) clientMessageURL(ctx context.Context, conv sqlc.Conversation) string {
	user, err := w.q.GetUserByEmail(ctx, conv.ClientEmail)
	if err == nil && user.Role == "user" {
		return w.frontendURL + "/dashboard/client/messages?c=" + conv.ID.String()
	}
	return w.frontendURL + "/messages/" + conv.ClientAccessToken
}

func (w *worker) artistContact(ctx context.Context, artistID uuid.UUID) (name, addr string) {
	artist, err := w.q.GetArtistByID(ctx, artistID)
	if err != nil {
		return "", ""
	}
	user, err := w.q.GetUserByID(ctx, artist.UserID)
	if err != nil {
		return "", ""
	}
	name = strings.TrimSpace(deref(user.FirstName) + " " + deref(user.LastName))
	if name == "" {
		name = deref(user.Username)
	}
	return name, user.Email
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
