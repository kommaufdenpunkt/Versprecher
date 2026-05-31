package fidolin

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"
	"time"

	"github.com/kommaufdenpunkt/insider/internal/moderation"
)

type appliedResult struct {
	score       float64
	kind, meant string
	status      string
}

// fakeStore: In-Memory-Ersatz für die DB.
type fakeStore struct {
	queued  []QueuedPost
	applied map[int64]appliedResult
	failed  map[int64]bool
}

func newFakeStore() *fakeStore {
	return &fakeStore{applied: map[int64]appliedResult{}, failed: map[int64]bool{}}
}

func (s *fakeStore) ClaimQueued(_ context.Context, limit int) ([]QueuedPost, error) {
	if len(s.queued) > limit {
		batch := s.queued[:limit]
		s.queued = s.queued[limit:]
		return batch, nil
	}
	batch := s.queued
	s.queued = nil
	return batch, nil
}

func (s *fakeStore) ApplyAnalysis(_ context.Context, id int64, score float64, kind, meant, status string) error {
	s.applied[id] = appliedResult{score: score, kind: kind, meant: meant, status: status}
	return nil
}

func (s *fakeStore) MarkFailed(_ context.Context, id int64) error {
	s.failed[id] = true
	return nil
}

func (s *fakeStore) ReclaimStale(_ context.Context, _ time.Duration) (int64, error) { return 0, nil }

type fakeSettings struct{ s moderation.Settings }

func (f fakeSettings) Get(context.Context) (moderation.Settings, error) { return f.s, nil }

// stubAnalyzer liefert ein festes Ergebnis oder einen Fehler.
type stubAnalyzer struct {
	res Analysis
	err error
}

func (a stubAnalyzer) Analyze(context.Context, string, string) (Analysis, error) {
	return a.res, a.err
}

func quietLogger() *log.Logger { return log.New(io.Discard, "", 0) }

func newTestFidolin(store Store, an Analyzer) *Fidolin {
	return New(store, fakeSettings{moderation.Defaults}, an, Config{Workers: 1, BatchSize: 10}, quietLogger())
}

func TestRunOnceHarmlessBecomesVisible(t *testing.T) {
	store := newFakeStore()
	store.queued = []QueuedPost{{ID: 1, Word: "Spabl", Explanation: "Spatzl + Baby"}}
	f := newTestFidolin(store, NewHeuristicAnalyzer(DefaultBlocklist))

	n, err := f.RunOnce(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("RunOnce: n=%d err=%v", n, err)
	}
	if store.applied[1].status != "visible" {
		t.Errorf("harmloser Beitrag sollte visible sein, ist %q", store.applied[1].status)
	}
}

func TestRunOnceHateBlocked(t *testing.T) {
	store := newFakeStore()
	store.queued = []QueuedPost{{ID: 2, Word: "x", Explanation: "boeseswort"}}
	f := newTestFidolin(store, NewHeuristicAnalyzer([]string{"boeseswort"}))

	if _, err := f.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if store.applied[2].status != "blocked" {
		t.Errorf("Hassinhalt sollte blocked sein, ist %q", store.applied[2].status)
	}
}

func TestRunOnceAnalyzerErrorFailsClosed(t *testing.T) {
	store := newFakeStore()
	store.queued = []QueuedPost{{ID: 3, Word: "x"}}
	f := newTestFidolin(store, stubAnalyzer{err: errors.New("LLM weg")})

	if _, err := f.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if !store.failed[3] {
		t.Error("bei KI-Fehler sollte der Beitrag als failed (Mensch prüft) markiert werden")
	}
	if _, ok := store.applied[3]; ok {
		t.Error("bei KI-Fehler sollte kein Status automatisch gesetzt werden")
	}
}

func TestDecideStatusThresholds(t *testing.T) {
	s := moderation.Settings{AutoRejectThreshold: 0.85, HandoffThreshold: 0.60}
	cases := []struct {
		score float64
		want  string
	}{
		{0.05, "visible"},
		{0.60, "pending_review"},
		{0.84, "pending_review"},
		{0.85, "blocked"},
		{0.99, "blocked"},
	}
	for _, c := range cases {
		if got := decideStatus(c.score, s); got != c.want {
			t.Errorf("decideStatus(%v) = %q, erwartet %q", c.score, got, c.want)
		}
	}
}
