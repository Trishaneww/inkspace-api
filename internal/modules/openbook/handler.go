package openbook

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/trishaneupnexx/inkspace-api/internal/httpx"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetProfile(c *gin.Context) {
	profile, err := h.svc.GetProfile(c.Request.Context(), c.Param("slug"))
	if err != nil {
		buildErrorResponse(c, err)
		return
	}
	httpx.OK(c, profile)
}

func (h *Handler) PresignReference(c *gin.Context) {
	var input PresignReferenceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", err.Error())
		return
	}
	result, err := h.svc.PresignReferenceUpload(c.Request.Context(), c.Param("slug"), input.ContentType)
	if err != nil {
		buildErrorResponse(c, err)
		return
	}
	httpx.OK(c, result)
}

func (h *Handler) CreateRequest(c *gin.Context) {
	var input CreateRequestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, 400, "invalid_request", err.Error())
		return
	}
	result, err := h.svc.CreateRequest(c.Request.Context(), c.Param("slug"), input)
	if err != nil {
		buildErrorResponse(c, err)
		return
	}
	httpx.Created(c, result)
}

func buildErrorResponse(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, 404, "not_found", "We couldn't find that booking page.")
	case errors.Is(err, ErrNotAccepting):
		httpx.Error(c, 409, "not_accepting", "This artist isn't accepting bookings right now.")
	case errors.Is(err, ErrInvalidInput):
		httpx.Error(c, 400, "invalid_request", err.Error())
	default:
		httpx.Error(c, 500, "internal_error", "Something went wrong. Please try again.")
	}
}
