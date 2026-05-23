package messaging

import (
	"github.com/gin-gonic/gin"
	"github.com/trishaneupnexx/inkspace-api/internal/httpx"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ListConversations(c *gin.Context)  { httpx.NotImplemented(c) }
func (h *Handler) CreateConversation(c *gin.Context) { httpx.NotImplemented(c) }
func (h *Handler) GetConversation(c *gin.Context)    { httpx.NotImplemented(c) }
func (h *Handler) ListMessages(c *gin.Context)       { httpx.NotImplemented(c) }
func (h *Handler) SendMessage(c *gin.Context)        { httpx.NotImplemented(c) }
func (h *Handler) MarkRead(c *gin.Context)           { httpx.NotImplemented(c) }
