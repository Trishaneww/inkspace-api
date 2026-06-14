package bookings

import (
	"github.com/gin-gonic/gin"

	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/auth"
)

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	bookings := rg.Group("/current-user/bookings")
	bookings.Use(middleware.RequireAuth(m.cfg.JWTSecret))
	bookings.Use(middleware.RequireRole(string(auth.RoleArtist)))

	bookings.GET("", m.Handler.List)
	bookings.POST("/dev-seed", m.Handler.DevSeed)
	bookings.GET("/:id", m.Handler.Get)
	bookings.POST("/:id/accept", m.Handler.Accept)
	bookings.POST("/:id/request-consultation", m.Handler.RequestConsultation)
	bookings.POST("/:id/reschedule", m.Handler.Reschedule)
	bookings.POST("/:id/decline", m.Handler.Decline)
	bookings.POST("/:id/reopen", m.Handler.Reopen)
}
