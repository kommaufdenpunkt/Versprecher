// Package fahrstunden ist das Nebenbuch zum FS Manager.
//
// Der FS Manager lässt pro Tag nur eine begrenzte Arbeitszeit zu (Standard 495
// Minuten). Wird an einem Tag mehr gefahren, muss die Stunde unter einem anderen
// Tag verbucht werden. Damit die Dokumentation lückenlos bleibt, hält dieses
// Paket immer beide Daten fest: „gefahren am“ und „eingetragen am“ — plus Notiz
// und Unterschrift der Fahrschülerin bzw. des Fahrschülers.
package fahrstunden

import (
	"fmt"
	"strconv"
	"time"
)

// StandardTageslimit ist die Arbeitszeit, die der FS Manager pro Tag zulässt
// (8 h 15 min). Über die Konfiguration änderbar, falls sich die Vorgabe ändert.
const StandardTageslimit = 495

// Arten einer Fahrstunde. Die Sonderfahrten sind bewusst einzeln aufgeführt,
// damit der Nachweis zeigt, was schon gefahren wurde.
const (
	ArtGrundausbildung       = "grundausbildung"
	ArtUebungsstunde         = "uebungsstunde"
	ArtUeberlandfahrt        = "ueberlandfahrt"
	ArtAutobahnfahrt         = "autobahnfahrt"
	ArtNachtfahrt            = "nachtfahrt"
	ArtPruefungsvorbereitung = "pruefungsvorbereitung"
	ArtPruefungsfahrt        = "pruefungsfahrt"
	ArtSonstiges             = "sonstiges"
)

// ArtLabels sind die ausgeschriebenen Bezeichnungen (Oberfläche und PDF).
var ArtLabels = map[string]string{
	ArtGrundausbildung:       "Grundausbildung",
	ArtUebungsstunde:         "Übungsstunde",
	ArtUeberlandfahrt:        "Überlandfahrt",
	ArtAutobahnfahrt:         "Autobahnfahrt",
	ArtNachtfahrt:            "Nachtfahrt",
	ArtPruefungsvorbereitung: "Prüfungsvorbereitung",
	ArtPruefungsfahrt:        "Prüfungsfahrt",
	ArtSonstiges:             "Sonstiges",
}

// ArtReihenfolge legt die Anzeigereihenfolge fest (Maps sind unsortiert).
var ArtReihenfolge = []string{
	ArtGrundausbildung, ArtUebungsstunde, ArtUeberlandfahrt, ArtAutobahnfahrt,
	ArtNachtfahrt, ArtPruefungsvorbereitung, ArtPruefungsfahrt, ArtSonstiges,
}

// ArtLabel liefert die ausgeschriebene Bezeichnung, sonst den Rohwert.
func ArtLabel(art string) string {
	if l, ok := ArtLabels[art]; ok {
		return l
	}
	return art
}

// Fahrschueler entspricht der Tabelle fahrschueler.
type Fahrschueler struct {
	ID           int64
	FahrlehrerID int64
	Name         string
	Klasse       string
	Notiz        string
	Aktiv        bool
	CreatedAt    time.Time
}

// Fahrstunde entspricht der Tabelle fahrstunden. SchuelerName kommt aus dem
// JOIN und ist nur beim Lesen gefüllt (Listen, PDF).
type Fahrstunde struct {
	ID             int64
	FahrlehrerID   int64
	FahrschuelerID int64
	SchuelerName   string
	SchuelerKlasse string

	GefahrenAm    time.Time
	EingetragenAm time.Time
	DauerMinuten  int
	Art           string
	Notiz         string

	UnterschriftPNG  *string
	UnterschriebenAm *time.Time

	LimitUebersteuert bool
	LimitGrund        string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// AbweichungTage sagt, wie viele Tage zwischen Fahrt und Eintragung liegen.
// Negativ bedeutet: früher eingetragen als gefahren (kommt vor, wenn der
// Fahrtag im FS Manager schon voll ist).
func (f *Fahrstunde) AbweichungTage() int {
	return int(tagesbeginn(f.EingetragenAm).Sub(tagesbeginn(f.GefahrenAm)).Hours() / 24)
}

// Unterschrieben sagt, ob eine Unterschrift hinterlegt ist.
func (f *Fahrstunde) Unterschrieben() bool {
	return f.UnterschriftPNG != nil && *f.UnterschriftPNG != ""
}

// SchuelerParams sind die Felder zum Anlegen einer Fahrschülerin/eines Fahrschülers.
type SchuelerParams struct {
	FahrlehrerID int64
	Name         string
	Klasse       string
	Notiz        string
}

// SchuelerUpdate sind die änderbaren Felder. nil = unverändert lassen.
type SchuelerUpdate struct {
	Name   *string
	Klasse *string
	Notiz  *string
	Aktiv  *bool
}

// StundeParams sind die Felder zum Anlegen einer Fahrstunde.
type StundeParams struct {
	FahrlehrerID      int64
	FahrschuelerID    int64
	GefahrenAm        time.Time
	EingetragenAm     time.Time
	DauerMinuten      int
	Art               string
	Notiz             string
	UnterschriftPNG   *string
	LimitUebersteuert bool
	LimitGrund        string
}

// StundeUpdate sind die änderbaren Felder einer Fahrstunde. nil = unverändert.
type StundeUpdate struct {
	GefahrenAm    *time.Time
	EingetragenAm *time.Time
	DauerMinuten  *int
	Art           *string
	Notiz         *string
	// LimitGrund wird nur gesetzt, wenn das Limit bewusst übersteuert wird.
	LimitGrund *string
}

// Filter grenzt eine Stundenliste ein. Nullwerte heißen „keine Einschränkung“.
type Filter struct {
	FahrlehrerID   int64
	FahrschuelerID int64
	// Von/Bis filtern auf gefahren_am (der Tag, an dem tatsächlich gefahren wurde).
	Von   time.Time
	Bis   time.Time
	Limit int
}

// Tageskapazitaet ist die Auslastung eines Eintragetages im FS Manager.
type Tageskapazitaet struct {
	Datum         time.Time
	BelegtMinuten int
	LimitMinuten  int
	FreiMinuten   int
	Anzahl        int
	// Uebersteuert: an diesem Tag wurde das Limit bewusst überschritten.
	Uebersteuert bool
}

// Vorschlag ist ein Eintragetag, an dem die gewünschte Dauer noch passt.
type Vorschlag struct {
	Datum       time.Time
	FreiMinuten int
	// AbstandTage zum Fahrtag; negativ = vor dem Fahrtag.
	AbstandTage int
}

// Summe fasst eine Stundenliste für den Nachweis zusammen.
type Summe struct {
	Anzahl         int
	Minuten        int
	JeArt          map[string]int // Art -> Minuten
	Unterschrieben int            // Anzahl unterschriebener Stunden
	Verschoben     int            // Anzahl Stunden mit gefahren_am != eingetragen_am
}

// Zusammenfassen bildet die Summen über eine Stundenliste.
func Zusammenfassen(list []Fahrstunde) Summe {
	s := Summe{JeArt: map[string]int{}}
	for i := range list {
		f := &list[i]
		s.Anzahl++
		s.Minuten += f.DauerMinuten
		s.JeArt[f.Art] += f.DauerMinuten
		if f.Unterschrieben() {
			s.Unterschrieben++
		}
		if f.AbweichungTage() != 0 {
			s.Verschoben++
		}
	}
	return s
}

// tagesbeginn schneidet die Uhrzeit ab. Alle Datumsfelder sind reine Kalendertage.
func tagesbeginn(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// DatumsFormat ist das Austauschformat der API (ISO, sortierbar, eindeutig).
const DatumsFormat = "2006-01-02"

// ParseDatum liest ein Datum im Format JJJJ-MM-TT.
func ParseDatum(s string) (time.Time, error) {
	t, err := time.Parse(DatumsFormat, s)
	if err != nil {
		return time.Time{}, err
	}
	return tagesbeginn(t), nil
}

// FormatDatum schreibt ein Datum als JJJJ-MM-TT.
func FormatDatum(t time.Time) string { return t.Format(DatumsFormat) }

// FormatDatumDE schreibt ein Datum als TT.MM.JJJJ (für PDF und Ausdruck).
func FormatDatumDE(t time.Time) string { return t.Format("02.01.2006") }

// FormatDauer macht aus Minuten „90 Min (1:30 h)“ — im Nachweis leichter zu lesen.
func FormatDauer(minuten int) string {
	h, m := minuten/60, minuten%60
	if h == 0 {
		return strconv.Itoa(m) + " Min"
	}
	return fmt.Sprintf("%d Min (%d:%02d h)", minuten, h, m)
}
