package portfolios

import (
	"context"
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
	filter, ok := parseListFilter(c, false)
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

func (h *Handler) ListForCurrentUser(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	filter, ok := parseListFilter(c, true)
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
	var input CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", err.Error())
		return
	}
	item, err := h.svc.Create(c.Request.Context(), userID, input)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.Created(c, item)
}

func (h *Handler) Update(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	itemID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var input UpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", err.Error())
		return
	}
	item, err := h.svc.Update(c.Request.Context(), userID, itemID, input)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, item)
}

func (h *Handler) Publish(c *gin.Context)   { h.transition(c, h.svc.Publish) }
func (h *Handler) Archive(c *gin.Context)   { h.transition(c, h.svc.Archive) }
func (h *Handler) Unarchive(c *gin.Context) { h.transition(c, h.svc.Unarchive) }

func (h *Handler) Delete(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	itemID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), userID, itemID); err != nil {
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
	resp, err := h.svc.PresignImageUpload(c.Request.Context(), userID, input.ContentType)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, resp)
}

// transition runs a status-change service method that returns the updated item.
func (h *Handler) transition(
	c *gin.Context,
	fn func(ctx context.Context, userID, itemID uuid.UUID) (PortfolioItem, error),
) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	itemID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	item, err := fn(c.Request.Context(), userID, itemID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	httpx.OK(c, item)
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

// parseListFilter reads paging + an optional status filter. Public callers may
// not filter by status (they only ever get published items).
func parseListFilter(c *gin.Context, allowStatus bool) (ListFilter, bool) {
	filter := ListFilter{}

	if allowStatus {
		if raw := c.Query("status"); raw != "" {
			status := Status(raw)
			switch status {
			case StatusDraft, StatusPublished, StatusArchived:
				filter.Status = &status
			default:
				httpx.Error(c, 400, "invalid_status", "status must be draft, published, or archived")
				return ListFilter{}, false
			}
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
	case errors.Is(err, ErrInvalidInput):
		httpx.Error(c, 400, "invalid_input", err.Error())
	default:
		httpx.Error(c, 500, "internal_error", err.Error())
	}
}
