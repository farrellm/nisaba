package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// getIP extracts the remote IP from the request.
func getIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// RateLimit implements an IP-based token bucket rate limiter.
func RateLimit(limit rate.Limit, burst int) func(http.Handler) http.Handler {
	var (
		mu       sync.Mutex
		visitors = make(map[string]*visitor)
	)

	// Background cleanup of old entries
	go func() {
		for {
			time.Sleep(time.Minute)

			mu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIP(r)

			mu.Lock()
			v, exists := visitors[ip]
			if !exists {
				limiter := rate.NewLimiter(limit, burst)
				v = &visitor{limiter: limiter, lastSeen: time.Now()}
				visitors[ip] = v
			}

			v.lastSeen = time.Now()
			limiter := v.limiter
			mu.Unlock()

			if !limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"Too many requests"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
