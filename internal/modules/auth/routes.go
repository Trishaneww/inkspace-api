package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
)

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/auth")
	g.POST("/register", m.Handler.Register)
	g.POST("/login", m.Handler.Login)
	g.POST("/refresh", m.Handler.Refresh)
	g.POST("/logout", m.Handler.Logout)

	authed := g.Group("")
	authed.Use(middleware.RequireAuth(m.cfg.JWTSecret))
	authed.GET("/me", m.Handler.Me)
}
