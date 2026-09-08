package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// serveSPA builds a temp frontend/dist and returns the handler over it.
func serveSPA(t *testing.T) http.Handler {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"index.html":    "<!doctype html>index",
		"assets/app.js": "console.log(1)",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A secret next to, but outside, the served directory.
	if err := os.WriteFile(filepath.Join(dir, "..", "secret.txt"), []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	return SPA(dir)
}

func TestSPAServesFilesAndFallsBack(t *testing.T) {
	h := serveSPA(t)

	tests := []struct {
		name, path, want string
	}{
		{"real file", "/assets/app.js", "console.log(1)"},
		{"root", "/", "<!doctype html>index"},
		{"client-side route", "/documents/5", "<!doctype html>index"},
		{"unknown asset", "/assets/missing.js", "<!doctype html>index"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if got := rec.Body.String(); got != tc.want {
				t.Errorf("body = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSPADoesNotEscapeDir(t *testing.T) {
	h := serveSPA(t)

	// The file one level up must never come back, in either the raw or the
	// encoded form. http.ServeFile rejects a dotted path outright rather than
	// falling back, which is fine — all that matters is the content never
	// leaves the directory.
	for _, path := range []string{"/../secret.txt", "/%2e%2e/secret.txt"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

		if strings.Contains(rec.Body.String(), "nope") {
			t.Errorf("%s: served the file outside the directory", path)
		}
	}
}
