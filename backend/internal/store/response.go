package store

import (
	"context"

	"github.com/farrellm/nisaba/internal/model"
)

// CreateResponse appends a response to a block.
func (s *Store) CreateResponse(ctx context.Context, r model.Response) (model.Response, error) {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO responses (block_id, value, model, position)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		r.BlockID, r.Value, r.Model, r.Position,
	).Scan(&r.ID)
	return r, err
}

// ListResponses returns a block's responses ordered by position.
func (s *Store) ListResponses(ctx context.Context, blockID int64) ([]model.Response, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, block_id, value, model, position
		   FROM responses WHERE block_id = $1 ORDER BY position, id`, blockID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var responses []model.Response
	for rows.Next() {
		var r model.Response
		if err := rows.Scan(&r.ID, &r.BlockID, &r.Value, &r.Model, &r.Position); err != nil {
			return nil, err
		}
		responses = append(responses, r)
	}
	return responses, rows.Err()
}

// UpdateResponse replaces a response's text. The value is stored raw (no
// trimming), matching CreateResponse.
func (s *Store) UpdateResponse(ctx context.Context, r model.Response) error {
	ct, err := s.pool.Exec(ctx,
		`UPDATE responses SET value = $2 WHERE id = $1`, r.ID, r.Value)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteResponse removes a response.
func (s *Store) DeleteResponse(ctx context.Context, id int64) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM responses WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
