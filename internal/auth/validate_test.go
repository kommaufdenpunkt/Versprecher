package auth

import "testing"

func TestValidEmail(t *testing.T) {
	ok := []string{"a@b.de", "max.mustermann@example.com"}
	bad := []string{"", "keinemail", "a@b", "a b@c.de", "@b.de"}
	for _, e := range ok {
		if !validEmail(e) {
			t.Errorf("E-Mail sollte gültig sein: %q", e)
		}
	}
	for _, e := range bad {
		if validEmail(e) {
			t.Errorf("E-Mail sollte ungültig sein: %q", e)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Max@Example.DE "); got != "max@example.de" {
		t.Errorf("NormalizeEmail = %q", got)
	}
}

func TestValidPasswordAndName(t *testing.T) {
	if validPassword("kurz") {
		t.Error("zu kurzes Passwort sollte abgelehnt werden")
	}
	if !validPassword("langgenug1") {
		t.Error("ausreichend langes Passwort sollte gültig sein")
	}
	if validDisplayName("   ") {
		t.Error("leerer Anzeigename sollte abgelehnt werden")
	}
	if !validDisplayName("Lisa") {
		t.Error("normaler Name sollte gültig sein")
	}
}
