package portfolios

import (
	"github.com/gin-gonic/gin"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
)

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	// Public portfolio browsing
	rg.GET("/artists/:id/portfolio", m.Handler.GetArtistPortfolio)

	// Public flash marketplace
	flash := rg.Group("/flash")
	flash.GET("", m.Handler.ListFlash)
	flash.GET("/:id", m.Handler.GetFlash)

	// Authed: artist portfolio + flash management, and client flash reservation
	authed := rg.Group("")
	authed.Use(middleware.RequireAuth(m.cfg.JWTSecret))

	authed.POST("/portfolios/items", m.Handler.CreatePortfolioItem)
	authed.PATCH("/portfolios/items/:id", m.Handler.UpdatePortfolioItem)
	authed.DELETE("/portfolios/items/:id", m.Handler.DeletePortfolioItem)

	authed.POST("/flash", m.Handler.CreateFlash)
	authed.PATCH("/flash/:id", m.Handler.UpdateFlash)
	authed.DELETE("/flash/:id", m.Handler.DeleteFlash)
	authed.POST("/flash/:id/reserve", m.Handler.ReserveFlash)
}
