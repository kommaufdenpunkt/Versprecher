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
	}

	if len(cfg.JWTSecret) < 16 {
		return nil, fmt.Errorf("JWT_SECRET fehlt oder ist zu kurz (mind. 16 Zeichen)")
	}
	return cfg, nil
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
