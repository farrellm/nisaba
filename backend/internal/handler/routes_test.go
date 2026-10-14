package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
)

// TestKeyByClientIPIgnoresSpoofedXFF pins the login limiter's trust model: only
// the rightmost X-Forwarded-For entry (appended by tailscale serve) counts, so a
// client can't pick its own bucket by sending a forged header.
func TestKeyByClientIPIgnoresSpoofedXFF(t *testing.T) {
	cases := []struct {
		name, xff, want string
	}{
		{"single hop", "203.0.113.7", "203.0.113.7"},
		{"spoofed prefix", "198.51.100.1, 203.0.113.7", "203.0.113.7"},
		{"no header", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			h := middleware.ClientIPFromXFF()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got, _ = keyByClientIP(r)
			}))
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/auth/login", nil)
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			if got != tc.want {
				t.Errorf("key = %q, want %q", got, tc.want)
			}
		})
	}
}
