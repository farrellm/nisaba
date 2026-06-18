package store

import (
	"context"

	"github.com/farrellm/nisaba/internal/model"
)

// CreateLabel inserts a label for a user. The (user_id, name) uniqueness
// constraint surfaces as an error if the name already exists for that user.
func (s *Store) CreateLabel(ctx context.Context, userID int64, name string) (model.Label, error) {
	l := model.Label{UserID: userID, Name: name}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO labels (user_id, name) VALUES ($1, $2) RETURNING id`,
		userID, name,
	).Scan(&l.ID)
	return l, err
}

// ListLabels returns a user's labels ordered by name.
func (s *Store) ListLabels(ctx context.Context, userID int64) ([]model.Label, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, name FROM labels WHERE user_id = $1 ORDER BY name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labels []model.Label
	for rows.Next() {
		var l model.Label
		if err := rows.Scan(&l.ID, &l.UserID, &l.Name); err != nil {
			return nil, err
		}
		labels = append(labels, l)
	}
	return labels, rows.Err()
}

// DeleteLabel removes a label owned by the user (cascading to its document
// taggings). Scoping by user_id prevents deleting another user's label.
func (s *Store) DeleteLabel(ctx context.Context, userID, id int64) error {
	ct, err := s.pool.Exec(ctx,
		`DELETE FROM labels WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AddDocumentLabel tags a document with a label. It is idempotent.
func (s *Store) AddDocumentLabel(ctx context.Context, documentID, labelID int64) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO document_labels (document_id, label_id) VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`,
		documentID, labelID)
	return err
}

// RemoveDocumentLabel removes a label tag from a document.
func (s *Store) RemoveDocumentLabel(ctx context.Context, documentID, labelID int64) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM document_labels WHERE document_id = $1 AND label_id = $2`,
		documentID, labelID)
	return err
}
