package auth

import (
	"testing"
	"time"
)

func TestJWTIssueAndParse(t *testing.T) {
	m := NewJWTManager("test-secret-mindestens-16", time.Hour)

	token, err := m.Issue(42, true)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	claims, err := m.Parse(token)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if claims.UID != 42 || !claims.Admin {
		t.Fatalf("Claims falsch: %+v", claims)
	}
}

func TestJWTRejectsTamperedToken(t *testing.T) {
	m := NewJWTManager("test-secret-mindestens-16", time.Hour)
	other := NewJWTManager("ein-anderes-secret-1234", time.Hour)

	token, _ := m.Issue(1, false)
	if _, err := other.Parse(token); err == nil {
		t.Fatal("Token mit falschem Secret wurde akzeptiert")
	}
}

func TestJWTExpired(t *testing.T) {
	m := NewJWTManager("test-secret-mindestens-16", -time.Minute) // bereits abgelaufen
	token, _ := m.Issue(1, false)
	if _, err := m.Parse(token); err == nil {
		t.Fatal("abgelaufenes Token wurde akzeptiert")
	}
}
