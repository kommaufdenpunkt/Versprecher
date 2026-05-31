// Package fidolin ist der stille KI-Worker (§7): er prüft neue Beiträge auf
// Hass/Übergriffiges und schlägt vor, was gemeint war. Fidolin taucht nie als
// Nutzer auf. Er läuft als Goroutine-Worker-Pool mit Polling.
package fidolin

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/kommaufdenpunkt/insider/internal/moderation"
)

// settingsProvider liefert die aktuellen Schwellen (von moderation.Repository erfüllt).
type settingsProvider interface {
	Get(ctx context.Context) (moderation.Settings, error)
}

// Fidolin bündelt Store, Analyzer und Einstellungen samt Worker-Konfiguration.
type Fidolin struct {
	store      Store
	settings   settingsProvider
	analyzer   Analyzer
	workers    int
	batchSize  int
	pollEvery  time.Duration
	staleAfter time.Duration
	log        *log.Logger
}

// Config steuert das Laufzeitverhalten.
type Config struct {
	Workers      int
	BatchSize    int
	PollInterval time.Duration
	StaleAfter   time.Duration
}

func New(store Store, settings settingsProvider, analyzer Analyzer, cfg Config, logger *log.Logger) *Fidolin {
	if cfg.Workers < 1 {
		cfg.Workers = 2
	}
	if cfg.BatchSize < 1 {
		cfg.BatchSize = 10
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.StaleAfter <= 0 {
		cfg.StaleAfter = 5 * time.Minute
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Fidolin{
		store: store, settings: settings, analyzer: analyzer,
		workers: cfg.Workers, batchSize: cfg.BatchSize,
		pollEvery: cfg.PollInterval, staleAfter: cfg.StaleAfter, log: logger,
	}
}

// Run startet den Worker-Pool und pollt, bis ctx abgebrochen wird (graceful).
func (f *Fidolin) Run(ctx context.Context) {
	jobs := make(chan QueuedPost)
	var wg sync.WaitGroup
	for i := 0; i < f.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				f.process(ctx, p)
			}
		}()
	}
	f.log.Printf("Fidolin gestartet (%d Worker, Poll alle %s)", f.workers, f.pollEvery)

	ticker := time.NewTicker(f.pollEvery)
	defer ticker.Stop()

	dispatch := func() bool { // false = beenden
		if _, err := f.store.ReclaimStale(ctx, f.staleAfter); err != nil {
			f.log.Printf("Fidolin: ReclaimStale-Fehler: %v", err)
		}
		batch, err := f.store.ClaimQueued(ctx, f.batchSize)
		if err != nil {
			f.log.Printf("Fidolin: ClaimQueued-Fehler: %v", err)
			return true
		}
		for _, p := range batch {
			select {
			case jobs <- p:
			case <-ctx.Done():
				return false
			}
		}
		return true
	}

	// Sofort einmal pollen, dann im Takt.
	if dispatch() {
		for {
			select {
			case <-ctx.Done():
				close(jobs)
				wg.Wait()
				f.log.Printf("Fidolin gestoppt")
				return
			case <-ticker.C:
				if !dispatch() {
					close(jobs)
					wg.Wait()
					f.log.Printf("Fidolin gestoppt")
					return
				}
			}
		}
	}
	close(jobs)
	wg.Wait()
}

// RunOnce verarbeitet eine Runde synchron (claim + analysieren). Praktisch für
// Tests und einfache Setups.
func (f *Fidolin) RunOnce(ctx context.Context) (int, error) {
	if _, err := f.store.ReclaimStale(ctx, f.staleAfter); err != nil {
		return 0, err
	}
	batch, err := f.store.ClaimQueued(ctx, f.batchSize)
	if err != nil {
		return 0, err
	}
	for _, p := range batch {
		f.process(ctx, p)
	}
	return len(batch), nil
}

// process analysiert einen Beitrag und schreibt das Ergebnis. Fail-closed:
// bei jedem KI-Fehler geht der Beitrag in die menschliche Prüfung.
func (f *Fidolin) process(ctx context.Context, p QueuedPost) {
	set, err := f.settings.Get(ctx)
	if err != nil {
		set = moderation.Defaults
	}
	a, err := f.analyzer.Analyze(ctx, p.Word, p.Explanation)
	if err != nil {
		f.log.Printf("Fidolin: Analyse fehlgeschlagen (post %d): %v", p.ID, err)
		if mErr := f.store.MarkFailed(ctx, p.ID); mErr != nil {
			f.log.Printf("Fidolin: MarkFailed-Fehler (post %d): %v", p.ID, mErr)
		}
		return
	}
	status := decideStatus(a.Score, set)
	if err := f.store.ApplyAnalysis(ctx, p.ID, a.Score, a.KindSuggestion, a.MeantSuggestion, status); err != nil {
		f.log.Printf("Fidolin: ApplyAnalysis-Fehler (post %d): %v", p.ID, err)
	}
}

// decideStatus bildet den Score über die Schwellen auf den Status ab (§7).
func decideStatus(score float64, s moderation.Settings) string {
	switch {
	case score >= s.AutoRejectThreshold:
		return "blocked"
	case score >= s.HandoffThreshold:
		return "pending_review"
	default:
		return "visible"
	}
}
