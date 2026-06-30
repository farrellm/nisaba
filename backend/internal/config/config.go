package config

import (
	"os"
	"strings"
)

type Config struct {
	Addr               string
	DatabaseURL        string
	CORSOrigins        []string
	SessionSecret      string
	ModeTemplatesDir   string
	ReflexDBPath       string
	RedditClientID     string
	RedditClientSecret string
	RedditUsername     string
	RedditPassword     string
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

	// Reddit application-only OAuth credentials, from a registered app at
	// https://www.reddit.com/prefs/apps. Without these the Reddit posts endpoint
	// reports that the integration is not configured. REDDIT_USERNAME/PASSWORD are
	// the script-app account credentials used to submit posts (password grant);
	// without them the submit endpoint reports it is not configured.
	return Config{
		Addr:               addr,
		DatabaseURL:        dbURL,
		CORSOrigins:        strings.Split(originsEnv, ","),
		SessionSecret:      sessionSecret,
		ModeTemplatesDir:   modeTemplatesDir,
		ReflexDBPath:       reflexDBPath,
		RedditClientID:     os.Getenv("REDDIT_CLIENT_ID"),
		RedditClientSecret: os.Getenv("REDDIT_CLIENT_SECRET"),
		RedditUsername:     os.Getenv("REDDIT_USERNAME"),
		RedditPassword:     os.Getenv("REDDIT_PASSWORD"),
	}
}
