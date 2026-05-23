package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/trishaneupnexx/inkspace-api/internal/httpx"
)

const CtxUserIDKey = "user_id"

// RequireAuth is a placeholder JWT middleware. Real verification lands with the auth module.
func RequireAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			httpx.Error(c, 401, "unauthorized", "missing bearer token")
			return
		}
		// TODO: verify JWT, extract user id, set c.Set(CtxUserIDKey, ...)
		c.Next()
	}
}
