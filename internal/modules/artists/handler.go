package artists

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

func (h *Handler) List(c *gin.Context)               { httpx.NotImplemented(c) }
func (h *Handler) Get(c *gin.Context)                { httpx.NotImplemented(c) }
func (h *Handler) GetByHandle(c *gin.Context)        { httpx.NotImplemented(c) }
func (h *Handler) Create(c *gin.Context)             { httpx.NotImplemented(c) }
func (h *Handler) Update(c *gin.Context)             { httpx.NotImplemented(c) }
func (h *Handler) GetAvailability(c *gin.Context)    { httpx.NotImplemented(c) }
func (h *Handler) UpdateAvailability(c *gin.Context) { httpx.NotImplemented(c) }
