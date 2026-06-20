package dashboard

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type Repository interface {
	GetArtistByUserID(ctx context.Context, userID uuid.UUID) (sqlc.Artist, error)
	GetArtistSettings(ctx context.Context, artistID uuid.UUID) (sqlc.ArtistSetting, error)
	GetDashboardCounts(ctx context.Context, artistID uuid.UUID) (sqlc.GetDashboardCountsRow, error)
	GetBookingMixForRange(ctx context.Context, params sqlc.GetBookingMixForRangeParams) (sqlc.GetBookingMixForRangeRow, error)
	GetMonthlyEarnings(ctx context.Context, params sqlc.GetMonthlyEarningsParams) ([]sqlc.GetMonthlyEarningsRow, error)
	ListUpcomingAppointments(ctx context.Context, artistID uuid.UUID) ([]sqlc.ListUpcomingAppointmentsRow, error)
}

type repository struct {
	q *sqlc.Queries
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{q: sqlc.New(db)}
}

func (r *repository) GetArtistByUserID(ctx context.Context, userID uuid.UUID) (sqlc.Artist, error) {
	return r.q.GetArtistByUserID(ctx, userID)
}

func (r *repository) GetArtistSettings(ctx context.Context, artistID uuid.UUID) (sqlc.ArtistSetting, error) {
	return r.q.GetArtistSettings(ctx, artistID)
}

func (r *repository) GetDashboardCounts(ctx context.Context, artistID uuid.UUID) (sqlc.GetDashboardCountsRow, error) {
	return r.q.GetDashboardCounts(ctx, artistID)
}

func (r *repository) GetBookingMixForRange(ctx context.Context, params sqlc.GetBookingMixForRangeParams) (sqlc.GetBookingMixForRangeRow, error) {
	return r.q.GetBookingMixForRange(ctx, params)
}

func (r *repository) GetMonthlyEarnings(ctx context.Context, params sqlc.GetMonthlyEarningsParams) ([]sqlc.GetMonthlyEarningsRow, error) {
	return r.q.GetMonthlyEarnings(ctx, params)
}

func (r *repository) ListUpcomingAppointments(ctx context.Context, artistID uuid.UUID) ([]sqlc.ListUpcomingAppointmentsRow, error) {
	return r.q.ListUpcomingAppointments(ctx, artistID)
}
