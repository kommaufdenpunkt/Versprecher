package posts

import (
	"regexp"
	"strings"
)

// Hashtag-Regel (§5): genau ein Wort, max. 12 Buchstaben (auch Umlaute/ß),
// optionales führendes '#'. Das geteilte Wort *ist* das Hashtag.
var wordRe = regexp.MustCompile(`^#?[A-Za-zÄÖÜäöüß]{1,12}$`)

const maxExplanationLen = 2000

// validWord prüft die Hashtag-Regel auf der getrimmten Eingabe.
func validWord(raw string) bool {
	return wordRe.MatchString(strings.TrimSpace(raw))
}

// cleanWord entfernt Whitespace und ein führendes '#' (Anzeigeform).
func cleanWord(raw string) string {
	return strings.TrimPrefix(strings.TrimSpace(raw), "#")
}

// normalizeWord ist der Aggregations-Schlüssel: kleingeschrieben, ohne '#'.
func normalizeWord(raw string) string {
	return strings.ToLower(cleanWord(raw))
}

func validKind(kind string) bool {
	return kind == "versprecher" || kind == "verhoerer"
}
