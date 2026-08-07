package fahrstunden

import (
	"context"
	"errors"
	"sort"
	"time"
)

var (
	ErrUngueltigeEingabe = errors.New("ungültige Eingabe")
	// ErrGrundFehlt: das Tageslimit bewusst zu überschreiten ist erlaubt, aber
	// nur mit Begründung — sonst ist im Nachweis später nicht klar, warum.
	ErrGrundFehlt = errors.New("für das bewusste Überschreiten des Tageslimits ist ein Grund nötig")
	// ErrKeinFreierTag: im geprüften Fenster passt die Stunde an keinem Tag.
	ErrKeinFreierTag = errors.New("im geprüften Zeitraum hat kein Tag genug freie Zeit")
)

const (
	standardFenster    = 14  // Tage vor/nach dem Fahrtag, in denen nach Platz gesucht wird
	maxFenster         = 120 // Obergrenze für Vorschläge
	maxVorschlaege     = 6
	maxKapazitaetsTage = 400 // Schutz gegen versehentlich riesige Zeiträume
	standardListLimit  = 500
	maxListLimit       = 2000
)

// Service bündelt die Fahrstunden-Logik inklusive Tageslimit.
type Service struct {
	repo  Repository
	limit int
}

// NewService erstellt den Service. limitMinuten <= 0 fällt auf das
// FS-Manager-Standardlimit (495 Minuten) zurück.
func NewService(repo Repository, limitMinuten int) *Service {
	if limitMinuten <= 0 {
		limitMinuten = StandardTageslimit
	}
	return &Service{repo: repo, limit: limitMinuten}
}

// TageslimitMinuten ist die pro Eintragetag zulässige Zeit.
func (s *Service) TageslimitMinuten() int { return s.limit }

// ---------------------------------------------------------------- Fahrschüler

func (s *Service) AnlegenSchueler(ctx context.Context, fahrlehrerID int64, name, klasse, notiz string) (*Fahrschueler, error) {
	name = bereinigeName(name)
	if name == "" || fahrlehrerID <= 0 {
		return nil, ErrUngueltigeEingabe
	}
	return s.repo.CreateSchueler(ctx, SchuelerParams{
		FahrlehrerID: fahrlehrerID,
		Name:         name,
		Klasse:       kuerze(klasse, maxKlasseLen),
		Notiz:        kuerze(notiz, maxNotizLen),
	})
}

func (s *Service) AendernSchueler(ctx context.Context, id, fahrlehrerID int64, u SchuelerUpdate) (*Fahrschueler, error) {
	if id <= 0 || fahrlehrerID <= 0 {
		return nil, ErrUngueltigeEingabe
	}
	if u.Name != nil {
		n := bereinigeName(*u.Name)
		if n == "" {
			return nil, ErrUngueltigeEingabe
		}
		u.Name = &n
	}
	if u.Klasse != nil {
		k := kuerze(*u.Klasse, maxKlasseLen)
		u.Klasse = &k
	}
	if u.Notiz != nil {
		n := kuerze(*u.Notiz, maxNotizLen)
		u.Notiz = &n
	}
	return s.repo.UpdateSchueler(ctx, id, fahrlehrerID, u)
}

func (s *Service) HoleSchueler(ctx context.Context, id, fahrlehrerID int64) (*Fahrschueler, error) {
	if id <= 0 || fahrlehrerID <= 0 {
		return nil, ErrUngueltigeEingabe
	}
	return s.repo.GetSchueler(ctx, id, fahrlehrerID)
}

func (s *Service) ListeSchueler(ctx context.Context, fahrlehrerID int64, nurAktive bool) ([]Fahrschueler, error) {
	if fahrlehrerID <= 0 {
		return nil, ErrUngueltigeEingabe
	}
	return s.repo.ListSchueler(ctx, fahrlehrerID, nurAktive)
}

// ---------------------------------------------------------------- Fahrstunden

// AnlegenInput sind die Eingaben beim Eintragen einer Fahrstunde.
type AnlegenInput struct {
	FahrlehrerID   int64
	FahrschuelerID int64
	GefahrenAm     time.Time
	// GefahrenVon ist der Beginn der Fahrstunde (HH:MM, optional).
	GefahrenVon string
	// EingetragenAm leer lassen heißt: den nächstmöglichen Tag automatisch
	// wählen (zuerst der Fahrtag selbst, dann die Tage rundherum).
	EingetragenAm time.Time
	// EingetragenUm ist die Uhrzeit im FS Manager (HH:MM, optional).
	EingetragenUm     string
	DauerMinuten      int
	Art               string
	Notiz             string
	Unterschrift      string // optional, PNG als data-URL
	LimitUebersteuern bool
	LimitGrund        string
}

// AnlegenStunde trägt eine Fahrstunde ein. Der Eintragetag wird gegen das
// Tageslimit geprüft; passt die Stunde nicht, kommt ein *LimitFehler zurück
// (mit den konkreten Zahlen), statt still das Limit zu sprengen.
func (s *Service) AnlegenStunde(ctx context.Context, in AnlegenInput) (*Fahrstunde, error) {
	if in.FahrlehrerID <= 0 || in.FahrschuelerID <= 0 {
		return nil, ErrUngueltigeEingabe
	}
	if in.DauerMinuten < minDauer || in.DauerMinuten > maxDauer {
		return nil, ErrUngueltigeEingabe
	}
	if in.Art == "" {
		in.Art = ArtUebungsstunde
	}
	if !gueltigeArt(in.Art) {
		return nil, ErrUngueltigeEingabe
	}
	in.GefahrenAm = tagesbeginn(in.GefahrenAm)
	if !plausiblesDatum(in.GefahrenAm) {
		return nil, ErrUngueltigeEingabe
	}

	gefahrenVon, ok := bereinigeUhrzeit(in.GefahrenVon)
	if !ok {
		return nil, ErrUngueltigeEingabe
	}
	eingetragenUm, ok := bereinigeUhrzeit(in.EingetragenUm)
	if !ok {
		return nil, ErrUngueltigeEingabe
	}

	grund := kuerze(in.LimitGrund, maxGrundLen)
	if in.LimitUebersteuern && grund == "" {
		return nil, ErrGrundFehlt
	}

	var unterschrift *string
	if in.Unterschrift != "" {
		if !gueltigeUnterschrift(in.Unterschrift) {
			return nil, ErrUngueltigeEingabe
		}
		u := in.Unterschrift
		unterschrift = &u
	}

	eingetragenAm := tagesbeginn(in.EingetragenAm)
	if in.EingetragenAm.IsZero() {
		// Kein Eintragetag angegeben: den nächstmöglichen suchen.
		gewaehlt, err := s.naechsterFreierTag(ctx, in.FahrlehrerID, in.GefahrenAm, in.DauerMinuten)
		if err != nil {
			if errors.Is(err, ErrKeinFreierTag) && in.LimitUebersteuern {
				eingetragenAm = in.GefahrenAm
			} else {
				return nil, err
			}
		} else {
			eingetragenAm = gewaehlt
		}
	} else if !plausiblesDatum(eingetragenAm) {
		return nil, ErrUngueltigeEingabe
	}

	return s.repo.CreateStunde(ctx, StundeParams{
		FahrlehrerID:      in.FahrlehrerID,
		FahrschuelerID:    in.FahrschuelerID,
		GefahrenAm:        in.GefahrenAm,
		GefahrenVon:       gefahrenVon,
		EingetragenAm:     eingetragenAm,
		EingetragenUm:     eingetragenUm,
		DauerMinuten:      in.DauerMinuten,
		Art:               in.Art,
		Notiz:             kuerze(in.Notiz, maxNotizLen),
		UnterschriftPNG:   unterschrift,
		LimitUebersteuert: in.LimitUebersteuern,
		LimitGrund:        grund,
	}, LimitPruefung{LimitMinuten: s.limit, Uebersteuern: in.LimitUebersteuern})
}

// AendernInput sind die änderbaren Felder einer Fahrstunde. nil heißt
// „unverändert“; bei den Uhrzeiten heißt ein Zeiger auf "" „Uhrzeit entfernen“.
type AendernInput struct {
	GefahrenAm        *time.Time
	GefahrenVon       *string
	EingetragenAm     *time.Time
	EingetragenUm     *string
	DauerMinuten      *int
	Art               *string
	Notiz             *string
	LimitUebersteuern bool
	LimitGrund        string
}

func (s *Service) AendernStunde(ctx context.Context, id, fahrlehrerID int64, in AendernInput) (*Fahrstunde, error) {
	if id <= 0 || fahrlehrerID <= 0 {
		return nil, ErrUngueltigeEingabe
	}
	u := StundeUpdate{DauerMinuten: in.DauerMinuten, Art: in.Art}

	if in.GefahrenAm != nil {
		d := tagesbeginn(*in.GefahrenAm)
		if !plausiblesDatum(d) {
			return nil, ErrUngueltigeEingabe
		}
		u.GefahrenAm = &d
	}
	if in.EingetragenAm != nil {
		d := tagesbeginn(*in.EingetragenAm)
		if !plausiblesDatum(d) {
			return nil, ErrUngueltigeEingabe
		}
		u.EingetragenAm = &d
	}
	if in.GefahrenVon != nil {
		z, ok := bereinigeUhrzeit(*in.GefahrenVon)
		if !ok {
			return nil, ErrUngueltigeEingabe
		}
		u.GefahrenVon = &z
	}
	if in.EingetragenUm != nil {
		z, ok := bereinigeUhrzeit(*in.EingetragenUm)
		if !ok {
			return nil, ErrUngueltigeEingabe
		}
		u.EingetragenUm = &z
	}
	if in.DauerMinuten != nil && (*in.DauerMinuten < minDauer || *in.DauerMinuten > maxDauer) {
		return nil, ErrUngueltigeEingabe
	}
	if in.Art != nil && !gueltigeArt(*in.Art) {
		return nil, ErrUngueltigeEingabe
	}
	if in.Notiz != nil {
		n := kuerze(*in.Notiz, maxNotizLen)
		u.Notiz = &n
	}
	grund := kuerze(in.LimitGrund, maxGrundLen)
	if in.LimitUebersteuern {
		if grund == "" {
			return nil, ErrGrundFehlt
		}
		u.LimitGrund = &grund
	}

	return s.repo.UpdateStunde(ctx, id, fahrlehrerID, u, LimitPruefung{
		LimitMinuten: s.limit,
		Uebersteuern: in.LimitUebersteuern,
	})
}

func (s *Service) LoeschenStunde(ctx context.Context, id, fahrlehrerID int64) error {
	if id <= 0 || fahrlehrerID <= 0 {
		return ErrUngueltigeEingabe
	}
	return s.repo.DeleteStunde(ctx, id, fahrlehrerID)
}

func (s *Service) HoleStunde(ctx context.Context, id, fahrlehrerID int64) (*Fahrstunde, error) {
	if id <= 0 || fahrlehrerID <= 0 {
		return nil, ErrUngueltigeEingabe
	}
	return s.repo.GetStunde(ctx, id, fahrlehrerID)
}

func (s *Service) ListeStunden(ctx context.Context, f Filter) ([]Fahrstunde, error) {
	if f.FahrlehrerID <= 0 {
		return nil, ErrUngueltigeEingabe
	}
	if f.Limit <= 0 || f.Limit > maxListLimit {
		f.Limit = standardListLimit
	}
	return s.repo.ListStunden(ctx, f)
}

// Unterschreiben hinterlegt die Unterschrift der Fahrschülerin/des Fahrschülers.
// Ein leerer Wert entfernt sie wieder (falls jemand danebengemalt hat).
func (s *Service) Unterschreiben(ctx context.Context, id, fahrlehrerID int64, png string) (*Fahrstunde, error) {
	if id <= 0 || fahrlehrerID <= 0 {
		return nil, ErrUngueltigeEingabe
	}
	if png == "" {
		return s.repo.SetUnterschrift(ctx, id, fahrlehrerID, nil)
	}
	if !gueltigeUnterschrift(png) {
		return nil, ErrUngueltigeEingabe
	}
	return s.repo.SetUnterschrift(ctx, id, fahrlehrerID, &png)
}

// ----------------------------------------------------------------- Kapazität

// Kapazitaet liefert für jeden Tag im Zeitraum die verbuchte und die freie Zeit
// — auch für Tage ohne Eintrag. Das ist die Sicht „was passt noch wohin?“.
func (s *Service) Kapazitaet(ctx context.Context, fahrlehrerID int64, von, bis time.Time) ([]Tageskapazitaet, error) {
	if fahrlehrerID <= 0 {
		return nil, ErrUngueltigeEingabe
	}
	von, bis = tagesbeginn(von), tagesbeginn(bis)
	if !plausiblesDatum(von) || !plausiblesDatum(bis) || bis.Before(von) {
		return nil, ErrUngueltigeEingabe
	}
	if int(bis.Sub(von).Hours()/24) > maxKapazitaetsTage {
		return nil, ErrUngueltigeEingabe
	}

	belegt, err := s.repo.Belegung(ctx, fahrlehrerID, von, bis)
	if err != nil {
		return nil, err
	}
	nachTag := make(map[string]Tageskapazitaet, len(belegt))
	for _, b := range belegt {
		nachTag[FormatDatum(b.Datum)] = b
	}

	out := make([]Tageskapazitaet, 0, int(bis.Sub(von).Hours()/24)+1)
	for d := von; !d.After(bis); d = d.AddDate(0, 0, 1) {
		t := nachTag[FormatDatum(d)]
		t.Datum = d
		t.LimitMinuten = s.limit
		t.FreiMinuten = s.limit - t.BelegtMinuten
		if t.FreiMinuten < 0 {
			t.FreiMinuten = 0
		}
		out = append(out, t)
	}
	return out, nil
}

// Vorschlaege nennt Eintragetage, an denen die gewünschte Dauer noch passt —
// sortiert nach Nähe zum Fahrtag. Bei gleichem Abstand kommt der frühere Tag
// zuerst: in der Praxis wird eher auf einen Tag davor verschoben.
func (s *Service) Vorschlaege(ctx context.Context, fahrlehrerID int64, gefahrenAm time.Time, dauer, fenster int) ([]Vorschlag, error) {
	if fahrlehrerID <= 0 || dauer < minDauer || dauer > maxDauer {
		return nil, ErrUngueltigeEingabe
	}
	gefahrenAm = tagesbeginn(gefahrenAm)
	if !plausiblesDatum(gefahrenAm) {
		return nil, ErrUngueltigeEingabe
	}
	if fenster <= 0 || fenster > maxFenster {
		fenster = standardFenster
	}

	kap, err := s.Kapazitaet(ctx, fahrlehrerID,
		gefahrenAm.AddDate(0, 0, -fenster), gefahrenAm.AddDate(0, 0, fenster))
	if err != nil {
		return nil, err
	}

	passend := make([]Vorschlag, 0, len(kap))
	for _, t := range kap {
		if t.FreiMinuten < dauer {
			continue
		}
		passend = append(passend, Vorschlag{
			Datum:       t.Datum,
			FreiMinuten: t.FreiMinuten,
			AbstandTage: int(t.Datum.Sub(gefahrenAm).Hours() / 24),
		})
	}
	sort.SliceStable(passend, func(i, j int) bool {
		ai, aj := abs(passend[i].AbstandTage), abs(passend[j].AbstandTage)
		if ai != aj {
			return ai < aj
		}
		return passend[i].AbstandTage < passend[j].AbstandTage
	})
	if len(passend) > maxVorschlaege {
		passend = passend[:maxVorschlaege]
	}
	return passend, nil
}

// naechsterFreierTag wählt automatisch den besten Eintragetag.
func (s *Service) naechsterFreierTag(ctx context.Context, fahrlehrerID int64, gefahrenAm time.Time, dauer int) (time.Time, error) {
	v, err := s.Vorschlaege(ctx, fahrlehrerID, gefahrenAm, dauer, standardFenster)
	if err != nil {
		return time.Time{}, err
	}
	if len(v) == 0 {
		return time.Time{}, ErrKeinFreierTag
	}
	return v[0].Datum, nil
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
