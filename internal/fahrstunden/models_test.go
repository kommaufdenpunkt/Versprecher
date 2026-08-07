package fahrstunden

import "testing"

func TestAbweichungTage(t *testing.T) {
	faelle := []struct {
		gefahren, eingetragen string
		erwartet              int
	}{
		{"2026-08-09", "2026-08-09", 0},
		{"2026-08-09", "2026-08-08", -1}, // Samstag davor eingetragen
		{"2026-08-09", "2026-08-12", 3},
		{"2026-02-28", "2026-03-01", 1},
		// Über die Sommerzeit-Umstellung hinweg (26.10.2026 ist der Montag danach).
		{"2026-10-24", "2026-10-26", 2},
	}
	for _, f := range faelle {
		s := Fahrstunde{GefahrenAm: tag(t, f.gefahren), EingetragenAm: tag(t, f.eingetragen)}
		if got := s.AbweichungTage(); got != f.erwartet {
			t.Errorf("%s → %s: %d Tage, erwartet %d", f.gefahren, f.eingetragen, got, f.erwartet)
		}
	}
}

func TestZusammenfassen(t *testing.T) {
	sig := testUnterschrift(t)
	list := []Fahrstunde{
		{GefahrenAm: tag(t, "2026-08-09"), EingetragenAm: tag(t, "2026-08-08"),
			DauerMinuten: 90, Art: ArtUeberlandfahrt, UnterschriftPNG: &sig},
		{GefahrenAm: tag(t, "2026-08-10"), EingetragenAm: tag(t, "2026-08-10"),
			DauerMinuten: 45, Art: ArtUebungsstunde},
		{GefahrenAm: tag(t, "2026-08-11"), EingetragenAm: tag(t, "2026-08-12"),
			DauerMinuten: 135, Art: ArtUeberlandfahrt},
	}
	s := Zusammenfassen(list)

	if s.Anzahl != 3 || s.Minuten != 270 {
		t.Errorf("Anzahl=%d Minuten=%d, erwartet 3/270", s.Anzahl, s.Minuten)
	}
	if s.Unterschrieben != 1 {
		t.Errorf("unterschrieben=%d, erwartet 1", s.Unterschrieben)
	}
	if s.Verschoben != 2 {
		t.Errorf("verschoben=%d, erwartet 2", s.Verschoben)
	}
	if s.JeArt[ArtUeberlandfahrt] != 225 || s.JeArt[ArtUebungsstunde] != 45 {
		t.Errorf("je Art falsch: %v", s.JeArt)
	}
}

func TestFormatDauer(t *testing.T) {
	faelle := map[int]string{
		45:  "45 Min",
		90:  "90 Min (1:30 h)",
		135: "135 Min (2:15 h)",
		495: "495 Min (8:15 h)",
		60:  "60 Min (1:00 h)",
		0:   "0 Min",
	}
	for minuten, erwartet := range faelle {
		if got := FormatDauer(minuten); got != erwartet {
			t.Errorf("FormatDauer(%d) = %q, erwartet %q", minuten, got, erwartet)
		}
	}
}

func TestParseUndFormatDatum(t *testing.T) {
	d, err := ParseDatum("2026-08-09")
	if err != nil {
		t.Fatal(err)
	}
	if FormatDatum(d) != "2026-08-09" {
		t.Errorf("FormatDatum = %q", FormatDatum(d))
	}
	if FormatDatumDE(d) != "09.08.2026" {
		t.Errorf("FormatDatumDE = %q, erwartet 09.08.2026", FormatDatumDE(d))
	}
	if d.Hour() != 0 || d.Minute() != 0 {
		t.Errorf("Uhrzeit nicht abgeschnitten: %v", d)
	}
	for _, murks := range []string{"", "09.08.2026", "2026-13-01", "2026-08-32", "morgen"} {
		if _, err := ParseDatum(murks); err == nil {
			t.Errorf("ParseDatum(%q) hätte scheitern müssen", murks)
		}
	}
}

func TestArtLabel(t *testing.T) {
	if ArtLabel(ArtUeberlandfahrt) != "Überlandfahrt" {
		t.Errorf("ArtLabel(%q) = %q", ArtUeberlandfahrt, ArtLabel(ArtUeberlandfahrt))
	}
	// Unbekannte Werte kommen unverändert zurück, statt zu verschwinden.
	if ArtLabel("etwas-neues") != "etwas-neues" {
		t.Error("unbekannte Art wurde verschluckt")
	}
	// Jede Art in der Reihenfolge braucht eine Beschriftung.
	for _, art := range ArtReihenfolge {
		if _, ok := ArtLabels[art]; !ok {
			t.Errorf("Art %q hat keine Beschriftung", art)
		}
	}
	if len(ArtReihenfolge) != len(ArtLabels) {
		t.Errorf("Reihenfolge (%d) und Beschriftungen (%d) laufen auseinander",
			len(ArtReihenfolge), len(ArtLabels))
	}
}

func TestGueltigeUnterschrift(t *testing.T) {
	if !gueltigeUnterschrift(testUnterschrift(t)) {
		t.Error("gültige Unterschrift abgelehnt")
	}
	abzulehnen := []string{
		"",
		"data:image/png;base64,",
		"data:image/jpeg;base64,AAAA",
		"javascript:alert(1)",
		unterschriftPrefix + string(make([]byte, maxUnterschriftLen)), // zu groß
	}
	for _, s := range abzulehnen {
		if gueltigeUnterschrift(s) {
			t.Errorf("ungültige Unterschrift angenommen: %.40q", s)
		}
	}
}
