package config

import (
	"os"
	"strings"
)

type Config struct {
	Addr        string
	DatabaseURL string
	CORSOrigins []string
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

	return Config{
		Addr:        addr,
		DatabaseURL: dbURL,
		CORSOrigins: strings.Split(originsEnv, ","),
	}
}
