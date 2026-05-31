package posts

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidInput    = errors.New("ungültige Eingabe")
	ErrNotMember       = errors.New("kein Mitglied dieser Gruppe")
	ErrTaggedNotMember = errors.New("markierte Person ist kein Mitglied der Gruppe")
	ErrForbidden       = errors.New("nicht erlaubt")
	ErrPostNotFound    = errors.New("Beitrag nicht gefunden")
)

const (
	defaultFeedLimit = 20
	maxFeedLimit     = 100
)

// Membership entkoppelt posts von groups (kein Import-Zyklus). Implementiert von
// groups.Service.
type Membership interface {
	IsMember(ctx context.Context, groupID, userID int64) (bool, error)
}

// Service bündelt die Beitrags-Logik.
type Service struct {
	repo    Repository
	members Membership
}

func NewService(repo Repository, members Membership) *Service {
	return &Service{repo: repo, members: members}
}

// CreateInput sind die Eingaben beim Anlegen eines Beitrags.
type CreateInput struct {
	GroupID      int64
	AuthorID     int64
	TaggedUserID *int64
	Word         string
	Explanation  string
	Kind         string // optional ("" = offen, Fidolin schlägt vor)
	VoiceURL     string // optional
}

// Create legt einen Beitrag an. Sicherheit: nur Mitglieder dürfen posten; eine
// markierte Person muss ebenfalls Mitglied der Gruppe sein. Der Beitrag startet
// als pending_review (nicht im Feed) und wird von Fidolin geprüft.
func (s *Service) Create(ctx context.Context, in CreateInput) (*Post, error) {
	if !validWord(in.Word) || len(in.Explanation) > maxExplanationLen {
		return nil, ErrInvalidInput
	}
	var kind *string
	if in.Kind != "" {
		if !validKind(in.Kind) {
			return nil, ErrInvalidInput
		}
		kind = &in.Kind
	}

	isMember, err := s.members.IsMember(ctx, in.GroupID, in.AuthorID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotMember
	}

	if in.TaggedUserID != nil {
		taggedMember, err := s.members.IsMember(ctx, in.GroupID, *in.TaggedUserID)
		if err != nil {
			return nil, err
		}
		if !taggedMember {
			return nil, ErrTaggedNotMember
		}
	}

	var voice *string
	if v := strings.TrimSpace(in.VoiceURL); v != "" {
		voice = &v
	}

	return s.repo.Create(ctx, CreateParams{
		GroupID:        in.GroupID,
		AuthorID:       in.AuthorID,
		TaggedUserID:   in.TaggedUserID,
		Word:           in.Word,
		WordNormalized: normalizeWord(in.Word),
		Explanation:    strings.TrimSpace(in.Explanation),
		Kind:           kind,
		VoiceURL:       voice,
	})
}

// Feed liefert die sichtbaren Beiträge einer Gruppe — nur für Mitglieder.
func (s *Service) Feed(ctx context.Context, userID, groupID, beforeID int64, limit int) ([]Post, error) {
	isMember, err := s.members.IsMember(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotMember
	}
	if limit <= 0 || limit > maxFeedLimit {
		limit = defaultFeedLimit
	}
	if beforeID < 0 {
		beforeID = 0
	}
	return s.repo.ListFeed(ctx, groupID, beforeID, limit)
}

// ConfirmMeant lässt den Autor das „gemeint" (und optional die Sorte) bestätigen.
func (s *Service) ConfirmMeant(ctx context.Context, userID, postID int64, meant, kind string) (*Post, error) {
	meant = strings.TrimSpace(meant)
	if meant == "" || len(meant) > maxExplanationLen {
		return nil, ErrInvalidInput
	}
	var kindPtr *string
	if kind != "" {
		if !validKind(kind) {
			return nil, ErrInvalidInput
		}
		kindPtr = &kind
	}

	post, err := s.repo.GetByID(ctx, postID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrPostNotFound
	}
	if err != nil {
		return nil, err
	}
	if post.AuthorID != userID {
		return nil, ErrForbidden
	}
	return s.repo.UpdateMeant(ctx, postID, meant, kindPtr)
}
