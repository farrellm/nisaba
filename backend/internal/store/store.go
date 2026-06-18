// Package store holds the data-access layer: SQL queries over the pgx pool that
// read and write the domain types in internal/model. Methods hang off Store so
// handlers depend on one value rather than the raw pool.
package store

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a query that expects to find a row finds none.
var ErrNotFound = errors.New("store: not found")

// Store provides data-access methods backed by a pgx connection pool.
type Store struct {
	pool *pgxpool.Pool
}

// New returns a Store backed by the given pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}
