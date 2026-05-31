package auth

import (
	"context"
	"errors"
	"time"
)

// Fehler des Service. Generisch gehalten, um keine Rückschlüsse zu erlauben
// (z. B. ob eine E-Mail existiert) — Schutz vor User-Enumeration.
var (
	ErrInvalidInput    = errors.New("ungültige Eingabe")
	ErrInviteRequired  = errors.New("gültiger Einladungs-Token erforderlich")
	ErrInviteInvalid   = errors.New("Einladung ungültig oder abgelaufen")
	ErrEmailTaken      = errors.New("Registrierung nicht möglich")
	ErrInvalidLogin    = errors.New("E-Mail oder Passwort falsch")
	ErrEmailUnverified = errors.New("E-Mail noch nicht bestätigt")
	ErrAccountBlocked  = errors.New("Konto gesperrt")
	ErrTokenInvalid    = errors.New("Token ungültig oder abgelaufen")
	ErrGroupFull       = errors.New("Gruppe ist voll")
)

// GroupJoiner verbindet die Registrierung mit dem Gruppenbeitritt (Phase 2).
// Wird per Interface eingebunden, damit auth nicht von groups abhängt (kein Zyklus).
// Bei nil (z. B. in Phase-1-Tests) erfolgt kein Gruppenbeitritt.
type GroupJoiner interface {
	HasCapacity(ctx context.Context, groupID int64) (bool, error)
	AddMember(ctx context.Context, groupID, userID int64, invitedBy *int64) error
}

// Service bündelt die Auth-Geschäftslogik. Hängt nur am Repository-Interface,
// am JWTManager und an wenigen Konfig-Flags — bewusst schlank gehalten.
type Service struct {
	repo                     Repository
	jwt                      *JWTManager
	joiner                   GroupJoiner // optional; bindet den Gruppenbeitritt an die Registrierung
	requireEmailVerification bool
	emailVerificationTTL     time.Duration
	now                      func() time.Time // für Tests überschreibbar
}

func NewService(repo Repository, jwt *JWTManager, joiner GroupJoiner, requireEmailVerification bool, emailVerificationTTL time.Duration) *Service {
	return &Service{
		repo:                     repo,
		jwt:                      jwt,
		joiner:                   joiner,
		requireEmailVerification: requireEmailVerification,
		emailVerificationTTL:     emailVerificationTTL,
		now:                      time.Now,
	}
}

// RegisterInput sind die Eingaben der Registrierung.
type RegisterInput struct {
	InviteToken string
	Email       string
	Password    string
	DisplayName string
}

// RegisterResult enthält den neuen Nutzer und das (im Dev zurückgegebene)
// E-Mail-Verifizierungs-Token. In Produktion wird das Token per Mail versandt.
type RegisterResult struct {
	User                   *User
	EmailVerificationToken string
}

// Register legt ein Konto an. Regeln (§2, §11):
//   - Nur auf Einladung. Ausnahme: der allererste Account (Bootstrap) — wird als
//     admin angelegt und hat invited_by = NULL.
//   - Einladung muss pending, nicht abgelaufen und (falls vorgegeben) für diese
//     E-Mail bestimmt sein. Sie ist einmalig (wird auf accepted gesetzt).
func (s *Service) Register(ctx context.Context, in RegisterInput) (*RegisterResult, error) {
	email := NormalizeEmail(in.Email)
	if !validEmail(email) || !validPassword(in.Password) || !validDisplayName(in.DisplayName) {
		return nil, ErrInvalidInput
	}

	count, err := s.repo.CountUsers(ctx)
	if err != nil {
		return nil, err
	}

	role := "user"
	var invitedBy *int64
	var inviteToAccept *Invitation

	if count == 0 {
		// Bootstrap: erster Account wird Admin, ohne Einladung.
		role = "admin"
	} else {
		if in.InviteToken == "" {
			return nil, ErrInviteRequired
		}
		inv, err := s.repo.GetInvitationByToken(ctx, HashToken(in.InviteToken))
		if errors.Is(err, ErrNotFound) {
			return nil, ErrInviteInvalid
		}
		if err != nil {
			return nil, err
		}
		if !s.invitationUsable(inv, email) {
			return nil, ErrInviteInvalid
		}
		invitedBy = &inv.InviterID
		inviteToAccept = inv
	}

	// Gehört die Einladung zu einer Gruppe? Dann früh prüfen, ob noch Platz ist,
	// bevor wir ein Konto anlegen (vermeidet "Konto ohne Gruppe").
	if inviteToAccept != nil && inviteToAccept.GroupID != nil && s.joiner != nil {
		hasRoom, err := s.joiner.HasCapacity(ctx, *inviteToAccept.GroupID)
		if err != nil {
			return nil, err
		}
		if !hasRoom {
			return nil, ErrGroupFull
		}
	}

	// E-Mail bereits vergeben? Generische Fehlermeldung (keine Enumeration).
	if existing, err := s.repo.GetUserByEmail(ctx, email); err == nil && existing != nil {
		return nil, ErrEmailTaken
	} else if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	hash, err := HashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.CreateUser(ctx, CreateUserParams{
		Email:        email,
		PasswordHash: hash,
		DisplayName:  in.DisplayName,
		Role:         role,
		InvitedBy:    invitedBy,
	})
	if err != nil {
		return nil, err
	}

	if inviteToAccept != nil {
		// Bei Gruppen-Einladung den neuen Nutzer direkt der Gruppe hinzufügen.
		if inviteToAccept.GroupID != nil && s.joiner != nil {
			if err := s.joiner.AddMember(ctx, *inviteToAccept.GroupID, user.ID, &inviteToAccept.InviterID); err != nil {
				return nil, err
			}
		}
		if err := s.repo.AcceptInvitation(ctx, inviteToAccept.ID); err != nil {
			return nil, err
		}
	}

	// E-Mail-Verifizierungs-Token erzeugen.
	raw, tokenHash, err := NewToken()
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateEmailVerification(ctx, user.ID, tokenHash, s.now().Add(s.emailVerificationTTL)); err != nil {
		return nil, err
	}

	return &RegisterResult{User: user, EmailVerificationToken: raw}, nil
}

func (s *Service) invitationUsable(inv *Invitation, email string) bool {
	if inv.Status != "pending" {
		return false
	}
	if inv.ExpiresAt != nil && inv.ExpiresAt.Before(s.now()) {
		return false
	}
	if inv.Email != nil && NormalizeEmail(*inv.Email) != email {
		return false
	}
	return true
}

// Login prüft Zugangsdaten und gibt ein JWT zurück.
func (s *Service) Login(ctx context.Context, email, password string) (string, *User, error) {
	email = NormalizeEmail(email)

	user, err := s.repo.GetUserByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		// Trotzdem einen bcrypt-Vergleich ausführen wäre ideal gegen Timing;
		// hier generische Meldung — keine Enumeration.
		return "", nil, ErrInvalidLogin
	}
	if err != nil {
		return "", nil, err
	}
	if !CheckPassword(user.PasswordHash, password) {
		return "", nil, ErrInvalidLogin
	}
	if user.Status != "active" {
		return "", nil, ErrAccountBlocked
	}
	if s.requireEmailVerification && !user.EmailVerified() {
		return "", nil, ErrEmailUnverified
	}

	token, err := s.jwt.Issue(user.ID, user.IsAdmin())
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

// VerifyEmail bestätigt eine E-Mail anhand des Verifizierungs-Tokens.
func (s *Service) VerifyEmail(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return ErrTokenInvalid
	}
	ev, err := s.repo.GetEmailVerificationByToken(ctx, HashToken(rawToken))
	if errors.Is(err, ErrNotFound) {
		return ErrTokenInvalid
	}
	if err != nil {
		return err
	}
	if ev.ExpiresAt.Before(s.now()) {
		return ErrTokenInvalid
	}
	if err := s.repo.MarkEmailVerified(ctx, ev.UserID); err != nil {
		return err
	}
	return s.repo.DeleteEmailVerification(ctx, ev.ID)
}

// Me liefert das Profil zur User-ID (für GET /v1/me).
func (s *Service) Me(ctx context.Context, uid int64) (*User, error) {
	return s.repo.GetUserByID(ctx, uid)
}
