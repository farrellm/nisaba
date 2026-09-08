package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// SPA serves the built frontend out of dir, falling back to index.html for any
// path that isn't a file on disk — the equivalent of nginx's
// `try_files $uri /index.html`, so client-side routes like /documents/5 survive
// a page load. Mounted at "/*" only when WEB_DIR is set, which is what lets
// `tailscale serve` point at a single upstream for both the app and the API.
func SPA(dir string) http.Handler {
	index := filepath.Join(dir, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name, ok := resolve(dir, r.URL.Path)
		if !ok {
			http.ServeFile(w, r, index)
			return
		}
		http.ServeFile(w, r, name)
	})
}

// resolve maps a request path onto a regular file inside dir, reporting false
// when there is no such file (the SPA-fallback case) or when the path escapes
// dir. Directories are deliberately not ok: serving one would expose a listing,
// and "/" must reach index.html through the fallback anyway.
func resolve(dir, urlPath string) (string, bool) {
	// filepath.Clean on a rooted path drops any "..", so the join cannot escape
	// dir; the prefix check below is a belt-and-braces guard on that.
	name := filepath.Join(dir, filepath.Clean("/"+urlPath))
	if name != dir && !strings.HasPrefix(name, dir+string(os.PathSeparator)) {
		return "", false
	}

	info, err := os.Stat(name)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	return name, true
}
