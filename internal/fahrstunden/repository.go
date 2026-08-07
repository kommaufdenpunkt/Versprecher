package fahrstunden

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrNotFound: Datensatz existiert nicht (oder gehört einem anderen Fahrlehrer).
	ErrNotFound = errors.New("nicht gefunden")
	// ErrNameVergeben: Fahrschüler mit diesem Namen gibt es bereits.
	ErrNameVergeben = errors.New("Fahrschüler mit diesem Namen existiert bereits")
)

// LimitPruefung steuert, wie beim Schreiben mit dem Tageslimit umgegangen wird.
type LimitPruefung struct {
	LimitMinuten int
	// Uebersteuern: bewusst überschreiten. Wird an der Stunde vermerkt, damit
	// im Nachweis nachvollziehbar bleibt, warum der Tag über dem Limit liegt.
	Uebersteuern bool
}

// LimitFehler meldet, dass ein Eintragetag das Tageslimit sprengen würde.
// Er trägt die Zahlen mit, damit die Oberfläche direkt sagen kann, was noch passt.
type LimitFehler struct {
	Datum         time.Time
	LimitMinuten  int
	BelegtMinuten int
	FreiMinuten   int
	WunschMinuten int
}

func (e *LimitFehler) Error() string {
	return "Tageslimit überschritten: am " + FormatDatumDE(e.Datum) + " sind noch " +
		FormatDauer(e.FreiMinuten) + " frei, gewünscht sind " + FormatDauer(e.WunschMinuten)
}

// Repository ist der Datenzugriff (über einen Fake testbar).
type Repository interface {
	CreateSchueler(ctx context.Context, p SchuelerParams) (*Fahrschueler, error)
	UpdateSchueler(ctx context.Context, id, fahrlehrerID int64, u SchuelerUpdate) (*Fahrschueler, error)
	GetSchueler(ctx context.Context, id, fahrlehrerID int64) (*Fahrschueler, error)
	ListSchueler(ctx context.Context, fahrlehrerID int64, nurAktive bool) ([]Fahrschueler, error)

	CreateStunde(ctx context.Context, p StundeParams, l LimitPruefung) (*Fahrstunde, error)
	UpdateStunde(ctx context.Context, id, fahrlehrerID int64, u StundeUpdate, l LimitPruefung) (*Fahrstunde, error)
	DeleteStunde(ctx context.Context, id, fahrlehrerID int64) error
	GetStunde(ctx context.Context, id, fahrlehrerID int64) (*Fahrstunde, error)
	ListStunden(ctx context.Context, f Filter) ([]Fahrstunde, error)
	SetUnterschrift(ctx context.Context, id, fahrlehrerID int64, png *string) (*Fahrstunde, error)

	// Belegung liefert je Eintragetag im Zeitraum die verbuchten Minuten.
	Belegung(ctx context.Context, fahrlehrerID int64, von, bis time.Time) ([]Tageskapazitaet, error)
}

type PgRepository struct {
	pool *pgxpool.Pool
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository { return &PgRepository{pool: pool} }

const schuelerColumns = `id, fahrlehrer_id, name, klasse, notiz, aktiv, created_at`

func scanSchueler(row pgx.Row) (*Fahrschueler, error) {
	var s Fahrschueler
	err := row.Scan(&s.ID, &s.FahrlehrerID, &s.Name, &s.Klasse, &s.Notiz, &s.Aktiv, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// istUniqueVerletzung erkennt den Konflikt auf (fahrlehrer_id, lower(name)).
func istUniqueVerletzung(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (r *PgRepository) CreateSchueler(ctx context.Context, p SchuelerParams) (*Fahrschueler, error) {
	s, err := scanSchueler(r.pool.QueryRow(ctx,
		`INSERT INTO fahrschueler (fahrlehrer_id, name, klasse, notiz)
		 VALUES ($1, $2, $3, $4)
		 RETURNING `+schuelerColumns,
		p.FahrlehrerID, p.Name, p.Klasse, p.Notiz))
	if istUniqueVerletzung(err) {
		return nil, ErrNameVergeben
	}
	return s, err
}

func (r *PgRepository) UpdateSchueler(ctx context.Context, id, fahrlehrerID int64, u SchuelerUpdate) (*Fahrschueler, error) {
	s, err := scanSchueler(r.pool.QueryRow(ctx,
		`UPDATE fahrschueler
		 SET name   = COALESCE($3, name),
		     klasse = COALESCE($4, klasse),
		     notiz  = COALESCE($5, notiz),
		     aktiv  = COALESCE($6, aktiv)
		 WHERE id = $1 AND fahrlehrer_id = $2
		 RETURNING `+schuelerColumns,
		id, fahrlehrerID, u.Name, u.Klasse, u.Notiz, u.Aktiv))
	if istUniqueVerletzung(err) {
		return nil, ErrNameVergeben
	}
	return s, err
}

func (r *PgRepository) GetSchueler(ctx context.Context, id, fahrlehrerID int64) (*Fahrschueler, error) {
	return scanSchueler(r.pool.QueryRow(ctx,
		`SELECT `+schuelerColumns+` FROM fahrschueler WHERE id = $1 AND fahrlehrer_id = $2`,
		id, fahrlehrerID))
}

func (r *PgRepository) ListSchueler(ctx context.Context, fahrlehrerID int64, nurAktive bool) ([]Fahrschueler, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+schuelerColumns+`
		 FROM fahrschueler
		 WHERE fahrlehrer_id = $1 AND ($2 = false OR aktiv)
		 ORDER BY aktiv DESC, lower(name)`, fahrlehrerID, nurAktive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Fahrschueler{}
	for rows.Next() {
		var s Fahrschueler
		if err := rows.Scan(&s.ID, &s.FahrlehrerID, &s.Name, &s.Klasse, &s.Notiz, &s.Aktiv, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

const stundeColumns = `f.id, f.fahrlehrer_id, f.fahrschueler_id, f.gefahren_am, f.eingetragen_am,
	f.dauer_minuten, f.art, f.notiz, f.unterschrift_png, f.unterschrieben_am,
	f.limit_uebersteuert, f.limit_grund, f.created_at, f.updated_at, s.name, s.klasse`

func scanStunde(row pgx.Row) (*Fahrstunde, error) {
	var f Fahrstunde
	err := row.Scan(&f.ID, &f.FahrlehrerID, &f.FahrschuelerID, &f.GefahrenAm, &f.EingetragenAm,
		&f.DauerMinuten, &f.Art, &f.Notiz, &f.UnterschriftPNG, &f.UnterschriebenAm,
		&f.LimitUebersteuert, &f.LimitGrund, &f.CreatedAt, &f.UpdatedAt, &f.SchuelerName, &f.SchuelerKlasse)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// sperreTag serialisiert alle Schreibzugriffe auf einen Eintragetag eines
// Fahrlehrers. Ohne diese Sperre könnten zwei gleichzeitige Eintragungen beide
// die Limit-Prüfung bestehen und den Tag zusammen über 495 Minuten heben.
func sperreTag(ctx context.Context, tx pgx.Tx, fahrlehrerID int64, tag time.Time) error {
	tage := int32(tagesbeginn(tag).Unix() / 86400)
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1::int4, $2::int4)`, int32(fahrlehrerID), tage)
	return err
}

// belegtAm summiert die Minuten eines Eintragetages; ausser ist eine optional
// auszunehmende Stunde (beim Ändern zählt der eigene alte Wert nicht mit).
func belegtAm(ctx context.Context, tx pgx.Tx, fahrlehrerID int64, tag time.Time, ausser int64) (int, error) {
	var belegt int
	err := tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(dauer_minuten), 0)
		 FROM fahrstunden
		 WHERE fahrlehrer_id = $1 AND eingetragen_am = $2 AND ($3 = 0 OR id <> $3)`,
		fahrlehrerID, tag, ausser).Scan(&belegt)
	return belegt, err
}

// pruefeLimit stellt sicher, dass der Eintragetag das Limit einhält.
func pruefeLimit(ctx context.Context, tx pgx.Tx, fahrlehrerID int64, tag time.Time, dauer int, ausser int64, l LimitPruefung) error {
	if err := sperreTag(ctx, tx, fahrlehrerID, tag); err != nil {
		return err
	}
	belegt, err := belegtAm(ctx, tx, fahrlehrerID, tag, ausser)
	if err != nil {
		return err
	}
	frei := l.LimitMinuten - belegt
	if frei < 0 {
		frei = 0
	}
	if !l.Uebersteuern && belegt+dauer > l.LimitMinuten {
		return &LimitFehler{
			Datum:         tagesbeginn(tag),
			LimitMinuten:  l.LimitMinuten,
			BelegtMinuten: belegt,
			FreiMinuten:   frei,
			WunschMinuten: dauer,
		}
	}
	return nil
}

func (r *PgRepository) CreateStunde(ctx context.Context, p StundeParams, l LimitPruefung) (*Fahrstunde, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Der Schüler muss demselben Fahrlehrer gehören — sonst gäbe es einen Weg,
	// in fremde Nachweise zu schreiben.
	var vorhanden bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM fahrschueler WHERE id = $1 AND fahrlehrer_id = $2)`,
		p.FahrschuelerID, p.FahrlehrerID).Scan(&vorhanden); err != nil {
		return nil, err
	}
	if !vorhanden {
		return nil, ErrNotFound
	}

	if err := pruefeLimit(ctx, tx, p.FahrlehrerID, p.EingetragenAm, p.DauerMinuten, 0, l); err != nil {
		return nil, err
	}

	var unterschriebenAm *time.Time
	if p.UnterschriftPNG != nil {
		jetzt := time.Now()
		unterschriebenAm = &jetzt
	}

	f, err := scanStunde(tx.QueryRow(ctx,
		`WITH neu AS (
		     INSERT INTO fahrstunden (fahrlehrer_id, fahrschueler_id, gefahren_am, eingetragen_am,
		                              dauer_minuten, art, notiz, unterschrift_png, unterschrieben_am,
		                              limit_uebersteuert, limit_grund)
		     VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		     RETURNING *
		 )
		 SELECT `+stundeColumns+`
		 FROM neu f JOIN fahrschueler s ON s.id = f.fahrschueler_id`,
		p.FahrlehrerID, p.FahrschuelerID, p.GefahrenAm, p.EingetragenAm, p.DauerMinuten,
		p.Art, p.Notiz, p.UnterschriftPNG, unterschriebenAm, p.LimitUebersteuert, p.LimitGrund))
	if err != nil {
		return nil, err
	}
	return f, tx.Commit(ctx)
}

func (r *PgRepository) UpdateStunde(ctx context.Context, id, fahrlehrerID int64, u StundeUpdate, l LimitPruefung) (*Fahrstunde, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var altTag time.Time
	var altDauer int
	err = tx.QueryRow(ctx,
		`SELECT eingetragen_am, dauer_minuten FROM fahrstunden WHERE id = $1 AND fahrlehrer_id = $2`,
		id, fahrlehrerID).Scan(&altTag, &altDauer)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	neuTag, neuDauer := altTag, altDauer
	if u.EingetragenAm != nil {
		neuTag = *u.EingetragenAm
	}
	if u.DauerMinuten != nil {
		neuDauer = *u.DauerMinuten
	}
	// Nur prüfen, wenn sich am Eintragetag oder an der Dauer etwas ändert.
	if !neuTag.Equal(altTag) || neuDauer != altDauer {
		if err := pruefeLimit(ctx, tx, fahrlehrerID, neuTag, neuDauer, id, l); err != nil {
			return nil, err
		}
	}

	f, err := scanStunde(tx.QueryRow(ctx,
		`WITH geaendert AS (
		     UPDATE fahrstunden
		     SET gefahren_am        = COALESCE($3, gefahren_am),
		         eingetragen_am     = COALESCE($4, eingetragen_am),
		         dauer_minuten      = COALESCE($5, dauer_minuten),
		         art                = COALESCE($6, art),
		         notiz              = COALESCE($7, notiz),
		         limit_uebersteuert = CASE WHEN $8 THEN true ELSE limit_uebersteuert END,
		         limit_grund        = COALESCE($9, limit_grund),
		         updated_at         = now()
		     WHERE id = $1 AND fahrlehrer_id = $2
		     RETURNING *
		 )
		 SELECT `+stundeColumns+`
		 FROM geaendert f JOIN fahrschueler s ON s.id = f.fahrschueler_id`,
		id, fahrlehrerID, u.GefahrenAm, u.EingetragenAm, u.DauerMinuten, u.Art, u.Notiz,
		l.Uebersteuern, u.LimitGrund))
	if err != nil {
		return nil, err
	}
	return f, tx.Commit(ctx)
}

func (r *PgRepository) DeleteStunde(ctx context.Context, id, fahrlehrerID int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM fahrstunden WHERE id = $1 AND fahrlehrer_id = $2`, id, fahrlehrerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PgRepository) GetStunde(ctx context.Context, id, fahrlehrerID int64) (*Fahrstunde, error) {
	return scanStunde(r.pool.QueryRow(ctx,
		`SELECT `+stundeColumns+`
		 FROM fahrstunden f JOIN fahrschueler s ON s.id = f.fahrschueler_id
		 WHERE f.id = $1 AND f.fahrlehrer_id = $2`, id, fahrlehrerID))
}

// ListStunden liefert den Nachweis: nach Fahrtag sortiert (ältester zuerst),
// denn so wird die Liste später auch ausgedruckt und unterschrieben.
func (r *PgRepository) ListStunden(ctx context.Context, f Filter) ([]Fahrstunde, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+stundeColumns+`
		 FROM fahrstunden f JOIN fahrschueler s ON s.id = f.fahrschueler_id
		 WHERE f.fahrlehrer_id = $1
		   AND ($2 = 0 OR f.fahrschueler_id = $2)
		   AND ($3::date IS NULL OR f.gefahren_am >= $3)
		   AND ($4::date IS NULL OR f.gefahren_am <= $4)
		 ORDER BY f.gefahren_am, f.id
		 LIMIT $5`,
		f.FahrlehrerID, f.FahrschuelerID, nullDatum(f.Von), nullDatum(f.Bis), f.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Fahrstunde{}
	for rows.Next() {
		var s Fahrstunde
		if err := rows.Scan(&s.ID, &s.FahrlehrerID, &s.FahrschuelerID, &s.GefahrenAm, &s.EingetragenAm,
			&s.DauerMinuten, &s.Art, &s.Notiz, &s.UnterschriftPNG, &s.UnterschriebenAm,
			&s.LimitUebersteuert, &s.LimitGrund, &s.CreatedAt, &s.UpdatedAt,
			&s.SchuelerName, &s.SchuelerKlasse); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PgRepository) SetUnterschrift(ctx context.Context, id, fahrlehrerID int64, png *string) (*Fahrstunde, error) {
	var unterschriebenAm *time.Time
	if png != nil {
		jetzt := time.Now()
		unterschriebenAm = &jetzt
	}
	return scanStunde(r.pool.QueryRow(ctx,
		`WITH geaendert AS (
		     UPDATE fahrstunden
		     SET unterschrift_png = $3, unterschrieben_am = $4, updated_at = now()
		     WHERE id = $1 AND fahrlehrer_id = $2
		     RETURNING *
		 )
		 SELECT `+stundeColumns+`
		 FROM geaendert f JOIN fahrschueler s ON s.id = f.fahrschueler_id`,
		id, fahrlehrerID, png, unterschriebenAm))
}

// Belegung liefert nur Tage, an denen etwas eingetragen ist. Freie Tage
// ergänzt der Service — so bleibt die Abfrage klein.
func (r *PgRepository) Belegung(ctx context.Context, fahrlehrerID int64, von, bis time.Time) ([]Tageskapazitaet, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT eingetragen_am, SUM(dauer_minuten)::int, COUNT(*)::int, bool_or(limit_uebersteuert)
		 FROM fahrstunden
		 WHERE fahrlehrer_id = $1 AND eingetragen_am >= $2 AND eingetragen_am <= $3
		 GROUP BY eingetragen_am
		 ORDER BY eingetragen_am`,
		fahrlehrerID, tagesbeginn(von), tagesbeginn(bis))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Tageskapazitaet{}
	for rows.Next() {
		var t Tageskapazitaet
		if err := rows.Scan(&t.Datum, &t.BelegtMinuten, &t.Anzahl, &t.Uebersteuert); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// nullDatum macht aus dem Nullwert ein SQL-NULL (= keine Einschränkung).
func nullDatum(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	d := tagesbeginn(t)
	return &d
}
