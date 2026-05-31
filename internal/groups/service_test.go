package groups

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kommaufdenpunkt/insider/internal/auth"
)

// fakeRepo: In-Memory-Repository für Tests (keine echte DB nötig).
type fakeRepo struct {
	groups      map[int64]*Group
	members     map[string]*Member // Schlüssel: "groupID:userID"
	invites     map[string]*auth.Invitation
	nextGroupID int64
	nextInvID   int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		groups:  map[int64]*Group{},
		members: map[string]*Member{},
		invites: map[string]*auth.Invitation{},
	}
}

func memberKey(g, u int64) string { return fmt.Sprintf("%d:%d", g, u) }

func (f *fakeRepo) CreateGroup(_ context.Context, name string, ownerID int64, maxMembers int) (*Group, error) {
	f.nextGroupID++
	g := &Group{ID: f.nextGroupID, Name: name, OwnerID: ownerID, MaxMembers: maxMembers, CreatedAt: time.Now()}
	f.groups[g.ID] = g
	return g, nil
}

func (f *fakeRepo) GetGroup(_ context.Context, id int64) (*Group, error) {
	if g, ok := f.groups[id]; ok {
		return g, nil
	}
	return nil, ErrNotFound
}

func (f *fakeRepo) ListGroupsForUser(_ context.Context, userID int64) ([]Group, error) {
	var out []Group
	for _, m := range f.members {
		if m.UserID == userID {
			out = append(out, *f.groups[m.GroupID])
		}
	}
	return out, nil
}

func (f *fakeRepo) MemberRole(_ context.Context, groupID, userID int64) (string, error) {
	if m, ok := f.members[memberKey(groupID, userID)]; ok {
		return m.Role, nil
	}
	return "", ErrNotFound
}

func (f *fakeRepo) CountMembers(_ context.Context, groupID int64) (int, error) {
	n := 0
	for _, m := range f.members {
		if m.GroupID == groupID {
			n++
		}
	}
	return n, nil
}

func (f *fakeRepo) AddMember(_ context.Context, groupID, userID int64, invitedBy *int64, role string) error {
	f.members[memberKey(groupID, userID)] = &Member{GroupID: groupID, UserID: userID, Role: role, InvitedBy: invitedBy, JoinedAt: time.Now()}
	return nil
}

func (f *fakeRepo) CreateInvitation(_ context.Context, groupID, inviterID int64, tokenHash string, email *string, expiresAt *time.Time) (*auth.Invitation, error) {
	f.nextInvID++
	gid := groupID
	inv := &auth.Invitation{ID: f.nextInvID, GroupID: &gid, InviterID: inviterID, Token: tokenHash, Email: email, Status: "pending", ExpiresAt: expiresAt}
	f.invites[tokenHash] = inv
	return inv, nil
}

func (f *fakeRepo) GetInvitationByToken(_ context.Context, tokenHash string) (*auth.Invitation, error) {
	if inv, ok := f.invites[tokenHash]; ok {
		return inv, nil
	}
	return nil, ErrNotFound
}

func (f *fakeRepo) AcceptInvitation(_ context.Context, id int64) error {
	for _, inv := range f.invites {
		if inv.ID == id {
			inv.Status = "accepted"
		}
	}
	return nil
}

func TestCreateGroupAddsOwner(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	g, err := svc.CreateGroup(context.Background(), 1, "Familie")
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if g.MaxMembers != defaultMaxMembers {
		t.Errorf("max_members sollte %d sein, ist %d", defaultMaxMembers, g.MaxMembers)
	}
	role, err := repo.MemberRole(context.Background(), g.ID, 1)
	if err != nil || role != "owner" {
		t.Errorf("Ersteller sollte owner sein, role=%q err=%v", role, err)
	}
}

func TestCreateGroupRejectsEmptyName(t *testing.T) {
	svc := NewService(newFakeRepo())
	if _, err := svc.CreateGroup(context.Background(), 1, "   "); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("erwartet ErrInvalidInput, bekam %v", err)
	}
}

func TestGetGroupRequiresMembership(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	g, _ := svc.CreateGroup(context.Background(), 1, "Familie")

	if _, err := svc.GetGroup(context.Background(), 99, g.ID); !errors.Is(err, ErrNotMember) {
		t.Fatalf("Nicht-Mitglied sollte ErrNotMember bekommen, bekam %v", err)
	}
	d, err := svc.GetGroup(context.Background(), 1, g.ID)
	if err != nil {
		t.Fatalf("GetGroup (Owner): %v", err)
	}
	if d.MemberCount != 1 || d.MyRole != "owner" {
		t.Errorf("Details falsch: %+v", d)
	}
}

func TestInviteAndAccept(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	g, _ := svc.CreateGroup(context.Background(), 1, "Familie")

	// Nicht-Mitglied darf nicht einladen.
	if _, _, err := svc.CreateInvite(context.Background(), 99, g.ID, nil); !errors.Is(err, ErrNotMember) {
		t.Fatalf("Nicht-Mitglied sollte nicht einladen dürfen, bekam %v", err)
	}

	raw, _, err := svc.CreateInvite(context.Background(), 1, g.ID, nil)
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	// Neuer Nutzer 2 tritt bei.
	joined, err := svc.AcceptInvite(context.Background(), 2, raw)
	if err != nil {
		t.Fatalf("AcceptInvite: %v", err)
	}
	if joined.ID != g.ID {
		t.Error("falsche Gruppe beigetreten")
	}
	if role, _ := repo.MemberRole(context.Background(), g.ID, 2); role != "member" {
		t.Errorf("User 2 sollte member sein, ist %q", role)
	}

	// Denselben (verbrauchten) Token erneut nutzen → einmalig, also ungültig.
	if _, err := svc.AcceptInvite(context.Background(), 2, raw); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("verbrauchter Token sollte ErrInviteInvalid geben, bekam %v", err)
	}

	// Neue, gültige Einladung, aber User 2 ist schon Mitglied → ErrAlreadyMember.
	raw2, _, _ := svc.CreateInvite(context.Background(), 1, g.ID, nil)
	if _, err := svc.AcceptInvite(context.Background(), 2, raw2); !errors.Is(err, ErrAlreadyMember) {
		t.Fatalf("erwartet ErrAlreadyMember, bekam %v", err)
	}
}

func TestAcceptInviteInvalidToken(t *testing.T) {
	svc := NewService(newFakeRepo())
	if _, err := svc.AcceptInvite(context.Background(), 2, "gibtsnicht"); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("erwartet ErrInviteInvalid, bekam %v", err)
	}
}

func TestAcceptInviteGroupFull(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	g, _ := svc.CreateGroup(context.Background(), 1, "Klein")
	g.MaxMembers = 1 // Owner füllt die Gruppe bereits

	raw, _, _ := svc.CreateInvite(context.Background(), 1, g.ID, nil)
	if _, err := svc.AcceptInvite(context.Background(), 2, raw); !errors.Is(err, ErrGroupFull) {
		t.Fatalf("erwartet ErrGroupFull, bekam %v", err)
	}
}

func TestJoinerCapacityAndAddMember(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	g, _ := svc.CreateGroup(context.Background(), 1, "Familie")

	ok, err := svc.HasCapacity(context.Background(), g.ID)
	if err != nil || !ok {
		t.Fatalf("HasCapacity sollte true sein: ok=%v err=%v", ok, err)
	}
	inviter := int64(1)
	if err := svc.AddMember(context.Background(), g.ID, 5, &inviter); err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	if role, _ := repo.MemberRole(context.Background(), g.ID, 5); role != "member" {
		t.Errorf("User 5 sollte member sein, ist %q", role)
	}
}
