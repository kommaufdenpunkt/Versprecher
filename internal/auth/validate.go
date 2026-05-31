package auth

import (
	"regexp"
	"strings"
)

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

const minPasswordLen = 8

// NormalizeEmail vereinheitlicht E-Mails (trim + lowercase).
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validEmail(email string) bool {
	return len(email) <= 254 && emailRe.MatchString(email)
}

// validPassword: einfache, aber sinnvolle Mindestanforderung.
func validPassword(pw string) bool {
	return len(pw) >= minPasswordLen && len(pw) <= 200
}

func validDisplayName(name string) bool {
	n := strings.TrimSpace(name)
	return len(n) >= 1 && len(n) <= 80
}
