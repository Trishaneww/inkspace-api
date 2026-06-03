package flashes

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/trishaneupnexx/inkspace-api/internal/httpx"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ListByArtist(c *gin.Context) {
	artistID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	filter, ok := parseListFilter(c)
	if !ok {
		return
	}

	result, err := h.svc.ListByArtist(c.Request.Context(), artistID, filter)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, result)
}

func (h *Handler) Get(c *gin.Context) {
	flashID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	flash, err := h.svc.Get(c.Request.Context(), flashID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, flash)
}

func (h *Handler) ListForCurrentUser(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	filter, ok := parseListFilter(c)
	if !ok {
		return
	}
	result, err := h.svc.ListForCurrentUser(c.Request.Context(), userID, filter)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, result)
}

func (h *Handler) Create(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var input CreateFlashInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", err.Error())
		return
	}
	flash, err := h.svc.Create(c.Request.Context(), userID, input)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.Created(c, flash)
}

func (h *Handler) Update(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	flashID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var input UpdateFlashInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", err.Error())
		return
	}
	flash, err := h.svc.Update(c.Request.Context(), userID, flashID, input)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, flash)
}

func (h *Handler) Publish(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	flashID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	flash, err := h.svc.Publish(c.Request.Context(), userID, flashID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, flash)
}

func (h *Handler) Archive(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	flashID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	flash, err := h.svc.Archive(c.Request.Context(), userID, flashID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, flash)
}

func (h *Handler) Unarchive(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	flashID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	flash, err := h.svc.Unarchive(c.Request.Context(), userID, flashID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, flash)
}

func (h *Handler) Delete(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	flashID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), userID, flashID); err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) PresignUpload(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var input PresignUploadInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", err.Error())
		return
	}
	resp, err := h.svc.PresignFlashImageUpload(c.Request.Context(), userID, input.ContentType)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, resp)
}

func requireUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.Error(c, 401, "unauthorized", "missing user context")
		return uuid.Nil, false
	}
	return userID, true
}

func parseIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		httpx.Error(c, 400, "invalid_id", "invalid "+name)
		return uuid.Nil, false
	}
	return id, true
}

func parseListFilter(c *gin.Context) (ListFilter, bool) {
	filter := ListFilter{}

	if raw := c.Query("status"); raw != "" {
		status := FlashStatus(raw)
		switch status {
		case FlashStatusDraft, FlashStatusAvailable, FlashStatusClaimed, FlashStatusArchived:
			filter.Status = &status
		default:
			httpx.Error(c, 400, "invalid_status", "status must be draft, available, claimed, or archived")
			return ListFilter{}, false
		}
	}
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			httpx.Error(c, 400, "invalid_limit", "limit must be a non-negative integer")
			return ListFilter{}, false
		}
		filter.Limit = int32(n)
	}
	if raw := c.Query("offset"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			httpx.Error(c, 400, "invalid_offset", "offset must be a non-negative integer")
			return ListFilter{}, false
		}
		filter.Offset = int32(n)
	}
	return filter, true
}

func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, 404, "not_found", err.Error())
	case errors.Is(err, ErrForbidden):
		httpx.Error(c, 403, "forbidden", err.Error())
	case errors.Is(err, ErrArtistMissing):
		httpx.Error(c, 403, "no_artist_profile", err.Error())
	case errors.Is(err, ErrInvalidInput):
		httpx.Error(c, 400, "invalid_input", err.Error())
	default:
		httpx.Error(c, 500, "internal_error", err.Error())
	}
}
