package settings

import (
	"github.com/gin-gonic/gin"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/auth"
)

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	// Public: Stripe calls this server-to-server. Authenticity comes from the
	// Stripe-Signature header (verified in the service), not a JWT.
	rg.POST("/webhooks/stripe", m.Handler.StripeWebhook)

	authed := rg.Group("/current-user")
	authed.Use(middleware.RequireAuth(m.cfg.JWTSecret))

	authed.PATCH("/profile", m.Handler.UpdateProfile)
	authed.POST("/email", m.Handler.ChangeEmail)
	authed.POST("/password", m.Handler.ChangePassword)
	authed.POST("/avatar/presign", m.Handler.PresignAvatar)
	authed.DELETE("/account", m.Handler.DeleteAccount) // Phase-2 stub

	// Artist business configuration
	artist := authed.Group("")
	artist.Use(middleware.RequireRole(string(auth.RoleArtist)))

	artist.GET("/settings", m.Handler.GetSettings)

	artist.PATCH("/settings", m.Handler.UpdateSettings)
	artist.PUT("/settings/availability", m.Handler.SetAvailability)
	artist.POST("/settings/waiver/presign", m.Handler.PresignWaiver)

	artist.POST("/settings/session-presets", m.Handler.CreatePreset)
	artist.PATCH("/settings/session-presets/:id", m.Handler.UpdatePreset)
	artist.DELETE("/settings/session-presets/:id", m.Handler.DeletePreset)

	artist.POST("/settings/days-off", m.Handler.AddDayOff)
	artist.DELETE("/settings/days-off/:day", m.Handler.RemoveDayOff)

	artist.POST("/settings/blocklist", m.Handler.AddBlocklistEntry)
	artist.DELETE("/settings/blocklist/:id", m.Handler.RemoveBlocklistEntry)

	artist.POST("/settings/stripe/connect", m.Handler.ConnectStripe)
	artist.POST("/settings/stripe/refresh", m.Handler.RefreshStripeStatus)
	artist.DELETE("/settings/stripe", m.Handler.DisconnectStripe)

	artist.POST("/settings/google-calendar/connect", m.Handler.ConnectGoogleCalendar)
	artist.DELETE("/settings/google-calendar", m.Handler.DisconnectGoogleCalendar)
}
