package posts

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound: Beitrag existiert nicht.
var ErrNotFound = errors.New("nicht gefunden")

// Repository ist die Datenzugriffs-Schnittstelle für Beiträge (testbar via Fake).
type Repository interface {
	Create(ctx context.Context, p CreateParams) (*Post, error)
	GetByID(ctx context.Context, id int64) (*Post, error)
	ListFeed(ctx context.Context, groupID int64, beforeID int64, limit int) ([]Post, error)
	UpdateMeant(ctx context.Context, id int64, meant string, kind *string) (*Post, error)
}

type PgRepository struct {
	pool *pgxpool.Pool
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository { return &PgRepository{pool: pool} }

// ai_score wird als float8 gelesen (NUMERIC -> *float64).
const postColumns = `id, group_id, author_id, tagged_user_id, word, word_normalized,
	explanation, kind, ai_kind_suggestion, ai_meant_suggestion, meant_confirmed,
	voice_url, ai_score::float8, status, is_pinned, created_at`

func scanPost(row pgx.Row) (*Post, error) {
	var p Post
	err := row.Scan(&p.ID, &p.GroupID, &p.AuthorID, &p.TaggedUserID, &p.Word, &p.WordNormalized,
		&p.Explanation, &p.Kind, &p.AIKindSuggestion, &p.AIMeantSuggestion, &p.MeantConfirmed,
		&p.VoiceURL, &p.AIScore, &p.Status, &p.IsPinned, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Create legt einen Beitrag an. status/ai_status kommen aus den DB-Defaults
// ('pending_review' bzw. 'queued') — der Beitrag ist also zunächst nicht im Feed.
func (r *PgRepository) Create(ctx context.Context, p CreateParams) (*Post, error) {
	return scanPost(r.pool.QueryRow(ctx,
		`INSERT INTO posts (group_id, author_id, tagged_user_id, word, word_normalized, explanation, kind, voice_url)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING `+postColumns,
		p.GroupID, p.AuthorID, p.TaggedUserID, cleanWord(p.Word), p.WordNormalized,
		p.Explanation, p.Kind, p.VoiceURL))
}

func (r *PgRepository) GetByID(ctx context.Context, id int64) (*Post, error) {
	return scanPost(r.pool.QueryRow(ctx, `SELECT `+postColumns+` FROM posts WHERE id = $1`, id))
}

// ListFeed liefert sichtbare Beiträge einer Gruppe, neueste zuerst (Keyset über id).
// beforeID = 0 liefert die erste Seite.
func (r *PgRepository) ListFeed(ctx context.Context, groupID, beforeID int64, limit int) ([]Post, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+postColumns+`
		 FROM posts
		 WHERE group_id = $1 AND status = 'visible' AND ($2 = 0 OR id < $2)
		 ORDER BY id DESC
		 LIMIT $3`, groupID, beforeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.GroupID, &p.AuthorID, &p.TaggedUserID, &p.Word, &p.WordNormalized,
			&p.Explanation, &p.Kind, &p.AIKindSuggestion, &p.AIMeantSuggestion, &p.MeantConfirmed,
			&p.VoiceURL, &p.AIScore, &p.Status, &p.IsPinned, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpdateMeant setzt das vom Autor bestätigte „gemeint" und optional die Sorte (kind).
func (r *PgRepository) UpdateMeant(ctx context.Context, id int64, meant string, kind *string) (*Post, error) {
	return scanPost(r.pool.QueryRow(ctx,
		`UPDATE posts SET meant_confirmed = $2, kind = COALESCE($3, kind)
		 WHERE id = $1
		 RETURNING `+postColumns,
		id, meant, kind))
}
