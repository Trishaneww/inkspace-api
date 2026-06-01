package flashes

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type Repository interface {
	// Flash rows
	Create(ctx context.Context, params sqlc.CreateFlashParams) (sqlc.Flash, error)
	GetByID(ctx context.Context, id uuid.UUID) (sqlc.Flash, error)
	ListByArtist(ctx context.Context, params sqlc.ListFlashesByArtistParams) ([]sqlc.Flash, error)
	CountByArtist(ctx context.Context, params sqlc.CountFlashesByArtistParams) (sqlc.CountFlashesByArtistRow, error)
	Update(ctx context.Context, params sqlc.UpdateFlashParams) (sqlc.Flash, error)
	Publish(ctx context.Context, id uuid.UUID) (sqlc.Flash, error)
	Archive(ctx context.Context, id uuid.UUID) (sqlc.Flash, error)
	Unarchive(ctx context.Context, id uuid.UUID) (sqlc.Flash, error)
	Delete(ctx context.Context, id uuid.UUID) error
	IncrementViewCount(ctx context.Context, id uuid.UUID) error

	// Pricing tiers
	ListPricingTiers(ctx context.Context, flashID uuid.UUID) ([]sqlc.FlashPricingTier, error)
	ListPricingTiersForFlashes(ctx context.Context, flashIDs []uuid.UUID) ([]sqlc.FlashPricingTier, error)
	UpsertPricingTier(ctx context.Context, params sqlc.UpsertFlashPricingTierParams) (sqlc.FlashPricingTier, error)
	DeleteAllPricingTiers(ctx context.Context, flashID uuid.UUID) error

	// Cross-table lookup: JWT carries user_id, but flashes belong to an
	// artist_id. The 1:1 mapping lives in the artists table.
	GetArtistByUserID(ctx context.Context, userID uuid.UUID) (sqlc.Artist, error)

	// RunInTx runs `fn` inside a transaction, handing it a Repository
	// bound to the transaction. Used when create/update must touch both
	// the flash row and its pricing tiers atomically.
	RunInTx(ctx context.Context, fn func(Repository) error) error
}

type repository struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{db: db, q: sqlc.New(db)}
}

// withTx returns a copy of the repo bound to the given transaction.
func (r *repository) withTx(tx pgx.Tx) *repository {
	return &repository{db: r.db, q: r.q.WithTx(tx)}
}

func (r *repository) Create(ctx context.Context, params sqlc.CreateFlashParams) (sqlc.Flash, error) {
	return r.q.CreateFlash(ctx, params)
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (sqlc.Flash, error) {
	return r.q.GetFlash(ctx, id)
}

func (r *repository) ListByArtist(ctx context.Context, params sqlc.ListFlashesByArtistParams) ([]sqlc.Flash, error) {
	return r.q.ListFlashesByArtist(ctx, params)
}

func (r *repository) CountByArtist(ctx context.Context, params sqlc.CountFlashesByArtistParams) (sqlc.CountFlashesByArtistRow, error) {
	return r.q.CountFlashesByArtist(ctx, params)
}

func (r *repository) Update(ctx context.Context, params sqlc.UpdateFlashParams) (sqlc.Flash, error) {
	return r.q.UpdateFlash(ctx, params)
}

func (r *repository) Publish(ctx context.Context, id uuid.UUID) (sqlc.Flash, error) {
	return r.q.PublishFlash(ctx, id)
}

func (r *repository) Archive(ctx context.Context, id uuid.UUID) (sqlc.Flash, error) {
	return r.q.ArchiveFlash(ctx, id)
}

func (r *repository) Unarchive(ctx context.Context, id uuid.UUID) (sqlc.Flash, error) {
	return r.q.UnarchiveFlash(ctx, id)
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteFlash(ctx, id)
}

func (r *repository) IncrementViewCount(ctx context.Context, id uuid.UUID) error {
	return r.q.IncrementFlashViewCount(ctx, id)
}

func (r *repository) ListPricingTiers(ctx context.Context, flashID uuid.UUID) ([]sqlc.FlashPricingTier, error) {
	return r.q.ListFlashPricingTiers(ctx, flashID)
}

func (r *repository) ListPricingTiersForFlashes(ctx context.Context, flashIDs []uuid.UUID) ([]sqlc.FlashPricingTier, error) {
	return r.q.ListFlashPricingTiersForFlashes(ctx, flashIDs)
}

func (r *repository) UpsertPricingTier(ctx context.Context, params sqlc.UpsertFlashPricingTierParams) (sqlc.FlashPricingTier, error) {
	return r.q.UpsertFlashPricingTier(ctx, params)
}

func (r *repository) DeleteAllPricingTiers(ctx context.Context, flashID uuid.UUID) error {
	return r.q.DeleteAllFlashPricingTiers(ctx, flashID)
}

func (r *repository) GetArtistByUserID(ctx context.Context, userID uuid.UUID) (sqlc.Artist, error) {
	return r.q.GetArtistByUserID(ctx, userID)
}

func (r *repository) RunInTx(ctx context.Context, fn func(Repository) error) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(r.withTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
