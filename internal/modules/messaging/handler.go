package messaging

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

// ── Artist ──────────────────────────────────────────────────────────────────
func (h *Handler) ListArtistConversations(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	list, err := h.svc.ListArtistConversations(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, list)
}

func (h *Handler) GetArtistConversation(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	convID, ok := parseConvID(c)
	if !ok {
		return
	}
	thread, err := h.svc.GetArtistConversation(c.Request.Context(), userID, convID)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, thread)
}

func (h *Handler) SendArtistMessage(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	convID, ok := parseConvID(c)
	if !ok {
		return
	}
	input, ok := bindInput(c)
	if !ok {
		return
	}
	msg, err := h.svc.SendArtistMessage(c.Request.Context(), userID, convID, input)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.Created(c, msg)
}

func (h *Handler) MarkArtistRead(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	convID, ok := parseConvID(c)
	if !ok {
		return
	}
	if err := h.svc.MarkArtistRead(c.Request.Context(), userID, convID); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

// ── Client ────────────────────────────────────────────────────────────────────
func (h *Handler) ListClientConversations(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	list, err := h.svc.ListClientConversations(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, list)
}

func (h *Handler) GetClientConversation(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	convID, ok := parseConvID(c)
	if !ok {
		return
	}
	thread, err := h.svc.GetClientConversation(c.Request.Context(), userID, convID)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, thread)
}

func (h *Handler) SendClientMessage(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	convID, ok := parseConvID(c)
	if !ok {
		return
	}
	input, ok := bindInput(c)
	if !ok {
		return
	}
	msg, err := h.svc.SendClientMessage(c.Request.Context(), userID, convID, input)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.Created(c, msg)
}

func (h *Handler) MarkClientRead(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	convID, ok := parseConvID(c)
	if !ok {
		return
	}
	if err := h.svc.MarkClientRead(c.Request.Context(), userID, convID); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

// ── Guest ─────────────────────────────────────────────────────────
func (h *Handler) GetGuestConversation(c *gin.Context) {
	thread, err := h.svc.GetGuestConversation(c.Request.Context(), c.Param("token"))
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, thread)
}

func (h *Handler) SendGuestMessage(c *gin.Context) {
	input, ok := bindInput(c)
	if !ok {
		return
	}
	msg, err := h.svc.SendGuestMessage(c.Request.Context(), c.Param("token"), input)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.Created(c, msg)
}

func (h *Handler) MarkGuestRead(c *gin.Context) {
	if err := h.svc.MarkGuestRead(c.Request.Context(), c.Param("token")); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

func requireUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "missing user context")
		return uuid.Nil, false
	}
	return userID, true
}

func parseConvID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_id", "invalid conversation id")
		return uuid.Nil, false
	}
	return id, true
}

func bindInput(c *gin.Context) (SendMessageInput, bool) {
	var input SendMessageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_input", "invalid request body")
		return SendMessageInput{}, false
	}
	return input, true
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "conversation not found")
	case errors.Is(err, ErrEmptyMessage):
		httpx.Error(c, http.StatusBadRequest, "empty_message", "message cannot be empty")
	case errors.Is(err, ErrTooLong):
		httpx.Error(c, http.StatusBadRequest, "message_too_long", "message is too long")
	default:
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
