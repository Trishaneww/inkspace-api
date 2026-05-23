package matching

import (
	"github.com/gin-gonic/gin"
	"github.com/trishaneupnexx/inkspace-api/internal/middleware"
)

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/matching")
	g.Use(middleware.RequireAuth(m.cfg.JWTSecret))

	// Client side: submit a tattoo idea and view matches
	g.POST("/requests", m.Handler.CreateRequest)
	g.GET("/requests", m.Handler.ListMyRequests)
	g.GET("/requests/:id", m.Handler.GetRequest)
	g.GET("/requests/:id/matches", m.Handler.ListMatches)

	// Artist side: lead inbox (bounty system)
	g.GET("/leads", m.Handler.ListLeads)
	g.POST("/leads/:requestId/interest", m.Handler.ExpressInterest)
	g.POST("/leads/:requestId/pass", m.Handler.PassLead)
}
