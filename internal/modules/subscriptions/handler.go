package subscriptions

import (
	"errors"
	"io"
	"net/http"

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

func (h *Handler) GetSubscription(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	status, err := h.svc.GetSubscription(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, status)
}

func (h *Handler) CreateCheckout(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	resp, err := h.svc.CreateCheckout(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) CreatePortalSession(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	resp, err := h.svc.CreatePortalSession(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) Webhook(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_request", "could not read request body")
		return
	}
	if err := h.svc.ProcessWebhook(c.Request.Context(), payload, c.GetHeader("Stripe-Signature")); err != nil {
		httpx.Error(c, http.StatusBadRequest, "webhook_error", "could not process event")
		return
	}
	httpx.OK(c, gin.H{"received": true})
}

func requireUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "missing user context")
		return uuid.Nil, false
	}
	return userID, true
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotConfigured):
		httpx.Error(c, http.StatusServiceUnavailable, "not_configured", "subscriptions are not available right now")
	case errors.Is(err, ErrAlreadySubscribed):
		httpx.Error(c, http.StatusConflict, "already_subscribed", "you already have an active subscription")
	case errors.Is(err, ErrNoSubscription):
		httpx.Error(c, http.StatusConflict, "no_subscription", "you don't have a subscription to manage yet")
	default:
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
