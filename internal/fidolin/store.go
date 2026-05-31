package fidolin

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// QueuedPost ist ein zur Prüfung übernommener Beitrag (Minimaldaten).
type QueuedPost struct {
	ID          int64
	Word        string
	Explanation string
}

// Store ist Fidolins Zugriff auf die Moderations-Spalten der posts-Tabelle.
// Eigenes Interface (kein Import des posts-Pakets) — beide Pakete teilen sich
// nur die Tabelle, nicht die Go-Typen.
type Store interface {
	// ClaimQueued übernimmt bis zu limit offene Beiträge atomar (markiert sie als
	// 'processing'). Dank FOR UPDATE SKIP LOCKED greifen mehrere Worker/Instanzen
	// nie denselben Beitrag.
	ClaimQueued(ctx context.Context, limit int) ([]QueuedPost, error)
	// ApplyAnalysis schreibt das Ergebnis und den finalen Moderationsstatus.
	ApplyAnalysis(ctx context.Context, id int64, score float64, kindSuggestion, meantSuggestion, status string) error
	// MarkFailed: bei KI-Fehler sicherheitshalber zum Menschen (pending_review).
	MarkFailed(ctx context.Context, id int64) error
	// ReclaimStale stellt zu lange 'processing' gebliebene Beiträge zurück auf
	// 'queued' (z. B. nach einem Absturz).
	ReclaimStale(ctx context.Context, olderThan time.Duration) (int64, error)
}

type PgStore struct {
	pool *pgxpool.Pool
}

func NewPgStore(pool *pgxpool.Pool) *PgStore { return &PgStore{pool: pool} }

func (s *PgStore) ClaimQueued(ctx context.Context, limit int) ([]QueuedPost, error) {
	rows, err := s.pool.Query(ctx,
		`WITH claimed AS (
		     SELECT id FROM posts
		     WHERE ai_status = 'queued'
		     ORDER BY created_at
		     LIMIT $1
		     FOR UPDATE SKIP LOCKED
		 )
		 UPDATE posts p
		 SET ai_status = 'processing', ai_claimed_at = now()
		 FROM claimed
		 WHERE p.id = claimed.id
		 RETURNING p.id, p.word, p.explanation`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []QueuedPost
	for rows.Next() {
		var q QueuedPost
		if err := rows.Scan(&q.ID, &q.Word, &q.Explanation); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

func (s *PgStore) ApplyAnalysis(ctx context.Context, id int64, score float64, kindSuggestion, meantSuggestion, status string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE posts
		 SET ai_score = $2,
		     ai_kind_suggestion = NULLIF($3, ''),
		     ai_meant_suggestion = NULLIF($4, ''),
		     status = $5,
		     ai_status = 'analyzed'
		 WHERE id = $1`,
		id, score, kindSuggestion, meantSuggestion, status)
	return err
}

func (s *PgStore) MarkFailed(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE posts SET status = 'pending_review', ai_status = 'failed' WHERE id = $1`, id)
	return err
}

func (s *PgStore) ReclaimStale(ctx context.Context, olderThan time.Duration) (int64, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE posts SET ai_status = 'queued'
		 WHERE ai_status = 'processing'
		   AND ai_claimed_at < now() - make_interval(secs => $1)`,
		olderThan.Seconds())
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
