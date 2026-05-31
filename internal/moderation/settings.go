// Package moderation hält die Moderations-Einstellungen (Schwellen für Fidolin, §7).
// Die Schreib-Endpoints für Moderatoren kommen in Phase 7; Lesen wird hier bereits
// von Fidolin genutzt.
package moderation

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Settings sind die Moderations-Schwellen (0–1).
type Settings struct {
	AutoRejectThreshold float64 // >= -> blocked
	HandoffThreshold    float64 // >= -> pending_review (Mensch)
}

// Defaults gemäß §7.
var Defaults = Settings{AutoRejectThreshold: 0.85, HandoffThreshold: 0.60}

// Repository liest/schreibt die (einzige) Einstellungs-Zeile.
type Repository interface {
	Get(ctx context.Context) (Settings, error)
	Update(ctx context.Context, s Settings) error
}

type PgRepository struct {
	pool *pgxpool.Pool
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository { return &PgRepository{pool: pool} }

// Get liefert die Schwellen; fehlt die Zeile, gelten die Defaults.
func (r *PgRepository) Get(ctx context.Context) (Settings, error) {
	var s Settings
	err := r.pool.QueryRow(ctx,
		`SELECT auto_reject_threshold::float8, handoff_threshold::float8
		 FROM moderation_settings WHERE id = 1`).
		Scan(&s.AutoRejectThreshold, &s.HandoffThreshold)
	if errors.Is(err, pgx.ErrNoRows) {
		return Defaults, nil
	}
	if err != nil {
		return Defaults, err
	}
	return s, nil
}

// Update setzt die Schwellen (genutzt vom Moderations-Tool in Phase 7).
func (r *PgRepository) Update(ctx context.Context, s Settings) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO moderation_settings (id, auto_reject_threshold, handoff_threshold)
		 VALUES (1, $1, $2)
		 ON CONFLICT (id) DO UPDATE
		 SET auto_reject_threshold = EXCLUDED.auto_reject_threshold,
		     handoff_threshold = EXCLUDED.handoff_threshold`,
		s.AutoRejectThreshold, s.HandoffThreshold)
	return err
}
