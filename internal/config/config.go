// Package config lädt die Anwendungskonfiguration aus Umgebungsvariablen.
// Geheimnisse (JWT-Secret, DB-Passwort) kommen ausschließlich aus der Umgebung,
// niemals aus dem Code oder dem Repo.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port        string
	DatabaseURL string

	JWTSecret string
	JWTTTL    time.Duration

	// TrustedProxies: Liste vertrauenswürdiger Proxy-IPs für die echte Client-IP.
	TrustedProxies []string

	// RequireEmailVerification: Login nur mit verifizierter E-Mail (Default true).
	RequireEmailVerification bool

	EmailVerificationTTL time.Duration

	// Schreib-Endpoints: einfaches Rate-Limit pro IP.
	RateLimitPerMinute int

	// Fidolin-Worker (Moderation).
	FidolinWorkers      int
	FidolinBatchSize    int
	FidolinPollInterval time.Duration
	FidolinStaleAfter   time.Duration

	// Optionale Blockliste für die Heuristik (sonst Default). Komma-getrennt.
	ModerationBlocklist []string

	// Fahrstunden-Nachweis (Nebenbuch zum FS Manager).
	//
	// FahrstundenTageslimit ist die Arbeitszeit, die der FS Manager pro Tag
	// zulässt (Standard 495 Minuten). Änderbar, falls sich die Vorgabe ändert.
	FahrstundenTageslimit int
	// FahrstundenBasisPfad ist die URL, unter der die Oberfläche erreichbar ist.
	// So kann der Nachweis später unter einem eigenen Namen laufen, ohne dass
	// im Code etwas geändert werden muss (z. B. FAHRSTUNDEN_BASIS_PFAD=/gino).
	FahrstundenBasisPfad string
	// FahrstundenFahrschule steht als Kopfzeile im PDF.
	FahrstundenFahrschule string
	// FahrstundenAppName ist der Name in Oberfläche und Titelzeile.
	FahrstundenAppName string
}

// Load liest die Konfiguration. JWTSecret ist Pflicht (kein unsicherer Default).
func Load() (*Config, error) {
	cfg := &Config{
		Port:                     getEnv("PORT", "8080"),
		DatabaseURL:              getEnv("DATABASE_URL", "postgres://insider_app:insider_dev_pw@127.0.0.1:5432/insider_dev"),
		JWTSecret:                os.Getenv("JWT_SECRET"),
		JWTTTL:                   getDuration("JWT_TTL", 24*time.Hour),
		TrustedProxies:           getList("TRUSTED_PROXIES"),
		RequireEmailVerification: getBool("REQUIRE_EMAIL_VERIFICATION", true),
		EmailVerificationTTL:     getDuration("EMAIL_VERIFICATION_TTL", 48*time.Hour),
		RateLimitPerMinute:       getInt("RATE_LIMIT_PER_MINUTE", 20),
		FidolinWorkers:           getInt("FIDOLIN_WORKERS", 2),
		FidolinBatchSize:         getInt("FIDOLIN_BATCH_SIZE", 10),
		FidolinPollInterval:      getDuration("FIDOLIN_POLL_INTERVAL", 2*time.Second),
		FidolinStaleAfter:        getDuration("FIDOLIN_STALE_AFTER", 5*time.Minute),
		ModerationBlocklist:      getList("MODERATION_BLOCKLIST"),
		FahrstundenTageslimit:    getInt("FAHRSTUNDEN_TAGESLIMIT_MINUTEN", 495),
		FahrstundenBasisPfad:     getEnv("FAHRSTUNDEN_BASIS_PFAD", "/fahrstunden"),
		FahrstundenFahrschule:    os.Getenv("FAHRSTUNDEN_FAHRSCHULE"),
		FahrstundenAppName:       getEnv("FAHRSTUNDEN_APP_NAME", "Fahrstunden-Nachweis"),
	}

	if len(cfg.JWTSecret) < 16 {
		return nil, fmt.Errorf("JWT_SECRET fehlt oder ist zu kurz (mind. 16 Zeichen)")
	}
	if cfg.FahrstundenTageslimit <= 0 || cfg.FahrstundenTageslimit > 1440 {
		return nil, fmt.Errorf("FAHRSTUNDEN_TAGESLIMIT_MINUTEN muss zwischen 1 und 1440 liegen")
	}
	cfg.FahrstundenBasisPfad = normalisierePfad(cfg.FahrstundenBasisPfad)
	return cfg, nil
}

// normalisierePfad macht aus „gino“, „/gino/“ usw. immer „/gino“.
func normalisierePfad(p string) string {
	p = "/" + strings.Trim(strings.TrimSpace(p), "/")
	if p == "/" {
		return "/fahrstunden" // die Wurzel bleibt frei für andere Routen
	}
	return p
}

func (c *Config) Addr() string { return ":" + c.Port }

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}

func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return def
}

func getDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return def
}

func getList(key string) []string {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
