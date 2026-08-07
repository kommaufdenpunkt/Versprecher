package fahrstunden

import (
	"context"
	"errors"
	"testing"
	"time"
)

const lehrer = int64(1)

// aufbau liefert Service und Repo mit einer angelegten Fahrschülerin.
func aufbau(t *testing.T, limit int) (*Service, *fakeRepo, int64) {
	t.Helper()
	repo := neuerFakeRepo()
	svc := NewService(repo, limit)
	s, err := svc.AnlegenSchueler(context.Background(), lehrer, "Änne Müller", "B", "")
	if err != nil {
		t.Fatalf("Schüler anlegen: %v", err)
	}
	return svc, repo, s.ID
}

func datum(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := ParseDatum(s)
	if err != nil {
		t.Fatalf("Datum %q: %v", s, err)
	}
	return d
}

func eintragen(t *testing.T, svc *Service, schueler int64, gefahren, eingetragen string, dauer int) (*Fahrstunde, error) {
	t.Helper()
	in := AnlegenInput{
		FahrlehrerID:   lehrer,
		FahrschuelerID: schueler,
		GefahrenAm:     datum(t, gefahren),
		DauerMinuten:   dauer,
		Art:            ArtUebungsstunde,
	}
	if eingetragen != "" {
		in.EingetragenAm = datum(t, eingetragen)
	}
	return svc.AnlegenStunde(context.Background(), in)
}

func TestTageslimitGreiftAufEintragetag(t *testing.T) {
	svc, _, schueler := aufbau(t, 495)

	// 5 × 90 = 450 Minuten passen noch.
	for i := 0; i < 5; i++ {
		if _, err := eintragen(t, svc, schueler, "2026-08-10", "2026-08-10", 90); err != nil {
			t.Fatalf("Stunde %d: %v", i+1, err)
		}
	}
	// Die sechste (90 Min) sprengt die 495.
	_, err := eintragen(t, svc, schueler, "2026-08-10", "2026-08-10", 90)
	var limitErr *LimitFehler
	if !errors.As(err, &limitErr) {
		t.Fatalf("erwartet *LimitFehler, bekommen: %v", err)
	}
	if limitErr.BelegtMinuten != 450 || limitErr.FreiMinuten != 45 || limitErr.WunschMinuten != 90 {
		t.Errorf("falsche Zahlen im Fehler: belegt=%d frei=%d wunsch=%d",
			limitErr.BelegtMinuten, limitErr.FreiMinuten, limitErr.WunschMinuten)
	}
	// Der Rest passt aber noch exakt.
	if _, err := eintragen(t, svc, schueler, "2026-08-10", "2026-08-10", 45); err != nil {
		t.Errorf("45 Min hätten noch passen müssen: %v", err)
	}
}

func TestTageslimitZaehltNurEintragetagNichtFahrtag(t *testing.T) {
	svc, _, schueler := aufbau(t, 495)

	// Alles am selben Fahrtag gefahren, aber auf zwei Eintragetage verteilt:
	// beide Tage bleiben unter dem Limit, also darf nichts abgelehnt werden.
	for i := 0; i < 5; i++ {
		if _, err := eintragen(t, svc, schueler, "2026-08-10", "2026-08-10", 90); err != nil {
			t.Fatalf("Tag 1, Stunde %d: %v", i+1, err)
		}
	}
	for i := 0; i < 5; i++ {
		if _, err := eintragen(t, svc, schueler, "2026-08-10", "2026-08-09", 90); err != nil {
			t.Fatalf("Tag 2, Stunde %d: %v", i+1, err)
		}
	}
	kap, err := svc.Kapazitaet(context.Background(), lehrer, datum(t, "2026-08-09"), datum(t, "2026-08-10"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tag := range kap {
		if tag.BelegtMinuten != 450 {
			t.Errorf("%s: belegt=%d, erwartet 450", FormatDatum(tag.Datum), tag.BelegtMinuten)
		}
	}
}

func TestOhneEintragetagWirdDerNaechsteFreieGewaehlt(t *testing.T) {
	svc, _, schueler := aufbau(t, 495)

	// Fahrtag komplett vollmachen.
	for i := 0; i < 11; i++ {
		if _, err := eintragen(t, svc, schueler, "2026-08-10", "2026-08-10", 45); err != nil {
			t.Fatalf("Füllstunde %d: %v", i+1, err)
		}
	}
	// Jetzt ohne eingetragen_am buchen: der Dienst muss ausweichen.
	f, err := eintragen(t, svc, schueler, "2026-08-10", "", 90)
	if err != nil {
		t.Fatalf("automatische Wahl: %v", err)
	}
	if FormatDatum(f.GefahrenAm) != "2026-08-10" {
		t.Errorf("Fahrtag verändert: %s", FormatDatum(f.GefahrenAm))
	}
	// Bei gleichem Abstand gewinnt der frühere Tag — so wird in der Praxis
	// verschoben („am Neunten gefahren, am Samstag davor eingetragen“).
	if FormatDatum(f.EingetragenAm) != "2026-08-09" {
		t.Errorf("eingetragen am %s, erwartet 2026-08-09", FormatDatum(f.EingetragenAm))
	}
	if f.AbweichungTage() != -1 {
		t.Errorf("Abweichung %d, erwartet -1", f.AbweichungTage())
	}
}

func TestVorschlaegeSindNachNaeheSortiert(t *testing.T) {
	svc, _, schueler := aufbau(t, 495)

	// Fahrtag voll, Vortag halb voll.
	for i := 0; i < 11; i++ {
		if _, err := eintragen(t, svc, schueler, "2026-08-10", "2026-08-10", 45); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := eintragen(t, svc, schueler, "2026-08-09", "2026-08-09", 450); err != nil {
		t.Fatal(err)
	}

	v, err := svc.Vorschlaege(context.Background(), lehrer, datum(t, "2026-08-10"), 90, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) == 0 {
		t.Fatal("keine Vorschläge")
	}
	// 10.08. ist voll, 09.08. hat nur 45 frei ⇒ erster Treffer ist der 11.08.
	if FormatDatum(v[0].Datum) != "2026-08-11" || v[0].AbstandTage != 1 {
		t.Errorf("erster Vorschlag %s (Abstand %d), erwartet 2026-08-11 (+1)",
			FormatDatum(v[0].Datum), v[0].AbstandTage)
	}
	// Danach mit wachsendem Abstand.
	for i := 1; i < len(v); i++ {
		if abs(v[i].AbstandTage) < abs(v[i-1].AbstandTage) {
			t.Errorf("Vorschläge nicht nach Nähe sortiert: %v", v)
			break
		}
	}
	for _, vorschlag := range v {
		if vorschlag.FreiMinuten < 90 {
			t.Errorf("Vorschlag %s hat nur %d Min frei", FormatDatum(vorschlag.Datum), vorschlag.FreiMinuten)
		}
	}
}

func TestUebersteuernBrauchtEinenGrund(t *testing.T) {
	svc, _, schueler := aufbau(t, 495)
	for i := 0; i < 11; i++ {
		if _, err := eintragen(t, svc, schueler, "2026-08-10", "2026-08-10", 45); err != nil {
			t.Fatal(err)
		}
	}

	basis := AnlegenInput{
		FahrlehrerID: lehrer, FahrschuelerID: schueler,
		GefahrenAm: datum(t, "2026-08-10"), EingetragenAm: datum(t, "2026-08-10"),
		DauerMinuten: 90, Art: ArtPruefungsfahrt, LimitUebersteuern: true,
	}

	if _, err := svc.AnlegenStunde(context.Background(), basis); !errors.Is(err, ErrGrundFehlt) {
		t.Fatalf("ohne Grund erwartet ErrGrundFehlt, bekommen: %v", err)
	}

	basis.LimitGrund = "Prüfungstermin, ließ sich nicht verschieben"
	f, err := svc.AnlegenStunde(context.Background(), basis)
	if err != nil {
		t.Fatalf("mit Grund: %v", err)
	}
	if !f.LimitUebersteuert || f.LimitGrund == "" {
		t.Errorf("Übersteuerung nicht vermerkt: uebersteuert=%v grund=%q", f.LimitUebersteuert, f.LimitGrund)
	}
}

func TestKapazitaetFuelltTageOhneEintraege(t *testing.T) {
	svc, _, schueler := aufbau(t, 495)
	if _, err := eintragen(t, svc, schueler, "2026-08-10", "2026-08-10", 90); err != nil {
		t.Fatal(err)
	}

	kap, err := svc.Kapazitaet(context.Background(), lehrer, datum(t, "2026-08-08"), datum(t, "2026-08-12"))
	if err != nil {
		t.Fatal(err)
	}
	if len(kap) != 5 {
		t.Fatalf("%d Tage, erwartet 5", len(kap))
	}
	for _, tag := range kap {
		erwartetBelegt := 0
		if FormatDatum(tag.Datum) == "2026-08-10" {
			erwartetBelegt = 90
		}
		if tag.BelegtMinuten != erwartetBelegt {
			t.Errorf("%s: belegt=%d, erwartet %d", FormatDatum(tag.Datum), tag.BelegtMinuten, erwartetBelegt)
		}
		if tag.FreiMinuten != 495-erwartetBelegt || tag.LimitMinuten != 495 {
			t.Errorf("%s: frei=%d limit=%d", FormatDatum(tag.Datum), tag.FreiMinuten, tag.LimitMinuten)
		}
	}
}

func TestAendernPrueftDasLimitErneut(t *testing.T) {
	svc, _, schueler := aufbau(t, 495)
	for i := 0; i < 5; i++ {
		if _, err := eintragen(t, svc, schueler, "2026-08-10", "2026-08-10", 90); err != nil {
			t.Fatal(err)
		}
	}
	f, err := eintragen(t, svc, schueler, "2026-08-11", "2026-08-11", 45)
	if err != nil {
		t.Fatal(err)
	}

	// Auf den vollen Tag umbuchen: passt (45 ≤ 45 frei).
	ziel := datum(t, "2026-08-10")
	if _, err := svc.AendernStunde(context.Background(), f.ID, lehrer,
		AendernInput{EingetragenAm: &ziel}); err != nil {
		t.Fatalf("Umbuchen auf freie 45 Min: %v", err)
	}
	// Jetzt die Dauer erhöhen — das muss scheitern.
	neu := 90
	_, err = svc.AendernStunde(context.Background(), f.ID, lehrer, AendernInput{DauerMinuten: &neu})
	var limitErr *LimitFehler
	if !errors.As(err, &limitErr) {
		t.Fatalf("erwartet *LimitFehler, bekommen: %v", err)
	}
	// Die eigene alte Dauer darf beim Prüfen nicht doppelt zählen.
	if limitErr.BelegtMinuten != 450 {
		t.Errorf("belegt=%d, erwartet 450 (eigener Altwert darf nicht mitzählen)", limitErr.BelegtMinuten)
	}
}

func TestUngueltigeEingaben(t *testing.T) {
	svc, _, schueler := aufbau(t, 495)

	faelle := map[string]AnlegenInput{
		"Dauer 0": {FahrlehrerID: lehrer, FahrschuelerID: schueler,
			GefahrenAm: datum(t, "2026-08-10"), DauerMinuten: 0},
		"Dauer negativ": {FahrlehrerID: lehrer, FahrschuelerID: schueler,
			GefahrenAm: datum(t, "2026-08-10"), DauerMinuten: -30},
		"unbekannte Art": {FahrlehrerID: lehrer, FahrschuelerID: schueler,
			GefahrenAm: datum(t, "2026-08-10"), DauerMinuten: 90, Art: "kaffeepause"},
		"kein Fahrtag": {FahrlehrerID: lehrer, FahrschuelerID: schueler, DauerMinuten: 90},
		"kaputte Unterschrift": {FahrlehrerID: lehrer, FahrschuelerID: schueler,
			GefahrenAm: datum(t, "2026-08-10"), DauerMinuten: 90, Unterschrift: "nicht-wirklich-ein-png"},
	}
	for name, in := range faelle {
		if _, err := svc.AnlegenStunde(context.Background(), in); !errors.Is(err, ErrUngueltigeEingabe) {
			t.Errorf("%s: erwartet ErrUngueltigeEingabe, bekommen %v", name, err)
		}
	}
}

func TestFremdeDatenBleibenUnerreichbar(t *testing.T) {
	svc, _, schueler := aufbau(t, 495)
	f, err := eintragen(t, svc, schueler, "2026-08-10", "2026-08-10", 90)
	if err != nil {
		t.Fatal(err)
	}

	const andererLehrer = int64(99)
	if _, err := svc.HoleStunde(context.Background(), f.ID, andererLehrer); !errors.Is(err, ErrNotFound) {
		t.Errorf("fremde Stunde lesbar: %v", err)
	}
	if err := svc.LoeschenStunde(context.Background(), f.ID, andererLehrer); !errors.Is(err, ErrNotFound) {
		t.Errorf("fremde Stunde löschbar: %v", err)
	}
	if _, err := svc.HoleSchueler(context.Background(), schueler, andererLehrer); !errors.Is(err, ErrNotFound) {
		t.Errorf("fremder Schüler lesbar: %v", err)
	}
	// Und die Stunde eines fremden Schülers lässt sich nicht anlegen.
	_, err = svc.AnlegenStunde(context.Background(), AnlegenInput{
		FahrlehrerID: andererLehrer, FahrschuelerID: schueler,
		GefahrenAm: datum(t, "2026-08-10"), DauerMinuten: 90,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Stunde für fremden Schüler angelegt: %v", err)
	}
}

func TestUnterschriftSetzenUndEntfernen(t *testing.T) {
	svc, _, schueler := aufbau(t, 495)
	f, err := eintragen(t, svc, schueler, "2026-08-10", "2026-08-10", 90)
	if err != nil {
		t.Fatal(err)
	}
	if f.Unterschrieben() {
		t.Fatal("frisch angelegte Stunde ist unterschrieben")
	}

	png := testUnterschrift(t)
	f, err = svc.Unterschreiben(context.Background(), f.ID, lehrer, png)
	if err != nil {
		t.Fatalf("unterschreiben: %v", err)
	}
	if !f.Unterschrieben() || f.UnterschriebenAm == nil {
		t.Error("Unterschrift nicht gesetzt")
	}

	if _, err := svc.Unterschreiben(context.Background(), f.ID, lehrer, "data:image/gif;base64,AAAA"); !errors.Is(err, ErrUngueltigeEingabe) {
		t.Error("fremdes Bildformat wurde angenommen")
	}

	f, err = svc.Unterschreiben(context.Background(), f.ID, lehrer, "")
	if err != nil {
		t.Fatalf("Unterschrift entfernen: %v", err)
	}
	if f.Unterschrieben() {
		t.Error("Unterschrift nicht entfernt")
	}
}

func TestSchuelerNameWirdBereinigtUndIstEindeutig(t *testing.T) {
	svc, _, _ := aufbau(t, 495)

	s, err := svc.AnlegenSchueler(context.Background(), lehrer, "  Tom   Schmidt  ", "BE", "")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "Tom Schmidt" {
		t.Errorf("Name %q, erwartet %q", s.Name, "Tom Schmidt")
	}
	if _, err := svc.AnlegenSchueler(context.Background(), lehrer, "tom schmidt", "", ""); !errors.Is(err, ErrNameVergeben) {
		t.Errorf("Doppelanlage erlaubt: %v", err)
	}
	if _, err := svc.AnlegenSchueler(context.Background(), lehrer, "   ", "", ""); !errors.Is(err, ErrUngueltigeEingabe) {
		t.Errorf("leerer Name erlaubt: %v", err)
	}
}
