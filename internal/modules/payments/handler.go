package payments

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

func (h *Handler) List(c *gin.Context)                { httpx.NotImplemented(c) }
func (h *Handler) Get(c *gin.Context)                 { httpx.NotImplemented(c) }
func (h *Handler) CreateDepositIntent(c *gin.Context) { httpx.NotImplemented(c) }
func (h *Handler) Refund(c *gin.Context)              { httpx.NotImplemented(c) }
func (h *Handler) StripeWebhook(c *gin.Context)       { httpx.NotImplemented(c) }
