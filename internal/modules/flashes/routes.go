package flashes

import (
	"github.com/gin-gonic/gin"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
)

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	// Public reads.
	rg.GET("/artists/:id/flashes", m.Handler.ListByArtist)
	rg.GET("/flashes/:id", m.Handler.Get)

	// Authed — artist manages their own flashbook.
	authed := rg.Group("")
	authed.Use(middleware.RequireAuth(m.cfg.JWTSecret))

	authed.GET("/current-user/flashes", m.Handler.ListForCurrentUser)
	authed.POST("/flashes/uploads/presign", m.Handler.PresignUpload)
	authed.POST("/flashes", m.Handler.Create)
	authed.PATCH("/flashes/:id", m.Handler.Update)
	authed.POST("/flashes/:id/publish", m.Handler.Publish)
	authed.POST("/flashes/:id/archive", m.Handler.Archive)
	authed.POST("/flashes/:id/unarchive", m.Handler.Unarchive)
	authed.DELETE("/flashes/:id", m.Handler.Delete)
}
