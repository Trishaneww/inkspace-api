package artists

import (
	"github.com/gin-gonic/gin"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
)

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/artists")
	g.GET("", m.Handler.List)
	g.GET("/:id", m.Handler.Get)
	g.GET("/:id/availability", m.Handler.GetAvailability)
	g.GET("/by-handle/:handle", m.Handler.GetByHandle)

	authed := g.Group("")
	authed.Use(middleware.RequireAuth(m.cfg.JWTSecret))
	authed.POST("", m.Handler.Create)
	authed.PATCH("/:id", m.Handler.Update)
	authed.PUT("/:id/availability", m.Handler.UpdateAvailability)
}
