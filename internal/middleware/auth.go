package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/trishaneupnexx/inkspace-api/internal/httpx"
)

const (
	CtxUserIDKey = "user_id"
	CtxRoleKey   = "role"
)

// RequireAuth verifies a Bearer JWT signed with `secret`, then exposes
// the user id + role on the gin context for downstream handlers.
func RequireAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authz := c.GetHeader("Authorization")
		if authz == "" {
			httpx.Error(c, 401, "unauthorized", "missing bearer token")
			return
		}
		parts := strings.SplitN(authz, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			httpx.Error(c, 401, "unauthorized", "malformed Authorization header")
			return
		}
		raw := parts[1]

		token, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			httpx.Error(c, 401, "unauthorized", "invalid or expired token")
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			httpx.Error(c, 401, "unauthorized", "invalid token claims")
			return
		}

		sub, _ := claims["sub"].(string)
		userID, err := uuid.Parse(sub)
		if err != nil {
			httpx.Error(c, 401, "unauthorized", "invalid subject")
			return
		}
		c.Set(CtxUserIDKey, userID)
		if role, ok := claims["role"].(string); ok {
			c.Set(CtxRoleKey, role)
		}
		c.Next()
	}
}

// UserIDFromContext pulls the verified user id set by RequireAuth.
func UserIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	v, exists := c.Get(CtxUserIDKey)
	if !exists {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

// RoleFromContext returns the role claim set by RequireAuth, if present.
func RoleFromContext(c *gin.Context) (string, bool) {
	v, exists := c.Get(CtxRoleKey)
	if !exists {
		return "", false
	}
	r, ok := v.(string)
	return r, ok
}
