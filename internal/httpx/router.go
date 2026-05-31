// Package httpx baut den Gin-Router mit allen Routen zusammen.
package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kommaufdenpunkt/insider/internal/auth"
	"github.com/kommaufdenpunkt/insider/internal/config"
	"github.com/kommaufdenpunkt/insider/internal/middleware"
)

// NewRouter erstellt den Router. Alle App-Routen liegen unter /v1 (§3, §10).
func NewRouter(cfg *config.Config, jwt *auth.JWTManager, authH *auth.Handler) (*gin.Engine, error) {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())

	// Echte Client-IP nur Proxys vertrauen, die wir kennen (§11).
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}

	// Health-Check (ungeschützt).
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	writeLimit := middleware.NewRateLimiter(cfg.RateLimitPerMinute).Middleware()

	v1 := r.Group("/v1")
	{
		// Auth-Endpoints: offen, aber ratenbegrenzt (Brute-Force-Schutz).
		a := v1.Group("/auth")
		a.Use(writeLimit)
		{
			a.POST("/register", authH.Register)
			a.POST("/login", authH.Login)
			a.POST("/verify-email", authH.VerifyEmail)
		}

		// Geschützte Endpoints.
		secured := v1.Group("")
		secured.Use(middleware.RequireAuth(jwt))
		{
			secured.GET("/me", authH.Me)
		}
	}

	return r, nil
}
