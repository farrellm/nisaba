package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/farrellm/nisaba/internal/model"
)

// ReflexStore is a read-only data-access layer over the legacy reflex.db SQLite
// database that powers the "Anansi" pages. It mirrors the shapes the live
// Store returns (model.Document and friends) so the same API/frontend types are
// reused, but it never writes and is unrelated to the current Postgres data.
type ReflexStore struct {
	db *sql.DB
}

// NewReflexStore returns a ReflexStore backed by the given SQLite handle.
func NewReflexStore(db *sql.DB) *ReflexStore {
	return &ReflexStore{db: db}
}

// ListReflexDocuments returns every legacy document as a summary (no blocks or
// attributes), most-recently-updated first, archived included. Slices are
// defaulted to empty so the JSON is [] rather than null.
func (s *ReflexStore) ListReflexDocuments(ctx context.Context) ([]model.Document, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, title, url, create_ts, update_ts, archived
		   FROM doc ORDER BY update_ts DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	docs := []model.Document{}
	for rows.Next() {
		d, err := scanReflexDoc(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	labels, err := s.labelsByDocument(ctx)
	if err != nil {
		return nil, err
	}
	for i := range docs {
		if names := labels[docs[i].ID]; names != nil {
			docs[i].Labels = names
		}
		docs[i].PostURLs = []string{}
	}
	return docs, nil
}

// GetReflexDocument returns a single legacy document fully populated with its
// attributes, label names, and blocks (each with attributes and responses), or
// ErrNotFound.
func (s *ReflexStore) GetReflexDocument(ctx context.Context, id int64) (model.Document, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, title, url, create_ts, update_ts, archived
		   FROM doc WHERE id = ?`, id)
	d, err := scanReflexDoc(row)
	if errors.Is(err, sql.ErrNoRows) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, err
	}

	if d.Attributes, err = s.documentTags(ctx, id); err != nil {
		return d, err
	}
	if d.Labels, err = s.documentLabels(ctx, id); err != nil {
		return d, err
	}
	if d.Blocks, err = s.documentBlocks(ctx, id); err != nil {
		return d, err
	}
	d.PostURLs = []string{}
	return d, nil
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows so a single scan
// routine serves the list and get paths.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanReflexDoc reads a doc row into a model.Document, translating the legacy
// columns (archived -> IsArchived). The SQLite driver returns the DATETIME
// create_ts/update_ts columns as time.Time directly.
func scanReflexDoc(sc rowScanner) (model.Document, error) {
	var d model.Document
	var url sql.NullString
	if err := sc.Scan(&d.ID, &d.UserID, &d.Title, &url, &d.CreatedAt, &d.UpdatedAt, &d.IsArchived); err != nil {
		return d, err
	}
	if url.Valid {
		d.URL = &url.String
	}
	d.Attributes = map[string]string{}
	d.Labels = []string{}
	d.Blocks = []model.Block{}
	d.PostURLs = []string{}
	return d, nil
}

// documentTags loads a document's key/value attributes from doctag.
func (s *ReflexStore) documentTags(ctx context.Context, docID int64) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT key, value FROM doctag WHERE doc_id = ?`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	attrs := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		attrs[k] = v
	}
	return attrs, rows.Err()
}

// documentLabels loads the names of the labels a document is tagged with.
func (s *ReflexStore) documentLabels(ctx context.Context, docID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT l.value
		   FROM label l
		   JOIN doclabellink dl ON dl.label_id = l.id
		  WHERE dl.doc_id = ?
		  ORDER BY l.value`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	names := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// labelsByDocument returns every document's label names keyed by document id.
// Documents without labels are absent from the map.
func (s *ReflexStore) labelsByDocument(ctx context.Context) (map[int64][]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT dl.doc_id, l.value
		   FROM label l
		   JOIN doclabellink dl ON dl.label_id = l.id
		  ORDER BY l.value`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byDoc := map[int64][]string{}
	for rows.Next() {
		var docID int64
		var name string
		if err := rows.Scan(&docID, &name); err != nil {
			return nil, err
		}
		byDoc[docID] = append(byDoc[docID], name)
	}
	return byDoc, rows.Err()
}

// documentBlocks loads a document's blocks (ordered by id) with their
// attributes and responses. The legacy block carries the model; it is surfaced
// on each Response.Model since the live model.Block has no model field.
func (s *ReflexStore) documentBlocks(ctx context.Context, docID int64) ([]model.Block, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, mode, model FROM block WHERE doc_id = ? ORDER BY id`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocks := []model.Block{}
	models := map[int64]string{}
	for rows.Next() {
		var b model.Block
		var blockModel string
		if err := rows.Scan(&b.ID, &b.Mode, &blockModel); err != nil {
			return nil, err
		}
		b.DocumentID = docID
		b.Position = len(blocks)
		b.Attributes = map[string]string{}
		b.Responses = []model.Response{}
		models[b.ID] = blockModel
		blocks = append(blocks, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range blocks {
		if blocks[i].Attributes, err = s.blockTags(ctx, blocks[i].ID); err != nil {
			return nil, err
		}
		if blocks[i].Responses, err = s.blockResponses(ctx, blocks[i].ID, models[blocks[i].ID]); err != nil {
			return nil, err
		}
	}
	return blocks, nil
}

// blockTags loads a block's key/value attributes from blocktag.
func (s *ReflexStore) blockTags(ctx context.Context, blockID int64) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT key, value FROM blocktag WHERE block_id = ?`, blockID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	attrs := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		attrs[k] = v
	}
	return attrs, rows.Err()
}

// blockResponses loads a block's responses (ordered by id), stamping each with
// the owning block's model and a positional index.
func (s *ReflexStore) blockResponses(ctx context.Context, blockID int64, blockModel string) ([]model.Response, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, value FROM blockresponse WHERE block_id = ? ORDER BY id`, blockID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	responses := []model.Response{}
	for rows.Next() {
		var r model.Response
		if err := rows.Scan(&r.ID, &r.Value); err != nil {
			return nil, err
		}
		r.BlockID = blockID
		r.Model = blockModel
		r.Position = len(responses)
		responses = append(responses, r)
	}
	return responses, rows.Err()
}
