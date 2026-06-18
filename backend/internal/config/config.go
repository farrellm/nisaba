package config

import (
	"os"
	"strings"
)

type Config struct {
	Addr          string
	DatabaseURL   string
	CORSOrigins   []string
	SessionSecret string
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

	return Config{
		Addr:          addr,
		DatabaseURL:   dbURL,
		CORSOrigins:   strings.Split(originsEnv, ","),
		SessionSecret: sessionSecret,
	}
}
