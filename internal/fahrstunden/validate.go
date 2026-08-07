package fahrstunden

import (
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxNameLen   = 120
	maxKlasseLen = 20
	maxNotizLen  = 500
	maxGrundLen  = 300

	// Plausibilitätsgrenzen für eine einzelne Fahrstunde (das echte Tageslimit
	// prüft der Service gegen die Konfiguration).
	minDauer = 1
	maxDauer = 1440

	// Unterschrift: PNG als data-URL. 250 KB reichen für ein Unterschriftsfeld
	// bequem und begrenzen zugleich, was die Datenbank aufnimmt.
	maxUnterschriftLen = 250 * 1024

	unterschriftPrefix = "data:image/png;base64,"
)

// gueltigeArt prüft die Art gegen die erlaubten Werte (identisch zum CHECK in der Migration).
func gueltigeArt(art string) bool {
	_, ok := ArtLabels[art]
	return ok
}

// bereinigeName trimmt und begrenzt einen Namen; "" heißt ungültig.
func bereinigeName(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" || utf8.RuneCountInString(s) > maxNameLen {
		return ""
	}
	return s
}

// kuerze schneidet Freitext auf die erlaubte Länge (Zeichen, nicht Bytes).
func kuerze(s string, max int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max])
}

// plausiblesDatum fängt Tippfehler ab (Jahreszahl völlig daneben).
func plausiblesDatum(t time.Time) bool {
	return !t.IsZero() && t.Year() >= 2000 && t.Year() <= 2100
}

// gueltigeUnterschrift prüft Format und Größe der erfassten Unterschrift.
func gueltigeUnterschrift(s string) bool {
	return strings.HasPrefix(s, unterschriftPrefix) &&
		len(s) > len(unterschriftPrefix) &&
		len(s) <= maxUnterschriftLen
}
