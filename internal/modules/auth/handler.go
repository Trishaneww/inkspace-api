package auth

import (
	"errors"
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

func (h *Handler) Register(c *gin.Context) {
	var in RegisterInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	resp, err := h.svc.Register(c.Request.Context(), in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	httpx.Created(c, resp)
}

func (h *Handler) Login(c *gin.Context) {
	var in LoginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	resp, err := h.svc.Login(c.Request.Context(), in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) VerifyPhone(c *gin.Context) {
	var in VerifyPhoneInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	resp, err := h.svc.VerifyPhone(c.Request.Context(), in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) ResendPhoneCode(c *gin.Context) {
	var in ResendPhoneInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err := h.svc.ResendPhoneCode(c.Request.Context(), in); err != nil {
		writeServiceError(c, err)
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

func (h *Handler) OAuthCallback(c *gin.Context) {
	provider := c.Param("provider")
	var in OAuthCallbackInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	in.Provider = provider

	resp, err := h.svc.OAuthCallback(c.Request.Context(), in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) CompleteOAuthSignup(c *gin.Context) {
	var in OAuthCompleteInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	resp, err := h.svc.CompleteOAuthSignup(c.Request.Context(), in)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	httpx.Created(c, resp)
}

func (h *Handler) GetCurrentUser(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "missing user context")
		return
	}
	user, err := h.svc.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	httpx.OK(c, user)
}

func (h *Handler) Logout(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "missing user context")
		return
	}
	if err := h.svc.Logout(c.Request.Context(), userID); err != nil {
		writeServiceError(c, err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) Refresh(c *gin.Context) {
	var in RefreshInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	pair, err := h.svc.RefreshSession(c.Request.Context(), in.RefreshToken)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	httpx.OK(c, pair)
}

func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Error(c, http.StatusUnauthorized, "invalid_credentials", err.Error())
	case errors.Is(err, ErrInvalidRefreshToken):
		httpx.Error(c, http.StatusUnauthorized, "invalid_refresh_token", err.Error())
	case errors.Is(err, ErrEmailTaken):
		httpx.Error(c, http.StatusConflict, "email_taken", err.Error())
	case errors.Is(err, ErrPhoneTaken):
		httpx.Error(c, http.StatusConflict, "phone_taken", err.Error())
	case errors.Is(err, ErrUsernameTaken):
		httpx.Error(c, http.StatusConflict, "username_taken", err.Error())
	case errors.Is(err, ErrUsernameRequired):
		httpx.Error(c, http.StatusBadRequest, "username_required", err.Error())
	case errors.Is(err, ErrPhoneVerificationNotFound):
		httpx.Error(c, http.StatusNotFound, "phone_verification_not_found", err.Error())
	case errors.Is(err, ErrInvalidVerificationCode):
		httpx.Error(c, http.StatusBadRequest, "invalid_verification_code", err.Error())
	case errors.Is(err, ErrTooManyVerificationAttempts):
		httpx.Error(c, http.StatusTooManyRequests, "too_many_verification_attempts", err.Error())
	case errors.Is(err, ErrProviderNotConfigured):
		httpx.Error(c, http.StatusNotImplemented, "oauth_not_configured", err.Error())
	case errors.Is(err, ErrProviderNotImplemented):
		httpx.Error(c, http.StatusNotImplemented, "oauth_not_implemented", err.Error())
	default:
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}

// _ keeps uuid imported in case future handlers parse path params.
var _ = uuid.Nil
