package model

// Label is a user-scoped tag for documents. Names are unique per user, keeping
// labels private to each user.
type Label struct {
	ID     int64  `json:"id"`
	UserID int64  `json:"userId"`
	Name   string `json:"name"`
}
