package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeRepo ist ein In-Memory-Repository für Tests (keine echte DB nötig).
type fakeRepo struct {
	users      map[int64]*User
	byEmail    map[string]*User
	invites    map[string]*Invitation // Schlüssel: token-Hash
	evs        map[string]*EmailVerification
	nextUserID int64
	nextEvID   int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		users:   map[int64]*User{},
		byEmail: map[string]*User{},
		invites: map[string]*Invitation{},
		evs:     map[string]*EmailVerification{},
	}
}

func (f *fakeRepo) CountUsers(context.Context) (int, error) { return len(f.users), nil }

func (f *fakeRepo) GetUserByEmail(_ context.Context, email string) (*User, error) {
	if u, ok := f.byEmail[email]; ok {
		return u, nil
	}
	return nil, ErrNotFound
}

func (f *fakeRepo) GetUserByID(_ context.Context, id int64) (*User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, ErrNotFound
}

func (f *fakeRepo) CreateUser(_ context.Context, p CreateUserParams) (*User, error) {
	f.nextUserID++
	u := &User{
		ID:           f.nextUserID,
		Email:        p.Email,
		PasswordHash: p.PasswordHash,
		DisplayName:  p.DisplayName,
		Role:         p.Role,
		Status:       "active",
		InvitedBy:    p.InvitedBy,
		CreatedAt:    time.Now(),
	}
	f.users[u.ID] = u
	f.byEmail[u.Email] = u
	return u, nil
}

func (f *fakeRepo) GetInvitationByToken(_ context.Context, tokenHash string) (*Invitation, error) {
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

func (f *fakeRepo) CreateEmailVerification(_ context.Context, userID int64, tokenHash string, exp time.Time) error {
	f.nextEvID++
	f.evs[tokenHash] = &EmailVerification{ID: f.nextEvID, UserID: userID, Token: tokenHash, ExpiresAt: exp}
	return nil
}

func (f *fakeRepo) GetEmailVerificationByToken(_ context.Context, tokenHash string) (*EmailVerification, error) {
	if ev, ok := f.evs[tokenHash]; ok {
		return ev, nil
	}
	return nil, ErrNotFound
}

func (f *fakeRepo) MarkEmailVerified(_ context.Context, userID int64) error {
	if u, ok := f.users[userID]; ok {
		now := time.Now()
		u.EmailVerifiedAt = &now
	}
	return nil
}

func (f *fakeRepo) DeleteEmailVerification(_ context.Context, id int64) error {
	for k, ev := range f.evs {
		if ev.ID == id {
			delete(f.evs, k)
		}
	}
	return nil
}

func newTestService(repo Repository) *Service {
	jwt := NewJWTManager("test-secret-mindestens-16", time.Hour)
	return NewService(repo, jwt, true, 48*time.Hour)
}

func TestRegisterBootstrapFirstUserIsAdmin(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(repo)

	res, err := svc.Register(context.Background(), RegisterInput{
		Email: "erster@example.de", Password: "geheim1234", DisplayName: "Erster",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if res.User.Role != "admin" {
		t.Errorf("erster Nutzer sollte admin sein, ist %q", res.User.Role)
	}
	if res.User.InvitedBy != nil {
		t.Error("erster Nutzer sollte invited_by = nil haben")
	}
	if res.EmailVerificationToken == "" {
		t.Error("Verifizierungs-Token fehlt")
	}
}

func TestRegisterRequiresInviteAfterFirstUser(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(repo)
	// Erster Nutzer (Bootstrap).
	_, _ = svc.Register(context.Background(), RegisterInput{
		Email: "erster@example.de", Password: "geheim1234", DisplayName: "Erster",
	})
	// Zweiter ohne Token → abgelehnt.
	_, err := svc.Register(context.Background(), RegisterInput{
		Email: "zweiter@example.de", Password: "geheim1234", DisplayName: "Zwei",
	})
	if !errors.Is(err, ErrInviteRequired) {
		t.Fatalf("erwartet ErrInviteRequired, bekam %v", err)
	}
}

func TestRegisterWithValidInvite(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(repo)
	_, _ = svc.Register(context.Background(), RegisterInput{
		Email: "erster@example.de", Password: "geheim1234", DisplayName: "Erster",
	})

	raw := "einladungstoken-roh"
	repo.invites[HashToken(raw)] = &Invitation{ID: 1, InviterID: 1, Token: HashToken(raw), Status: "pending"}

	res, err := svc.Register(context.Background(), RegisterInput{
		InviteToken: raw, Email: "zwei@example.de", Password: "geheim1234", DisplayName: "Zwei",
	})
	if err != nil {
		t.Fatalf("Register mit Invite: %v", err)
	}
	if res.User.InvitedBy == nil || *res.User.InvitedBy != 1 {
		t.Error("invited_by sollte auf den Inviter zeigen")
	}
	if repo.invites[HashToken(raw)].Status != "accepted" {
		t.Error("Einladung sollte als accepted markiert sein")
	}
}

func TestRegisterRejectsExpiredInvite(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(repo)
	_, _ = svc.Register(context.Background(), RegisterInput{
		Email: "erster@example.de", Password: "geheim1234", DisplayName: "Erster",
	})

	raw := "abgelaufen"
	past := time.Now().Add(-time.Hour)
	repo.invites[HashToken(raw)] = &Invitation{ID: 2, InviterID: 1, Token: HashToken(raw), Status: "pending", ExpiresAt: &past}

	_, err := svc.Register(context.Background(), RegisterInput{
		InviteToken: raw, Email: "x@example.de", Password: "geheim1234", DisplayName: "X",
	})
	if !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("erwartet ErrInviteInvalid, bekam %v", err)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(repo)
	_, _ = svc.Register(context.Background(), RegisterInput{
		Email: "erster@example.de", Password: "geheim1234", DisplayName: "Erster",
	})
	raw := "tok"
	repo.invites[HashToken(raw)] = &Invitation{ID: 1, InviterID: 1, Token: HashToken(raw), Status: "pending"}

	_, err := svc.Register(context.Background(), RegisterInput{
		InviteToken: raw, Email: "ERSTER@example.de", Password: "geheim1234", DisplayName: "Doppelt",
	})
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("erwartet ErrEmailTaken, bekam %v", err)
	}
}

func TestLoginFlow(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(repo)
	res, _ := svc.Register(context.Background(), RegisterInput{
		Email: "erster@example.de", Password: "geheim1234", DisplayName: "Erster",
	})

	// Falsches Passwort.
	if _, _, err := svc.Login(context.Background(), "erster@example.de", "falschfalsch"); !errors.Is(err, ErrInvalidLogin) {
		t.Fatalf("erwartet ErrInvalidLogin, bekam %v", err)
	}
	// Noch nicht verifiziert.
	if _, _, err := svc.Login(context.Background(), "erster@example.de", "geheim1234"); !errors.Is(err, ErrEmailUnverified) {
		t.Fatalf("erwartet ErrEmailUnverified, bekam %v", err)
	}
	// Verifizieren, dann Login.
	if err := svc.VerifyEmail(context.Background(), res.EmailVerificationToken); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}
	token, _, err := svc.Login(context.Background(), "erster@example.de", "geheim1234")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if token == "" {
		t.Error("Login sollte ein Token liefern")
	}
}

func TestVerifyEmailInvalidToken(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(repo)
	if err := svc.VerifyEmail(context.Background(), "gibtsnicht"); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("erwartet ErrTokenInvalid, bekam %v", err)
	}
}
