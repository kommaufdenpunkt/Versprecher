package config

import "testing"

func TestNormalisierePfad(t *testing.T) {
	faelle := map[string]string{
		// Unterpfad, egal wie geschrieben.
		"/ginos":     "/ginos",
		"ginos":      "/ginos",
		"/ginos/":    "/ginos",
		"  /ginos  ": "/ginos",
		"//ginos//":  "/ginos",
		// Eigene Domain: die Oberfläche liegt direkt auf der Wurzel.
		"/":  "/",
		"//": "/",
		// Nicht gesetzt ⇒ Standardpfad, damit die Wurzel frei bleibt.
		"":    "/fahrstunden",
		"   ": "/fahrstunden",
		// Mehrstufige Pfade bleiben erhalten.
		"/app/fahrstunden/": "/app/fahrstunden",
	}
	for eingabe, erwartet := range faelle {
		if got := normalisierePfad(eingabe); got != erwartet {
			t.Errorf("normalisierePfad(%q) = %q, erwartet %q", eingabe, got, erwartet)
		}
	}
}

func TestLoadPruefungen(t *testing.T) {
	// JWT_SECRET ist Pflicht — ohne startet der Dienst bewusst nicht.
	t.Setenv("JWT_SECRET", "")
	if _, err := Load(); err == nil {
		t.Error("Load() ohne JWT_SECRET war erfolgreich")
	}
	t.Setenv("JWT_SECRET", "zu-kurz")
	if _, err := Load(); err == nil {
		t.Error("Load() mit zu kurzem JWT_SECRET war erfolgreich")
	}

	t.Setenv("JWT_SECRET", "ein-ausreichend-langes-geheimnis")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if cfg.FahrstundenTageslimit != 495 {
		t.Errorf("Tageslimit = %d, erwartet 495", cfg.FahrstundenTageslimit)
	}
	if cfg.FahrstundenBasisPfad != "/fahrstunden" {
		t.Errorf("Basispfad = %q", cfg.FahrstundenBasisPfad)
	}

	// Ein unsinniges Tageslimit fällt beim Start auf, nicht erst im Betrieb.
	for _, murks := range []string{"0", "-30", "2000"} {
		t.Setenv("FAHRSTUNDEN_TAGESLIMIT_MINUTEN", murks)
		if _, err := Load(); err == nil {
			t.Errorf("Tageslimit %q wurde angenommen", murks)
		}
	}
}
