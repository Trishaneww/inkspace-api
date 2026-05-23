package messaging

import (
	"github.com/gin-gonic/gin"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
)

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/conversations")
	g.Use(middleware.RequireAuth(m.cfg.JWTSecret))

	g.GET("", m.Handler.ListConversations)
	g.POST("", m.Handler.CreateConversation)
	g.GET("/:id", m.Handler.GetConversation)
	g.GET("/:id/messages", m.Handler.ListMessages)
	g.POST("/:id/messages", m.Handler.SendMessage)
	g.POST("/:id/read", m.Handler.MarkRead)
}
