package db

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"

	_ "modernc.org/sqlite"
)

func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// OpenSQLite opens the legacy reflex.db read-only and pings it to fail fast.
// The database is treated as immutable: it is never written, so the connection
// uses SQLite's read-only/immutable open flags. The caller owns Close.
func OpenSQLite(path string) (*sql.DB, error) {
	dsn := "file:" + path + "?mode=ro&immutable=1"
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}
