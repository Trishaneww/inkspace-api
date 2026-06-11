package openbook

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type Repository interface {
	GetUserByUsername(ctx context.Context, username string) (sqlc.User, error)
	GetArtistByUserID(ctx context.Context, userID uuid.UUID) (sqlc.Artist, error)
	GetArtistSettings(ctx context.Context, artistID uuid.UUID) (sqlc.ArtistSetting, error)
	GetOpenBookByArtist(ctx context.Context, artistID uuid.UUID) (sqlc.OpenBook, error)
	ListAvailabilityWindows(ctx context.Context, artistID uuid.UUID) ([]sqlc.ArtistAvailabilityWindow, error)
	ListArtistLocations(ctx context.Context, artistID uuid.UUID) ([]sqlc.ArtistLocation, error)
	CreateBookingRequest(ctx context.Context, params sqlc.CreateBookingRequestParams) (sqlc.BookingRequest, error)
}

type repository struct {
	q *sqlc.Queries
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{q: sqlc.New(db)}
}

func (r *repository) GetUserByUsername(ctx context.Context, username string) (sqlc.User, error) {
	return r.q.GetUserByUsername(ctx, &username)
}

func (r *repository) GetArtistByUserID(ctx context.Context, userID uuid.UUID) (sqlc.Artist, error) {
	return r.q.GetArtistByUserID(ctx, userID)
}

func (r *repository) GetArtistSettings(ctx context.Context, artistID uuid.UUID) (sqlc.ArtistSetting, error) {
	return r.q.GetArtistSettings(ctx, artistID)
}

func (r *repository) GetOpenBookByArtist(ctx context.Context, artistID uuid.UUID) (sqlc.OpenBook, error) {
	return r.q.GetOpenBookByArtist(ctx, artistID)
}

func (r *repository) ListAvailabilityWindows(ctx context.Context, artistID uuid.UUID) ([]sqlc.ArtistAvailabilityWindow, error) {
	return r.q.ListAvailabilityWindows(ctx, artistID)
}

func (r *repository) ListArtistLocations(ctx context.Context, artistID uuid.UUID) ([]sqlc.ArtistLocation, error) {
	return r.q.ListArtistLocations(ctx, artistID)
}

func (r *repository) CreateBookingRequest(ctx context.Context, params sqlc.CreateBookingRequestParams) (sqlc.BookingRequest, error) {
	return r.q.CreateBookingRequest(ctx, params)
}
