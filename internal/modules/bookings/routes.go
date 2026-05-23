package bookings

import (
	"github.com/gin-gonic/gin"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
)

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/bookings")
	g.Use(middleware.RequireAuth(m.cfg.JWTSecret))

	g.GET("", m.Handler.List)
	g.POST("", m.Handler.Create)
	g.GET("/:id", m.Handler.Get)
	g.PATCH("/:id", m.Handler.Update)
	g.POST("/:id/confirm", m.Handler.Confirm)
	g.POST("/:id/cancel", m.Handler.Cancel)
	g.POST("/:id/reschedule", m.Handler.Reschedule)
}
