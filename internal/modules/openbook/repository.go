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
	CountAvailableFlashes(ctx context.Context, artistID uuid.UUID) (int64, error)
	CountPublishedPortfolio(ctx context.Context, artistID uuid.UUID) (int64, error)
	GetFlash(ctx context.Context, flashID uuid.UUID) (sqlc.Flash, error)
	ListFlashPricingTiers(ctx context.Context, flashID uuid.UUID) ([]sqlc.FlashPricingTier, error)
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

func (r *repository) CountAvailableFlashes(ctx context.Context, artistID uuid.UUID) (int64, error) {
	row, err := r.q.CountFlashesByArtist(ctx, sqlc.CountFlashesByArtistParams{ArtistID: artistID})
	if err != nil {
		return 0, err
	}
	return row.Available, nil
}

func (r *repository) CountPublishedPortfolio(ctx context.Context, artistID uuid.UUID) (int64, error) {
	row, err := r.q.CountPortfolioItemsByArtist(ctx, sqlc.CountPortfolioItemsByArtistParams{ArtistID: artistID})
	if err != nil {
		return 0, err
	}
	return row.Published, nil
}

func (r *repository) GetFlash(ctx context.Context, flashID uuid.UUID) (sqlc.Flash, error) {
	return r.q.GetFlash(ctx, flashID)
}

func (r *repository) ListFlashPricingTiers(ctx context.Context, flashID uuid.UUID) ([]sqlc.FlashPricingTier, error) {
	return r.q.ListFlashPricingTiers(ctx, flashID)
}

func (r *repository) CreateBookingRequest(ctx context.Context, params sqlc.CreateBookingRequestParams) (sqlc.BookingRequest, error) {
	return r.q.CreateBookingRequest(ctx, params)
}
