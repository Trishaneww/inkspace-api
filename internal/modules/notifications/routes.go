package notifications

import (
	"github.com/gin-gonic/gin"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
)

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/notifications")
	g.Use(middleware.RequireAuth(m.cfg.JWTSecret))

	g.GET("", m.Handler.List)
	g.PATCH("/:id/read", m.Handler.MarkRead)
	g.POST("/read-all", m.Handler.MarkAllRead)

	g.GET("/preferences", m.Handler.GetPreferences)
	g.PUT("/preferences", m.Handler.UpdatePreferences)
}
