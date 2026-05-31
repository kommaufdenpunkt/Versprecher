package posts

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// fakeRepo: In-Memory-Beiträge.
type fakeRepo struct {
	posts  map[int64]*Post
	nextID int64
}

func newFakeRepo() *fakeRepo { return &fakeRepo{posts: map[int64]*Post{}} }

func (f *fakeRepo) Create(_ context.Context, p CreateParams) (*Post, error) {
	f.nextID++
	post := &Post{
		ID: f.nextID, GroupID: p.GroupID, AuthorID: p.AuthorID, TaggedUserID: p.TaggedUserID,
		Word: cleanWord(p.Word), WordNormalized: p.WordNormalized, Explanation: p.Explanation,
		Kind: p.Kind, VoiceURL: p.VoiceURL, Status: "pending_review", CreatedAt: time.Now(),
	}
	f.posts[post.ID] = post
	return post, nil
}

func (f *fakeRepo) GetByID(_ context.Context, id int64) (*Post, error) {
	if p, ok := f.posts[id]; ok {
		return p, nil
	}
	return nil, ErrNotFound
}

func (f *fakeRepo) ListFeed(_ context.Context, groupID, beforeID int64, limit int) ([]Post, error) {
	var out []Post
	for _, p := range f.posts {
		if p.GroupID == groupID && p.Status == "visible" && (beforeID == 0 || p.ID < beforeID) {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (f *fakeRepo) UpdateMeant(_ context.Context, id int64, meant string, kind *string) (*Post, error) {
	p := f.posts[id]
	p.MeantConfirmed = &meant
	if kind != nil {
		p.Kind = kind
	}
	return p, nil
}

// fakeMembers: konfigurierbare Mitgliedschaften ("groupID:userID").
type fakeMembers struct{ members map[string]bool }

func newFakeMembers() *fakeMembers    { return &fakeMembers{members: map[string]bool{}} }
func key(g, u int64) string           { return fmt.Sprintf("%d:%d", g, u) }
func (m *fakeMembers) add(g, u int64) { m.members[key(g, u)] = true }
func (m *fakeMembers) IsMember(_ context.Context, g, u int64) (bool, error) {
	return m.members[key(g, u)], nil
}

func TestCreateRequiresMembership(t *testing.T) {
	svc := NewService(newFakeRepo(), newFakeMembers())
	_, err := svc.Create(context.Background(), CreateInput{GroupID: 1, AuthorID: 7, Word: "Spabl"})
	if !errors.Is(err, ErrNotMember) {
		t.Fatalf("erwartet ErrNotMember, bekam %v", err)
	}
}

func TestCreateValidatesWord(t *testing.T) {
	mem := newFakeMembers()
	mem.add(1, 7)
	svc := NewService(newFakeRepo(), mem)

	bad := []string{"", "viel zu langes wort", "Mit Leerzeichen", "Zahl123", "Dreizehnzeich"}
	for _, w := range bad {
		if _, err := svc.Create(context.Background(), CreateInput{GroupID: 1, AuthorID: 7, Word: w}); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("Wort %q sollte ungültig sein, err=%v", w, err)
		}
	}
	// Gültig: max. 12 Buchstaben, optional führendes '#', Umlaute erlaubt.
	for _, w := range []string{"Spabl", "#Spabl", "Fernsehturm", "Tütü"} {
		if _, err := svc.Create(context.Background(), CreateInput{GroupID: 1, AuthorID: 7, Word: w}); err != nil {
			t.Errorf("Wort %q sollte gültig sein, err=%v", w, err)
		}
	}
}

func TestCreateStartsPendingAndNormalizes(t *testing.T) {
	mem := newFakeMembers()
	mem.add(1, 7)
	svc := NewService(newFakeRepo(), mem)

	p, err := svc.Create(context.Background(), CreateInput{GroupID: 1, AuthorID: 7, Word: "#Spabl", Explanation: " Spatzl + Baby "})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.Status != "pending_review" {
		t.Errorf("neuer Beitrag sollte pending_review sein, ist %q", p.Status)
	}
	if p.Word != "Spabl" || p.WordNormalized != "spabl" {
		t.Errorf("Wort/Normalisierung falsch: %q / %q", p.Word, p.WordNormalized)
	}
	if p.Explanation != "Spatzl + Baby" {
		t.Errorf("Erklärung nicht getrimmt: %q", p.Explanation)
	}
}

func TestCreateTaggedMustBeMember(t *testing.T) {
	mem := newFakeMembers()
	mem.add(1, 7) // Autor
	svc := NewService(newFakeRepo(), mem)

	tagged := int64(99)
	if _, err := svc.Create(context.Background(), CreateInput{GroupID: 1, AuthorID: 7, Word: "Spabl", TaggedUserID: &tagged}); !errors.Is(err, ErrTaggedNotMember) {
		t.Fatalf("erwartet ErrTaggedNotMember, bekam %v", err)
	}
	mem.add(1, 99) // jetzt Mitglied
	if _, err := svc.Create(context.Background(), CreateInput{GroupID: 1, AuthorID: 7, Word: "Spabl", TaggedUserID: &tagged}); err != nil {
		t.Fatalf("markierte Person ist Mitglied, sollte klappen: %v", err)
	}
}

func TestFeedRequiresMembership(t *testing.T) {
	svc := NewService(newFakeRepo(), newFakeMembers())
	if _, err := svc.Feed(context.Background(), 7, 1, 0, 20); !errors.Is(err, ErrNotMember) {
		t.Fatalf("erwartet ErrNotMember, bekam %v", err)
	}
}

func TestConfirmMeantOnlyAuthor(t *testing.T) {
	repo := newFakeRepo()
	mem := newFakeMembers()
	mem.add(1, 7)
	svc := NewService(repo, mem)
	p, _ := svc.Create(context.Background(), CreateInput{GroupID: 1, AuthorID: 7, Word: "Spabl"})

	if _, err := svc.ConfirmMeant(context.Background(), 8, p.ID, "Spatzl + Baby", "versprecher"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("fremder Nutzer sollte ErrForbidden bekommen, bekam %v", err)
	}
	updated, err := svc.ConfirmMeant(context.Background(), 7, p.ID, "Spatzl + Baby", "versprecher")
	if err != nil {
		t.Fatalf("Autor-Bestätigung: %v", err)
	}
	if updated.MeantConfirmed == nil || *updated.MeantConfirmed != "Spatzl + Baby" {
		t.Error("meant_confirmed nicht gesetzt")
	}
	if updated.Kind == nil || *updated.Kind != "versprecher" {
		t.Error("kind nicht gesetzt")
	}
}
