package config

import (
	"os"
	"strings"
)

// charlotteCLIDefault is the fallback for CHARLOTTE_CLI. The Makefile sets it at
// link time (-ldflags -X) to the absolute path of charlotte-cli as found on the
// build shell's PATH; a plain `go build` leaves the bare name, resolved on PATH.
var charlotteCLIDefault = "charlotte-cli"

// Config holds every environment-derived setting, with local-dev defaults so
// `make backend` runs with no environment at all.
type Config struct {
	Addr             string
	DatabaseURL      string
	CORSOrigins      []string
	WebDir           string
	SessionSecret    string
	SessionSecure    bool
	ModeTemplatesDir string
	ReflexDBPath     string
	CharlotteCLI     string
	RedditSession    string
}

func Load() Config {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://nisaba:nisaba@localhost:5432/nisaba?sslmode=disable"
	}

	originsEnv := os.Getenv("CORS_ORIGINS")
	if originsEnv == "" {
		originsEnv = "http://localhost:5173"
	}

	// webDir is the built frontend (frontend/dist) served with an SPA fallback
	// at "/", so `tailscale serve` has a single upstream for both the app and
	// the API. Empty — the default — disables static serving entirely, which is
	// what local dev wants: Vite serves the app and proxies /api here.
	webDir := os.Getenv("WEB_DIR")

	// SessionSecret signs and encrypts the session cookie. The default is for
	// local dev only — production MUST set SESSION_SECRET to a long random value.
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "dev-insecure-session-secret-change-me"
	}

	// modeTemplatesDir is the base templates directory; per-user overrides live
	// in siblings named "<modeTemplatesDir>-<username>". The default matches the
	// `make backend` working directory (backend/).
	modeTemplatesDir := os.Getenv("MODE_TEMPLATES_DIR")
	if modeTemplatesDir == "" {
		modeTemplatesDir = "internal/mode/templates"
	}

	// reflexDBPath points at the legacy SQLite database (reflex.db) browsed
	// read-only by the "Anansi" pages. The default is relative to the
	// `make backend` working directory (backend/); the file lives at the repo root.
	reflexDBPath := os.Getenv("REFLEX_DB_PATH")
	if reflexDBPath == "" {
		reflexDBPath = "../reflex.db"
	}

	// charlotteCLI is the executable browsed read-only by the "Charlotte" pages,
	// an older file-based version of this app (`--list` / `--doc <name>`). The
	// default is charlotteCLIDefault; handlers report errors per request if it is
	// missing.
	charlotteCLI := os.Getenv("CHARLOTTE_CLI")
	if charlotteCLI == "" {
		charlotteCLI = charlotteCLIDefault
	}

	// REDDIT_SESSION is the reddit_session cookie of a logged-in old.reddit.com
	// browser session, used both to read and to submit posts. Without it the
	// Reddit endpoints report that the integration is not configured.
	return Config{
		Addr:          addr,
		DatabaseURL:   dbURL,
		CORSOrigins:   strings.Split(originsEnv, ","),
		WebDir:        webDir,
		SessionSecret: sessionSecret,
		// Mark the cookie Secure in production (HTTPS); SESSION_SECURE=true enables it.
		SessionSecure:    os.Getenv("SESSION_SECURE") == "true",
		ModeTemplatesDir: modeTemplatesDir,
		ReflexDBPath:     reflexDBPath,
		CharlotteCLI:     charlotteCLI,
		RedditSession:    os.Getenv("REDDIT_SESSION"),
	}
}
