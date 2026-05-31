package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter: einfacher Per-IP-Begrenzer gegen Brute-Force auf Schreib-Endpoints.
// Bewusst simpel (In-Memory). Für mehrere Instanzen später z. B. Redis.
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	perMin   int
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter erlaubt perMinute Anfragen pro IP (mit kleinem Burst).
func NewRateLimiter(perMinute int) *RateLimiter {
	rl := &RateLimiter{visitors: make(map[string]*visitor), perMin: perMinute}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) limiterFor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	v, ok := rl.visitors[ip]
	if !ok {
		// perMin Anfragen/Minute, Burst = max(perMin/4, 1).
		burst := rl.perMin / 4
		if burst < 1 {
			burst = 1
		}
		lim := rate.NewLimiter(rate.Every(time.Minute/time.Duration(rl.perMin)), burst)
		rl.visitors[ip] = &visitor{limiter: lim, lastSeen: time.Now()}
		return lim
	}
	v.lastSeen = time.Now()
	return v.limiter
}

func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(3 * time.Minute)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 10*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware begrenzt Anfragen pro Client-IP (gin.ClientIP berücksichtigt TrustedProxies).
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.limiterFor(c.ClientIP()).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "zu viele Anfragen, bitte kurz warten"})
			return
		}
		c.Next()
	}
}
