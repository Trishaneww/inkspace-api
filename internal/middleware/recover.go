package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/trishaneupnexx/inkspace-api/internal/httpx"
)

func Recover(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic_recovered", "error", r, "path", c.Request.URL.Path)
				httpx.Error(c, 500, "internal_error", "something went wrong")
			}
		}()
		c.Next()
	}
}
