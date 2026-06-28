// Package model defines the domain types persisted by the backend. The structs
// here mirror the database schema in db/migrations and are shaped for use as
// API request/response bodies. They carry no data-access logic.
package model

import "time"

// User owns documents and labels. The stored password hash is never serialized.
type User struct {
	ID               int64     `json:"id"`
	Username         string    `json:"username"`
	CreatedAt        time.Time `json:"createdAt"`
	Subreddit        string    `json:"subreddit"`
	StreamingEnabled bool      `json:"streamingEnabled"`
}
