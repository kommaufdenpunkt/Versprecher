package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound signalisiert: Datensatz existiert nicht.
var ErrNotFound = errors.New("nicht gefunden")

// Repository ist die Datenzugriffs-Schnittstelle. Dank Interface lässt sich der
// Service ohne echte DB testen (siehe service_test.go).
type Repository interface {
	CountUsers(ctx context.Context) (int, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id int64) (*User, error)
	CreateUser(ctx context.Context, p CreateUserParams) (*User, error)

	GetInvitationByToken(ctx context.Context, tokenHash string) (*Invitation, error)
	AcceptInvitation(ctx context.Context, id int64) error

	CreateEmailVerification(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error
	GetEmailVerificationByToken(ctx context.Context, tokenHash string) (*EmailVerification, error)
	MarkEmailVerified(ctx context.Context, userID int64) error
	DeleteEmailVerification(ctx context.Context, id int64) error
}

// PgRepository ist die PostgreSQL-Implementierung (App-User, nur DML).
type PgRepository struct {
	pool *pgxpool.Pool
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

func (r *PgRepository) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&n)
	return n, err
}

const userColumns = `id, email, email_verified_at, password_hash, display_name, role, status, invited_by, created_at`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.EmailVerifiedAt, &u.PasswordHash,
		&u.DisplayName, &u.Role, &u.Status, &u.InvitedBy, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *PgRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return scanUser(r.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE email = $1`, email))
}

func (r *PgRepository) GetUserByID(ctx context.Context, id int64) (*User, error) {
	return scanUser(r.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1`, id))
}

func (r *PgRepository) CreateUser(ctx context.Context, p CreateUserParams) (*User, error) {
	return scanUser(r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, display_name, role, invited_by)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+userColumns,
		p.Email, p.PasswordHash, p.DisplayName, p.Role, p.InvitedBy))
}

func (r *PgRepository) GetInvitationByToken(ctx context.Context, tokenHash string) (*Invitation, error) {
	var inv Invitation
	err := r.pool.QueryRow(ctx,
		`SELECT id, group_id, inviter_id, token, email, status, expires_at, created_at
		 FROM invitations WHERE token = $1`, tokenHash).
		Scan(&inv.ID, &inv.GroupID, &inv.InviterID, &inv.Token, &inv.Email,
			&inv.Status, &inv.ExpiresAt, &inv.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *PgRepository) AcceptInvitation(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE invitations SET status = 'accepted' WHERE id = $1`, id)
	return err
}

func (r *PgRepository) CreateEmailVerification(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_verifications (user_id, token, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt)
	return err
}

func (r *PgRepository) GetEmailVerificationByToken(ctx context.Context, tokenHash string) (*EmailVerification, error) {
	var ev EmailVerification
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token, expires_at FROM email_verifications WHERE token = $1`, tokenHash).
		Scan(&ev.ID, &ev.UserID, &ev.Token, &ev.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &ev, nil
}

func (r *PgRepository) MarkEmailVerified(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET email_verified_at = now() WHERE id = $1`, userID)
	return err
}

func (r *PgRepository) DeleteEmailVerification(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM email_verifications WHERE id = $1`, id)
	return err
}
