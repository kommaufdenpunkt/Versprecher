package fahrstunden

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

// NachweisDaten ist alles, was für den ausdruckbaren Nachweis gebraucht wird.
type NachweisDaten struct {
	Fahrschule string // Kopfzeile, z. B. „Fahrschule Muster“
	Fahrlehrer string // Name der Fahrlehrerin/des Fahrlehrers
	Schueler   Fahrschueler
	Stunden    []Fahrstunde

	// Von/Bis dokumentieren den Zeitraum der Liste (Nullwert = alles).
	Von, Bis time.Time

	ErstelltAm        time.Time
	TageslimitMinuten int
}

// Seitenmaße und Spaltenbreiten in mm (A4 hoch, 12 mm Rand ⇒ 186 mm nutzbar).
const (
	seiteBreite  = 210.0
	seiteHoehe   = 297.0
	randLinks    = 12.0
	randOben     = 12.0
	randUnten    = 16.0
	inhaltBreite = seiteBreite - 2*randLinks

	spNr            = 9.0
	spGefahren      = 24.0
	spEingetragen   = 24.0
	spArt           = 34.0
	spDauer         = 17.0
	spNotiz         = 44.0
	spUnterschrift  = 34.0
	zeileMin        = 7.0
	zeileMitBild    = 13.0
	notizZeileHoehe = 3.8
)

// NachweisPDF baut den Fahrstunden-Nachweis als PDF.
//
// Der Nachweis zeigt bewusst beide Daten nebeneinander — „gefahren am“ und
// „eingetragen am“ —, dazu Notiz und Unterschrift. So ist auf einem Blatt
// belegt, wann tatsächlich gefahren und wann es im FS Manager verbucht wurde.
func NachweisPDF(d NachweisDaten) ([]byte, error) {
	if d.ErstelltAm.IsZero() {
		d.ErstelltAm = time.Now()
	}
	if d.TageslimitMinuten <= 0 {
		d.TageslimitMinuten = StandardTageslimit
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(randLinks, randOben, randLinks)
	pdf.SetAutoPageBreak(false, randUnten) // Seitenumbruch steuern wir selbst
	pdf.SetTitle("Fahrstunden-Nachweis "+d.Schueler.Name, true)
	pdf.SetCreator("Fahrstunden-Nachweis", true)

	// Kernschriften können kein UTF-8: Text vor der Ausgabe nach CP1252 wandeln.
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	seite := &nachweisSeite{pdf: pdf, tr: tr, daten: &d}
	seite.neueSeite(true)

	for i := range d.Stunden {
		seite.zeile(i+1, &d.Stunden[i])
	}
	seite.abschluss()
	seite.fussnoten()

	if err := pdf.Error(); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// nachweisSeite hält den Zeichenzustand (aktuelle Höhe, Seitenzahl) zusammen.
type nachweisSeite struct {
	pdf    *fpdf.Fpdf
	tr     func(string) string
	daten  *NachweisDaten
	y      float64
	seiten int
}

func (s *nachweisSeite) neueSeite(erste bool) {
	s.pdf.AddPage()
	s.seiten++
	s.y = randOben

	if erste {
		s.kopf()
	} else {
		s.pdf.SetFont("Helvetica", "B", 10)
		s.pdf.SetXY(randLinks, s.y)
		s.pdf.CellFormat(inhaltBreite, 6,
			s.tr("Fahrstunden-Nachweis · "+s.daten.Schueler.Name+" (Fortsetzung)"), "", 1, "L", false, 0, "")
		s.y += 8
	}
	s.tabellenKopf()
}

// kopf zeichnet den Briefkopf: wer, für wen, welcher Zeitraum.
func (s *nachweisSeite) kopf() {
	p := s.pdf

	if s.daten.Fahrschule != "" {
		p.SetFont("Helvetica", "", 9)
		p.SetTextColor(110, 110, 110)
		p.SetXY(randLinks, s.y)
		p.CellFormat(inhaltBreite, 5, s.tr(s.daten.Fahrschule), "", 1, "L", false, 0, "")
		p.SetTextColor(0, 0, 0)
		s.y += 5
	}

	p.SetFont("Helvetica", "B", 16)
	p.SetXY(randLinks, s.y)
	p.CellFormat(inhaltBreite, 9, s.tr("Fahrstunden-Nachweis"), "", 1, "L", false, 0, "")
	s.y += 11

	// Angaben als zwei Spalten — kompakt und trotzdem gut lesbar.
	links := [][2]string{
		{"Fahrschüler:in", s.daten.Schueler.Name},
		{"Klasse", leerAls(s.daten.Schueler.Klasse, "—")},
	}
	rechts := [][2]string{
		{"Zeitraum", s.zeitraum()},
		{"Erstellt am", FormatDatumDE(s.daten.ErstelltAm)},
	}
	if s.daten.Fahrlehrer != "" {
		rechts = append(rechts, [2]string{"Fahrlehrer:in", s.daten.Fahrlehrer})
	}

	startY := s.y
	s.angabenBlock(randLinks, startY, links)
	endeRechts := s.angabenBlock(randLinks+inhaltBreite/2, startY, rechts)
	endeLinks := startY + float64(len(links))*5
	s.y = maxF(endeLinks, endeRechts) + 4

	// Kurze Erklärung, warum zwei Daten in der Liste stehen.
	p.SetFont("Helvetica", "", 8)
	p.SetTextColor(90, 90, 90)
	p.SetXY(randLinks, s.y)
	p.MultiCell(inhaltBreite, 3.8, s.tr(
		"Gefahren am = Tag der tatsächlichen Fahrstunde. Eingetragen am = Tag, unter dem die Stunde "+
			"im FS Manager verbucht ist. Beides kann abweichen, weil der FS Manager pro Tag höchstens "+
			strconv.Itoa(s.daten.TageslimitMinuten)+" Minuten zulässt."), "", "L", false)
	p.SetTextColor(0, 0, 0)
	s.y = p.GetY() + 3
}

// angabenBlock schreibt Label/Wert-Paare und liefert die Endhöhe.
func (s *nachweisSeite) angabenBlock(x, y float64, felder [][2]string) float64 {
	p := s.pdf
	for _, f := range felder {
		p.SetFont("Helvetica", "", 9)
		p.SetTextColor(110, 110, 110)
		p.SetXY(x, y)
		p.CellFormat(26, 5, s.tr(f[0]), "", 0, "L", false, 0, "")
		p.SetFont("Helvetica", "B", 9)
		p.SetTextColor(0, 0, 0)
		p.CellFormat(inhaltBreite/2-26, 5, s.tr(f[1]), "", 0, "L", false, 0, "")
		y += 5
	}
	return y
}

func (s *nachweisSeite) zeitraum() string {
	von, bis := s.daten.Von, s.daten.Bis
	// Ohne ausdrücklichen Filter den tatsächlich abgedeckten Bereich zeigen.
	if von.IsZero() && bis.IsZero() && len(s.daten.Stunden) > 0 {
		von = s.daten.Stunden[0].GefahrenAm
		bis = s.daten.Stunden[len(s.daten.Stunden)-1].GefahrenAm
	}
	switch {
	case von.IsZero() && bis.IsZero():
		return "—"
	case von.IsZero():
		return "bis " + FormatDatumDE(bis)
	case bis.IsZero():
		return "ab " + FormatDatumDE(von)
	default:
		return FormatDatumDE(von) + " – " + FormatDatumDE(bis)
	}
}

var spalten = []struct {
	titel string
	w     float64
	align string
}{
	{"Nr.", spNr, "C"},
	{"Gefahren am", spGefahren, "L"},
	{"Eingetragen am", spEingetragen, "L"},
	{"Art", spArt, "L"},
	{"Dauer", spDauer, "R"},
	{"Notiz", spNotiz, "L"},
	{"Unterschrift", spUnterschrift, "C"},
}

func (s *nachweisSeite) tabellenKopf() {
	p := s.pdf
	p.SetFont("Helvetica", "B", 8)
	p.SetFillColor(238, 240, 243)
	p.SetDrawColor(190, 195, 200)
	x := randLinks
	for _, c := range spalten {
		p.SetXY(x, s.y)
		p.CellFormat(c.w, 7, s.tr(c.titel), "1", 0, c.align, true, 0, "")
		x += c.w
	}
	s.y += 7
}

// zeile zeichnet eine Fahrstunde. Die Zeilenhöhe wächst mit der Notiz und mit
// einer vorhandenen Unterschrift, damit nichts abgeschnitten wird.
func (s *nachweisSeite) zeile(nr int, f *Fahrstunde) {
	p := s.pdf

	p.SetFont("Helvetica", "", 8)
	notizZeilen := p.SplitLines([]byte(s.tr(f.Notiz)), spNotiz-2)
	if len(notizZeilen) > 3 {
		notizZeilen = notizZeilen[:3] // im Ausdruck reichen drei Zeilen
	}

	hoehe := zeileMin
	if n := float64(len(notizZeilen))*notizZeileHoehe + 3; n > hoehe {
		hoehe = n
	}
	if f.Unterschrieben() && hoehe < zeileMitBild {
		hoehe = zeileMitBild
	}

	if s.y+hoehe > seiteHoehe-randUnten {
		s.neueSeite(false)
	}

	// Verschobene Stunden dezent hinterlegen — sie sind der Grund für dieses Blatt.
	verschoben := f.AbweichungTage() != 0
	if verschoben {
		p.SetFillColor(252, 247, 235)
	} else {
		p.SetFillColor(255, 255, 255)
	}
	p.SetDrawColor(210, 214, 219)
	p.Rect(randLinks, s.y, inhaltBreite, hoehe, "FD")

	x := randLinks
	zelle := func(w float64, txt, align string, fett bool) {
		stil := ""
		if fett {
			stil = "B"
		}
		p.SetFont("Helvetica", stil, 8)
		p.SetXY(x, s.y)
		p.CellFormat(w, hoehe, s.tr(txt), "", 0, align, false, 0, "")
		x += w
	}

	zelle(spNr, strconv.Itoa(nr), "C", false)
	zelle(spGefahren, FormatDatumDE(f.GefahrenAm), "L", false)

	// Beim Eintragetag die Verschiebung direkt danebenschreiben (+2 / −3 Tage).
	p.SetFont("Helvetica", "", 8)
	p.SetXY(x, s.y)
	p.CellFormat(spEingetragen, hoehe, s.tr(FormatDatumDE(f.EingetragenAm)+abweichungKurz(f)), "", 0, "L", false, 0, "")
	x += spEingetragen

	zelle(spArt, ArtLabel(f.Art), "L", false)
	zelle(spDauer, strconv.Itoa(f.DauerMinuten)+" Min", "R", false)

	// Notiz zeilenweise, im Zellenblock senkrecht mittig wie die übrigen Spalten.
	p.SetFont("Helvetica", "", 7.5)
	ny := s.y + (hoehe-float64(len(notizZeilen))*notizZeileHoehe)/2
	for _, z := range notizZeilen {
		p.SetXY(x, ny)
		p.CellFormat(spNotiz, notizZeileHoehe, string(z), "", 0, "L", false, 0, "")
		ny += notizZeileHoehe
	}
	x += spNotiz

	s.unterschrift(x, f, hoehe)

	// Senkrechte Trennlinien der Spalten.
	gx := randLinks
	for _, c := range spalten[:len(spalten)-1] {
		gx += c.w
		p.Line(gx, s.y, gx, s.y+hoehe)
	}

	s.y += hoehe
}

// unterschrift zeichnet das Unterschriftsbild — oder eine Linie zum Unterschreiben
// auf Papier, falls digital noch keine vorliegt.
func (s *nachweisSeite) unterschrift(x float64, f *Fahrstunde, hoehe float64) {
	p := s.pdf
	if !f.Unterschrieben() {
		p.SetDrawColor(170, 175, 180)
		y := s.y + hoehe - 2.2
		p.Line(x+3, y, x+spUnterschrift-3, y)
		p.SetDrawColor(210, 214, 219)
		return
	}

	roh, err := dekodiereUnterschrift(*f.UnterschriftPNG)
	if err != nil {
		return // defektes Bild darf den Nachweis nicht kippen
	}
	name := "sig" + strconv.FormatInt(f.ID, 10)
	info := p.RegisterImageOptionsReader(name, fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(roh))
	if info == nil || p.Err() {
		p.SetError(nil)
		return
	}

	// Bild proportional in die Zelle einpassen (mit etwas Luft).
	maxB, maxH := spUnterschrift-4, hoehe-3
	b, h := info.Extent()
	if b <= 0 || h <= 0 {
		return
	}
	faktor := minF(maxB/b, maxH/h)
	zb, zh := b*faktor, h*faktor
	p.ImageOptions(name, x+(spUnterschrift-zb)/2, s.y+(hoehe-zh)/2, zb, zh, false,
		fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
}

// abschluss zeichnet Summen und den Unterschriftsblock zum Gegenzeichnen.
func (s *nachweisSeite) abschluss() {
	p := s.pdf
	summe := Zusammenfassen(s.daten.Stunden)

	benoetigt := 46.0
	if s.y+benoetigt > seiteHoehe-randUnten {
		p.AddPage()
		s.seiten++
		s.y = randOben
		p.SetFont("Helvetica", "B", 10)
		p.SetXY(randLinks, s.y)
		p.CellFormat(inhaltBreite, 6,
			s.tr("Fahrstunden-Nachweis · "+s.daten.Schueler.Name+" (Zusammenfassung)"), "", 1, "L", false, 0, "")
		s.y += 10
	} else {
		s.y += 4
	}

	// Summenzeile.
	p.SetFillColor(238, 240, 243)
	p.SetDrawColor(190, 195, 200)
	p.Rect(randLinks, s.y, inhaltBreite, 8, "FD")
	p.SetFont("Helvetica", "B", 9)
	p.SetXY(randLinks+2, s.y)
	p.CellFormat(inhaltBreite/2, 8, s.tr(
		fmt.Sprintf("Gesamt: %d Fahrstunden · %s", summe.Anzahl, FormatDauer(summe.Minuten))),
		"", 0, "L", false, 0, "")
	p.SetFont("Helvetica", "", 9)
	p.CellFormat(inhaltBreite/2-2, 8, s.tr(
		fmt.Sprintf("davon unterschrieben: %d · verschoben eingetragen: %d",
			summe.Unterschrieben, summe.Verschoben)),
		"", 0, "R", false, 0, "")
	s.y += 10

	// Aufschlüsselung nach Art (nur was vorkommt).
	teile := make([]string, 0, len(ArtReihenfolge))
	for _, art := range ArtReihenfolge {
		if m := summe.JeArt[art]; m > 0 {
			teile = append(teile, ArtLabel(art)+": "+strconv.Itoa(m)+" Min")
		}
	}
	if len(teile) > 0 {
		p.SetFont("Helvetica", "", 8)
		p.SetTextColor(90, 90, 90)
		p.SetXY(randLinks, s.y)
		p.MultiCell(inhaltBreite, 4, s.tr(strings.Join(teile, "  ·  ")), "", "L", false)
		p.SetTextColor(0, 0, 0)
		s.y = p.GetY() + 3
	}

	// Bestätigung + Unterschriftslinien.
	p.SetFont("Helvetica", "", 8.5)
	p.SetXY(randLinks, s.y)
	p.MultiCell(inhaltBreite, 4, s.tr(
		"Hiermit bestätige ich, dass die oben aufgeführten Fahrstunden wie angegeben stattgefunden haben."),
		"", "L", false)
	s.y = p.GetY() + 12

	linien := []string{"Ort, Datum", "Unterschrift Fahrschüler:in", "Unterschrift Fahrlehrer:in"}
	breite := inhaltBreite / 3
	for i, titel := range linien {
		x := randLinks + float64(i)*breite
		p.SetDrawColor(120, 125, 130)
		p.Line(x, s.y, x+breite-8, s.y)
		p.SetFont("Helvetica", "", 7.5)
		p.SetTextColor(110, 110, 110)
		p.SetXY(x, s.y+0.5)
		p.CellFormat(breite-8, 4, s.tr(titel), "", 0, "L", false, 0, "")
	}
	p.SetTextColor(0, 0, 0)
	s.y += 8
}

// fussnoten schreibt die Seitenzahlen, wenn die Gesamtzahl feststeht.
func (s *nachweisSeite) fussnoten() {
	p := s.pdf
	gesamt := p.PageCount()
	for i := 1; i <= gesamt; i++ {
		p.SetPage(i)
		p.SetFont("Helvetica", "", 7.5)
		p.SetTextColor(130, 130, 130)
		p.SetXY(randLinks, seiteHoehe-randUnten+4)
		p.CellFormat(inhaltBreite, 4, s.tr(
			fmt.Sprintf("%s · Seite %d von %d · erstellt am %s",
				s.daten.Schueler.Name, i, gesamt, FormatDatumDE(s.daten.ErstelltAm))),
			"", 0, "C", false, 0, "")
	}
	p.SetTextColor(0, 0, 0)
}

// dekodiereUnterschrift macht aus der data-URL die rohen PNG-Bytes.
func dekodiereUnterschrift(dataURL string) ([]byte, error) {
	roh, ok := strings.CutPrefix(dataURL, unterschriftPrefix)
	if !ok {
		return nil, fmt.Errorf("unerwartetes Unterschriftsformat")
	}
	return base64.StdEncoding.DecodeString(roh)
}

// abweichungKurz macht aus der Verschiebung ein knappes „ (+2 T)“.
func abweichungKurz(f *Fahrstunde) string {
	d := f.AbweichungTage()
	if d == 0 {
		return ""
	}
	return fmt.Sprintf(" (%+d T)", d)
}

func leerAls(s, ersatz string) string {
	if strings.TrimSpace(s) == "" {
		return ersatz
	}
	return s
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
