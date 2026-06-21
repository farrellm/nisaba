package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/farrellm/nisaba/internal/model"
)

// CreateBlock inserts a block row (not its attributes or responses) and returns
// it with id populated.
func (s *Store) CreateBlock(ctx context.Context, b model.Block) (model.Block, error) {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO blocks (document_id, mode, position)
		 VALUES ($1, $2, $3)
		 RETURNING id`,
		b.DocumentID, b.Mode, b.Position,
	).Scan(&b.ID)
	if b.Attributes == nil {
		b.Attributes = map[string]string{}
	}
	return b, err
}

// GetBlock returns a single block with its attributes and responses populated.
func (s *Store) GetBlock(ctx context.Context, id int64) (model.Block, error) {
	var b model.Block
	err := s.pool.QueryRow(ctx,
		`SELECT id, document_id, mode, position FROM blocks WHERE id = $1`, id,
	).Scan(&b.ID, &b.DocumentID, &b.Mode, &b.Position)
	if errors.Is(err, pgx.ErrNoRows) {
		return b, ErrNotFound
	}
	if err != nil {
		return b, err
	}

	attrs, err := s.blockAttributes(ctx, []int64{id})
	if err != nil {
		return b, err
	}
	if b.Attributes = attrs[id]; b.Attributes == nil {
		b.Attributes = map[string]string{}
	}

	responses, err := s.blockResponses(ctx, []int64{id})
	if err != nil {
		return b, err
	}
	if b.Responses = responses[id]; b.Responses == nil {
		b.Responses = []model.Response{}
	}
	return b, nil
}

// ListBlocks returns a document's blocks (with attributes and responses) ordered
// by position.
func (s *Store) ListBlocks(ctx context.Context, documentID int64) ([]model.Block, error) {
	return s.documentBlocks(ctx, documentID)
}

// UpdateBlock updates a block's mode and position.
func (s *Store) UpdateBlock(ctx context.Context, b model.Block) error {
	ct, err := s.pool.Exec(ctx,
		`UPDATE blocks SET mode = $2, position = $3 WHERE id = $1`,
		b.ID, b.Mode, b.Position)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteBlock removes a block (cascading to its attributes and responses).
func (s *Store) DeleteBlock(ctx context.Context, id int64) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM blocks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetBlockAttribute inserts or updates a single key/value attribute on a block.
func (s *Store) SetBlockAttribute(ctx context.Context, blockID int64, key, value string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO block_attributes (block_id, key, value)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (block_id, key) DO UPDATE SET value = EXCLUDED.value`,
		blockID, key, value)
	return err
}

// DeleteBlockAttribute removes a single attribute key from a block.
func (s *Store) DeleteBlockAttribute(ctx context.Context, blockID int64, key string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM block_attributes WHERE block_id = $1 AND key = $2`, blockID, key)
	return err
}

// ReplaceBlockAttributes atomically replaces all of a block's attributes.
func (s *Store) ReplaceBlockAttributes(ctx context.Context, blockID int64, attrs map[string]string) error {
	attrs = trimAttrs(attrs)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`DELETE FROM block_attributes WHERE block_id = $1`, blockID); err != nil {
		return err
	}
	for k, v := range attrs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO block_attributes (block_id, key, value) VALUES ($1, $2, $3)`,
			blockID, k, v); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// blockAttributes loads attributes for the given block ids, keyed by block id.
func (s *Store) blockAttributes(ctx context.Context, blockIDs []int64) (map[int64]map[string]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT block_id, key, value FROM block_attributes WHERE block_id = ANY($1)`, blockIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int64]map[string]string{}
	for rows.Next() {
		var bid int64
		var k, v string
		if err := rows.Scan(&bid, &k, &v); err != nil {
			return nil, err
		}
		if out[bid] == nil {
			out[bid] = map[string]string{}
		}
		out[bid][k] = v
	}
	return out, rows.Err()
}

// blockResponses loads responses for the given block ids, keyed by block id and
// ordered by position.
func (s *Store) blockResponses(ctx context.Context, blockIDs []int64) (map[int64][]model.Response, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, block_id, value, model, position
		   FROM responses WHERE block_id = ANY($1) ORDER BY position, id`, blockIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int64][]model.Response{}
	for rows.Next() {
		var r model.Response
		if err := rows.Scan(&r.ID, &r.BlockID, &r.Value, &r.Model, &r.Position); err != nil {
			return nil, err
		}
		out[r.BlockID] = append(out[r.BlockID], r)
	}
	return out, rows.Err()
}

// documentBlocks loads a document's blocks fully populated with attributes and
// responses, using batched queries to avoid an N+1 per block.
func (s *Store) documentBlocks(ctx context.Context, documentID int64) ([]model.Block, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, document_id, mode, position
		   FROM blocks WHERE document_id = $1 ORDER BY position, id`, documentID)
	if err != nil {
		return nil, err
	}

	blocks := []model.Block{}
	var ids []int64
	for rows.Next() {
		var b model.Block
		if err := rows.Scan(&b.ID, &b.DocumentID, &b.Mode, &b.Position); err != nil {
			rows.Close()
			return nil, err
		}
		b.Attributes = map[string]string{}
		blocks = append(blocks, b)
		ids = append(ids, b.ID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(blocks) == 0 {
		return blocks, nil
	}

	attrs, err := s.blockAttributes(ctx, ids)
	if err != nil {
		return nil, err
	}
	responses, err := s.blockResponses(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range blocks {
		if a := attrs[blocks[i].ID]; a != nil {
			blocks[i].Attributes = a
		}
		if r := responses[blocks[i].ID]; r != nil {
			blocks[i].Responses = r
		} else {
			blocks[i].Responses = []model.Response{}
		}
	}
	return blocks, nil
}
