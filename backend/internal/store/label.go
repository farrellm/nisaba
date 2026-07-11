package store

import (
	"context"
	"errors"
	"strings"

	"github.com/farrellm/nisaba/internal/model"
	"github.com/jackc/pgx/v5"
)

// deleteOrphanLabelsSQL removes a user's labels that are no longer attached to any
// document. Run it after any operation that can detach a label (label reconcile,
// document delete) so labels exist only while associated with at least one document.
const deleteOrphanLabelsSQL = `DELETE FROM labels l
	 WHERE l.user_id = $1
	   AND NOT EXISTS (SELECT 1 FROM document_labels dl WHERE dl.label_id = l.id)`

// CreateLabel inserts a label for a user. The (user_id, name) uniqueness
// constraint surfaces as an error if the name already exists for that user.
func (s *Store) CreateLabel(ctx context.Context, userID int64, name string) (_ model.Label, err error) {
	defer wrap(&err, "create label")
	l := model.Label{UserID: userID, Name: name}
	err = s.pool.QueryRow(ctx,
		`INSERT INTO labels (user_id, name) VALUES ($1, $2) RETURNING id`,
		userID, name,
	).Scan(&l.ID)
	return l, err
}

// ListLabels returns a user's labels ordered by name.
func (s *Store) ListLabels(ctx context.Context, userID int64) (_ []model.Label, err error) {
	defer wrap(&err, "list labels")
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
func (s *Store) DeleteLabel(ctx context.Context, userID, id int64) (err error) {
	defer wrap(&err, "delete label")
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

// DeleteLabelByName removes a user's label by its name (cascading to its document
// taggings), detaching it from every document at once. Scoping by user_id prevents
// touching another user's label. Returns ErrNotFound if the name doesn't exist.
func (s *Store) DeleteLabelByName(ctx context.Context, userID int64, name string) (err error) {
	defer wrap(&err, "delete label by name")
	ct, err := s.pool.Exec(ctx,
		`DELETE FROM labels WHERE user_id = $1 AND name = $2`, userID, name)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RenameLabel renames a user's label across every document at once. newName is
// trimmed; an empty result is rejected with ErrEmptyName. When no label named
// newName exists the row is renamed in place. When one already exists (a different
// label), the two are merged: the old label's taggings are repointed onto the
// existing label and the old row is deleted — merged is true in that case. Returns
// ErrNotFound if oldName doesn't exist. Renaming a label to its own name is a no-op.
func (s *Store) RenameLabel(ctx context.Context, userID int64, oldName, newName string) (merged bool, err error) {
	defer wrap(&err, "rename label")
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return false, ErrEmptyName
	}
	if oldName == newName {
		return false, nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var oldID int64
	if err := tx.QueryRow(ctx,
		`SELECT id FROM labels WHERE user_id = $1 AND name = $2`,
		userID, oldName).Scan(&oldID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrNotFound
		}
		return false, err
	}

	var newID int64
	err = tx.QueryRow(ctx,
		`SELECT id FROM labels WHERE user_id = $1 AND name = $2`,
		userID, newName).Scan(&newID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// No collision: rename the row in place.
		if _, err := tx.Exec(ctx,
			`UPDATE labels SET name = $1 WHERE id = $2`, newName, oldID); err != nil {
			return false, err
		}
	case err == nil:
		// Collision: repoint the old label's taggings onto the existing label,
		// then drop the old row.
		merged = true
		if _, err := tx.Exec(ctx,
			`INSERT INTO document_labels (document_id, label_id)
			 SELECT document_id, $1 FROM document_labels WHERE label_id = $2
			 ON CONFLICT DO NOTHING`,
			newID, oldID); err != nil {
			return false, err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM labels WHERE id = $1`, oldID); err != nil {
			return false, err
		}
	default:
		return false, err
	}

	return merged, tx.Commit(ctx)
}

// AddDocumentLabel tags a document with a label. It is idempotent.
func (s *Store) AddDocumentLabel(ctx context.Context, documentID, labelID int64) (err error) {
	defer wrap(&err, "add document label")
	_, err = s.pool.Exec(ctx,
		`INSERT INTO document_labels (document_id, label_id) VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`,
		documentID, labelID)
	return err
}

// RemoveDocumentLabel removes a label tag from a document.
func (s *Store) RemoveDocumentLabel(ctx context.Context, documentID, labelID int64) (err error) {
	defer wrap(&err, "remove document label")
	_, err = s.pool.Exec(ctx,
		`DELETE FROM document_labels WHERE document_id = $1 AND label_id = $2`,
		documentID, labelID)
	return err
}

// SetDocumentLabels makes a document's labels exactly the given names. Each name
// is get-or-created for the user (existing labels are reused, never duplicated,
// via the (user_id, name) uniqueness constraint), then the document_labels join
// is reconciled to that set. Names are trimmed; blanks and duplicates are dropped.
func (s *Store) SetDocumentLabels(ctx context.Context, userID, documentID int64, names []string) (err error) {
	defer wrap(&err, "set document labels")
	// Dedupe trimmed, non-empty names preserving first-seen order.
	seen := map[string]struct{}{}
	var clean []string
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, dup := seen[n]; dup {
			continue
		}
		seen[n] = struct{}{}
		clean = append(clean, n)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	ids := make([]int64, 0, len(clean))
	for _, name := range clean {
		var id int64
		// The no-op DO UPDATE makes RETURNING fire on conflict, so an existing
		// label is reused (its id returned) rather than duplicated.
		if err := tx.QueryRow(ctx,
			`INSERT INTO labels (user_id, name) VALUES ($1, $2)
			 ON CONFLICT (user_id, name) DO UPDATE SET name = EXCLUDED.name
			 RETURNING id`,
			userID, name).Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}

	// Drop taggings no longer in the set (all of them when the set is empty).
	if _, err := tx.Exec(ctx,
		`DELETE FROM document_labels WHERE document_id = $1 AND NOT (label_id = ANY($2))`,
		documentID, ids); err != nil {
		return err
	}

	for _, id := range ids {
		if _, err := tx.Exec(ctx,
			`INSERT INTO document_labels (document_id, label_id) VALUES ($1, $2)
			 ON CONFLICT DO NOTHING`,
			documentID, id); err != nil {
			return err
		}
	}

	// A removed label may have been on no other document; delete it if so.
	if _, err := tx.Exec(ctx, deleteOrphanLabelsSQL, userID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
