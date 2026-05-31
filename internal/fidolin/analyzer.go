package fidolin

import (
	"context"
	"strings"
)

// Analysis ist Fidolins Ergebnis zu einem Beitrag (§7).
type Analysis struct {
	Score           float64 // 0–1, Problemgrad (1 = klar problematisch)
	Reason          string
	KindSuggestion  string // "versprecher" | "verhoerer"
	MeantSuggestion string // Vorschlag, was gemeint war (nur Vorschlag)
}

// Analyzer prüft einen Beitrag. Bewusst als Interface: die mitgelieferte
// Heuristik läuft offline; ein echter LLM-Analyzer kann später eingesteckt
// werden, ohne den Worker zu ändern.
type Analyzer interface {
	Analyze(ctx context.Context, word, explanation string) (Analysis, error)
}

// DefaultBlocklist: minimaler Platzhalter für eindeutig hasserfüllte/gewaltvolle
// Begriffe. Produktiv sollte der LLM-Analyzer + eine gepflegte Liste genutzt
// werden (per MODERATION_BLOCKLIST überschreibbar). Wichtig (§7): lustige
// Versprecher selbst werden NICHT gefiltert — nur Hass/Übergriffiges.
var DefaultBlocklist = []string{
	"judensau", "neger", "schwuchtel", "vergasen", "heil hitler",
}

// HeuristicAnalyzer ist eine einfache, deterministische Offline-Prüfung:
// enthält der Text einen Begriff der Blockliste, ist der Score hoch; sonst
// niedrig. Die „gemeint"-Erkennung bleibt bewusst leer (rät die KI später).
type HeuristicAnalyzer struct {
	blocklist []string
}

func NewHeuristicAnalyzer(blocklist []string) *HeuristicAnalyzer {
	lowered := make([]string, 0, len(blocklist))
	for _, w := range blocklist {
		if s := strings.ToLower(strings.TrimSpace(w)); s != "" {
			lowered = append(lowered, s)
		}
	}
	return &HeuristicAnalyzer{blocklist: lowered}
}

func (h *HeuristicAnalyzer) Analyze(_ context.Context, word, explanation string) (Analysis, error) {
	text := strings.ToLower(word + " " + explanation)
	for _, bad := range h.blocklist {
		if strings.Contains(text, bad) {
			return Analysis{
				Score:          0.95,
				Reason:         "möglicher Hassinhalt",
				KindSuggestion: guessKind(explanation),
			}, nil
		}
	}
	return Analysis{
		Score:          0.05,
		Reason:         "unauffällig",
		KindSuggestion: guessKind(explanation),
	}, nil
}

// guessKind: grobe Heuristik für die Sorte. Nur ein Vorschlag — der Autor
// bestätigt/ändert ihn (§7).
func guessKind(explanation string) string {
	e := strings.ToLower(explanation)
	for _, hint := range []string{"gehört", "gehoert", "verstanden", "verhört", "verhoert", "klang wie", "klingt wie"} {
		if strings.Contains(e, hint) {
			return "verhoerer"
		}
	}
	return "versprecher"
}
