package fahrstunden

import (
	"context"
	"errors"
	"testing"
)

func TestUhrzeitMinuten(t *testing.T) {
	gueltig := map[Uhrzeit]int{
		"00:00": 0,
		"09:05": 545,
		"14:00": 840,
		"23:59": 1439,
	}
	for u, erwartet := range gueltig {
		if got := u.Minuten(); got != erwartet {
			t.Errorf("%q.Minuten() = %d, erwartet %d", u, got, erwartet)
		}
		if !u.Gueltig() || !u.Gesetzt() {
			t.Errorf("%q wurde als ungültig eingestuft", u)
		}
	}
	for _, u := range []Uhrzeit{"", "24:00", "12:60", "9:05", "14:00:00", "abc", "1:2"} {
		if u.Minuten() >= 0 {
			t.Errorf("%q wurde als gültige Uhrzeit gelesen", u)
		}
	}
}

func TestUhrzeitPlus(t *testing.T) {
	faelle := []struct {
		von      Uhrzeit
		minuten  int
		erwartet Uhrzeit
	}{
		{"14:00", 90, "15:30"},
		{"14:00", 135, "16:15"},
		{"09:05", 45, "09:50"},
		{"08:00", 0, "08:00"},
		// Nachtfahrt über Mitternacht.
		{"23:15", 90, "00:45"},
		{"23:59", 1, "00:00"},
		// Ohne Anfangszeit gibt es auch keine Endzeit.
		{"", 90, ""},
	}
	for _, f := range faelle {
		if got := f.von.Plus(f.minuten); got != f.erwartet {
			t.Errorf("%q.Plus(%d) = %q, erwartet %q", f.von, f.minuten, got, f.erwartet)
		}
	}
}

func TestBereinigeUhrzeit(t *testing.T) {
	// Browser, Datenbank und Tippfehler liefern verschiedene Schreibweisen.
	gueltig := map[string]Uhrzeit{
		"14:00":    "14:00",
		"14:00:00": "14:00",
		"9:05":     "09:05",
		" 09:05 ":  "09:05",
		"0:00":     "00:00",
		"23:59":    "23:59",
		"":         "", // nicht notiert bleibt nicht notiert
	}
	for eingabe, erwartet := range gueltig {
		got, ok := bereinigeUhrzeit(eingabe)
		if !ok || got != erwartet {
			t.Errorf("bereinigeUhrzeit(%q) = %q, %v — erwartet %q, true", eingabe, got, ok, erwartet)
		}
	}
	for _, murks := range []string{"25:00", "12:60", "halb drei", "14", "-1:00", "::", "14:xx"} {
		if _, ok := bereinigeUhrzeit(murks); ok {
			t.Errorf("bereinigeUhrzeit(%q) wurde angenommen", murks)
		}
	}
}

func TestFahrzeitraum(t *testing.T) {
	mit := Fahrstunde{GefahrenVon: "14:00", DauerMinuten: 90}
	if got := mit.Fahrzeitraum(); got != "14:00–15:30" {
		t.Errorf("Fahrzeitraum = %q, erwartet 14:00–15:30", got)
	}
	if got := mit.GefahrenBis(); got != "15:30" {
		t.Errorf("GefahrenBis = %q, erwartet 15:30", got)
	}
	ohne := Fahrstunde{DauerMinuten: 90}
	if got := ohne.Fahrzeitraum(); got != "" {
		t.Errorf("ohne Anfangszeit: Fahrzeitraum = %q, erwartet leer", got)
	}
}

func TestUhrzeitenWerdenGespeichertUndGeaendert(t *testing.T) {
	svc, _, schueler := aufbau(t, 495)

	f, err := svc.AnlegenStunde(context.Background(), AnlegenInput{
		FahrlehrerID: lehrer, FahrschuelerID: schueler,
		GefahrenAm: datum(t, "2026-09-09"), GefahrenVon: "14:00",
		EingetragenAm: datum(t, "2026-09-05"), EingetragenUm: "19:30",
		DauerMinuten: 90, Art: ArtUeberlandfahrt,
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.GefahrenVon != "14:00" || f.EingetragenUm != "19:30" {
		t.Errorf("Uhrzeiten nicht übernommen: von=%q um=%q", f.GefahrenVon, f.EingetragenUm)
	}
	if f.Fahrzeitraum() != "14:00–15:30" {
		t.Errorf("Fahrzeitraum = %q", f.Fahrzeitraum())
	}
	// Der Fahrtag bleibt unabhängig von der Uhrzeit erhalten.
	if FormatDatum(f.GefahrenAm) != "2026-09-09" || f.AbweichungTage() != -4 {
		t.Errorf("Datum/Abweichung falsch: %s / %d", FormatDatum(f.GefahrenAm), f.AbweichungTage())
	}

	// Ändern: neue Anfangszeit, Endzeit rechnet mit der neuen Dauer.
	neueZeit, neueDauer := "08:30", 135
	f, err = svc.AendernStunde(context.Background(), f.ID, lehrer, AendernInput{
		GefahrenVon: &neueZeit, DauerMinuten: &neueDauer,
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Fahrzeitraum() != "08:30–10:45" {
		t.Errorf("nach Änderung: Fahrzeitraum = %q, erwartet 08:30–10:45", f.Fahrzeitraum())
	}

	// Leerer Wert entfernt die Uhrzeit wieder; die übrigen Felder bleiben.
	leer := ""
	f, err = svc.AendernStunde(context.Background(), f.ID, lehrer, AendernInput{GefahrenVon: &leer})
	if err != nil {
		t.Fatal(err)
	}
	if f.GefahrenVon.Gesetzt() {
		t.Errorf("Uhrzeit nicht entfernt: %q", f.GefahrenVon)
	}
	if f.EingetragenUm != "19:30" {
		t.Errorf("fremde Uhrzeit mitentfernt: %q", f.EingetragenUm)
	}
}

func TestUngueltigeUhrzeitWirdAbgelehnt(t *testing.T) {
	svc, _, schueler := aufbau(t, 495)

	basis := AnlegenInput{
		FahrlehrerID: lehrer, FahrschuelerID: schueler,
		GefahrenAm: datum(t, "2026-09-09"), DauerMinuten: 90,
	}
	for _, murks := range []string{"25:00", "halb drei", "12:60"} {
		in := basis
		in.GefahrenVon = murks
		if _, err := svc.AnlegenStunde(context.Background(), in); !errors.Is(err, ErrUngueltigeEingabe) {
			t.Errorf("gefahren_von %q angenommen: %v", murks, err)
		}
		in = basis
		in.EingetragenUm = murks
		if _, err := svc.AnlegenStunde(context.Background(), in); !errors.Is(err, ErrUngueltigeEingabe) {
			t.Errorf("eingetragen_um %q angenommen: %v", murks, err)
		}
	}
	// Ohne Uhrzeit muss es weiterhin gehen — sie ist bewusst optional.
	if _, err := svc.AnlegenStunde(context.Background(), basis); err != nil {
		t.Errorf("Eintrag ohne Uhrzeit abgelehnt: %v", err)
	}
}
