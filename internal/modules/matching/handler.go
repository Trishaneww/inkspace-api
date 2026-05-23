package matching

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

func (h *Handler) CreateRequest(c *gin.Context)   { httpx.NotImplemented(c) }
func (h *Handler) ListMyRequests(c *gin.Context)  { httpx.NotImplemented(c) }
func (h *Handler) GetRequest(c *gin.Context)      { httpx.NotImplemented(c) }
func (h *Handler) ListMatches(c *gin.Context)     { httpx.NotImplemented(c) }
func (h *Handler) ListLeads(c *gin.Context)       { httpx.NotImplemented(c) }
func (h *Handler) ExpressInterest(c *gin.Context) { httpx.NotImplemented(c) }
func (h *Handler) PassLead(c *gin.Context)        { httpx.NotImplemented(c) }
