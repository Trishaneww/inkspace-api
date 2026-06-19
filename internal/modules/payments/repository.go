package payments

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type Repository interface {
	GetArtistByUserID(ctx context.Context, userID uuid.UUID) (sqlc.Artist, error)
	GetArtistByID(ctx context.Context, id uuid.UUID) (sqlc.Artist, error)
	GetArtistSettings(ctx context.Context, artistID uuid.UUID) (sqlc.ArtistSetting, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error)
	GetUserByEmail(ctx context.Context, email string) (sqlc.User, error)
	CreateUser(ctx context.Context, params sqlc.CreateUserParams) (sqlc.User, error)
	CreateRefreshToken(ctx context.Context, params sqlc.CreateRefreshTokenParams) (sqlc.RefreshToken, error)
	SetUserMarketingOptIn(ctx context.Context, params sqlc.SetUserMarketingOptInParams) error
	LinkBookingRequestsToClient(ctx context.Context, params sqlc.LinkBookingRequestsToClientParams) error
	GetBookingRequest(ctx context.Context, params sqlc.GetBookingRequestParams) (sqlc.BookingRequest, error)

	GetArtistEarnings(ctx context.Context, artistID uuid.UUID) (sqlc.GetArtistEarningsRow, error)
	ListRecentPaidPaymentsForArtist(ctx context.Context, artistID uuid.UUID) ([]sqlc.ListRecentPaidPaymentsForArtistRow, error)

	CreatePaymentRequest(ctx context.Context, params sqlc.CreatePaymentRequestParams) (sqlc.PaymentRequest, error)
	GetPaymentRequestByToken(ctx context.Context, token string) (sqlc.PaymentRequest, error)
	GetPaymentRequestForArtist(ctx context.Context, params sqlc.GetPaymentRequestForArtistParams) (sqlc.PaymentRequest, error)
	MarkPaymentRequestEmailed(ctx context.Context, params sqlc.MarkPaymentRequestEmailedParams) (sqlc.PaymentRequest, error)
	CountLivePaymentRequests(ctx context.Context, params sqlc.CountLivePaymentRequestsParams) (int64, error)
	AttachCheckoutSession(ctx context.Context, params sqlc.AttachCheckoutSessionParams) (sqlc.PaymentRequest, error)
	CancelPaymentRequest(ctx context.Context, params sqlc.CancelPaymentRequestParams) (sqlc.PaymentRequest, error)

	MarkPaymentRequestPaidBySession(ctx context.Context, params sqlc.MarkPaymentRequestPaidBySessionParams) (sqlc.PaymentRequest, error)
	MarkPaymentRequestPaidByPaymentIntent(ctx context.Context, paymentIntentID *string) (sqlc.PaymentRequest, error)
	MarkPaymentRequestFailedByPaymentIntent(ctx context.Context, paymentIntentID *string) (sqlc.PaymentRequest, error)
	MarkPaymentRequestExpiredBySession(ctx context.Context, sessionID *string) (sqlc.PaymentRequest, error)
	RefundPaymentRequestByPaymentIntent(ctx context.Context, params sqlc.RefundPaymentRequestByPaymentIntentParams) (sqlc.PaymentRequest, error)
	SetBookingDepositStatus(ctx context.Context, params sqlc.SetBookingDepositStatusParams) error

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

func (r *repository) GetArtistByID(ctx context.Context, id uuid.UUID) (sqlc.Artist, error) {
	return r.q.GetArtistByID(ctx, id)
}

func (r *repository) GetArtistSettings(ctx context.Context, artistID uuid.UUID) (sqlc.ArtistSetting, error) {
	return r.q.GetArtistSettings(ctx, artistID)
}

func (r *repository) GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error) {
	return r.q.GetUserByID(ctx, id)
}

func (r *repository) GetArtistEarnings(ctx context.Context, artistID uuid.UUID) (sqlc.GetArtistEarningsRow, error) {
	return r.q.GetArtistEarnings(ctx, artistID)
}

func (r *repository) ListRecentPaidPaymentsForArtist(ctx context.Context, artistID uuid.UUID) ([]sqlc.ListRecentPaidPaymentsForArtistRow, error) {
	return r.q.ListRecentPaidPaymentsForArtist(ctx, artistID)
}

func (r *repository) GetUserByEmail(ctx context.Context, email string) (sqlc.User, error) {
	return r.q.GetUserByEmail(ctx, email)
}

func (r *repository) CreateUser(ctx context.Context, params sqlc.CreateUserParams) (sqlc.User, error) {
	return r.q.CreateUser(ctx, params)
}

func (r *repository) CreateRefreshToken(ctx context.Context, params sqlc.CreateRefreshTokenParams) (sqlc.RefreshToken, error) {
	return r.q.CreateRefreshToken(ctx, params)
}

func (r *repository) SetUserMarketingOptIn(ctx context.Context, params sqlc.SetUserMarketingOptInParams) error {
	return r.q.SetUserMarketingOptIn(ctx, params)
}

func (r *repository) LinkBookingRequestsToClient(ctx context.Context, params sqlc.LinkBookingRequestsToClientParams) error {
	return r.q.LinkBookingRequestsToClient(ctx, params)
}

func (r *repository) GetBookingRequest(ctx context.Context, params sqlc.GetBookingRequestParams) (sqlc.BookingRequest, error) {
	return r.q.GetBookingRequest(ctx, params)
}

func (r *repository) CreatePaymentRequest(ctx context.Context, params sqlc.CreatePaymentRequestParams) (sqlc.PaymentRequest, error) {
	return r.q.CreatePaymentRequest(ctx, params)
}

func (r *repository) GetPaymentRequestByToken(ctx context.Context, token string) (sqlc.PaymentRequest, error) {
	return r.q.GetPaymentRequestByToken(ctx, token)
}

func (r *repository) GetPaymentRequestForArtist(ctx context.Context, params sqlc.GetPaymentRequestForArtistParams) (sqlc.PaymentRequest, error) {
	return r.q.GetPaymentRequestForArtist(ctx, params)
}

func (r *repository) MarkPaymentRequestEmailed(ctx context.Context, params sqlc.MarkPaymentRequestEmailedParams) (sqlc.PaymentRequest, error) {
	return r.q.MarkPaymentRequestEmailed(ctx, params)
}

func (r *repository) CountLivePaymentRequests(ctx context.Context, params sqlc.CountLivePaymentRequestsParams) (int64, error) {
	return r.q.CountLivePaymentRequests(ctx, params)
}

func (r *repository) AttachCheckoutSession(ctx context.Context, params sqlc.AttachCheckoutSessionParams) (sqlc.PaymentRequest, error) {
	return r.q.AttachCheckoutSession(ctx, params)
}

func (r *repository) CancelPaymentRequest(ctx context.Context, params sqlc.CancelPaymentRequestParams) (sqlc.PaymentRequest, error) {
	return r.q.CancelPaymentRequest(ctx, params)
}

func (r *repository) MarkPaymentRequestPaidBySession(ctx context.Context, params sqlc.MarkPaymentRequestPaidBySessionParams) (sqlc.PaymentRequest, error) {
	return r.q.MarkPaymentRequestPaidBySession(ctx, params)
}

func (r *repository) MarkPaymentRequestPaidByPaymentIntent(ctx context.Context, paymentIntentID *string) (sqlc.PaymentRequest, error) {
	return r.q.MarkPaymentRequestPaidByPaymentIntent(ctx, paymentIntentID)
}

func (r *repository) MarkPaymentRequestFailedByPaymentIntent(ctx context.Context, paymentIntentID *string) (sqlc.PaymentRequest, error) {
	return r.q.MarkPaymentRequestFailedByPaymentIntent(ctx, paymentIntentID)
}

func (r *repository) MarkPaymentRequestExpiredBySession(ctx context.Context, sessionID *string) (sqlc.PaymentRequest, error) {
	return r.q.MarkPaymentRequestExpiredBySession(ctx, sessionID)
}

func (r *repository) RefundPaymentRequestByPaymentIntent(ctx context.Context, params sqlc.RefundPaymentRequestByPaymentIntentParams) (sqlc.PaymentRequest, error) {
	return r.q.RefundPaymentRequestByPaymentIntent(ctx, params)
}

func (r *repository) SetBookingDepositStatus(ctx context.Context, params sqlc.SetBookingDepositStatusParams) error {
	return r.q.SetBookingDepositStatus(ctx, params)
}

func (r *repository) InsertStripeEvent(ctx context.Context, params sqlc.InsertStripeEventParams) (int64, error) {
	return r.q.InsertStripeEvent(ctx, params)
}
