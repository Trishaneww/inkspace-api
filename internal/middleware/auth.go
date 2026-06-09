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

func RequireAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			httpx.Error(c, 401, "unauthorized", "missing bearer token")
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
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

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := RoleFromContext(c)
		if !ok {
			httpx.Error(c, 403, "forbidden", "role not available")
			return
		}
		for _, allowed := range roles {
			if role == allowed {
				c.Next()
				return
			}
		}
		httpx.Error(c, 403, "forbidden", "insufficient permissions for this resource")
	}
}

func UserIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	v, exists := c.Get(CtxUserIDKey)
	if !exists {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

func RoleFromContext(c *gin.Context) (string, bool) {
	v, exists := c.Get(CtxRoleKey)
	if !exists {
		return "", false
	}
	r, ok := v.(string)
	return r, ok
}
