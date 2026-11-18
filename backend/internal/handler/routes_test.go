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
		name, peer, xff, want string
	}{
		{"single hop", "127.0.0.1:1234", "203.0.113.7", "203.0.113.7"},
		{"spoofed prefix", "127.0.0.1:1234", "198.51.100.1, 203.0.113.7", "203.0.113.7"},
		{"no header", "127.0.0.1:1234", "", ""},
		{"untrusted peer", "192.0.2.9:1234", "203.0.113.7", "192.0.2.9"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			h := middleware.ClientIPFromXFF(trustedProxyPrefixes...)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got, _ = keyByClientIP(r)
			}))
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/auth/login", nil)
			req.RemoteAddr = tc.peer
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
