package payments

import (
	"github.com/gin-gonic/gin"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
)

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/payments")

	// Provider webhooks must NOT be behind auth — they're verified via signature.
	g.POST("/webhooks/stripe", m.Handler.StripeWebhook)

	authed := g.Group("")
	authed.Use(middleware.RequireAuth(m.cfg.JWTSecret))
	authed.GET("", m.Handler.List)
	authed.POST("/deposits", m.Handler.CreateDepositIntent)
	authed.GET("/:id", m.Handler.Get)
	authed.POST("/:id/refund", m.Handler.Refund)
}
