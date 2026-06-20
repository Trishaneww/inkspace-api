package dashboard

import (
	"github.com/gin-gonic/gin"

	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/auth"
)

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	artist := rg.Group("/current-user")
	artist.Use(middleware.RequireAuth(m.cfg.JWTSecret))
	artist.Use(middleware.RequireRole(string(auth.RoleArtist)))

	artist.GET("/dashboard", m.Handler.GetDashboard)
}
