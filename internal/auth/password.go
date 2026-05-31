package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword erzeugt einen bcrypt-Hash. Klartext-Passwörter werden nie
// gespeichert oder geloggt.
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword vergleicht Klartext mit Hash (konstante Laufzeit via bcrypt).
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
