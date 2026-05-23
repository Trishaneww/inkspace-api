package matching

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type Repository interface {
	// TODO: CreateMatchRequest, GetMatchRequestByID, ListMatchRequestsByClient,
	// TODO: UpdateMatchRequestStatus, ExpireOpenMatchRequests,
	// TODO: InsertMatch, GetMatchByID, ListMatchesByRequest, ListLeadsForArtist, UpdateMatchStatus
}

type repository struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{db: db, q: sqlc.New(db)}
}

func (r *repository) withTx(tx pgx.Tx) *repository {
	return &repository{db: r.db, q: r.q.WithTx(tx)}
}
