package bookings

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
)

type Repository interface {
	GetArtistByUserID(ctx context.Context, userID uuid.UUID) (sqlc.Artist, error)
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
	GetBookingRequest(ctx context.Context, params sqlc.GetBookingRequestParams) (sqlc.BookingRequest, error)
	UpdateBookingRequestStatus(ctx context.Context, params sqlc.UpdateBookingRequestStatusParams) (sqlc.BookingRequest, error)
	ReopenBookingRequest(ctx context.Context, params sqlc.ReopenBookingRequestParams) (sqlc.BookingRequest, error)
	GetFlash(ctx context.Context, flashID uuid.UUID) (sqlc.Flash, error)
	ClaimFlash(ctx context.Context, params sqlc.ClaimFlashParams) (sqlc.Flash, error)
	DeclineOtherFlashRequests(ctx context.Context, params sqlc.DeclineOtherFlashRequestsParams) error
	GetBookingStats(ctx context.Context, artistID uuid.UUID) (sqlc.GetBookingStatsRow, error)

	CreateAppointment(ctx context.Context, params sqlc.CreateAppointmentParams) (sqlc.Appointment, error)
	GetLatestAppointmentByRequest(ctx context.Context, bookingRequestID uuid.UUID) (sqlc.Appointment, error)
	ListLiveAppointmentsByRequest(ctx context.Context, bookingRequestID uuid.UUID) ([]sqlc.Appointment, error)
	ListLatestAppointmentsByArtist(ctx context.Context, artistID uuid.UUID) ([]sqlc.Appointment, error)
	ListAppointmentsByArtistInRange(ctx context.Context, params sqlc.ListAppointmentsByArtistInRangeParams) ([]sqlc.ListAppointmentsByArtistInRangeRow, error)
	UpdateAppointmentSchedule(ctx context.Context, params sqlc.UpdateAppointmentScheduleParams) (sqlc.Appointment, error)
	UpdateAppointmentStatus(ctx context.Context, params sqlc.UpdateAppointmentStatusParams) (sqlc.Appointment, error)
	SetAppointmentCalendarEvent(ctx context.Context, params sqlc.SetAppointmentCalendarEventParams) error
	CountOverlappingAppointments(ctx context.Context, params sqlc.CountOverlappingAppointmentsParams) (int64, error)
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

func (r *repository) GetBookingRequest(ctx context.Context, params sqlc.GetBookingRequestParams) (sqlc.BookingRequest, error) {
	return r.q.GetBookingRequest(ctx, params)
}

func (r *repository) UpdateBookingRequestStatus(ctx context.Context, params sqlc.UpdateBookingRequestStatusParams) (sqlc.BookingRequest, error) {
	return r.q.UpdateBookingRequestStatus(ctx, params)
}

func (r *repository) ReopenBookingRequest(ctx context.Context, params sqlc.ReopenBookingRequestParams) (sqlc.BookingRequest, error) {
	return r.q.ReopenBookingRequest(ctx, params)
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

func (r *repository) UpdateAppointmentSchedule(ctx context.Context, params sqlc.UpdateAppointmentScheduleParams) (sqlc.Appointment, error) {
	return r.q.UpdateAppointmentSchedule(ctx, params)
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
