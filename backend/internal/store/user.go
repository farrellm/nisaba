package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/farrellm/nisaba/internal/model"
)

// CreateUser inserts a user and returns the stored record (the password hash is
// never returned).
func (s *Store) CreateUser(ctx context.Context, username, passwordHash string) (model.User, error) {
	var u model.User
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (username, password_hash)
		 VALUES ($1, $2)
		 RETURNING id, username, created_at, subreddit, streaming_enabled`,
		username, passwordHash,
	).Scan(&u.ID, &u.Username, &u.CreatedAt, &u.Subreddit, &u.StreamingEnabled)
	return u, err
}

// GetUser returns the user with the given id, or ErrNotFound.
func (s *Store) GetUser(ctx context.Context, id int64) (model.User, error) {
	var u model.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, created_at, subreddit, streaming_enabled FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Username, &u.CreatedAt, &u.Subreddit, &u.StreamingEnabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

// UpdateUserSubreddit sets the user's configured subreddit and returns the
// refreshed record, or ErrNotFound if no such user exists.
func (s *Store) UpdateUserSubreddit(ctx context.Context, id int64, subreddit string) (model.User, error) {
	var u model.User
	err := s.pool.QueryRow(ctx,
		`UPDATE users SET subreddit = $2 WHERE id = $1
		 RETURNING id, username, created_at, subreddit, streaming_enabled`,
		id, subreddit,
	).Scan(&u.ID, &u.Username, &u.CreatedAt, &u.Subreddit, &u.StreamingEnabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

// UpdateUserStreamingEnabled sets whether the user wants streamed model
// responses and returns the refreshed record, or ErrNotFound if no such user
// exists.
func (s *Store) UpdateUserStreamingEnabled(ctx context.Context, id int64, enabled bool) (model.User, error) {
	var u model.User
	err := s.pool.QueryRow(ctx,
		`UPDATE users SET streaming_enabled = $2 WHERE id = $1
		 RETURNING id, username, created_at, subreddit, streaming_enabled`,
		id, enabled,
	).Scan(&u.ID, &u.Username, &u.CreatedAt, &u.Subreddit, &u.StreamingEnabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

// GetCredentialsByUsername returns the user id and stored password hash for
// authentication, or ErrNotFound if no such user exists.
func (s *Store) GetCredentialsByUsername(ctx context.Context, username string) (id int64, passwordHash string, err error) {
	err = s.pool.QueryRow(ctx,
		`SELECT id, password_hash FROM users WHERE username = $1`, username,
	).Scan(&id, &passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", ErrNotFound
	}
	return id, passwordHash, err
}

// DeleteUser removes a user (cascading to their documents and labels).
func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
