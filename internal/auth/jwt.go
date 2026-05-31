package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims sind die JWT-Claims (§3): uid = User-ID, adm = Admin-Flag.
type Claims struct {
	UID   int64 `json:"uid"`
	Admin bool  `json:"adm"`
	jwt.RegisteredClaims
}

// JWTManager signiert und prüft Tokens mit HMAC (HS256).
type JWTManager struct {
	secret []byte
	ttl    time.Duration
}

func NewJWTManager(secret string, ttl time.Duration) *JWTManager {
	return &JWTManager{secret: []byte(secret), ttl: ttl}
}

// Issue erstellt ein signiertes Token für den Nutzer.
func (m *JWTManager) Issue(uid int64, admin bool) (string, error) {
	now := time.Now()
	claims := Claims{
		UID:   uid,
		Admin: admin,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(m.secret)
}

// Parse prüft Signatur und Gültigkeit und gibt die Claims zurück.
func (m *JWTManager) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unerwartete Signaturmethode: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims.UID == 0 {
		return nil, errors.New("token ohne uid")
	}
	return claims, nil
}
