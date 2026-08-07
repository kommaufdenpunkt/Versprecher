package fahrstunden

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Namen liefert den Anzeigenamen zu einer User-ID. Entkoppelt fahrstunden von
// auth (kein Import-Zyklus); implementiert von auth.Service.
type Namen interface {
	AnzeigeName(ctx context.Context, userID int64) (string, error)
}

// Handler bedient die Endpoints unter /v1/fahrstunden.
type Handler struct {
	svc        *Service
	namen      Namen
	fahrschule string // Kopfzeile im PDF
}

func NewHandler(svc *Service, namen Namen, fahrschule string) *Handler {
	return &Handler{svc: svc, namen: namen, fahrschule: fahrschule}
}

// fahrlehrerName holt den Namen für den PDF-Kopf. Fehlt er, bleibt die Zeile
// einfach weg — ein Namensproblem darf den Nachweis nicht blockieren.
func (h *Handler) fahrlehrerName(c *gin.Context) string {
	if h.namen == nil {
		return ""
	}
	name, err := h.namen.AnzeigeName(c.Request.Context(), c.GetInt64("uid"))
	if err != nil {
		return ""
	}
	return name
}

func errorStatus(err error) int {
	var limitErr *LimitFehler
	switch {
	case errors.As(err, &limitErr):
		// 409: der Wunsch ist verstanden, kollidiert aber mit dem Tageslimit.
		return http.StatusConflict
	case errors.Is(err, ErrUngueltigeEingabe), errors.Is(err, ErrGrundFehlt), errors.Is(err, ErrKeinFreierTag):
		return http.StatusBadRequest
	case errors.Is(err, ErrNameVergeben):
		return http.StatusConflict
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func fail(c *gin.Context, err error) {
	status := errorStatus(err)

	// Beim Tageslimit die Zahlen mitgeben — die Oberfläche kann so direkt
	// sagen, was noch passt, statt nur „geht nicht“.
	var limitErr *LimitFehler
	if errors.As(err, &limitErr) {
		c.JSON(status, gin.H{
			"error": limitErr.Error(),
			"code":  "tageslimit",
			"tageslimit": gin.H{
				"datum":          FormatDatum(limitErr.Datum),
				"limit_minuten":  limitErr.LimitMinuten,
				"belegt_minuten": limitErr.BelegtMinuten,
				"frei_minuten":   limitErr.FreiMinuten,
				"wunsch_minuten": limitErr.WunschMinuten,
			},
		})
		return
	}

	msg := err.Error()
	if status == http.StatusInternalServerError {
		msg = "interner Fehler"
	}
	c.JSON(status, gin.H{"error": msg})
}

func schuelerView(s *Fahrschueler) gin.H {
	return gin.H{
		"id":         s.ID,
		"name":       s.Name,
		"klasse":     s.Klasse,
		"notiz":      s.Notiz,
		"aktiv":      s.Aktiv,
		"created_at": s.CreatedAt,
	}
}

func stundeView(f *Fahrstunde) gin.H {
	return gin.H{
		"id":                 f.ID,
		"fahrschueler_id":    f.FahrschuelerID,
		"schueler_name":      f.SchuelerName,
		"schueler_klasse":    f.SchuelerKlasse,
		"gefahren_am":        FormatDatum(f.GefahrenAm),
		"gefahren_von":       f.GefahrenVon,
		"gefahren_bis":       f.GefahrenBis(),
		"eingetragen_am":     FormatDatum(f.EingetragenAm),
		"eingetragen_um":     f.EingetragenUm,
		"abweichung_tage":    f.AbweichungTage(),
		"dauer_minuten":      f.DauerMinuten,
		"art":                f.Art,
		"art_label":          ArtLabel(f.Art),
		"notiz":              f.Notiz,
		"unterschrieben":     f.Unterschrieben(),
		"unterschrieben_am":  f.UnterschriebenAm,
		"limit_uebersteuert": f.LimitUebersteuert,
		"limit_grund":        f.LimitGrund,
		"created_at":         f.CreatedAt,
	}
}

func kapazitaetView(t *Tageskapazitaet) gin.H {
	return gin.H{
		"datum":          FormatDatum(t.Datum),
		"belegt_minuten": t.BelegtMinuten,
		"frei_minuten":   t.FreiMinuten,
		"limit_minuten":  t.LimitMinuten,
		"anzahl":         t.Anzahl,
		"uebersteuert":   t.Uebersteuert,
	}
}

func vorschlagView(v *Vorschlag) gin.H {
	return gin.H{
		"datum":        FormatDatum(v.Datum),
		"frei_minuten": v.FreiMinuten,
		"abstand_tage": v.AbstandTage,
	}
}

// ---------------------------------------------------------------- Fahrschüler

type schuelerRequest struct {
	Name   *string `json:"name"`
	Klasse *string `json:"klasse"`
	Notiz  *string `json:"notiz"`
	Aktiv  *bool   `json:"aktiv"`
}

// ListSchueler: GET /v1/fahrstunden/schueler?nur_aktive=1
func (h *Handler) ListSchueler(c *gin.Context) {
	nurAktive := c.Query("nur_aktive") == "1" || c.Query("nur_aktive") == "true"
	list, err := h.svc.ListeSchueler(c.Request.Context(), c.GetInt64("uid"), nurAktive)
	if err != nil {
		fail(c, err)
		return
	}
	out := make([]gin.H, 0, len(list))
	for i := range list {
		out = append(out, schuelerView(&list[i]))
	}
	c.JSON(http.StatusOK, gin.H{"schueler": out})
}

// CreateSchueler: POST /v1/fahrstunden/schueler
func (h *Handler) CreateSchueler(c *gin.Context) {
	var req schuelerRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == nil {
		fail(c, ErrUngueltigeEingabe)
		return
	}
	s, err := h.svc.AnlegenSchueler(c.Request.Context(), c.GetInt64("uid"),
		*req.Name, wert(req.Klasse), wert(req.Notiz))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"schueler": schuelerView(s)})
}

// UpdateSchueler: PATCH /v1/fahrstunden/schueler/:id
func (h *Handler) UpdateSchueler(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req schuelerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, ErrUngueltigeEingabe)
		return
	}
	s, err := h.svc.AendernSchueler(c.Request.Context(), id, c.GetInt64("uid"), SchuelerUpdate{
		Name: req.Name, Klasse: req.Klasse, Notiz: req.Notiz, Aktiv: req.Aktiv,
	})
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"schueler": schuelerView(s)})
}

// ---------------------------------------------------------------- Fahrstunden

type stundeRequest struct {
	FahrschuelerID    int64   `json:"fahrschueler_id"`
	GefahrenAm        *string `json:"gefahren_am"`
	GefahrenVon       *string `json:"gefahren_von"`
	EingetragenAm     *string `json:"eingetragen_am"`
	EingetragenUm     *string `json:"eingetragen_um"`
	DauerMinuten      *int    `json:"dauer_minuten"`
	Art               *string `json:"art"`
	Notiz             *string `json:"notiz"`
	Unterschrift      string  `json:"unterschrift"`
	LimitUebersteuern bool    `json:"limit_uebersteuern"`
	LimitGrund        string  `json:"limit_grund"`
}

// CreateStunde: POST /v1/fahrstunden/stunden
//
// eingetragen_am darf fehlen — dann sucht der Service selbst den nächsten Tag,
// an dem die Stunde ins Tageslimit passt.
func (h *Handler) CreateStunde(c *gin.Context) {
	var req stundeRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.GefahrenAm == nil || req.DauerMinuten == nil {
		fail(c, ErrUngueltigeEingabe)
		return
	}
	gefahren, err := ParseDatum(*req.GefahrenAm)
	if err != nil {
		fail(c, ErrUngueltigeEingabe)
		return
	}
	var eingetragen time.Time
	if req.EingetragenAm != nil && *req.EingetragenAm != "" {
		eingetragen, err = ParseDatum(*req.EingetragenAm)
		if err != nil {
			fail(c, ErrUngueltigeEingabe)
			return
		}
	}

	f, err := h.svc.AnlegenStunde(c.Request.Context(), AnlegenInput{
		FahrlehrerID:      c.GetInt64("uid"),
		FahrschuelerID:    req.FahrschuelerID,
		GefahrenAm:        gefahren,
		GefahrenVon:       wert(req.GefahrenVon),
		EingetragenAm:     eingetragen,
		EingetragenUm:     wert(req.EingetragenUm),
		DauerMinuten:      *req.DauerMinuten,
		Art:               wert(req.Art),
		Notiz:             wert(req.Notiz),
		Unterschrift:      req.Unterschrift,
		LimitUebersteuern: req.LimitUebersteuern,
		LimitGrund:        req.LimitGrund,
	})
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"stunde": stundeView(f)})
}

// UpdateStunde: PATCH /v1/fahrstunden/stunden/:id
func (h *Handler) UpdateStunde(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req stundeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, ErrUngueltigeEingabe)
		return
	}
	in := AendernInput{
		GefahrenVon:       req.GefahrenVon,
		EingetragenUm:     req.EingetragenUm,
		DauerMinuten:      req.DauerMinuten,
		Art:               req.Art,
		Notiz:             req.Notiz,
		LimitUebersteuern: req.LimitUebersteuern,
		LimitGrund:        req.LimitGrund,
	}
	if req.GefahrenAm != nil {
		d, err := ParseDatum(*req.GefahrenAm)
		if err != nil {
			fail(c, ErrUngueltigeEingabe)
			return
		}
		in.GefahrenAm = &d
	}
	if req.EingetragenAm != nil {
		d, err := ParseDatum(*req.EingetragenAm)
		if err != nil {
			fail(c, ErrUngueltigeEingabe)
			return
		}
		in.EingetragenAm = &d
	}

	f, err := h.svc.AendernStunde(c.Request.Context(), id, c.GetInt64("uid"), in)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"stunde": stundeView(f)})
}

// DeleteStunde: DELETE /v1/fahrstunden/stunden/:id
func (h *Handler) DeleteStunde(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.LoeschenStunde(c.Request.Context(), id, c.GetInt64("uid")); err != nil {
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListStunden: GET /v1/fahrstunden/stunden?fahrschueler_id=&von=&bis=&limit=
func (h *Handler) ListStunden(c *gin.Context) {
	f := Filter{FahrlehrerID: c.GetInt64("uid")}
	f.FahrschuelerID, _ = strconv.ParseInt(c.DefaultQuery("fahrschueler_id", "0"), 10, 64)
	f.Limit, _ = strconv.Atoi(c.Query("limit"))
	var ok bool
	if f.Von, ok = optDatum(c, "von"); !ok {
		return
	}
	if f.Bis, ok = optDatum(c, "bis"); !ok {
		return
	}

	list, err := h.svc.ListeStunden(c.Request.Context(), f)
	if err != nil {
		fail(c, err)
		return
	}
	out := make([]gin.H, 0, len(list))
	for i := range list {
		out = append(out, stundeView(&list[i]))
	}
	summe := Zusammenfassen(list)
	c.JSON(http.StatusOK, gin.H{
		"stunden": out,
		"summe": gin.H{
			"anzahl":         summe.Anzahl,
			"minuten":        summe.Minuten,
			"unterschrieben": summe.Unterschrieben,
			"verschoben":     summe.Verschoben,
			"je_art":         summe.JeArt,
		},
	})
}

type unterschriftRequest struct {
	Unterschrift string `json:"unterschrift"`
}

// Unterschreiben: PUT /v1/fahrstunden/stunden/:id/unterschrift
// Leerer Wert entfernt die Unterschrift wieder.
func (h *Handler) Unterschreiben(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req unterschriftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, ErrUngueltigeEingabe)
		return
	}
	f, err := h.svc.Unterschreiben(c.Request.Context(), id, c.GetInt64("uid"), req.Unterschrift)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"stunde": stundeView(f)})
}

// ----------------------------------------------------------------- Kapazität

// Kapazitaet: GET /v1/fahrstunden/kapazitaet?von=&bis=
// Zeigt je Eintragetag, wie viel Zeit im FS Manager schon verbucht ist.
func (h *Handler) Kapazitaet(c *gin.Context) {
	heute := tagesbeginn(time.Now())
	von, ok := optDatum(c, "von")
	if !ok {
		return
	}
	bis, ok := optDatum(c, "bis")
	if !ok {
		return
	}
	if von.IsZero() {
		von = heute.AddDate(0, 0, -7)
	}
	if bis.IsZero() {
		bis = von.AddDate(0, 0, 27)
	}

	list, err := h.svc.Kapazitaet(c.Request.Context(), c.GetInt64("uid"), von, bis)
	if err != nil {
		fail(c, err)
		return
	}
	out := make([]gin.H, 0, len(list))
	for i := range list {
		out = append(out, kapazitaetView(&list[i]))
	}
	c.JSON(http.StatusOK, gin.H{"tage": out, "limit_minuten": h.svc.TageslimitMinuten()})
}

// Vorschlaege: GET /v1/fahrstunden/kapazitaet/vorschlaege?gefahren_am=&dauer_minuten=&fenster=
// Nennt die Eintragetage, an denen die Stunde noch ins Limit passt.
func (h *Handler) Vorschlaege(c *gin.Context) {
	gefahren, err := ParseDatum(c.Query("gefahren_am"))
	if err != nil {
		fail(c, ErrUngueltigeEingabe)
		return
	}
	dauer, err := strconv.Atoi(c.Query("dauer_minuten"))
	if err != nil {
		fail(c, ErrUngueltigeEingabe)
		return
	}
	fenster, _ := strconv.Atoi(c.Query("fenster"))

	list, err := h.svc.Vorschlaege(c.Request.Context(), c.GetInt64("uid"), gefahren, dauer, fenster)
	if err != nil {
		fail(c, err)
		return
	}
	out := make([]gin.H, 0, len(list))
	for i := range list {
		out = append(out, vorschlagView(&list[i]))
	}
	c.JSON(http.StatusOK, gin.H{"vorschlaege": out, "limit_minuten": h.svc.TageslimitMinuten()})
}

// ------------------------------------------------------------------- Nachweis

// Nachweis: GET /v1/fahrstunden/schueler/:id/nachweis.pdf?von=&bis=
func (h *Handler) Nachweis(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	uid := c.GetInt64("uid")

	schueler, err := h.svc.HoleSchueler(c.Request.Context(), id, uid)
	if err != nil {
		fail(c, err)
		return
	}
	von, ok := optDatum(c, "von")
	if !ok {
		return
	}
	bis, ok := optDatum(c, "bis")
	if !ok {
		return
	}

	stunden, err := h.svc.ListeStunden(c.Request.Context(), Filter{
		FahrlehrerID: uid, FahrschuelerID: id, Von: von, Bis: bis,
	})
	if err != nil {
		fail(c, err)
		return
	}

	pdfBytes, err := NachweisPDF(NachweisDaten{
		Fahrschule:        h.fahrschule,
		Fahrlehrer:        h.fahrlehrerName(c),
		Schueler:          *schueler,
		Stunden:           stunden,
		Von:               von,
		Bis:               bis,
		ErstelltAm:        time.Now(),
		TageslimitMinuten: h.svc.TageslimitMinuten(),
	})
	if err != nil {
		fail(c, err)
		return
	}

	c.Header("Content-Disposition", `attachment; filename="`+dateiname(schueler.Name)+`"`)
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// dateiname baut einen unauffälligen, überall zulässigen Dateinamen: nur
// Buchstaben, Ziffern und einzelne Bindestriche — Umlaute werden umschrieben.
func dateiname(name string) string {
	var b strings.Builder
	trenner := false // sammelt Trennzeichen, damit keine Doppel-Bindestriche entstehen
	for _, r := range name {
		var teil string
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			teil = string(r)
		case r == 'ä' || r == 'Ä':
			teil = "ae"
		case r == 'ö' || r == 'Ö':
			teil = "oe"
		case r == 'ü' || r == 'Ü':
			teil = "ue"
		case r == 'ß':
			teil = "ss"
		default:
			// Alles andere (Leerzeichen, Satzzeichen, Pfadtrenner) trennt nur.
			trenner = b.Len() > 0
			continue
		}
		if trenner {
			b.WriteByte('-')
			trenner = false
		}
		b.WriteString(teil)
	}
	if b.Len() == 0 {
		return "Fahrstunden-Nachweis.pdf"
	}
	return "Fahrstunden-" + b.String() + ".pdf"
}

// Stammdaten: GET /v1/fahrstunden/stammdaten — was die Oberfläche zum Start braucht.
func (h *Handler) Stammdaten(c *gin.Context) {
	arten := make([]gin.H, 0, len(ArtReihenfolge))
	for _, a := range ArtReihenfolge {
		arten = append(arten, gin.H{"wert": a, "label": ArtLabel(a)})
	}
	c.JSON(http.StatusOK, gin.H{
		"arten":         arten,
		"limit_minuten": h.svc.TageslimitMinuten(),
		"fahrschule":    h.fahrschule,
		"heute":         FormatDatum(time.Now()),
	})
}

func parseID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		fail(c, ErrUngueltigeEingabe)
		return 0, false
	}
	return id, true
}

// optDatum liest einen optionalen Datums-Query-Parameter. Ein gesetzter, aber
// unlesbarer Wert ist ein Fehler — stillschweigend ignorieren wäre schlimmer,
// weil dann heimlich der falsche Zeitraum im Nachweis landet.
func optDatum(c *gin.Context, name string) (time.Time, bool) {
	s := strings.TrimSpace(c.Query(name))
	if s == "" {
		return time.Time{}, true
	}
	d, err := ParseDatum(s)
	if err != nil {
		fail(c, ErrUngueltigeEingabe)
		return time.Time{}, false
	}
	return d, true
}

func wert(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
