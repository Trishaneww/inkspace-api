package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
)

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/auth")

	g.POST("/signup", m.Handler.Register)
	g.POST("/login", m.Handler.Login)
	g.POST("/verify-phone", m.Handler.VerifyPhone)
	g.POST("/verify-phone/resend", m.Handler.ResendPhoneCode)
	g.POST("/oauth/complete", m.Handler.CompleteOAuthSignup)
	g.POST("/oauth/:provider", m.Handler.OAuthCallback)
	g.POST("/refresh", m.Handler.Refresh)

	authed := g.Group("")
	authed.Use(middleware.RequireAuth(m.cfg.JWTSecret))
	authed.GET("/current-user", m.Handler.GetCurrentUser)
	authed.POST("/logout", m.Handler.Logout)
}
