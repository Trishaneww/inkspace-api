package openbook

import "github.com/gin-gonic/gin"

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/book/:slug", m.Handler.GetProfile)
	rg.POST("/book/:slug/uploads/presign", m.Handler.PresignReference)
	rg.POST("/book/:slug/requests", m.Handler.CreateRequest)
}
