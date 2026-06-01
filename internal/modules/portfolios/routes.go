package portfolios

import (
	"github.com/gin-gonic/gin"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
)

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	// Public portfolio browsing
	rg.GET("/artists/:id/portfolio", m.Handler.GetArtistPortfolio)

	// Authed: artist portfolio management
	authed := rg.Group("")
	authed.Use(middleware.RequireAuth(m.cfg.JWTSecret))

	authed.POST("/portfolios/items", m.Handler.CreatePortfolioItem)
	authed.PATCH("/portfolios/items/:id", m.Handler.UpdatePortfolioItem)
	authed.DELETE("/portfolios/items/:id", m.Handler.DeletePortfolioItem)
}
