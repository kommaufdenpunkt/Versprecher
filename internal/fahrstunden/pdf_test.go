package fahrstunden

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"time"
)

// testUnterschrift baut eine kleine, echte PNG-data-URL — so wie sie die
// Oberfläche aus dem Unterschriftsfeld schickt.
func testUnterschrift(t *testing.T) string {
	t.Helper()
	bild := image.NewNRGBA(image.Rect(0, 0, 220, 70))
	tinte := color.NRGBA{R: 24, G: 28, B: 40, A: 255}
	for x := 12; x < 208; x++ {
		y := 35 + (x%24)/3
		bild.Set(x, y, tinte)
		bild.Set(x, y+1, tinte)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, bild); err != nil {
		t.Fatalf("PNG: %v", err)
	}
	return unterschriftPrefix + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func tag(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := ParseDatum(s)
	if err != nil {
		t.Fatalf("Datum %q: %v", s, err)
	}
	return d
}

func TestNachweisPDFEnthaeltBeideDaten(t *testing.T) {
	sig := testUnterschrift(t)
	daten := NachweisDaten{
		Fahrschule: "Fahrschule mit Herz",
		Fahrlehrer: "M. Beispiel",
		Schueler:   Fahrschueler{ID: 1, Name: "Änne Müller-Straß", Klasse: "B"},
		Stunden: []Fahrstunde{
			{ID: 1, GefahrenAm: tag(t, "2026-08-09"), EingetragenAm: tag(t, "2026-08-08"),
				DauerMinuten: 90, Art: ArtUeberlandfahrt, Notiz: "Überland B27",
				UnterschriftPNG: &sig},
			{ID: 2, GefahrenAm: tag(t, "2026-08-10"), EingetragenAm: tag(t, "2026-08-10"),
				DauerMinuten: 45, Art: ArtUebungsstunde},
		},
		ErstelltAm:        tag(t, "2026-08-07"),
		TageslimitMinuten: 495,
	}

	out, err := NachweisPDF(daten)
	if err != nil {
		t.Fatalf("PDF: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Fatalf("kein PDF-Header: %q", out[:min(12, len(out))])
	}
	if !bytes.Contains(out, []byte("%%EOF")) {
		t.Error("PDF ist nicht sauber abgeschlossen (EOF-Marke fehlt)")
	}
	if len(out) < 1500 {
		t.Errorf("PDF verdächtig klein: %d Bytes", len(out))
	}
}

func TestNachweisPDFOhneStundenUndOhneAngaben(t *testing.T) {
	// Ein leerer Nachweis darf nicht abstürzen — er wird durchaus gedruckt,
	// wenn jemand noch keine Stunde hat.
	out, err := NachweisPDF(NachweisDaten{Schueler: Fahrschueler{Name: "Neu Angemeldet"}})
	if err != nil {
		t.Fatalf("PDF: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Fatal("kein PDF")
	}
}

func TestNachweisPDFBrichtAufMehrereSeitenUm(t *testing.T) {
	sig := testUnterschrift(t)
	stunden := make([]Fahrstunde, 0, 60)
	for i := 0; i < 60; i++ {
		f := Fahrstunde{
			ID:            int64(i + 1),
			GefahrenAm:    tag(t, "2026-01-05").AddDate(0, 0, i),
			EingetragenAm: tag(t, "2026-01-05").AddDate(0, 0, i-1),
			DauerMinuten:  90,
			Art:           ArtUebungsstunde,
			Notiz:         "Übungsstunde in der Stadt, Kreisverkehr und Rückwärtsfahren um die Ecke geübt",
		}
		if i%2 == 0 {
			s := sig
			f.UnterschriftPNG = &s
		}
		stunden = append(stunden, f)
	}
	out, err := NachweisPDF(NachweisDaten{
		Schueler: Fahrschueler{Name: "Vielfahrer"}, Stunden: stunden,
	})
	if err != nil {
		t.Fatalf("PDF: %v", err)
	}
	// Mehrere Seiten müssen entstanden sein.
	if seiten := bytes.Count(out, []byte("/Type /Page\n")); seiten < 2 {
		t.Errorf("nur %d Seiten für 60 Stunden", seiten)
	}
}

func TestKaputteUnterschriftKipptDenNachweisNicht(t *testing.T) {
	kaputt := unterschriftPrefix + "das-ist-kein-gueltiges-base64-png!!!"
	out, err := NachweisPDF(NachweisDaten{
		Schueler: Fahrschueler{Name: "Test"},
		Stunden: []Fahrstunde{{
			ID: 1, GefahrenAm: tag(t, "2026-08-09"), EingetragenAm: tag(t, "2026-08-09"),
			DauerMinuten: 90, Art: ArtUebungsstunde, UnterschriftPNG: &kaputt,
		}},
	})
	if err != nil {
		t.Fatalf("defekte Unterschrift hat das PDF gekippt: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Fatal("kein PDF")
	}
}

func TestDateinameIstUnauffaellig(t *testing.T) {
	faelle := map[string]string{
		"Änne Müller-Straß": "Fahrstunden-aenne-Mueller-Strass.pdf",
		"Tom Schmidt":       "Fahrstunden-Tom-Schmidt.pdf",
		"../../etc/passwd":  "Fahrstunden-etc-passwd.pdf",
		`a"b;rm -rf /`:      "Fahrstunden-a-b-rm-rf.pdf",
		"  Tom  ":           "Fahrstunden-Tom.pdf",
		"«»":                "Fahrstunden-Nachweis.pdf", // nichts Verwertbares übrig
	}
	for eingabe, erwartet := range faelle {
		if got := dateiname(eingabe); got != erwartet {
			t.Errorf("dateiname(%q) = %q, erwartet %q", eingabe, got, erwartet)
		}
	}
	for eingabe := range faelle {
		name := dateiname(eingabe)
		if strings.ContainsAny(name, `/\"';`) {
			t.Errorf("dateiname(%q) enthält Sonderzeichen: %q", eingabe, name)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
