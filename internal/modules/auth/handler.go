package auth

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

func (h *Handler) Register(c *gin.Context) { httpx.NotImplemented(c) }
func (h *Handler) Login(c *gin.Context)    { httpx.NotImplemented(c) }
func (h *Handler) Refresh(c *gin.Context)  { httpx.NotImplemented(c) }
func (h *Handler) Logout(c *gin.Context)   { httpx.NotImplemented(c) }
func (h *Handler) Me(c *gin.Context)       { httpx.NotImplemented(c) }
