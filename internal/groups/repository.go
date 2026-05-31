package groups

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kommaufdenpunkt/insider/internal/auth"
)

// ErrNotFound: Datensatz existiert nicht.
var ErrNotFound = errors.New("nicht gefunden")

// Repository ist die Datenzugriffs-Schnittstelle für Gruppen (testbar via Fake).
type Repository interface {
	CreateGroup(ctx context.Context, name string, ownerID int64, maxMembers int) (*Group, error)
	GetGroup(ctx context.Context, id int64) (*Group, error)
	ListGroupsForUser(ctx context.Context, userID int64) ([]Group, error)

	MemberRole(ctx context.Context, groupID, userID int64) (string, error) // ErrNotFound = kein Mitglied
	CountMembers(ctx context.Context, groupID int64) (int, error)
	AddMember(ctx context.Context, groupID, userID int64, invitedBy *int64, role string) error

	CreateInvitation(ctx context.Context, groupID, inviterID int64, tokenHash string, email *string, expiresAt *time.Time) (*auth.Invitation, error)
	GetInvitationByToken(ctx context.Context, tokenHash string) (*auth.Invitation, error)
	AcceptInvitation(ctx context.Context, id int64) error
}

// PgRepository: PostgreSQL-Implementierung (App-User, nur DML).
type PgRepository struct {
	pool *pgxpool.Pool
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository { return &PgRepository{pool: pool} }

const groupColumns = `id, name, owner_id, max_members, created_at`

func scanGroup(row pgx.Row) (*Group, error) {
	var g Group
	err := row.Scan(&g.ID, &g.Name, &g.OwnerID, &g.MaxMembers, &g.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *PgRepository) CreateGroup(ctx context.Context, name string, ownerID int64, maxMembers int) (*Group, error) {
	return scanGroup(r.pool.QueryRow(ctx,
		`INSERT INTO groups (name, owner_id, max_members) VALUES ($1, $2, $3) RETURNING `+groupColumns,
		name, ownerID, maxMembers))
}

func (r *PgRepository) GetGroup(ctx context.Context, id int64) (*Group, error) {
	return scanGroup(r.pool.QueryRow(ctx, `SELECT `+groupColumns+` FROM groups WHERE id = $1`, id))
}

func (r *PgRepository) ListGroupsForUser(ctx context.Context, userID int64) ([]Group, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT g.id, g.name, g.owner_id, g.max_members, g.created_at
		 FROM groups g
		 JOIN group_members m ON m.group_id = g.id
		 WHERE m.user_id = $1
		 ORDER BY g.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name, &g.OwnerID, &g.MaxMembers, &g.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *PgRepository) MemberRole(ctx context.Context, groupID, userID int64) (string, error) {
	var role string
	err := r.pool.QueryRow(ctx,
		`SELECT role FROM group_members WHERE group_id = $1 AND user_id = $2`, groupID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return role, err
}

func (r *PgRepository) CountMembers(ctx context.Context, groupID int64) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM group_members WHERE group_id = $1`, groupID).Scan(&n)
	return n, err
}

func (r *PgRepository) AddMember(ctx context.Context, groupID, userID int64, invitedBy *int64, role string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO group_members (group_id, user_id, role, invited_by) VALUES ($1, $2, $3, $4)`,
		groupID, userID, role, invitedBy)
	return err
}

const invitationColumns = `id, group_id, inviter_id, token, email, status, expires_at, created_at`

func scanInvitation(row pgx.Row) (*auth.Invitation, error) {
	var inv auth.Invitation
	err := row.Scan(&inv.ID, &inv.GroupID, &inv.InviterID, &inv.Token, &inv.Email,
		&inv.Status, &inv.ExpiresAt, &inv.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *PgRepository) CreateInvitation(ctx context.Context, groupID, inviterID int64, tokenHash string, email *string, expiresAt *time.Time) (*auth.Invitation, error) {
	return scanInvitation(r.pool.QueryRow(ctx,
		`INSERT INTO invitations (group_id, inviter_id, token, email, expires_at, status)
		 VALUES ($1, $2, $3, $4, $5, 'pending') RETURNING `+invitationColumns,
		groupID, inviterID, tokenHash, email, expiresAt))
}

func (r *PgRepository) GetInvitationByToken(ctx context.Context, tokenHash string) (*auth.Invitation, error) {
	return scanInvitation(r.pool.QueryRow(ctx,
		`SELECT `+invitationColumns+` FROM invitations WHERE token = $1`, tokenHash))
}

func (r *PgRepository) AcceptInvitation(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE invitations SET status = 'accepted' WHERE id = $1`, id)
	return err
}
