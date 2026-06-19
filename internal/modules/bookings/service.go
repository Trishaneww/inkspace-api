package bookings

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/trishaneupnexx/inkspace-api/internal/clientaccount"
	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/crypto"
	"github.com/trishaneupnexx/inkspace-api/internal/database/sqlc"
	"github.com/trishaneupnexx/inkspace-api/internal/s3client"
)

const presignViewTTL = 1 * time.Hour

var (
	ErrNotFound          = errors.New("inquiry not found")
	ErrNoOpenBook        = errors.New("artist has no open book")
	ErrInvalidSchedule   = errors.New("invalid schedule: a session length is required, and a start time is required when you schedule clients yourself")
	ErrScheduleConflict  = errors.New("that time overlaps an appointment you've already scheduled")
	ErrInvalidInput      = errors.New("missing or invalid appointment details")
	ErrNotAwaitingClient = errors.New("this booking is not awaiting a time selection")
	ErrSlotUnavailable   = errors.New("that time is no longer available")
)

type Service interface {
	ListInquiries(ctx context.Context, userID uuid.UUID) (InquiryListResponse, error)
	ListClientInquiries(ctx context.Context, userID uuid.UUID) (ClientInquiryListResponse, error)
	ListAvailableSlots(ctx context.Context, userID, inquiryID uuid.UUID, date string) (SlotList, error)
	ScheduleClientBooking(ctx context.Context, userID, inquiryID uuid.UUID, input ScheduleBookingInput) (Inquiry, error)
	ListCalendarEvents(ctx context.Context, userID uuid.UUID, rangeStart, rangeEnd time.Time) ([]CalendarEvent, error)
	CreateManualAppointment(ctx context.Context, userID uuid.UUID, input CreateAppointmentInput) (Inquiry, error)
	GetInquiry(ctx context.Context, userID, inquiryID uuid.UUID) (Inquiry, error)
	AcceptInquiry(ctx context.Context, userID, inquiryID uuid.UUID, input AcceptInput) (Inquiry, error)
	RequestConsultation(ctx context.Context, userID, inquiryID uuid.UUID, input RequestConsultationInput) (Inquiry, error)
	RescheduleAppointment(ctx context.Context, userID, inquiryID uuid.UUID, input RescheduleInput) (Inquiry, error)
	ResendScheduleLink(ctx context.Context, userID, inquiryID uuid.UUID) (Inquiry, error)
	GetPublicBookingRequest(ctx context.Context, token string) (PublicBookingRequest, error)
	CreateClientAccountFromBooking(ctx context.Context, token string, input CreateClientAccountInput) (clientaccount.Session, error)
	CancelBooking(ctx context.Context, userID, inquiryID uuid.UUID, appointmentID *uuid.UUID) (Inquiry, error)
	DeclineInquiry(ctx context.Context, userID, inquiryID uuid.UUID) (Inquiry, error)
	ReopenInquiry(ctx context.Context, userID, inquiryID uuid.UUID) (Inquiry, error)
	SeedDevInquiries(ctx context.Context, userID uuid.UUID) (int, error)
}

type service struct {
	repo     Repository
	s3       *s3client.Client
	calendar *calendarClient
	cfg      *config.Config
	log      *slog.Logger
}

func NewService(repo Repository, s3 *s3client.Client, cfg *config.Config, cipher *crypto.Cipher) Service {
	return &service{
		repo:     repo,
		s3:       s3,
		calendar: newCalendarClient(cfg, cipher),
		cfg:      cfg,
		log:      slog.Default(),
	}
}

func (s *service) ListInquiries(ctx context.Context, userID uuid.UUID) (InquiryListResponse, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return InquiryListResponse{}, err
	}

	rows, err := s.repo.ListBookingRequestsByArtist(ctx, artist.ID)
	if err != nil {
		return InquiryListResponse{}, err
	}
	statsRow, err := s.repo.GetBookingStats(ctx, artist.ID)
	if err != nil {
		return InquiryListResponse{}, err
	}

	inquiries := make([]Inquiry, 0, len(rows))
	for _, row := range rows {
		inquiries = append(inquiries, inquiryFromRow(row))
	}
	s.attachLocations(ctx, artist.ID, inquiries)
	s.attachAppointments(ctx, artist.ID, inquiries)
	s.attachPaymentRequests(ctx, artist.ID, inquiries)
	return InquiryListResponse{Inquiries: inquiries, Stats: statsFromRow(statsRow)}, nil
}

func (s *service) ListClientInquiries(ctx context.Context, userID uuid.UUID) (ClientInquiryListResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return ClientInquiryListResponse{}, err
	}
	rows, err := s.repo.ListBookingRequestsByClientEmail(ctx, user.Email)
	if err != nil {
		return ClientInquiryListResponse{}, err
	}

	artists := make(map[uuid.UUID]artistRef)
	inquiries := make([]ClientInquiry, 0, len(rows))
	for _, row := range rows {
		inquiry := inquiryFromRow(row)
		s.attachClientPayments(ctx, row.ID, &inquiry)
		if appt, err := s.repo.GetLatestAppointmentByRequest(ctx, row.ID); err == nil {
			inquiry.Appointment = appointmentFromRow(appt)
		}
		ref := s.resolveArtistRef(ctx, row.ArtistID, artists)
		inquiries = append(inquiries, ClientInquiry{
			Inquiry:    inquiry,
			ArtistName: ref.name,
			ArtistSlug: ref.slug,
		})
	}
	return ClientInquiryListResponse{Inquiries: inquiries}, nil
}

func (s *service) attachClientPayments(ctx context.Context, bookingID uuid.UUID, inquiry *Inquiry) {
	payments, err := s.repo.ListPaymentRequestsByBooking(ctx, bookingID)
	if err != nil || len(payments) == 0 {
		return
	}
	list := make([]InquiryPayment, 0, len(payments))
	for _, p := range payments {
		list = append(list, paymentFromRow(p))
	}
	inquiry.Payments = list
}

type artistRef struct {
	name string
	slug string
}

func (s *service) resolveArtistRef(ctx context.Context, artistID uuid.UUID, cache map[uuid.UUID]artistRef) artistRef {
	if ref, ok := cache[artistID]; ok {
		return ref
	}
	ref := artistRef{name: s.artistDisplayName(ctx, artistID)}
	if book, err := s.repo.GetOpenBookByArtist(ctx, artistID); err == nil {
		ref.slug = book.Slug
	}
	cache[artistID] = ref
	return ref
}

func (s *service) artistDisplayName(ctx context.Context, artistID uuid.UUID) string {
	artist, err := s.repo.GetArtistByID(ctx, artistID)
	if err != nil {
		return ""
	}
	user, err := s.repo.GetUserByID(ctx, artist.UserID)
	if err != nil {
		return ""
	}
	name := strings.TrimSpace(strings.Join([]string{stringOrEmpty(user.FirstName), stringOrEmpty(user.LastName)}, " "))
	if name != "" {
		return name
	}
	return stringOrEmpty(user.Username)
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func newScheduleToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s *service) ListCalendarEvents(ctx context.Context, userID uuid.UUID, rangeStart, rangeEnd time.Time) ([]CalendarEvent, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return nil, err
	}

	rows, err := s.repo.ListAppointmentsByArtistInRange(ctx, sqlc.ListAppointmentsByArtistInRangeParams{
		ArtistID:   artist.ID,
		RangeStart: pgtype.Timestamptz{Time: rangeStart, Valid: true},
		RangeEnd:   pgtype.Timestamptz{Time: rangeEnd, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	events := make([]CalendarEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, calendarEventFromRow(row))
	}
	return events, nil
}

func (s *service) CreateManualAppointment(ctx context.Context, userID uuid.UUID, input CreateAppointmentInput) (Inquiry, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return Inquiry{}, err
	}
	book, err := s.requireOpenBook(ctx, artist.ID)
	if err != nil {
		return Inquiry{}, err
	}

	apptType := input.Type
	if apptType != string(AppointmentConsultation) && apptType != string(AppointmentSession) {
		return Inquiry{}, ErrInvalidInput
	}
	if strings.TrimSpace(input.ClientName) == "" || strings.TrimSpace(input.ClientEmail) == "" {
		return Inquiry{}, ErrInvalidInput
	}
	if input.ScheduledStart == nil || input.DurationMinutes <= 0 {
		return Inquiry{}, ErrInvalidSchedule
	}

	var format *string
	if apptType == string(AppointmentConsultation) {
		f := defaultConsultationFormat
		if input.Format != nil && *input.Format != "" {
			if !validFormat(*input.Format) {
				return Inquiry{}, ErrInvalidSchedule
			}
			f = *input.Format
		}
		format = &f
	}

	locationID := pgtype.UUID{}
	if input.LocationID != nil && *input.LocationID != "" {
		parsed, err := uuid.Parse(*input.LocationID)
		if err != nil {
			return Inquiry{}, ErrInvalidInput
		}
		locationID = pgtype.UUID{Bytes: parsed, Valid: true}
	}

	start := pgtype.Timestamptz{Time: *input.ScheduledStart, Valid: true}
	if err := s.assertNoConflict(ctx, artist.ID, uuid.Nil, start, input.DurationMinutes); err != nil {
		return Inquiry{}, err
	}

	var sessionDuration *int32
	if apptType == string(AppointmentSession) {
		d := input.DurationMinutes
		sessionDuration = &d
	}

	req, err := s.repo.CreateBookingRequest(ctx, sqlc.CreateBookingRequestParams{
		ArtistID:               artist.ID,
		OpenBookID:             book.ID,
		Type:                   string(RequestTypeCustom),
		Description:            input.Description,
		ReferenceImageKeys:     []string{},
		Placement:              input.Placement,
		ApproxSizeInches:       input.ApproxSizeInches,
		ColorType:              input.ColorType,
		LocationID:             locationID,
		Styles:                 []string{},
		ClientAvailability:     []byte("[]"),
		CustomAnswers:          []byte("[]"),
		ClientName:             strings.TrimSpace(input.ClientName),
		ClientEmail:            strings.TrimSpace(input.ClientEmail),
		ClientPhone:            input.ClientPhone,
		Status:                 "accepted",
		DepositStatus:          "not_required",
		WaiverStatus:           "not_required",
		SessionDurationMinutes: sessionDuration,
	})
	if err != nil {
		return Inquiry{}, err
	}

	appt, err := s.repo.CreateAppointment(ctx, sqlc.CreateAppointmentParams{
		ArtistID:         artist.ID,
		BookingRequestID: req.ID,
		Type:             apptType,
		Status:           appointmentScheduled,
		ScheduledStart:   start,
		DurationMinutes:  input.DurationMinutes,
		Format:           format,
		SchedulingOrigin: originArtistSet,
	})
	if err != nil {
		return Inquiry{}, err
	}

	inquiry, err := s.GetInquiry(ctx, userID, req.ID)
	if err != nil {
		return Inquiry{}, err
	}
	s.syncAppointmentToCalendar(ctx, artist.ID, appt, inquiry)
	return inquiry, nil
}

// attachAppointments hangs the latest appointment off each list inquiry so the
// inbox can distinguish a scheduled booking from one still proposed (awaiting
// the client's self-booking).
func (s *service) attachAppointments(ctx context.Context, artistID uuid.UUID, inquiries []Inquiry) {
	appointments, err := s.repo.ListLatestAppointmentsByArtist(ctx, artistID)
	if err != nil || len(appointments) == 0 {
		return
	}
	byRequest := make(map[string]sqlc.Appointment, len(appointments))
	for _, a := range appointments {
		byRequest[a.BookingRequestID.String()] = a
	}
	for i := range inquiries {
		if a, ok := byRequest[inquiries[i].ID]; ok {
			inquiries[i].Appointment = appointmentFromRow(a)
		}
	}
}

func (s *service) attachPaymentRequests(ctx context.Context, artistID uuid.UUID, inquiries []Inquiry) {
	payments, err := s.repo.ListPaymentRequestsByArtist(ctx, artistID)
	if err != nil || len(payments) == 0 {
		return
	}
	byRequest := make(map[string][]InquiryPayment)
	for _, p := range payments {
		key := p.BookingRequestID.String()
		byRequest[key] = append(byRequest[key], paymentFromRow(p))
	}
	for i := range inquiries {
		if list, ok := byRequest[inquiries[i].ID]; ok {
			inquiries[i].Payments = list
		}
	}
}

func (s *service) attachLocations(ctx context.Context, artistID uuid.UUID, inquiries []Inquiry) {
	locations, err := s.repo.ListAllArtistLocations(ctx, artistID)
	if err != nil || len(locations) == 0 {
		return
	}
	byID := make(map[string]sqlc.ArtistLocation, len(locations))
	for _, l := range locations {
		byID[l.ID.String()] = l
	}
	for i := range inquiries {
		if loc, ok := byID[inquiries[i].LocationID]; ok {
			inquiries[i].Location = inquiryLocationFromRow(loc)
		}
	}
}

func (s *service) GetInquiry(ctx context.Context, userID, inquiryID uuid.UUID) (Inquiry, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return Inquiry{}, err
	}
	row, err := s.repo.GetBookingRequest(ctx, sqlc.GetBookingRequestParams{ID: inquiryID, ArtistID: artist.ID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Inquiry{}, ErrNotFound
		}
		return Inquiry{}, err
	}
	return s.enrichInquiry(ctx, row)
}

func (s *service) AcceptInquiry(ctx context.Context, userID, inquiryID uuid.UUID, input AcceptInput) (Inquiry, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return Inquiry{}, err
	}
	book, err := s.requireOpenBook(ctx, artist.ID)
	if err != nil {
		return Inquiry{}, err
	}

	duration := int32Value(input.SessionDurationMinutes)
	if duration <= 0 {
		return Inquiry{}, ErrInvalidSchedule
	}
	mode := book.SchedulingMode
	if input.ClientScheduled != nil {
		if *input.ClientScheduled {
			mode = schedulingClient
		} else {
			mode = schedulingArtist
		}
	}
	apptStatus, origin, start, err := resolveSchedule(mode, input.ScheduledStart)
	if err != nil {
		return Inquiry{}, err
	}
	if err := s.assertNoConflict(ctx, artist.ID, inquiryID, start, duration); err != nil {
		return Inquiry{}, err
	}

	appt, err := s.repo.CreateAppointment(ctx, sqlc.CreateAppointmentParams{
		ArtistID:         artist.ID,
		BookingRequestID: inquiryID,
		Type:             string(AppointmentSession),
		Status:           apptStatus,
		ScheduledStart:   start,
		DurationMinutes:  duration,
		SchedulingOrigin: origin,
	})
	if err != nil {
		return Inquiry{}, err
	}

	depositStatus := s.resolveDepositStatus(ctx, artist.ID)
	inquiry, err := s.updateStatus(ctx, artist.ID, inquiryID, sqlc.UpdateBookingRequestStatusParams{
		Status:                 "accepted",
		SessionDurationMinutes: &duration,
		DepositStatus:          &depositStatus,
	})
	if err != nil {
		return Inquiry{}, err
	}
	if inquiry.Type == RequestTypeFlash && inquiry.FlashID != nil {
		if err := s.claimFlash(ctx, artist.ID, inquiryID, *inquiry.FlashID); err != nil {
			return Inquiry{}, err
		}
	}
	if mode == schedulingClient {
		s.sendScheduleInvite(ctx, artist.ID, inquiryID, duration)
	}
	s.syncAppointmentToCalendar(ctx, artist.ID, appt, inquiry)
	return inquiry, nil
}

func (s *service) RequestConsultation(ctx context.Context, userID, inquiryID uuid.UUID, input RequestConsultationInput) (Inquiry, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return Inquiry{}, err
	}

	duration := defaultConsultationDurationMinutes
	if input.DurationMinutes != nil && *input.DurationMinutes > 0 {
		duration = *input.DurationMinutes
	}
	format := defaultConsultationFormat
	if input.Format != nil && *input.Format != "" {
		format = *input.Format
	}
	if !validFormat(format) {
		return Inquiry{}, ErrInvalidSchedule
	}

	apptStatus, origin, start, err := resolveSchedule(schedulingArtist, input.ScheduledStart)
	if err != nil {
		return Inquiry{}, err
	}
	if err := s.assertNoConflict(ctx, artist.ID, inquiryID, start, duration); err != nil {
		return Inquiry{}, err
	}

	appt, err := s.repo.CreateAppointment(ctx, sqlc.CreateAppointmentParams{
		ArtistID:         artist.ID,
		BookingRequestID: inquiryID,
		Type:             string(AppointmentConsultation),
		Status:           apptStatus,
		ScheduledStart:   start,
		DurationMinutes:  duration,
		Format:           &format,
		SchedulingOrigin: origin,
	})
	if err != nil {
		return Inquiry{}, err
	}

	inquiry, err := s.updateStatus(ctx, artist.ID, inquiryID, sqlc.UpdateBookingRequestStatusParams{
		Status: "consultation_requested",
	})
	if err != nil {
		return Inquiry{}, err
	}
	s.syncAppointmentToCalendar(ctx, artist.ID, appt, inquiry)
	return inquiry, nil
}

func (s *service) RescheduleAppointment(ctx context.Context, userID, inquiryID uuid.UUID, input RescheduleInput) (Inquiry, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return Inquiry{}, err
	}
	if input.ScheduledStart == nil {
		return Inquiry{}, ErrInvalidSchedule
	}
	if input.Format != nil && *input.Format != "" && !validFormat(*input.Format) {
		return Inquiry{}, ErrInvalidSchedule
	}

	appt, err := s.resolveRescheduleTarget(ctx, inquiryID, input.AppointmentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Inquiry{}, ErrNotFound
		}
		return Inquiry{}, err
	}
	if appt.ArtistID != artist.ID {
		return Inquiry{}, ErrNotFound
	}

	duration := int32Value(input.DurationMinutes)
	if duration <= 0 {
		duration = appt.DurationMinutes
	}

	start := pgtype.Timestamptz{Time: *input.ScheduledStart, Valid: true}
	if err := s.assertNoConflict(ctx, artist.ID, inquiryID, start, duration); err != nil {
		return Inquiry{}, err
	}

	synced, err := s.repo.UpdateAppointmentSchedule(ctx, sqlc.UpdateAppointmentScheduleParams{
		ID:              appt.ID,
		ArtistID:        artist.ID,
		ScheduledStart:  start,
		DurationMinutes: input.DurationMinutes,
		Format:          input.Format,
	})
	if err != nil {
		return Inquiry{}, err
	}

	inquiry, err := s.GetInquiry(ctx, userID, inquiryID)
	if err != nil {
		return Inquiry{}, err
	}
	s.syncAppointmentToCalendar(ctx, artist.ID, synced, inquiry)
	return inquiry, nil
}

func (s *service) ResendScheduleLink(ctx context.Context, userID, inquiryID uuid.UUID) (Inquiry, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return Inquiry{}, err
	}
	row, err := s.repo.GetBookingRequest(ctx, sqlc.GetBookingRequestParams{ID: inquiryID, ArtistID: artist.ID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Inquiry{}, ErrNotFound
		}
		return Inquiry{}, err
	}
	if row.ScheduleToken == nil || *row.ScheduleToken == "" {
		return Inquiry{}, ErrNotFound
	}

	updated, err := s.repo.TouchScheduleEmailedAt(ctx, sqlc.TouchScheduleEmailedAtParams{ID: inquiryID, ArtistID: artist.ID})
	if err != nil {
		return Inquiry{}, err
	}
	duration := int32(0)
	if row.SessionDurationMinutes != nil {
		duration = *row.SessionDurationMinutes
	}
	s.sendBookingRequestEmail(updated, s.artistDisplayName(ctx, artist.ID), duration)
	return s.GetInquiry(ctx, userID, inquiryID)
}

func (s *service) ListAvailableSlots(ctx context.Context, userID, inquiryID uuid.UUID, date string) (SlotList, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return SlotList{}, err
	}
	row, err := s.repo.GetClientBookingRequest(ctx, sqlc.GetClientBookingRequestParams{
		ID:          inquiryID,
		ClientEmail: user.Email,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SlotList{}, ErrNotFound
		}
		return SlotList{}, err
	}

	appt, err := s.repo.GetLatestAppointmentByRequest(ctx, inquiryID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SlotList{}, ErrNotAwaitingClient
		}
		return SlotList{}, err
	}
	if appt.Status != appointmentProposed || appt.ScheduledStart.Valid {
		return SlotList{}, ErrNotAwaitingClient
	}

	settings, err := s.repo.GetArtistSettings(ctx, row.ArtistID)
	if err != nil {
		return SlotList{}, err
	}
	windows, err := s.repo.ListAvailabilityWindows(ctx, row.ArtistID)
	if err != nil {
		return SlotList{}, err
	}

	loc := loadLocation(settings.Timezone)
	day, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return SlotList{}, ErrInvalidInput
	}
	busy, err := s.repo.ListAppointmentsByArtistInRange(ctx, sqlc.ListAppointmentsByArtistInRangeParams{
		ArtistID:   row.ArtistID,
		RangeStart: pgtype.Timestamptz{Time: day, Valid: true},
		RangeEnd:   pgtype.Timestamptz{Time: day.AddDate(0, 0, 1), Valid: true},
	})
	if err != nil {
		return SlotList{}, err
	}

	slots := buildSlots(day, loc, windows, busy, appt.DurationMinutes, settings, time.Now())
	return SlotList{Slots: slots, DurationMinutes: appt.DurationMinutes}, nil
}

func (s *service) ScheduleClientBooking(ctx context.Context, userID, inquiryID uuid.UUID, input ScheduleBookingInput) (Inquiry, error) {
	if input.ScheduledStart.IsZero() {
		return Inquiry{}, ErrInvalidInput
	}
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return Inquiry{}, err
	}
	row, err := s.repo.GetClientBookingRequest(ctx, sqlc.GetClientBookingRequestParams{
		ID:          inquiryID,
		ClientEmail: user.Email,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Inquiry{}, ErrNotFound
		}
		return Inquiry{}, err
	}

	appt, err := s.repo.GetLatestAppointmentByRequest(ctx, inquiryID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Inquiry{}, ErrNotAwaitingClient
		}
		return Inquiry{}, err
	}
	if appt.Status != appointmentProposed || appt.ScheduledStart.Valid {
		return Inquiry{}, ErrNotAwaitingClient
	}

	settings, err := s.repo.GetArtistSettings(ctx, row.ArtistID)
	if err != nil {
		return Inquiry{}, err
	}
	windows, err := s.repo.ListAvailabilityWindows(ctx, row.ArtistID)
	if err != nil {
		return Inquiry{}, err
	}

	loc := loadLocation(settings.Timezone)
	startLocal := input.ScheduledStart.In(loc)
	day := time.Date(startLocal.Year(), startLocal.Month(), startLocal.Day(), 0, 0, 0, 0, loc)
	busy, err := s.repo.ListAppointmentsByArtistInRange(ctx, sqlc.ListAppointmentsByArtistInRangeParams{
		ArtistID:   row.ArtistID,
		RangeStart: pgtype.Timestamptz{Time: day, Valid: true},
		RangeEnd:   pgtype.Timestamptz{Time: day.AddDate(0, 0, 1), Valid: true},
	})
	if err != nil {
		return Inquiry{}, err
	}
	slots := buildSlots(day, loc, windows, busy, appt.DurationMinutes, settings, time.Now())
	if !containsSlot(slots, input.ScheduledStart.UTC().Format(time.RFC3339)) {
		return Inquiry{}, ErrSlotUnavailable
	}

	synced, err := s.repo.UpdateAppointmentSchedule(ctx, sqlc.UpdateAppointmentScheduleParams{
		ID:             appt.ID,
		ArtistID:       row.ArtistID,
		ScheduledStart: pgtype.Timestamptz{Time: input.ScheduledStart, Valid: true},
	})
	if err != nil {
		return Inquiry{}, err
	}

	inquiry := inquiryFromRow(row)
	inquiry.Appointment = appointmentFromRow(synced)
	s.syncAppointmentToCalendar(ctx, row.ArtistID, synced, inquiry)

	whenLabel := startLocal.Format("Monday, January 2 at 3:04 PM")
	s.sendBookingConfirmedEmail(row, s.artistDisplayName(ctx, row.ArtistID), whenLabel, appt.DurationMinutes)
	return inquiry, nil
}

func (s *service) GetPublicBookingRequest(ctx context.Context, token string) (PublicBookingRequest, error) {
	row, err := s.repo.GetBookingRequestByScheduleToken(ctx, &token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PublicBookingRequest{}, ErrNotFound
		}
		return PublicBookingRequest{}, err
	}
	email := clientaccount.NormalizeEmail(row.ClientEmail)
	_, userErr := s.repo.GetUserByEmail(ctx, email)

	duration := int32(0)
	if row.SessionDurationMinutes != nil {
		duration = *row.SessionDurationMinutes
	}
	return PublicBookingRequest{
		ArtistName:      s.artistDisplayName(ctx, row.ArtistID),
		ClientEmail:     email,
		ClientName:      row.ClientName,
		Status:          row.Status,
		DurationMinutes: duration,
		HasAccount:      userErr == nil,
	}, nil
}

func (s *service) CreateClientAccountFromBooking(ctx context.Context, token string, input CreateClientAccountInput) (clientaccount.Session, error) {
	row, err := s.repo.GetBookingRequestByScheduleToken(ctx, &token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return clientaccount.Session{}, ErrNotFound
		}
		return clientaccount.Session{}, err
	}
	return clientaccount.Create(ctx, s.repo, s.cfg, row.ClientEmail, clientaccount.Input{
		FirstName:      input.FirstName,
		LastName:       input.LastName,
		Phone:          input.Phone,
		Password:       input.Password,
		MarketingOptIn: input.MarketingOptIn,
	}, "book_page")
}

func (s *service) resolveRescheduleTarget(ctx context.Context, inquiryID uuid.UUID, appointmentID *string) (sqlc.Appointment, error) {
	if appointmentID == nil || *appointmentID == "" {
		return s.repo.GetLatestAppointmentByRequest(ctx, inquiryID)
	}
	id, err := uuid.Parse(*appointmentID)
	if err != nil {
		return sqlc.Appointment{}, ErrInvalidInput
	}
	appts, err := s.repo.ListLiveAppointmentsByRequest(ctx, inquiryID)
	if err != nil {
		return sqlc.Appointment{}, err
	}
	for _, a := range appts {
		if a.ID == id {
			return a, nil
		}
	}
	return sqlc.Appointment{}, pgx.ErrNoRows
}

func (s *service) CancelBooking(ctx context.Context, userID, inquiryID uuid.UUID, appointmentID *uuid.UUID) (Inquiry, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return Inquiry{}, err
	}

	live, err := s.repo.ListLiveAppointmentsByRequest(ctx, inquiryID)
	if err != nil {
		return Inquiry{}, err
	}
	if len(live) == 0 {
		return Inquiry{}, ErrNotFound
	}

	target := live[len(live)-1]
	if appointmentID != nil {
		found := false
		for _, a := range live {
			if a.ID == *appointmentID {
				target = a
				found = true
				break
			}
		}
		if !found {
			return Inquiry{}, ErrNotFound
		}
	}
	if target.ArtistID != artist.ID {
		return Inquiry{}, ErrNotFound
	}

	s.removeAppointmentFromCalendar(ctx, artist.ID, target)
	if _, err := s.repo.UpdateAppointmentStatus(ctx, sqlc.UpdateAppointmentStatusParams{
		ID:       target.ID,
		ArtistID: artist.ID,
		Status:   "cancelled",
	}); err != nil {
		return Inquiry{}, err
	}

	for _, a := range live {
		if a.ID != target.ID {
			return s.GetInquiry(ctx, userID, inquiryID)
		}
	}
	return s.updateStatus(ctx, artist.ID, inquiryID, sqlc.UpdateBookingRequestStatusParams{
		Status: "cancelled",
	})
}

// resolveSchedule maps the artist's scheduling mode to the appointment's
// initial status, origin, and (when the artist places it) start time.
func resolveSchedule(mode string, start *time.Time) (status, origin string, scheduledStart pgtype.Timestamptz, err error) {
	if mode == schedulingArtist {
		if start == nil {
			return "", "", pgtype.Timestamptz{}, ErrInvalidSchedule
		}
		return appointmentScheduled, originArtistSet, pgtype.Timestamptz{Time: *start, Valid: true}, nil
	}
	// client_scheduled: the client self-books a start later, so it stays proposed.
	return appointmentProposed, originClientBooked, pgtype.Timestamptz{}, nil
}

// assertNoConflict blocks placing an appointment whose window overlaps one the
// artist already has scheduled. Client-scheduled slots have no start yet, so
// there is nothing to conflict with. The request's own appointments are excluded
// (e.g. accepting a session after its consultation was scheduled).
func (s *service) assertNoConflict(ctx context.Context, artistID, requestID uuid.UUID, start pgtype.Timestamptz, durationMinutes int32) error {
	if !start.Valid {
		return nil
	}
	windowEnd := pgtype.Timestamptz{
		Time:  start.Time.Add(time.Duration(durationMinutes) * time.Minute),
		Valid: true,
	}
	count, err := s.repo.CountOverlappingAppointments(ctx, sqlc.CountOverlappingAppointmentsParams{
		ArtistID:         artistID,
		ExcludeRequestID: requestID,
		WindowStart:      start,
		WindowEnd:        windowEnd,
	})
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrScheduleConflict
	}
	return nil
}

func validFormat(format string) bool {
	switch format {
	case "in_person", "online", "phone":
		return true
	default:
		return false
	}
}

func int32Value(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

func (s *service) requireOpenBook(ctx context.Context, artistID uuid.UUID) (sqlc.OpenBook, error) {
	book, err := s.repo.GetOpenBookByArtist(ctx, artistID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.OpenBook{}, ErrNoOpenBook
		}
		return sqlc.OpenBook{}, err
	}
	return book, nil
}

func (s *service) resolveDepositStatus(ctx context.Context, artistID uuid.UUID) string {
	if err := s.repo.EnsureArtistSettings(ctx, artistID); err == nil {
		if settings, err := s.repo.GetArtistSettings(ctx, artistID); err == nil {
			if settings.DepositFlatFeeCents != nil && *settings.DepositFlatFeeCents > 0 {
				return "pending"
			}
		}
	}
	return "not_required"
}

func (s *service) claimFlash(ctx context.Context, artistID, bookingID uuid.UUID, flashIDStr string) error {
	flashID, err := uuid.Parse(flashIDStr)
	if err != nil {
		return nil
	}
	flash, err := s.repo.ClaimFlash(ctx, sqlc.ClaimFlashParams{
		ID:                 flashID,
		ClaimedByBookingID: pgtype.UUID{Bytes: bookingID, Valid: true},
	})
	if err != nil {
		// Already claimed/unavailable — nothing more to do.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if flash.Repeatable {
		return nil
	}
	return s.repo.DeclineOtherFlashRequests(ctx, sqlc.DeclineOtherFlashRequestsParams{
		ArtistID:  artistID,
		FlashID:   pgtype.UUID{Bytes: flashID, Valid: true},
		ExcludeID: bookingID,
	})
}

func (s *service) DeclineInquiry(ctx context.Context, userID, inquiryID uuid.UUID) (Inquiry, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return Inquiry{}, err
	}
	return s.updateStatus(ctx, artist.ID, inquiryID, sqlc.UpdateBookingRequestStatusParams{Status: "declined"})
}

func (s *service) ReopenInquiry(ctx context.Context, userID, inquiryID uuid.UUID) (Inquiry, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return Inquiry{}, err
	}
	row, err := s.repo.ReopenBookingRequest(ctx, sqlc.ReopenBookingRequestParams{ID: inquiryID, ArtistID: artist.ID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Inquiry{}, ErrNotFound
		}
		return Inquiry{}, err
	}
	return s.enrichInquiry(ctx, row)
}

func (s *service) updateStatus(ctx context.Context, artistID, inquiryID uuid.UUID, params sqlc.UpdateBookingRequestStatusParams) (Inquiry, error) {
	params.ID = inquiryID
	params.ArtistID = artistID
	row, err := s.repo.UpdateBookingRequestStatus(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Inquiry{}, ErrNotFound
		}
		return Inquiry{}, err
	}
	return s.enrichInquiry(ctx, row)
}

func (s *service) enrichInquiry(ctx context.Context, row sqlc.BookingRequest) (Inquiry, error) {
	inquiry := inquiryFromRow(row)
	if inquiry.LocationID != "" {
		if locations, err := s.repo.ListAllArtistLocations(ctx, row.ArtistID); err == nil {
			for _, l := range locations {
				if l.ID.String() == inquiry.LocationID {
					inquiry.Location = inquiryLocationFromRow(l)
					break
				}
			}
		}
	}
	if err := s.attachFlash(ctx, row, &inquiry); err != nil {
		return Inquiry{}, err
	}
	if err := s.attachSchedulingContext(ctx, row, &inquiry); err != nil {
		return Inquiry{}, err
	}
	if payments, err := s.repo.ListPaymentRequestsByBooking(ctx, row.ID); err == nil {
		list := make([]InquiryPayment, 0, len(payments))
		for _, p := range payments {
			list = append(list, paymentFromRow(p))
		}
		inquiry.Payments = list
	}
	if s.s3 == nil {
		return inquiry, nil
	}
	for _, key := range inquiry.ReferenceImageKeys {
		url, err := s.s3.PresignGet(ctx, key, presignViewTTL)
		if err != nil {
			return Inquiry{}, fmt.Errorf("presign reference image: %w", err)
		}
		inquiry.ReferenceImageURLs = append(inquiry.ReferenceImageURLs, url)
	}
	return inquiry, nil
}

// attachSchedulingContext hangs the artist's scheduling mode, consultation
// policy, and the request's latest appointment off the inquiry so the inbox can
// drive the accept / consultation actions and show the scheduled outcome.
func (s *service) attachSchedulingContext(ctx context.Context, row sqlc.BookingRequest, inquiry *Inquiry) error {
	if book, err := s.repo.GetOpenBookByArtist(ctx, row.ArtistID); err == nil {
		inquiry.SchedulingMode = book.SchedulingMode
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	if windows, err := s.repo.ListAvailabilityWindows(ctx, row.ArtistID); err == nil {
		inquiry.ArtistAvailability = availabilityWindowsFromRows(windows)
	}

	if live, err := s.repo.ListLiveAppointmentsByRequest(ctx, row.ID); err == nil {
		inquiry.LiveAppointments = make([]Appointment, 0, len(live))
		for _, a := range live {
			inquiry.LiveAppointments = append(inquiry.LiveAppointments, *appointmentFromRow(a))
		}
	}

	appt, err := s.repo.GetLatestAppointmentByRequest(ctx, row.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	inquiry.Appointment = appointmentFromRow(appt)
	return nil
}

// attachFlash hangs the claimed flash's title, images, and chosen size off a
// flash inquiry so the inbox can show the design instead of reference photos.
func (s *service) attachFlash(ctx context.Context, row sqlc.BookingRequest, inquiry *Inquiry) error {
	if row.Type != string(RequestTypeFlash) || !row.FlashID.Valid {
		return nil
	}
	flash, err := s.repo.GetFlash(ctx, uuid.UUID(row.FlashID.Bytes))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	out := &InquiryFlash{
		Title:     flash.Title,
		ImageURLs: []string{},
		SizeCode:  row.FlashSizeCode,
	}
	if s.s3 != nil {
		for _, key := range []*string{flash.S3Key, flash.ReferenceS3Key} {
			if key == nil || *key == "" {
				continue
			}
			url, err := s.s3.PresignGet(ctx, *key, presignViewTTL)
			if err != nil {
				return fmt.Errorf("presign flash image: %w", err)
			}
			out.ImageURLs = append(out.ImageURLs, url)
		}
	}
	inquiry.Flash = out
	return nil
}

func (s *service) requireArtist(ctx context.Context, userID uuid.UUID) (sqlc.Artist, error) {
	artist, err := s.repo.GetArtistByUserID(ctx, userID)
	if err == nil {
		return artist, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Artist{}, err
	}
	if err := s.repo.EnsureArtist(ctx, userID); err != nil {
		return sqlc.Artist{}, err
	}
	return s.repo.GetArtistByUserID(ctx, userID)
}

// ── Dev seed ─────────────────────────────────────────────────────────────────
func (s *service) SeedDevInquiries(ctx context.Context, userID uuid.UUID) (int, error) {
	artist, err := s.requireArtist(ctx, userID)
	if err != nil {
		return 0, err
	}
	book, err := s.repo.GetOpenBookByArtist(ctx, artist.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNoOpenBook
		}
		return 0, err
	}

	locations, err := s.repo.ListAllArtistLocations(ctx, artist.ID)
	if err != nil {
		return 0, err
	}

	availability := []byte(`[{"weekday":1,"ranges":[["600","1080"]]},{"weekday":4,"ranges":[["600","1080"]]}]`)

	samples := []struct {
		name, email, placement, description, pieceType, colorType string
		size                                                      int32
	}{
		{"Maya Chen", "maya@example.com", "forearm", "Fine-line botanical half sleeve, black and grey.", "custom", "black_and_grey", 8},
		{"Olivia West", "olivia@example.com", "spine", "Delicate wildflower trail down the spine.", "custom", "black_and_grey", 6},
		{"Hugo Reyes", "hugo@example.com", "calf", "Neo-traditional fox among autumn leaves.", "custom", "color", 8},
		{"Riley Quinn", "riley@example.com", "thigh", "Ornamental mandala, dotwork.", "custom", "black_and_grey", 7},
		{"Sam Rivera", "sam@example.com", "calf", "Traditional eagle, full colour.", "custom", "color", 10},
		{"Mason Cole", "mason@example.com", "ribs", "Loves the koi flash piece — slightly enlarged.", "flash", "color", 6},
		{"Emma Ford", "emma@example.com", "forearm", "Minimalist line portrait of her dog.", "custom", "black_and_grey", 4},
		{"Avery Brooks", "avery@example.com", "wrist", "Matching script with partner.", "custom", "either", 3},
		{"Maya Chen", "maya@example.com", "upper arm", "Second piece — continues the botanical sleeve.", "custom", "black_and_grey", 7},
	}

	customAnswers := []byte(`[{"prompt":"Is this your first tattoo?","answer":"No — this is my third."}]`)

	type sessionSeed struct {
		completedDaysAgo []int
		upcomingInDays   int
	}
	sessionsByIndex := map[int]sessionSeed{
		0: {completedDaysAgo: []int{21}},  // Maya — most recent piece (healing)
		2: {upcomingInDays: 5},            // Hugo — booked
		3: {completedDaysAgo: []int{63}},  // Riley — touch-up due
		4: {completedDaysAgo: []int{240}}, // Sam — settled
		6: {completedDaysAgo: []int{34}},  // Emma — touch-up due
		7: {upcomingInDays: 10},           // Avery — booked
		8: {completedDaysAgo: []int{120}}, // Maya — earlier piece, makes her a repeat client
	}

	for i, sample := range samples {
		size := sample.size
		phone := "+1 (555) 010-00" + strconv.Itoa(10+i)

		styles := []string{}
		if sample.pieceType == string(RequestTypeCustom) {
			styles = []string{"fine_line", "blackwork"}
		}

		locationID := pgtype.UUID{}
		if len(locations) > 0 {
			locationID = pgtype.UUID{Bytes: locations[i%len(locations)].ID, Valid: true}
		}

		created, err := s.repo.CreateBookingRequest(ctx, sqlc.CreateBookingRequestParams{
			ArtistID:           artist.ID,
			OpenBookID:         book.ID,
			Type:               sample.pieceType,
			Description:        sample.description,
			ReferenceImageKeys: []string{},
			Placement:          sample.placement,
			ApproxSizeInches:   &size,
			ColorType:          sample.colorType,
			LocationID:         locationID,
			Styles:             styles,
			ClientAvailability: availability,
			CustomAnswers:      customAnswers,
			ClientName:         sample.name,
			ClientEmail:        sample.email,
			ClientPhone:        phone,
			Status:             "pending",
			DepositStatus:      "not_required",
			WaiverStatus:       "not_required",
		})
		if err != nil {
			return i, fmt.Errorf("seed inquiry %d: %w", i, err)
		}

		switch i {
		case 1, 2:
			depositPending := "pending"
			if _, err := s.repo.UpdateBookingRequestStatus(ctx, sqlc.UpdateBookingRequestStatusParams{
				ID:            created.ID,
				ArtistID:      artist.ID,
				Status:        "accepted",
				DepositStatus: &depositPending,
			}); err != nil {
				return i, err
			}
		case 5:
			if _, err := s.repo.UpdateBookingRequestStatus(ctx, sqlc.UpdateBookingRequestStatusParams{
				ID:       created.ID,
				ArtistID: artist.ID,
				Status:   "declined",
			}); err != nil {
				return i, err
			}
		}

		seed := sessionsByIndex[i]
		for _, daysAgo := range seed.completedDaysAgo {
			if err := s.seedSession(ctx, artist.ID, created.ID, "completed", -daysAgo); err != nil {
				return i, err
			}
		}
		if seed.upcomingInDays > 0 {
			if err := s.seedSession(ctx, artist.ID, created.ID, "scheduled", seed.upcomingInDays); err != nil {
				return i, err
			}
		}
		if len(seed.completedDaysAgo) > 0 || seed.upcomingInDays > 0 {
			if _, err := s.repo.UpdateBookingRequestStatus(ctx, sqlc.UpdateBookingRequestStatusParams{
				ID:       created.ID,
				ArtistID: artist.ID,
				Status:   "accepted",
			}); err != nil {
				return i, err
			}
		}
	}

	return len(samples), nil
}

func (s *service) seedSession(ctx context.Context, artistID, requestID uuid.UUID, status string, offsetDays int) error {
	start := time.Now().AddDate(0, 0, offsetDays)
	_, err := s.repo.CreateAppointment(ctx, sqlc.CreateAppointmentParams{
		ArtistID:         artistID,
		BookingRequestID: requestID,
		Type:             "session",
		Status:           status,
		ScheduledStart:   pgtype.Timestamptz{Time: start, Valid: true},
		DurationMinutes:  180,
		SchedulingOrigin: "artist_set",
	})
	return err
}
