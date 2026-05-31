package groups

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/kommaufdenpunkt/insider/internal/auth"
)

var (
	ErrInvalidInput  = errors.New("ungültige Eingabe")
	ErrNotMember     = errors.New("kein Mitglied dieser Gruppe")
	ErrGroupNotFound = errors.New("Gruppe nicht gefunden")
	ErrAlreadyMember = errors.New("bereits Mitglied")
	ErrGroupFull     = errors.New("Gruppe ist voll")
	ErrInviteInvalid = errors.New("Einladung ungültig oder abgelaufen")
)

const (
	defaultMaxMembers = 30
	inviteTTL         = 7 * 24 * time.Hour
)

// Service bündelt die Gruppen-Logik. Hängt nur am Repository-Interface.
type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// CreateGroup gründet eine Gruppe; der Ersteller wird Owner und erstes Mitglied.
func (s *Service) CreateGroup(ctx context.Context, ownerID int64, name string) (*Group, error) {
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 80 {
		return nil, ErrInvalidInput
	}
	g, err := s.repo.CreateGroup(ctx, name, ownerID, defaultMaxMembers)
	if err != nil {
		return nil, err
	}
	if err := s.repo.AddMember(ctx, g.ID, ownerID, nil, "owner"); err != nil {
		return nil, err
	}
	return g, nil
}

// ListMyGroups liefert alle Gruppen, in denen der Nutzer Mitglied ist.
func (s *Service) ListMyGroups(ctx context.Context, userID int64) ([]Group, error) {
	return s.repo.ListGroupsForUser(ctx, userID)
}

// IsMember prüft, ob ein Nutzer Mitglied einer Gruppe ist (für andere Pakete,
// z. B. posts: Autorisierung beim Posten/Lesen).
func (s *Service) IsMember(ctx context.Context, groupID, userID int64) (bool, error) {
	_, err := s.repo.MemberRole(ctx, groupID, userID)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetGroup liefert Details — nur für Mitglieder.
func (s *Service) GetGroup(ctx context.Context, userID, groupID int64) (*GroupDetails, error) {
	role, err := s.repo.MemberRole(ctx, groupID, userID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrNotMember
	}
	if err != nil {
		return nil, err
	}
	g, err := s.repo.GetGroup(ctx, groupID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, err
	}
	count, err := s.repo.CountMembers(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return &GroupDetails{Group: *g, MyRole: role, MemberCount: count}, nil
}

// CreateInvite erstellt eine Einladung für eine Gruppe (nur Mitglieder).
// Gibt das Rohtoken zurück (für den Einladungslink); gespeichert wird nur der Hash.
func (s *Service) CreateInvite(ctx context.Context, userID, groupID int64, email *string) (rawToken string, inv *auth.Invitation, err error) {
	if _, err := s.repo.MemberRole(ctx, groupID, userID); errors.Is(err, ErrNotFound) {
		return "", nil, ErrNotMember
	} else if err != nil {
		return "", nil, err
	}

	raw, hash, err := auth.NewToken()
	if err != nil {
		return "", nil, err
	}
	expires := s.now().Add(inviteTTL)
	inv, err = s.repo.CreateInvitation(ctx, groupID, userID, hash, normalizeEmail(email), &expires)
	if err != nil {
		return "", nil, err
	}
	return raw, inv, nil
}

// AcceptInvite lässt einen bereits angemeldeten Nutzer einer Gruppe beitreten.
func (s *Service) AcceptInvite(ctx context.Context, userID int64, rawToken string) (*Group, error) {
	if rawToken == "" {
		return nil, ErrInviteInvalid
	}
	inv, err := s.repo.GetInvitationByToken(ctx, auth.HashToken(rawToken))
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInviteInvalid
	}
	if err != nil {
		return nil, err
	}
	if !s.invitationUsable(inv) || inv.GroupID == nil {
		return nil, ErrInviteInvalid
	}
	groupID := *inv.GroupID

	// Schon Mitglied?
	if _, err := s.repo.MemberRole(ctx, groupID, userID); err == nil {
		return nil, ErrAlreadyMember
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	g, err := s.repo.GetGroup(ctx, groupID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrGroupNotFound
	}
	if err != nil {
		return nil, err
	}

	count, err := s.repo.CountMembers(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if count >= g.MaxMembers {
		return nil, ErrGroupFull
	}

	inviter := inv.InviterID
	if err := s.repo.AddMember(ctx, groupID, userID, &inviter, "member"); err != nil {
		return nil, err
	}
	if err := s.repo.AcceptInvitation(ctx, inv.ID); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *Service) invitationUsable(inv *auth.Invitation) bool {
	if inv.Status != "pending" {
		return false
	}
	if inv.ExpiresAt != nil && inv.ExpiresAt.Before(s.now()) {
		return false
	}
	return true
}

// --- auth.GroupJoiner: wird bei der Registrierung mit Gruppen-Einladung genutzt ---

// HasCapacity meldet, ob die Gruppe noch Platz hat (max_members).
func (s *Service) HasCapacity(ctx context.Context, groupID int64) (bool, error) {
	g, err := s.repo.GetGroup(ctx, groupID)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	count, err := s.repo.CountMembers(ctx, groupID)
	if err != nil {
		return false, err
	}
	return count < g.MaxMembers, nil
}

// AddMember fügt einen Nutzer als einfaches Mitglied hinzu (für die Registrierung).
func (s *Service) AddMember(ctx context.Context, groupID, userID int64, invitedBy *int64) error {
	return s.repo.AddMember(ctx, groupID, userID, invitedBy, "member")
}

func normalizeEmail(email *string) *string {
	if email == nil {
		return nil
	}
	e := auth.NormalizeEmail(*email)
	if e == "" {
		return nil
	}
	return &e
}
