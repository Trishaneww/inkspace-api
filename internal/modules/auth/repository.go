package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type Repository interface {
	CreateUser(ctx context.Context, arg sqlc.CreateUserParams) (sqlc.User, error)
	UpdateUnverifiedUser(ctx context.Context, arg sqlc.UpdateUnverifiedUserParams) (sqlc.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error)
	GetUserByEmail(ctx context.Context, email string) (sqlc.User, error)
	GetUserByPhone(ctx context.Context, phone *string) (sqlc.User, error)
	GetUserByUsername(ctx context.Context, username *string) (sqlc.User, error)
	MarkPhoneVerified(ctx context.Context, id uuid.UUID) error

	EnsureArtist(ctx context.Context, userID uuid.UUID) error
	GetArtistOnboardedAt(ctx context.Context, userID uuid.UUID) (pgtype.Timestamptz, error)

	CreatePhoneVerification(
		ctx context.Context,
		arg sqlc.CreatePhoneVerificationParams,
	) (sqlc.PhoneVerification, error)
	GetActivePhoneVerification(
		ctx context.Context,
		id uuid.UUID,
	) (sqlc.PhoneVerification, error)
	IncrementPhoneVerificationAttempts(ctx context.Context, id uuid.UUID) error
	RefreshPhoneVerificationCode(
		ctx context.Context,
		arg sqlc.RefreshPhoneVerificationCodeParams,
	) error
	ConsumePhoneVerification(ctx context.Context, id uuid.UUID) error
	RevokeActivePhoneVerificationsForUser(ctx context.Context, userID uuid.UUID) error

	CreateRefreshToken(
		ctx context.Context,
		arg sqlc.CreateRefreshTokenParams,
	) (sqlc.RefreshToken, error)
	GetActiveRefreshTokenByHash(
		ctx context.Context,
		tokenHash string,
	) (sqlc.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllRefreshTokensForUser(ctx context.Context, userID uuid.UUID) error
}

type repository struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{db: db, q: sqlc.New(db)}
}

func (r *repository) CreateUser(
	ctx context.Context, arg sqlc.CreateUserParams,
) (sqlc.User, error) {
	return r.q.CreateUser(ctx, arg)
}

func (r *repository) UpdateUnverifiedUser(
	ctx context.Context, arg sqlc.UpdateUnverifiedUserParams,
) (sqlc.User, error) {
	return r.q.UpdateUnverifiedUser(ctx, arg)
}

func (r *repository) GetUserByID(
	ctx context.Context, id uuid.UUID,
) (sqlc.User, error) {
	return r.q.GetUserByID(ctx, id)
}

func (r *repository) GetUserByEmail(
	ctx context.Context, email string,
) (sqlc.User, error) {
	return r.q.GetUserByEmail(ctx, email)
}

func (r *repository) GetUserByPhone(
	ctx context.Context, phone *string,
) (sqlc.User, error) {
	return r.q.GetUserByPhone(ctx, phone)
}

func (r *repository) GetUserByUsername(
	ctx context.Context, username *string,
) (sqlc.User, error) {
	return r.q.GetUserByUsername(ctx, username)
}

func (r *repository) MarkPhoneVerified(
	ctx context.Context, id uuid.UUID,
) error {
	return r.q.MarkPhoneVerified(ctx, id)
}

func (r *repository) EnsureArtist(
	ctx context.Context, userID uuid.UUID,
) error {
	return r.q.EnsureArtist(ctx, userID)
}

func (r *repository) GetArtistOnboardedAt(
	ctx context.Context, userID uuid.UUID,
) (pgtype.Timestamptz, error) {
	return r.q.GetArtistOnboardedAt(ctx, userID)
}

func (r *repository) CreatePhoneVerification(
	ctx context.Context,
	arg sqlc.CreatePhoneVerificationParams,
) (sqlc.PhoneVerification, error) {
	return r.q.CreatePhoneVerification(ctx, arg)
}

func (r *repository) GetActivePhoneVerification(
	ctx context.Context, id uuid.UUID,
) (sqlc.PhoneVerification, error) {
	return r.q.GetActivePhoneVerification(ctx, id)
}

func (r *repository) IncrementPhoneVerificationAttempts(
	ctx context.Context, id uuid.UUID,
) error {
	return r.q.IncrementPhoneVerificationAttempts(ctx, id)
}

func (r *repository) RefreshPhoneVerificationCode(
	ctx context.Context,
	arg sqlc.RefreshPhoneVerificationCodeParams,
) error {
	return r.q.RefreshPhoneVerificationCode(ctx, arg)
}

func (r *repository) ConsumePhoneVerification(
	ctx context.Context, id uuid.UUID,
) error {
	return r.q.ConsumePhoneVerification(ctx, id)
}

func (r *repository) RevokeActivePhoneVerificationsForUser(
	ctx context.Context, userID uuid.UUID,
) error {
	return r.q.RevokeActivePhoneVerificationsForUser(ctx, userID)
}

func (r *repository) CreateRefreshToken(
	ctx context.Context, arg sqlc.CreateRefreshTokenParams,
) (sqlc.RefreshToken, error) {
	return r.q.CreateRefreshToken(ctx, arg)
}

func (r *repository) GetActiveRefreshTokenByHash(
	ctx context.Context, tokenHash string,
) (sqlc.RefreshToken, error) {
	return r.q.GetActiveRefreshTokenByHash(ctx, tokenHash)
}

func (r *repository) RevokeRefreshToken(
	ctx context.Context, tokenHash string,
) error {
	return r.q.RevokeRefreshToken(ctx, tokenHash)
}

func (r *repository) RevokeAllRefreshTokensForUser(
	ctx context.Context, userID uuid.UUID,
) error {
	return r.q.RevokeAllRefreshTokensForUser(ctx, userID)
}
