package portfolios

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

func (h *Handler) GetArtistPortfolio(c *gin.Context)   { httpx.NotImplemented(c) }
func (h *Handler) CreatePortfolioItem(c *gin.Context)  { httpx.NotImplemented(c) }
func (h *Handler) UpdatePortfolioItem(c *gin.Context)  { httpx.NotImplemented(c) }
func (h *Handler) DeletePortfolioItem(c *gin.Context)  { httpx.NotImplemented(c) }

func (h *Handler) ListFlash(c *gin.Context)    { httpx.NotImplemented(c) }
func (h *Handler) GetFlash(c *gin.Context)     { httpx.NotImplemented(c) }
func (h *Handler) CreateFlash(c *gin.Context)  { httpx.NotImplemented(c) }
func (h *Handler) UpdateFlash(c *gin.Context)  { httpx.NotImplemented(c) }
func (h *Handler) DeleteFlash(c *gin.Context)  { httpx.NotImplemented(c) }
func (h *Handler) ReserveFlash(c *gin.Context) { httpx.NotImplemented(c) }
