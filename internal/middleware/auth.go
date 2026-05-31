// Package middleware enthält HTTP-Middleware (Auth, Rate-Limit, Security-Header).
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/kommaufdenpunkt/insider/internal/auth"
)

// RequireAuth prüft das Bearer-JWT und legt uid/adm in den Kontext.
func RequireAuth(jwt *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "nicht angemeldet"})
			return
		}
		claims, err := jwt.Parse(strings.TrimPrefix(header, prefix))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "ungültiges Token"})
			return
		}
		c.Set("uid", claims.UID)
		c.Set("adm", claims.Admin)
		c.Next()
	}
}
