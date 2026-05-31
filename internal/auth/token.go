package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// NewToken erzeugt ein zufälliges Rohtoken (an den Nutzer ausgegeben) und dessen
// sha256-Hash (in der DB gespeichert). So liegt nie ein nutzbares Token im Klartext
// in der Datenbank.
func NewToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	return raw, HashToken(raw), nil
}

// HashToken bildet den sha256-Hash eines Rohtokens (hex).
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
