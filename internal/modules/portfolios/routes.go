package portfolios

import (
	"github.com/gin-gonic/gin"

	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/auth"
)

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/artists/:id/portfolio", m.Handler.ListByArtist)

	authed := rg.Group("")
	authed.Use(middleware.RequireAuth(m.cfg.JWTSecret))
	authed.Use(middleware.RequireRole(string(auth.RoleArtist)))

	authed.GET("/current-user/portfolio", m.Handler.ListForCurrentUser)
	authed.POST("/portfolio/uploads/presign", m.Handler.PresignUpload)
	authed.POST("/portfolio", m.Handler.Create)
	authed.PATCH("/portfolio/:id", m.Handler.Update)
	authed.POST("/portfolio/:id/publish", m.Handler.Publish)
	authed.POST("/portfolio/:id/archive", m.Handler.Archive)
	authed.POST("/portfolio/:id/unarchive", m.Handler.Unarchive)
	authed.DELETE("/portfolio/:id", m.Handler.Delete)
}
