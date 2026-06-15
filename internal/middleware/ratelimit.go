package middleware

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/trishaneupnexx/inkspace-api/internal/httpx"
	"github.com/trishaneupnexx/inkspace-api/internal/ratelimit"
)

// RateLimit throttles requests per client IP using the given limiter. `prefix`
// namespaces the Redis keys so independent buckets (e.g. global vs auth) don't
// collide for the same IP.
//
// It fails OPEN: if the limiter backend errors, the request is allowed and the
// failure is logged, so a Redis hiccup can never take the whole API down.
func RateLimit(limiter ratelimit.Limiter, prefix string, log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := prefix + ":" + c.ClientIP()

		res, err := limiter.CheckLimit(c.Request.Context(), key)
		if err != nil {
			log.Warn("rate_limiter_error", "prefix", prefix, "error", err)
			return // fail open
		}
		if !res.Allowed {
			if res.RetryAfter > 0 {
				secs := max(int(res.RetryAfter.Seconds()), 1)
				c.Header("Retry-After", strconv.Itoa(secs))
			}
			httpx.Error(c, http.StatusTooManyRequests, "rate_limited",
				"Too many requests. Please slow down and try again shortly.")
			return
		}
	}
}

// FilterByPath applies `mw` only to requests whose path starts with `prefix`.
// Lets a stricter limiter be attached to a route family (e.g. /v1/auth/) while
// staying registered at the engine level.
func FilterByPath(prefix string, mw gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, prefix) {
			mw(c)
		}
	}
}
