package subscriptions

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type Repository interface {
	GetArtistByUserID(ctx context.Context, userID uuid.UUID) (sqlc.Artist, error)
	GetArtistByStripeCustomerID(ctx context.Context, stripeCustomerID *string) (sqlc.Artist, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error)

	SetArtistStripeCustomerID(ctx context.Context, params sqlc.SetArtistStripeCustomerIDParams) error
	UpdateArtistSubscription(ctx context.Context, params sqlc.UpdateArtistSubscriptionParams) error

	InsertStripeEvent(ctx context.Context, params sqlc.InsertStripeEventParams) (int64, error)
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

func (r *repository) GetArtistByStripeCustomerID(ctx context.Context, stripeCustomerID *string) (sqlc.Artist, error) {
	return r.q.GetArtistByStripeCustomerID(ctx, stripeCustomerID)
}

func (r *repository) GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error) {
	return r.q.GetUserByID(ctx, id)
}

func (r *repository) SetArtistStripeCustomerID(ctx context.Context, params sqlc.SetArtistStripeCustomerIDParams) error {
	return r.q.SetArtistStripeCustomerID(ctx, params)
}

func (r *repository) UpdateArtistSubscription(ctx context.Context, params sqlc.UpdateArtistSubscriptionParams) error {
	return r.q.UpdateArtistSubscription(ctx, params)
}

func (r *repository) InsertStripeEvent(ctx context.Context, params sqlc.InsertStripeEventParams) (int64, error) {
	return r.q.InsertStripeEvent(ctx, params)
}
