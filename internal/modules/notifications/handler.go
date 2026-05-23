package notifications

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

func (h *Handler) List(c *gin.Context)              { httpx.NotImplemented(c) }
func (h *Handler) MarkRead(c *gin.Context)          { httpx.NotImplemented(c) }
func (h *Handler) MarkAllRead(c *gin.Context)       { httpx.NotImplemented(c) }
func (h *Handler) GetPreferences(c *gin.Context)    { httpx.NotImplemented(c) }
func (h *Handler) UpdatePreferences(c *gin.Context) { httpx.NotImplemented(c) }
