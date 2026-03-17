package middleware

import (
	"net/http"
	"strings"

	"payment-platform/pkg/auth"
	"payment-platform/pkg/config"

	"github.com/gin-gonic/gin"
)

func Auth(cfg config.JWTConfig) gin.HandlerFunc {
	manager := auth.NewJWTManager(cfg.Secret, cfg.Issuer, 0) // ttl=0: verify only

	publicPaths := map[string]bool{
		"/healthz":       true,
		"/metrics":       true,
		"/auth/register": true,
		"/auth/login":    true,
		"/auth/refresh":  true,
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if publicPaths[path] || strings.HasPrefix(path, "/swagger/") || strings.HasPrefix(path, "/docs") {
			c.Next()
			return
		}

		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		var token string
		if scheme, rest, found := strings.Cut(header, " "); found {
			if !strings.EqualFold(scheme, "bearer") {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
				return
			}
			token = rest
		} else {
			token = header
		}

		claims, err := manager.VerifyAccess(token)
		if err != nil {
			status := http.StatusUnauthorized
			msg := "invalid token"
			if err == auth.ErrTokenExpired {
				msg = "token expired â€” use POST /auth/refresh to renew"
			}
			c.AbortWithStatusJSON(status, gin.H{"error": msg})
			return
		}

		c.Set("user_id", claims.UserID)
		role := claims.Role
		if role == "" {
			role = "customer"
		}
		c.Set("user_role", role)

		c.Next()
	}
}
