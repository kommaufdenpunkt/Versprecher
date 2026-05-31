package auth

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("geheim1234")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "geheim1234" {
		t.Fatal("Passwort wurde nicht gehasht")
	}
	if !CheckPassword(hash, "geheim1234") {
		t.Fatal("korrektes Passwort wurde abgelehnt")
	}
	if CheckPassword(hash, "falsch") {
		t.Fatal("falsches Passwort wurde akzeptiert")
	}
}
