package fahrstunden

import (
	"context"
	"sort"
	"strings"
	"time"
)

// fakeRepo ist ein Repository im Speicher. Es bildet das Tageslimit genauso ab
// wie das echte Repository — sonst würden die Service-Tests am Kern vorbeitesten.
type fakeRepo struct {
	schueler []Fahrschueler
	stunden  []Fahrstunde
	naechste int64
}

func neuerFakeRepo() *fakeRepo { return &fakeRepo{naechste: 1} }

func (r *fakeRepo) id() int64 {
	id := r.naechste
	r.naechste++
	return id
}

func (r *fakeRepo) CreateSchueler(_ context.Context, p SchuelerParams) (*Fahrschueler, error) {
	for i := range r.schueler {
		if r.schueler[i].FahrlehrerID == p.FahrlehrerID &&
			strings.EqualFold(r.schueler[i].Name, p.Name) {
			return nil, ErrNameVergeben
		}
	}
	s := Fahrschueler{
		ID: r.id(), FahrlehrerID: p.FahrlehrerID, Name: p.Name,
		Klasse: p.Klasse, Notiz: p.Notiz, Aktiv: true, CreatedAt: time.Now(),
	}
	r.schueler = append(r.schueler, s)
	return &s, nil
}

func (r *fakeRepo) findeSchueler(id, fahrlehrerID int64) *Fahrschueler {
	for i := range r.schueler {
		if r.schueler[i].ID == id && r.schueler[i].FahrlehrerID == fahrlehrerID {
			return &r.schueler[i]
		}
	}
	return nil
}

func (r *fakeRepo) UpdateSchueler(_ context.Context, id, fahrlehrerID int64, u SchuelerUpdate) (*Fahrschueler, error) {
	s := r.findeSchueler(id, fahrlehrerID)
	if s == nil {
		return nil, ErrNotFound
	}
	if u.Name != nil {
		s.Name = *u.Name
	}
	if u.Klasse != nil {
		s.Klasse = *u.Klasse
	}
	if u.Notiz != nil {
		s.Notiz = *u.Notiz
	}
	if u.Aktiv != nil {
		s.Aktiv = *u.Aktiv
	}
	kopie := *s
	return &kopie, nil
}

func (r *fakeRepo) GetSchueler(_ context.Context, id, fahrlehrerID int64) (*Fahrschueler, error) {
	s := r.findeSchueler(id, fahrlehrerID)
	if s == nil {
		return nil, ErrNotFound
	}
	kopie := *s
	return &kopie, nil
}

func (r *fakeRepo) ListSchueler(_ context.Context, fahrlehrerID int64, nurAktive bool) ([]Fahrschueler, error) {
	out := []Fahrschueler{}
	for _, s := range r.schueler {
		if s.FahrlehrerID == fahrlehrerID && (!nurAktive || s.Aktiv) {
			out = append(out, s)
		}
	}
	return out, nil
}

// belegt summiert die Minuten eines Eintragetages (ohne die Stunde ausser).
func (r *fakeRepo) belegt(fahrlehrerID int64, tag time.Time, ausser int64) int {
	summe := 0
	for _, f := range r.stunden {
		if f.FahrlehrerID == fahrlehrerID && f.EingetragenAm.Equal(tagesbeginn(tag)) && f.ID != ausser {
			summe += f.DauerMinuten
		}
	}
	return summe
}

func (r *fakeRepo) pruefe(fahrlehrerID int64, tag time.Time, dauer int, ausser int64, l LimitPruefung) error {
	belegt := r.belegt(fahrlehrerID, tag, ausser)
	frei := l.LimitMinuten - belegt
	if frei < 0 {
		frei = 0
	}
	if !l.Uebersteuern && belegt+dauer > l.LimitMinuten {
		return &LimitFehler{
			Datum: tagesbeginn(tag), LimitMinuten: l.LimitMinuten,
			BelegtMinuten: belegt, FreiMinuten: frei, WunschMinuten: dauer,
		}
	}
	return nil
}

func (r *fakeRepo) CreateStunde(_ context.Context, p StundeParams, l LimitPruefung) (*Fahrstunde, error) {
	s := r.findeSchueler(p.FahrschuelerID, p.FahrlehrerID)
	if s == nil {
		return nil, ErrNotFound
	}
	if err := r.pruefe(p.FahrlehrerID, p.EingetragenAm, p.DauerMinuten, 0, l); err != nil {
		return nil, err
	}
	f := Fahrstunde{
		ID: r.id(), FahrlehrerID: p.FahrlehrerID, FahrschuelerID: p.FahrschuelerID,
		SchuelerName: s.Name, SchuelerKlasse: s.Klasse,
		GefahrenAm: tagesbeginn(p.GefahrenAm), EingetragenAm: tagesbeginn(p.EingetragenAm),
		DauerMinuten: p.DauerMinuten, Art: p.Art, Notiz: p.Notiz,
		UnterschriftPNG:   p.UnterschriftPNG,
		LimitUebersteuert: p.LimitUebersteuert, LimitGrund: p.LimitGrund,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if p.UnterschriftPNG != nil {
		jetzt := time.Now()
		f.UnterschriebenAm = &jetzt
	}
	r.stunden = append(r.stunden, f)
	return &f, nil
}

func (r *fakeRepo) findeStunde(id, fahrlehrerID int64) *Fahrstunde {
	for i := range r.stunden {
		if r.stunden[i].ID == id && r.stunden[i].FahrlehrerID == fahrlehrerID {
			return &r.stunden[i]
		}
	}
	return nil
}

func (r *fakeRepo) UpdateStunde(_ context.Context, id, fahrlehrerID int64, u StundeUpdate, l LimitPruefung) (*Fahrstunde, error) {
	f := r.findeStunde(id, fahrlehrerID)
	if f == nil {
		return nil, ErrNotFound
	}
	neuTag, neuDauer := f.EingetragenAm, f.DauerMinuten
	if u.EingetragenAm != nil {
		neuTag = tagesbeginn(*u.EingetragenAm)
	}
	if u.DauerMinuten != nil {
		neuDauer = *u.DauerMinuten
	}
	if !neuTag.Equal(f.EingetragenAm) || neuDauer != f.DauerMinuten {
		if err := r.pruefe(fahrlehrerID, neuTag, neuDauer, id, l); err != nil {
			return nil, err
		}
	}
	if u.GefahrenAm != nil {
		f.GefahrenAm = tagesbeginn(*u.GefahrenAm)
	}
	f.EingetragenAm, f.DauerMinuten = neuTag, neuDauer
	if u.Art != nil {
		f.Art = *u.Art
	}
	if u.Notiz != nil {
		f.Notiz = *u.Notiz
	}
	if l.Uebersteuern {
		f.LimitUebersteuert = true
	}
	if u.LimitGrund != nil {
		f.LimitGrund = *u.LimitGrund
	}
	f.UpdatedAt = time.Now()
	kopie := *f
	return &kopie, nil
}

func (r *fakeRepo) DeleteStunde(_ context.Context, id, fahrlehrerID int64) error {
	for i := range r.stunden {
		if r.stunden[i].ID == id && r.stunden[i].FahrlehrerID == fahrlehrerID {
			r.stunden = append(r.stunden[:i], r.stunden[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (r *fakeRepo) GetStunde(_ context.Context, id, fahrlehrerID int64) (*Fahrstunde, error) {
	f := r.findeStunde(id, fahrlehrerID)
	if f == nil {
		return nil, ErrNotFound
	}
	kopie := *f
	return &kopie, nil
}

func (r *fakeRepo) ListStunden(_ context.Context, filter Filter) ([]Fahrstunde, error) {
	out := []Fahrstunde{}
	for _, f := range r.stunden {
		if f.FahrlehrerID != filter.FahrlehrerID {
			continue
		}
		if filter.FahrschuelerID != 0 && f.FahrschuelerID != filter.FahrschuelerID {
			continue
		}
		if !filter.Von.IsZero() && f.GefahrenAm.Before(tagesbeginn(filter.Von)) {
			continue
		}
		if !filter.Bis.IsZero() && f.GefahrenAm.After(tagesbeginn(filter.Bis)) {
			continue
		}
		out = append(out, f)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].GefahrenAm.Equal(out[j].GefahrenAm) {
			return out[i].GefahrenAm.Before(out[j].GefahrenAm)
		}
		return out[i].ID < out[j].ID
	})
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

func (r *fakeRepo) SetUnterschrift(_ context.Context, id, fahrlehrerID int64, png *string) (*Fahrstunde, error) {
	f := r.findeStunde(id, fahrlehrerID)
	if f == nil {
		return nil, ErrNotFound
	}
	f.UnterschriftPNG = png
	if png == nil {
		f.UnterschriebenAm = nil
	} else {
		jetzt := time.Now()
		f.UnterschriebenAm = &jetzt
	}
	kopie := *f
	return &kopie, nil
}

func (r *fakeRepo) Belegung(_ context.Context, fahrlehrerID int64, von, bis time.Time) ([]Tageskapazitaet, error) {
	nachTag := map[string]*Tageskapazitaet{}
	for _, f := range r.stunden {
		if f.FahrlehrerID != fahrlehrerID {
			continue
		}
		if f.EingetragenAm.Before(tagesbeginn(von)) || f.EingetragenAm.After(tagesbeginn(bis)) {
			continue
		}
		schluessel := FormatDatum(f.EingetragenAm)
		t := nachTag[schluessel]
		if t == nil {
			t = &Tageskapazitaet{Datum: f.EingetragenAm}
			nachTag[schluessel] = t
		}
		t.BelegtMinuten += f.DauerMinuten
		t.Anzahl++
		t.Uebersteuert = t.Uebersteuert || f.LimitUebersteuert
	}
	out := []Tageskapazitaet{}
	for _, t := range nachTag {
		out = append(out, *t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Datum.Before(out[j].Datum) })
	return out, nil
}
