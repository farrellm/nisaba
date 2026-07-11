package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/farrellm/nisaba/internal/model"
)

// CreateDocument inserts the document row (not its blocks, attributes, or
// labels) and returns it with id and timestamps populated.
func (s *Store) CreateDocument(ctx context.Context, doc model.Document) (_ model.Document, err error) {
	defer wrap(&err, "create document")
	if doc.Metadata == nil {
		doc.Metadata = map[string]any{}
	}
	err = s.pool.QueryRow(ctx,
		`INSERT INTO documents (user_id, title, selected_model, metadata, is_archived, url)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at, updated_at`,
		doc.UserID, doc.Title, doc.SelectedModel, doc.Metadata, doc.IsArchived, doc.URL,
	).Scan(&doc.ID, &doc.CreatedAt, &doc.UpdatedAt)
	return doc, err
}

// GetDocument returns a single document fully populated with its attributes,
// label names, and blocks (each with their attributes and responses), or
// ErrNotFound.
func (s *Store) GetDocument(ctx context.Context, id int64) (_ model.Document, err error) {
	defer wrap(&err, "get document")
	var d model.Document
	err = s.pool.QueryRow(ctx,
		`SELECT id, user_id, created_at, updated_at, title, selected_model, metadata, is_archived, url
		   FROM documents WHERE id = $1`, id,
	).Scan(&d.ID, &d.UserID, &d.CreatedAt, &d.UpdatedAt,
		&d.Title, &d.SelectedModel, &d.Metadata, &d.IsArchived, &d.URL)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, err
	}

	if d.Attributes, err = s.documentAttributes(ctx, id); err != nil {
		return d, err
	}
	if d.Labels, err = s.documentLabelNames(ctx, id); err != nil {
		return d, err
	}
	if d.PostURLs, err = s.documentPostURLs(ctx, id); err != nil {
		return d, err
	}
	if d.Blocks, err = s.documentBlocks(ctx, id); err != nil {
		return d, err
	}
	return d, nil
}

// ListDocuments returns a user's documents as summaries (without nested blocks,
// attributes, or labels), most-recently-updated first. Archived documents are
// included only when includeArchived is true.
func (s *Store) ListDocuments(ctx context.Context, userID int64, includeArchived bool) (_ []model.Document, err error) {
	defer wrap(&err, "list documents")
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, created_at, updated_at, title, selected_model, metadata, is_archived, url
		   FROM documents
		  WHERE user_id = $1 AND ($2 OR NOT is_archived)
		  ORDER BY updated_at DESC`,
		userID, includeArchived)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []model.Document
	for rows.Next() {
		var d model.Document
		if err := rows.Scan(&d.ID, &d.UserID, &d.CreatedAt, &d.UpdatedAt,
			&d.Title, &d.SelectedModel, &d.Metadata, &d.IsArchived, &d.URL); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	labels, err := s.labelsByDocument(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range docs {
		if names := labels[docs[i].ID]; names != nil {
			docs[i].Labels = names
		} else {
			docs[i].Labels = []string{}
		}
		// Summaries don't load post URLs; default to an empty slice so the JSON
		// is [] rather than null.
		docs[i].PostURLs = []string{}
	}
	return docs, nil
}

// SearchDocuments returns a user's documents whose `story` attribute matches the
// full-text query, ranked by relevance then recency, as summaries (same shape as
// ListDocuments). Archived documents are always included; the caller marks them.
func (s *Store) SearchDocuments(ctx context.Context, userID int64, query string) (_ []model.Document, err error) {
	defer wrap(&err, "search documents")
	rows, err := s.pool.Query(ctx,
		`SELECT d.id, d.user_id, d.created_at, d.updated_at, d.title,
		        d.selected_model, d.metadata, d.is_archived, d.url
		   FROM documents d
		   JOIN document_attributes da ON da.document_id = d.id AND da.key = 'story'
		  WHERE d.user_id = $1
		    AND to_tsvector('english', da.value) @@ websearch_to_tsquery('english', $2)
		  ORDER BY ts_rank(to_tsvector('english', da.value),
		                   websearch_to_tsquery('english', $2)) DESC,
		           d.updated_at DESC`,
		userID, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []model.Document
	for rows.Next() {
		var d model.Document
		if err := rows.Scan(&d.ID, &d.UserID, &d.CreatedAt, &d.UpdatedAt,
			&d.Title, &d.SelectedModel, &d.Metadata, &d.IsArchived, &d.URL); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	labels, err := s.labelsByDocument(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range docs {
		if names := labels[docs[i].ID]; names != nil {
			docs[i].Labels = names
		} else {
			docs[i].Labels = []string{}
		}
		// Summaries don't load post URLs; default to an empty slice so the JSON
		// is [] rather than null.
		docs[i].PostURLs = []string{}
	}
	return docs, nil
}

// labelsByDocument returns, for every document owned by the user, its label names
// keyed by document id. Documents without labels are absent from the map.
func (s *Store) labelsByDocument(ctx context.Context, userID int64) (map[int64][]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT dl.document_id, l.name
		   FROM labels l
		   JOIN document_labels dl ON dl.label_id = l.id
		   JOIN documents d ON d.id = dl.document_id
		  WHERE d.user_id = $1
		  ORDER BY l.name`, userID)
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

// UpdateDocument updates a document's mutable columns and bumps updated_at to
// NOW(). It returns the refreshed row, or ErrNotFound.
func (s *Store) UpdateDocument(ctx context.Context, doc model.Document) (_ model.Document, err error) {
	defer wrap(&err, "update document")
	if doc.Metadata == nil {
		doc.Metadata = map[string]any{}
	}
	err = s.pool.QueryRow(ctx,
		`UPDATE documents
		    SET title = $2, selected_model = $3, metadata = $4, is_archived = $5, url = $6, updated_at = NOW()
		  WHERE id = $1
		 RETURNING user_id, created_at, updated_at`,
		doc.ID, doc.Title, doc.SelectedModel, doc.Metadata, doc.IsArchived, doc.URL,
	).Scan(&doc.UserID, &doc.CreatedAt, &doc.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return doc, ErrNotFound
	}
	return doc, err
}

// DeleteDocument removes a document (cascading to its blocks, attributes, and
// label taggings) and then deletes any of the owner's labels left attached to no
// document. userID scopes that orphan cleanup.
func (s *Store) DeleteDocument(ctx context.Context, userID, id int64) (err error) {
	defer wrap(&err, "delete document")
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	ct, err := tx.Exec(ctx, `DELETE FROM documents WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}

	// The document delete cascaded its label taggings; drop now-orphaned labels.
	if _, err := tx.Exec(ctx, deleteOrphanLabelsSQL, userID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// SetDocumentAttribute inserts or updates a single key/value attribute.
func (s *Store) SetDocumentAttribute(ctx context.Context, documentID int64, key, value string) (err error) {
	defer wrap(&err, "set document attribute")
	_, err = s.pool.Exec(ctx,
		`INSERT INTO document_attributes (document_id, key, value)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (document_id, key) DO UPDATE SET value = EXCLUDED.value`,
		documentID, key, value)
	return err
}

// DeleteDocumentAttribute removes a single attribute key from a document.
func (s *Store) DeleteDocumentAttribute(ctx context.Context, documentID int64, key string) (err error) {
	defer wrap(&err, "delete document attribute")
	_, err = s.pool.Exec(ctx,
		`DELETE FROM document_attributes WHERE document_id = $1 AND key = $2`, documentID, key)
	return err
}

// ReplaceDocumentAttributes atomically replaces all of a document's attributes.
func (s *Store) ReplaceDocumentAttributes(ctx context.Context, documentID int64, attrs map[string]string) (err error) {
	defer wrap(&err, "replace document attributes")
	attrs = trimAttrs(attrs)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`DELETE FROM document_attributes WHERE document_id = $1`, documentID); err != nil {
		return err
	}
	for k, v := range attrs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO document_attributes (document_id, key, value) VALUES ($1, $2, $3)`,
			documentID, k, v); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// MergeDocumentAttributes upserts each given key/value into a document's
// attributes without touching keys not present in attrs. Unlike
// ReplaceDocumentAttributes it does not wipe the shared namespace, so values set
// by other modes survive.
func (s *Store) MergeDocumentAttributes(ctx context.Context, documentID int64, attrs map[string]string) (err error) {
	defer wrap(&err, "merge document attributes")
	attrs = trimAttrs(attrs)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for k, v := range attrs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO document_attributes (document_id, key, value)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (document_id, key) DO UPDATE SET value = EXCLUDED.value`,
			documentID, k, v); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// documentAttributes loads a document's key/value attributes.
func (s *Store) documentAttributes(ctx context.Context, documentID int64) (map[string]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT key, value FROM document_attributes WHERE document_id = $1`, documentID)
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

// GetDocumentAttribute returns the value of a single document attribute, and
// whether it exists. A missing document is indistinguishable from a missing key
// (both return found=false) — callers needing existence checks must do them
// separately.
// GetDocumentTitle returns a document's title without loading the full aggregate.
func (s *Store) GetDocumentTitle(ctx context.Context, documentID int64) (_ string, err error) {
	defer wrap(&err, "get document title")
	var title string
	err = s.pool.QueryRow(ctx,
		`SELECT title FROM documents WHERE id = $1`, documentID).Scan(&title)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return title, nil
}

func (s *Store) GetDocumentAttribute(ctx context.Context, documentID int64, key string) (_ string, _ bool, err error) {
	defer wrap(&err, "get document attribute")
	var value string
	err = s.pool.QueryRow(ctx,
		`SELECT value FROM document_attributes WHERE document_id = $1 AND key = $2`,
		documentID, key).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

// documentLabelNames loads the names of the labels a document is tagged with.
func (s *Store) documentLabelNames(ctx context.Context, documentID int64) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT l.name
		   FROM labels l
		   JOIN document_labels dl ON dl.label_id = l.id
		  WHERE dl.document_id = $1
		  ORDER BY l.name`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// AddDocumentPost records a post URL published from a document. Re-adding an
// existing URL is a no-op (ON CONFLICT DO NOTHING), so retries don't error.
func (s *Store) AddDocumentPost(ctx context.Context, documentID int64, url string) (err error) {
	defer wrap(&err, "add document post")
	_, err = s.pool.Exec(ctx,
		`INSERT INTO document_posts (document_id, url) VALUES ($1, $2)
		 ON CONFLICT (document_id, url) DO NOTHING`,
		documentID, url)
	return err
}

// documentPostURLs loads a document's published post URLs, oldest first. It
// always returns a non-nil slice so the JSON is [] rather than null.
func (s *Store) documentPostURLs(ctx context.Context, documentID int64) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT url FROM document_posts WHERE document_id = $1 ORDER BY created_at, url`,
		documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	urls := []string{}
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	return urls, rows.Err()
}
