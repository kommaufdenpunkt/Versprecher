// Package httpx baut den Gin-Router mit allen Routen zusammen.
package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kommaufdenpunkt/insider/internal/auth"
	"github.com/kommaufdenpunkt/insider/internal/config"
	"github.com/kommaufdenpunkt/insider/internal/fahrstunden"
	"github.com/kommaufdenpunkt/insider/internal/groups"
	"github.com/kommaufdenpunkt/insider/internal/middleware"
	"github.com/kommaufdenpunkt/insider/internal/posts"
)

// NewRouter erstellt den Router. Alle App-Routen liegen unter /v1 (§3, §10).
func NewRouter(cfg *config.Config, jwt *auth.JWTManager, authH *auth.Handler, groupsH *groups.Handler,
	postsH *posts.Handler, fahrH *fahrstunden.Handler, fahrWeb *fahrstunden.WebHandler) (*gin.Engine, error) {
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

		// Geschützte Endpoints (gültiges JWT nötig).
		secured := v1.Group("")
		secured.Use(middleware.RequireAuth(jwt))
		{
			secured.GET("/me", authH.Me)

			// Gruppen (Phase 2). Lesen frei, Schreiben ratenbegrenzt.
			secured.GET("/groups", groupsH.ListMyGroups)
			secured.GET("/groups/:id", groupsH.GetGroup)
			secured.POST("/groups", writeLimit, groupsH.CreateGroup)
			secured.POST("/groups/:id/invite", writeLimit, groupsH.CreateInvite)
			secured.POST("/invitations/:token/accept", writeLimit, groupsH.AcceptInvite)

			// Beiträge + Feed (Phase 3).
			secured.GET("/groups/:id/feed", postsH.Feed)
			secured.POST("/groups/:id/posts", writeLimit, postsH.Create)
			secured.PATCH("/posts/:id", writeLimit, postsH.ConfirmMeant)

			// Fahrstunden-Nachweis (Nebenbuch zum FS Manager). Rein privat:
			// jede Route arbeitet ausschließlich auf den Daten der angemeldeten
			// Fahrlehrerin bzw. des angemeldeten Fahrlehrers (uid aus dem JWT).
			f := secured.Group("/fahrstunden")
			{
				f.GET("/stammdaten", fahrH.Stammdaten)

				f.GET("/schueler", fahrH.ListSchueler)
				f.POST("/schueler", writeLimit, fahrH.CreateSchueler)
				f.PATCH("/schueler/:id", writeLimit, fahrH.UpdateSchueler)
				f.GET("/schueler/:id/nachweis.pdf", fahrH.Nachweis)

				f.GET("/stunden", fahrH.ListStunden)
				f.POST("/stunden", writeLimit, fahrH.CreateStunde)
				f.PATCH("/stunden/:id", writeLimit, fahrH.UpdateStunde)
				f.DELETE("/stunden/:id", writeLimit, fahrH.DeleteStunde)
				f.PUT("/stunden/:id/unterschrift", writeLimit, fahrH.Unterschreiben)

				f.GET("/kapazitaet", fahrH.Kapazitaet)
				f.GET("/kapazitaet/vorschlaege", fahrH.Vorschlaege)
			}
		}
	}

	// Oberfläche des Fahrstunden-Nachweises unter einem frei wählbaren Pfad
	// (FAHRSTUNDEN_BASIS_PFAD) — so kann sie später unter eigenem Namen oder
	// einer eigenen Domain laufen, ohne Änderung im Code.
	basis := cfg.FahrstundenBasisPfad
	r.GET(basis, fahrWeb.Seite)
	r.GET(basis+"/", fahrWeb.Seite)
	r.GET(basis+"/konfig.json", fahrWeb.Konfig)

	return r, nil
}
