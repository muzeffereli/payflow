package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Role() gin.HandlerFunc {
	return func(c *gin.Context) {
		if role := c.GetHeader("X-User-Role"); role != "" {
			c.Set("user_role", role)
		}
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(c *gin.Context) {
		role := c.GetString("user_role")
		if !allowed[role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "forbidden: requires role " + roles[0],
			})
			return
		}
		c.Next()
	}
}
