package fidolin

import (
	"context"
	"testing"
)

func TestHeuristicHarmless(t *testing.T) {
	a := NewHeuristicAnalyzer(DefaultBlocklist)
	res, err := a.Analyze(context.Background(), "Spabl", "Spatzl + Baby")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if res.Score >= 0.5 {
		t.Errorf("harmloser Versprecher sollte niedrigen Score haben, ist %v", res.Score)
	}
	if res.KindSuggestion != "versprecher" {
		t.Errorf("KindSuggestion sollte 'versprecher' sein, ist %q", res.KindSuggestion)
	}
}

func TestHeuristicHateHighScore(t *testing.T) {
	a := NewHeuristicAnalyzer([]string{"boeseswort"})
	res, _ := a.Analyze(context.Background(), "Test", "enthält boeseswort hier")
	if res.Score < 0.85 {
		t.Errorf("Hassinhalt sollte hohen Score haben, ist %v", res.Score)
	}
}

func TestGuessKindVerhoerer(t *testing.T) {
	a := NewHeuristicAnalyzer(nil)
	res, _ := a.Analyze(context.Background(), "Fernsehturnier", "ich habe Fernsehturm gehört")
	if res.KindSuggestion != "verhoerer" {
		t.Errorf("sollte 'verhoerer' vorschlagen, ist %q", res.KindSuggestion)
	}
}
