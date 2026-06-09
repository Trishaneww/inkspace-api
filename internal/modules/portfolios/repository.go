package portfolios

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type Repository interface {
	// Portfolio items
	Create(ctx context.Context, params sqlc.CreatePortfolioItemParams) (sqlc.PortfolioItem, error)
	GetByID(ctx context.Context, id uuid.UUID) (sqlc.PortfolioItem, error)
	ListByArtist(ctx context.Context, params sqlc.ListPortfolioItemsByArtistParams) ([]sqlc.PortfolioItem, error)
	CountByArtist(ctx context.Context, params sqlc.CountPortfolioItemsByArtistParams) (sqlc.CountPortfolioItemsByArtistRow, error)
	Update(ctx context.Context, params sqlc.UpdatePortfolioItemParams) (sqlc.PortfolioItem, error)
	Publish(ctx context.Context, id uuid.UUID) (sqlc.PortfolioItem, error)
	Archive(ctx context.Context, id uuid.UUID) (sqlc.PortfolioItem, error)
	Unarchive(ctx context.Context, id uuid.UUID) (sqlc.PortfolioItem, error)
	Delete(ctx context.Context, id uuid.UUID) error

	GetArtistByUserID(ctx context.Context, userID uuid.UUID) (sqlc.Artist, error)
	EnsureArtist(ctx context.Context, userID uuid.UUID) error
}

type repository struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{db: db, q: sqlc.New(db)}
}

func (r *repository) Create(ctx context.Context, params sqlc.CreatePortfolioItemParams) (sqlc.PortfolioItem, error) {
	return r.q.CreatePortfolioItem(ctx, params)
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (sqlc.PortfolioItem, error) {
	return r.q.GetPortfolioItem(ctx, id)
}

func (r *repository) ListByArtist(ctx context.Context, params sqlc.ListPortfolioItemsByArtistParams) ([]sqlc.PortfolioItem, error) {
	return r.q.ListPortfolioItemsByArtist(ctx, params)
}

func (r *repository) CountByArtist(ctx context.Context, params sqlc.CountPortfolioItemsByArtistParams) (sqlc.CountPortfolioItemsByArtistRow, error) {
	return r.q.CountPortfolioItemsByArtist(ctx, params)
}

func (r *repository) Update(ctx context.Context, params sqlc.UpdatePortfolioItemParams) (sqlc.PortfolioItem, error) {
	return r.q.UpdatePortfolioItem(ctx, params)
}

func (r *repository) Publish(ctx context.Context, id uuid.UUID) (sqlc.PortfolioItem, error) {
	return r.q.PublishPortfolioItem(ctx, id)
}

func (r *repository) Archive(ctx context.Context, id uuid.UUID) (sqlc.PortfolioItem, error) {
	return r.q.ArchivePortfolioItem(ctx, id)
}

func (r *repository) Unarchive(ctx context.Context, id uuid.UUID) (sqlc.PortfolioItem, error) {
	return r.q.UnarchivePortfolioItem(ctx, id)
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeletePortfolioItem(ctx, id)
}

func (r *repository) GetArtistByUserID(ctx context.Context, userID uuid.UUID) (sqlc.Artist, error) {
	return r.q.GetArtistByUserID(ctx, userID)
}

func (r *repository) EnsureArtist(ctx context.Context, userID uuid.UUID) error {
	return r.q.EnsureArtist(ctx, userID)
}
