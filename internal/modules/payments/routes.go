package payments

import (
	"github.com/gin-gonic/gin"

	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
	"github.com/trishaneupnexx/inkspace-api/internal/modules/auth"
)

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/webhooks/stripe/payments", m.Handler.Webhook)

	pay := rg.Group("/pay")
	pay.GET("/:token", m.Handler.GetPaymentRequest)
	pay.POST("/:token/checkout", m.Handler.CreateCheckout)
	pay.POST("/:token/account", m.Handler.CreateClientAccount)

	artist := rg.Group("/current-user")
	artist.Use(middleware.RequireAuth(m.cfg.JWTSecret))
	artist.Use(middleware.RequireRole(string(auth.RoleArtist)))
	artist.GET("/earnings", m.Handler.GetEarnings)
	artist.GET("/payouts", m.Handler.GetPayouts)
	artist.POST("/bookings/:id/payment-requests", m.Handler.CreatePaymentRequest)
	artist.POST("/payment-requests/:id/cancel", m.Handler.CancelPaymentRequest)
	artist.POST("/payment-requests/:id/resend", m.Handler.ResendPaymentRequest)
	artist.POST("/payment-requests/:id/refund", m.Handler.RefundPaymentRequest)
}
