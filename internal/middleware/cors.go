package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS returns a permissive-per-origin-allowlist middleware.
// `allowedOrigins` is a comma-separated list (e.g. from CORS_ALLOWED_ORIGINS).
// Use "*" for any origin (do not combine with credentialed requests).
func CORS(allowedOrigins string) gin.HandlerFunc {
	origins := parseOrigins(allowedOrigins)
	allowAny := len(origins) == 1 && origins[0] == "*"

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if origin != "" {
			switch {
			case allowAny:
				c.Header("Access-Control-Allow-Origin", "*")
			case originAllowed(origins, origin):
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Credentials", "true")
			}
			c.Header("Access-Control-Allow-Methods",
				"GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers",
				"Authorization, Content-Type, X-Requested-With")
			c.Header("Access-Control-Max-Age", "600")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func parseOrigins(s string) []string {
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func originAllowed(allowed []string, origin string) bool {
	for _, a := range allowed {
		if a == origin {
			return true
		}
	}
	return false
}
