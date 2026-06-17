package payments

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

func (h *Handler) CreatePaymentRequest(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	bookingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_id", "invalid booking id")
		return
	}
	var input CreatePaymentRequestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_input", "invalid request body")
		return
	}
	pr, err := h.svc.CreatePaymentRequest(c.Request.Context(), userID, bookingID, input)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.Created(c, pr)
}

func (h *Handler) CancelPaymentRequest(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_id", "invalid payment request id")
		return
	}
	pr, err := h.svc.CancelPaymentRequest(c.Request.Context(), userID, id)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, pr)
}

func (h *Handler) ResendPaymentRequest(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_id", "invalid payment request id")
		return
	}
	pr, err := h.svc.ResendPaymentRequest(c.Request.Context(), userID, id)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, pr)
}

func (h *Handler) RefundPaymentRequest(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_id", "invalid payment request id")
		return
	}
	pr, err := h.svc.RefundPaymentRequest(c.Request.Context(), userID, id)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, pr)
}

func (h *Handler) GetPaymentRequest(c *gin.Context) {
	pr, err := h.svc.GetPublicPaymentRequest(c.Request.Context(), c.Param("token"))
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, pr)
}

func (h *Handler) CreateCheckout(c *gin.Context) {
	resp, err := h.svc.CreateCheckout(c.Request.Context(), c.Param("token"))
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) CreateClientAccount(c *gin.Context) {
	var input CreateClientAccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_input", "invalid request body")
		return
	}
	if err := h.svc.CreateClientAccount(c.Request.Context(), c.Param("token"), input); err != nil {
		respondError(c, err)
		return
	}
	httpx.Created(c, gin.H{"created": true})
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
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, ErrStripeNotConnected):
		httpx.Error(c, http.StatusConflict, "stripe_not_connected", "connect your Stripe account before requesting payments")
	case errors.Is(err, ErrAlreadyRequested):
		httpx.Error(c, http.StatusConflict, "already_requested", err.Error())
	case errors.Is(err, ErrNotPayable):
		httpx.Error(c, http.StatusConflict, "not_payable", "this payment link is no longer payable")
	case errors.Is(err, ErrResendTooSoon):
		httpx.Error(c, http.StatusTooManyRequests, "resend_too_soon",
			"You've already sent this recently. Please wait a few minutes before resending.")
	case errors.Is(err, ErrNotRefundable):
		httpx.Error(c, http.StatusConflict, "not_refundable", "this payment can't be refunded")
	case errors.Is(err, ErrAccountExists):
		httpx.Error(c, http.StatusConflict, "account_exists", "an account with this email already exists")
	case errors.Is(err, ErrWeakPassword):
		httpx.Error(c, http.StatusBadRequest, "weak_password", "password must be at least 8 characters")
	case errors.Is(err, ErrAmountTooLow):
		httpx.Error(c, http.StatusBadRequest, "amount_too_low", err.Error())
	case errors.Is(err, ErrNoDepositDefault):
		httpx.Error(c, http.StatusBadRequest, "no_deposit_default", "enter an amount or set a default deposit in settings")
	case errors.Is(err, ErrInvalidInput):
		httpx.Error(c, http.StatusBadRequest, "invalid_input", err.Error())
	default:
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
