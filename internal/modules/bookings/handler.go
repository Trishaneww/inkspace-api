package bookings

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/trishaneupnexx/inkspace-api/internal/config"
	"github.com/trishaneupnexx/inkspace-api/internal/httpx"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
)

type Handler struct {
	cfg *config.Config
	svc Service
}

func NewHandler(cfg *config.Config, svc Service) *Handler {
	return &Handler{cfg: cfg, svc: svc}
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	resp, err := h.svc.ListInquiries(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) Get(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	inquiry, err := h.svc.GetInquiry(c.Request.Context(), userID, id)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, inquiry)
}

func (h *Handler) Accept(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	var input AcceptInput
	_ = c.ShouldBindJSON(&input)

	inquiry, err := h.svc.AcceptInquiry(c.Request.Context(), userID, id, input)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, inquiry)
}

func (h *Handler) RequestConsultation(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	var input RequestConsultationInput
	_ = c.ShouldBindJSON(&input)

	inquiry, err := h.svc.RequestConsultation(c.Request.Context(), userID, id, input)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, inquiry)
}

func (h *Handler) Reschedule(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	var input RescheduleInput
	_ = c.ShouldBindJSON(&input)

	inquiry, err := h.svc.RescheduleAppointment(c.Request.Context(), userID, id, input)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, inquiry)
}

func (h *Handler) Cancel(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}

	var input struct {
		AppointmentID *string `json:"appointmentId"`
	}
	_ = c.ShouldBindJSON(&input)
	var appointmentID *uuid.UUID
	if input.AppointmentID != nil && *input.AppointmentID != "" {
		parsed, err := uuid.Parse(*input.AppointmentID)
		if err != nil {
			httpx.Error(c, http.StatusBadRequest, "invalid_id", "invalid appointment id")
			return
		}
		appointmentID = &parsed
	}

	inquiry, err := h.svc.CancelBooking(c.Request.Context(), userID, id, appointmentID)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, inquiry)
}

func (h *Handler) Decline(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	inquiry, err := h.svc.DeclineInquiry(c.Request.Context(), userID, id)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, inquiry)
}

func (h *Handler) Reopen(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	inquiry, err := h.svc.ReopenInquiry(c.Request.Context(), userID, id)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, inquiry)
}

func (h *Handler) DevSeed(c *gin.Context) {
	if h.cfg.AppEnv != "development" {
		httpx.Error(c, http.StatusNotFound, "not_found", "not available")
		return
	}
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	created, err := h.svc.SeedDevInquiries(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"created": created})
}

func requireUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "missing user context")
		return uuid.Nil, false
	}
	return userID, true
}

func parseID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_id", "invalid inquiry id")
		return uuid.Nil, false
	}
	return id, true
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, ErrNoOpenBook):
		httpx.Error(c, http.StatusConflict, "no_open_book", err.Error())
	case errors.Is(err, ErrInvalidSchedule):
		httpx.Error(c, http.StatusBadRequest, "invalid_schedule", err.Error())
	case errors.Is(err, ErrScheduleConflict):
		httpx.Error(c, http.StatusConflict, "schedule_conflict", err.Error())
	default:
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
