package messaging

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type Repository interface {
	GetArtistByUserID(ctx context.Context, userID uuid.UUID) (sqlc.Artist, error)
	GetArtistDisplayName(ctx context.Context, artistID uuid.UUID) (string, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error)

	GetOrCreateConversation(ctx context.Context, params sqlc.GetOrCreateConversationParams) (sqlc.Conversation, error)
	GetConversationForArtist(ctx context.Context, params sqlc.GetConversationForArtistParams) (sqlc.Conversation, error)
	GetConversationForClient(ctx context.Context, params sqlc.GetConversationForClientParams) (sqlc.Conversation, error)
	GetConversationByToken(ctx context.Context, token string) (sqlc.Conversation, error)
	ListConversationsForArtist(ctx context.Context, artistID uuid.UUID) ([]sqlc.ListConversationsForArtistRow, error)
	ListConversationsForClient(ctx context.Context, clientEmail string) ([]sqlc.ListConversationsForClientRow, error)
	TouchConversationAfterSend(ctx context.Context, params sqlc.TouchConversationAfterSendParams) error
	MarkConversationReadByArtist(ctx context.Context, params sqlc.MarkConversationReadByArtistParams) error
	MarkConversationReadByClient(ctx context.Context, id uuid.UUID) error

	CreateMessage(ctx context.Context, params sqlc.CreateMessageParams) (sqlc.Message, error)
	ListMessages(ctx context.Context, conversationID uuid.UUID) ([]sqlc.Message, error)

	GetBookingRequestPhone(ctx context.Context, bookingRequestID uuid.UUID) (string, error)
}

type repository struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{db: db, q: sqlc.New(db)}
}

func (r *repository) GetArtistByUserID(ctx context.Context, userID uuid.UUID) (sqlc.Artist, error) {
	return r.q.GetArtistByUserID(ctx, userID)
}

func (r *repository) GetArtistDisplayName(ctx context.Context, artistID uuid.UUID) (string, error) {
	return r.q.GetArtistDisplayName(ctx, artistID)
}

func (r *repository) GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error) {
	return r.q.GetUserByID(ctx, id)
}

func (r *repository) GetOrCreateConversation(ctx context.Context, params sqlc.GetOrCreateConversationParams) (sqlc.Conversation, error) {
	return r.q.GetOrCreateConversation(ctx, params)
}

func (r *repository) GetConversationForArtist(ctx context.Context, params sqlc.GetConversationForArtistParams) (sqlc.Conversation, error) {
	return r.q.GetConversationForArtist(ctx, params)
}

func (r *repository) GetConversationForClient(ctx context.Context, params sqlc.GetConversationForClientParams) (sqlc.Conversation, error) {
	return r.q.GetConversationForClient(ctx, params)
}

func (r *repository) GetConversationByToken(ctx context.Context, token string) (sqlc.Conversation, error) {
	return r.q.GetConversationByToken(ctx, token)
}

func (r *repository) ListConversationsForArtist(ctx context.Context, artistID uuid.UUID) ([]sqlc.ListConversationsForArtistRow, error) {
	return r.q.ListConversationsForArtist(ctx, artistID)
}

func (r *repository) ListConversationsForClient(ctx context.Context, clientEmail string) ([]sqlc.ListConversationsForClientRow, error) {
	return r.q.ListConversationsForClient(ctx, clientEmail)
}

func (r *repository) TouchConversationAfterSend(ctx context.Context, params sqlc.TouchConversationAfterSendParams) error {
	return r.q.TouchConversationAfterSend(ctx, params)
}

func (r *repository) MarkConversationReadByArtist(ctx context.Context, params sqlc.MarkConversationReadByArtistParams) error {
	return r.q.MarkConversationReadByArtist(ctx, params)
}

func (r *repository) MarkConversationReadByClient(ctx context.Context, id uuid.UUID) error {
	return r.q.MarkConversationReadByClient(ctx, id)
}

func (r *repository) CreateMessage(ctx context.Context, params sqlc.CreateMessageParams) (sqlc.Message, error) {
	return r.q.CreateMessage(ctx, params)
}

func (r *repository) ListMessages(ctx context.Context, conversationID uuid.UUID) ([]sqlc.Message, error) {
	return r.q.ListMessages(ctx, conversationID)
}

func (r *repository) GetBookingRequestPhone(ctx context.Context, bookingRequestID uuid.UUID) (string, error) {
	return r.q.GetBookingRequestPhone(ctx, bookingRequestID)
}
