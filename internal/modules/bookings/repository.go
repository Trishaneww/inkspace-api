package bookings

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type Repository interface {
	GetArtistByUserID(ctx context.Context, userID uuid.UUID) (sqlc.Artist, error)
	GetArtistByID(ctx context.Context, id uuid.UUID) (sqlc.Artist, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error)
	EnsureArtist(ctx context.Context, userID uuid.UUID) error

	EnsureArtistSettings(ctx context.Context, artistID uuid.UUID) error
	GetArtistSettings(ctx context.Context, artistID uuid.UUID) (sqlc.ArtistSetting, error)
	GetOpenBookByArtist(ctx context.Context, artistID uuid.UUID) (sqlc.OpenBook, error)
	// Includes closed spots so a request can resolve the location it was tagged
	// with even after the artist closes it.
	ListAllArtistLocations(ctx context.Context, artistID uuid.UUID) ([]sqlc.ArtistLocation, error)
	ListAvailabilityWindows(ctx context.Context, artistID uuid.UUID) ([]sqlc.ArtistAvailabilityWindow, error)

	CreateBookingRequest(ctx context.Context, params sqlc.CreateBookingRequestParams) (sqlc.BookingRequest, error)
	ListBookingRequestsByArtist(ctx context.Context, artistID uuid.UUID) ([]sqlc.BookingRequest, error)
	ListBookingRequestsByClientEmail(ctx context.Context, clientEmail string) ([]sqlc.BookingRequest, error)
	ListConversationFlagsForArtist(ctx context.Context, artistID uuid.UUID) ([]sqlc.ListConversationFlagsForArtistRow, error)
	GetBookingRequest(ctx context.Context, params sqlc.GetBookingRequestParams) (sqlc.BookingRequest, error)
	UpdateBookingRequestStatus(ctx context.Context, params sqlc.UpdateBookingRequestStatusParams) (sqlc.BookingRequest, error)
	ReopenBookingRequest(ctx context.Context, params sqlc.ReopenBookingRequestParams) (sqlc.BookingRequest, error)
	SetScheduleToken(ctx context.Context, params sqlc.SetScheduleTokenParams) (sqlc.BookingRequest, error)
	TouchScheduleEmailedAt(ctx context.Context, params sqlc.TouchScheduleEmailedAtParams) (sqlc.BookingRequest, error)
	GetBookingRequestByScheduleToken(ctx context.Context, scheduleToken *string) (sqlc.BookingRequest, error)
	GetClientBookingRequest(ctx context.Context, params sqlc.GetClientBookingRequestParams) (sqlc.BookingRequest, error)

	// Express client-account creation (satisfies clientaccount.Store).
	GetUserByEmail(ctx context.Context, email string) (sqlc.User, error)
	CreateUser(ctx context.Context, params sqlc.CreateUserParams) (sqlc.User, error)
	SetUserMarketingOptIn(ctx context.Context, params sqlc.SetUserMarketingOptInParams) error
	LinkBookingRequestsToClient(ctx context.Context, params sqlc.LinkBookingRequestsToClientParams) error
	CreateRefreshToken(ctx context.Context, params sqlc.CreateRefreshTokenParams) (sqlc.RefreshToken, error)
	GetFlash(ctx context.Context, flashID uuid.UUID) (sqlc.Flash, error)
	ClaimFlash(ctx context.Context, params sqlc.ClaimFlashParams) (sqlc.Flash, error)
	DeclineOtherFlashRequests(ctx context.Context, params sqlc.DeclineOtherFlashRequestsParams) error
	GetBookingStats(ctx context.Context, artistID uuid.UUID) (sqlc.GetBookingStatsRow, error)

	CreateAppointment(ctx context.Context, params sqlc.CreateAppointmentParams) (sqlc.Appointment, error)
	GetLatestAppointmentByRequest(ctx context.Context, bookingRequestID uuid.UUID) (sqlc.Appointment, error)
	ListLiveAppointmentsByRequest(ctx context.Context, bookingRequestID uuid.UUID) ([]sqlc.Appointment, error)
	ListLatestAppointmentsByArtist(ctx context.Context, artistID uuid.UUID) ([]sqlc.Appointment, error)
	ListAppointmentsByArtistInRange(ctx context.Context, params sqlc.ListAppointmentsByArtistInRangeParams) ([]sqlc.ListAppointmentsByArtistInRangeRow, error)
	ListBusyAppointmentsByArtistInRange(ctx context.Context, params sqlc.ListBusyAppointmentsByArtistInRangeParams) ([]sqlc.ListBusyAppointmentsByArtistInRangeRow, error)
	UpdateAppointmentSchedule(ctx context.Context, params sqlc.UpdateAppointmentScheduleParams) (sqlc.Appointment, error)
	HoldAppointmentSchedule(ctx context.Context, params sqlc.HoldAppointmentScheduleParams) (sqlc.Appointment, error)
	ConfirmHeldAppointment(ctx context.Context, bookingRequestID uuid.UUID) (sqlc.Appointment, error)
	UpdateAppointmentStatus(ctx context.Context, params sqlc.UpdateAppointmentStatusParams) (sqlc.Appointment, error)
	SetAppointmentCalendarEvent(ctx context.Context, params sqlc.SetAppointmentCalendarEventParams) error
	CountOverlappingAppointments(ctx context.Context, params sqlc.CountOverlappingAppointmentsParams) (int64, error)

	ListPaymentRequestsByArtist(ctx context.Context, artistID uuid.UUID) ([]sqlc.PaymentRequest, error)
	ListPaymentRequestsByBooking(ctx context.Context, bookingRequestID uuid.UUID) ([]sqlc.PaymentRequest, error)
	CreatePaymentRequest(ctx context.Context, params sqlc.CreatePaymentRequestParams) (sqlc.PaymentRequest, error)
	SetBookingDepositAmount(ctx context.Context, params sqlc.SetBookingDepositAmountParams) error
	SeedMarkPaymentPaid(ctx context.Context, params sqlc.SeedMarkPaymentPaidParams) error
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

func (r *repository) GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error) {
	return r.q.GetUserByID(ctx, id)
}

func (r *repository) EnsureArtist(ctx context.Context, userID uuid.UUID) error {
	return r.q.EnsureArtist(ctx, userID)
}

func (r *repository) EnsureArtistSettings(ctx context.Context, artistID uuid.UUID) error {
	return r.q.EnsureArtistSettings(ctx, artistID)
}

func (r *repository) GetArtistSettings(ctx context.Context, artistID uuid.UUID) (sqlc.ArtistSetting, error) {
	return r.q.GetArtistSettings(ctx, artistID)
}

func (r *repository) GetOpenBookByArtist(ctx context.Context, artistID uuid.UUID) (sqlc.OpenBook, error) {
	return r.q.GetOpenBookByArtist(ctx, artistID)
}

func (r *repository) ListAllArtistLocations(ctx context.Context, artistID uuid.UUID) ([]sqlc.ArtistLocation, error) {
	return r.q.ListAllArtistLocations(ctx, artistID)
}

func (r *repository) ListAvailabilityWindows(ctx context.Context, artistID uuid.UUID) ([]sqlc.ArtistAvailabilityWindow, error) {
	return r.q.ListAvailabilityWindows(ctx, artistID)
}

func (r *repository) CreateBookingRequest(ctx context.Context, params sqlc.CreateBookingRequestParams) (sqlc.BookingRequest, error) {
	return r.q.CreateBookingRequest(ctx, params)
}

func (r *repository) ListBookingRequestsByArtist(ctx context.Context, artistID uuid.UUID) ([]sqlc.BookingRequest, error) {
	return r.q.ListBookingRequestsByArtist(ctx, artistID)
}

func (r *repository) ListBookingRequestsByClientEmail(ctx context.Context, clientEmail string) ([]sqlc.BookingRequest, error) {
	return r.q.ListBookingRequestsByClientEmail(ctx, clientEmail)
}

func (r *repository) ListConversationFlagsForArtist(ctx context.Context, artistID uuid.UUID) ([]sqlc.ListConversationFlagsForArtistRow, error) {
	return r.q.ListConversationFlagsForArtist(ctx, artistID)
}

func (r *repository) GetBookingRequest(ctx context.Context, params sqlc.GetBookingRequestParams) (sqlc.BookingRequest, error) {
	return r.q.GetBookingRequest(ctx, params)
}

func (r *repository) UpdateBookingRequestStatus(ctx context.Context, params sqlc.UpdateBookingRequestStatusParams) (sqlc.BookingRequest, error) {
	return r.q.UpdateBookingRequestStatus(ctx, params)
}

func (r *repository) ReopenBookingRequest(ctx context.Context, params sqlc.ReopenBookingRequestParams) (sqlc.BookingRequest, error) {
	return r.q.ReopenBookingRequest(ctx, params)
}

func (r *repository) SetScheduleToken(ctx context.Context, params sqlc.SetScheduleTokenParams) (sqlc.BookingRequest, error) {
	return r.q.SetScheduleToken(ctx, params)
}

func (r *repository) TouchScheduleEmailedAt(ctx context.Context, params sqlc.TouchScheduleEmailedAtParams) (sqlc.BookingRequest, error) {
	return r.q.TouchScheduleEmailedAt(ctx, params)
}

func (r *repository) GetBookingRequestByScheduleToken(ctx context.Context, scheduleToken *string) (sqlc.BookingRequest, error) {
	return r.q.GetBookingRequestByScheduleToken(ctx, scheduleToken)
}

func (r *repository) GetClientBookingRequest(ctx context.Context, params sqlc.GetClientBookingRequestParams) (sqlc.BookingRequest, error) {
	return r.q.GetClientBookingRequest(ctx, params)
}

func (r *repository) GetUserByEmail(ctx context.Context, email string) (sqlc.User, error) {
	return r.q.GetUserByEmail(ctx, email)
}

func (r *repository) CreateUser(ctx context.Context, params sqlc.CreateUserParams) (sqlc.User, error) {
	return r.q.CreateUser(ctx, params)
}

func (r *repository) SetUserMarketingOptIn(ctx context.Context, params sqlc.SetUserMarketingOptInParams) error {
	return r.q.SetUserMarketingOptIn(ctx, params)
}

func (r *repository) LinkBookingRequestsToClient(ctx context.Context, params sqlc.LinkBookingRequestsToClientParams) error {
	return r.q.LinkBookingRequestsToClient(ctx, params)
}

func (r *repository) CreateRefreshToken(ctx context.Context, params sqlc.CreateRefreshTokenParams) (sqlc.RefreshToken, error) {
	return r.q.CreateRefreshToken(ctx, params)
}

func (r *repository) GetBookingStats(ctx context.Context, artistID uuid.UUID) (sqlc.GetBookingStatsRow, error) {
	return r.q.GetBookingStats(ctx, artistID)
}

func (r *repository) CreateAppointment(ctx context.Context, params sqlc.CreateAppointmentParams) (sqlc.Appointment, error) {
	return r.q.CreateAppointment(ctx, params)
}

func (r *repository) GetLatestAppointmentByRequest(ctx context.Context, bookingRequestID uuid.UUID) (sqlc.Appointment, error) {
	return r.q.GetLatestAppointmentByRequest(ctx, bookingRequestID)
}

func (r *repository) ListLiveAppointmentsByRequest(ctx context.Context, bookingRequestID uuid.UUID) ([]sqlc.Appointment, error) {
	return r.q.ListLiveAppointmentsByRequest(ctx, bookingRequestID)
}

func (r *repository) ListLatestAppointmentsByArtist(ctx context.Context, artistID uuid.UUID) ([]sqlc.Appointment, error) {
	return r.q.ListLatestAppointmentsByArtist(ctx, artistID)
}

func (r *repository) ListAppointmentsByArtistInRange(ctx context.Context, params sqlc.ListAppointmentsByArtistInRangeParams) ([]sqlc.ListAppointmentsByArtistInRangeRow, error) {
	return r.q.ListAppointmentsByArtistInRange(ctx, params)
}

func (r *repository) ListBusyAppointmentsByArtistInRange(ctx context.Context, params sqlc.ListBusyAppointmentsByArtistInRangeParams) ([]sqlc.ListBusyAppointmentsByArtistInRangeRow, error) {
	return r.q.ListBusyAppointmentsByArtistInRange(ctx, params)
}

func (r *repository) UpdateAppointmentSchedule(ctx context.Context, params sqlc.UpdateAppointmentScheduleParams) (sqlc.Appointment, error) {
	return r.q.UpdateAppointmentSchedule(ctx, params)
}

func (r *repository) HoldAppointmentSchedule(ctx context.Context, params sqlc.HoldAppointmentScheduleParams) (sqlc.Appointment, error) {
	return r.q.HoldAppointmentSchedule(ctx, params)
}

func (r *repository) ConfirmHeldAppointment(ctx context.Context, bookingRequestID uuid.UUID) (sqlc.Appointment, error) {
	return r.q.ConfirmHeldAppointment(ctx, bookingRequestID)
}

func (r *repository) UpdateAppointmentStatus(ctx context.Context, params sqlc.UpdateAppointmentStatusParams) (sqlc.Appointment, error) {
	return r.q.UpdateAppointmentStatus(ctx, params)
}

func (r *repository) SetAppointmentCalendarEvent(ctx context.Context, params sqlc.SetAppointmentCalendarEventParams) error {
	return r.q.SetAppointmentCalendarEvent(ctx, params)
}

func (r *repository) CountOverlappingAppointments(ctx context.Context, params sqlc.CountOverlappingAppointmentsParams) (int64, error) {
	return r.q.CountOverlappingAppointments(ctx, params)
}

func (r *repository) GetFlash(ctx context.Context, flashID uuid.UUID) (sqlc.Flash, error) {
	return r.q.GetFlash(ctx, flashID)
}

func (r *repository) ClaimFlash(ctx context.Context, params sqlc.ClaimFlashParams) (sqlc.Flash, error) {
	return r.q.ClaimFlash(ctx, params)
}

func (r *repository) DeclineOtherFlashRequests(ctx context.Context, params sqlc.DeclineOtherFlashRequestsParams) error {
	return r.q.DeclineOtherFlashRequests(ctx, params)
}

func (r *repository) ListPaymentRequestsByArtist(ctx context.Context, artistID uuid.UUID) ([]sqlc.PaymentRequest, error) {
	return r.q.ListPaymentRequestsByArtist(ctx, artistID)
}

func (r *repository) ListPaymentRequestsByBooking(ctx context.Context, bookingRequestID uuid.UUID) ([]sqlc.PaymentRequest, error) {
	return r.q.ListPaymentRequestsByBooking(ctx, bookingRequestID)
}

func (r *repository) CreatePaymentRequest(ctx context.Context, params sqlc.CreatePaymentRequestParams) (sqlc.PaymentRequest, error) {
	return r.q.CreatePaymentRequest(ctx, params)
}

func (r *repository) SetBookingDepositAmount(ctx context.Context, params sqlc.SetBookingDepositAmountParams) error {
	return r.q.SetBookingDepositAmount(ctx, params)
}

func (r *repository) SeedMarkPaymentPaid(ctx context.Context, params sqlc.SeedMarkPaymentPaidParams) error {
	return r.q.SeedMarkPaymentPaid(ctx, params)
}
