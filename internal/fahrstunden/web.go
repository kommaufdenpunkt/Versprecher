package fahrstunden

import (
	"embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed web/index.html
var webDateien embed.FS

// WebHandler liefert die Oberfläche des Fahrstunden-Nachweises.
//
// Die Seite ist bewusst eine einzelne, eingebettete Datei: kein Build-Schritt,
// keine externen Abhängigkeiten, funktioniert auch offline im Browser-Cache.
type WebHandler struct {
	appName    string
	apiBasis   string
	authBasis  string
	limit      int
	fahrschule string
}

func NewWebHandler(appName, apiBasis, authBasis string, limitMinuten int, fahrschule string) *WebHandler {
	if limitMinuten <= 0 {
		limitMinuten = StandardTageslimit
	}
	return &WebHandler{
		appName:    appName,
		apiBasis:   apiBasis,
		authBasis:  authBasis,
		limit:      limitMinuten,
		fahrschule: fahrschule,
	}
}

// Seite liefert die Oberfläche.
func (w *WebHandler) Seite(c *gin.Context) {
	daten, err := webDateien.ReadFile("web/index.html")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "interner Fehler"})
		return
	}
	// Kein Caching der HTML-Hülle: so kommt eine neue Version sofort an.
	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "text/html; charset=utf-8", daten)
}

// Konfig liefert der Oberfläche ihre Startwerte. Bewusst ohne Anmeldung:
// hier stehen nur Anzeigename, API-Pfade und das Tageslimit — keine Daten.
func (w *WebHandler) Konfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"app_name":      w.appName,
		"api_basis":     w.apiBasis,
		"auth_basis":    w.authBasis,
		"limit_minuten": w.limit,
		"fahrschule":    w.fahrschule,
	})
}
