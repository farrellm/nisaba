package store

import "context"

// AttributeValues returns the distinct, non-empty values stored under the given
// attribute key across all of the user's documents, drawn from both block- and
// document-level attributes, sorted alphabetically. Used to suggest past values
// (e.g. author names) when editing a block.
func (s *Store) AttributeValues(ctx context.Context, userID int64, key string) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT value FROM (
		   SELECT ba.value
		     FROM block_attributes ba
		     JOIN blocks b    ON b.id = ba.block_id
		     JOIN documents d ON d.id = b.document_id
		    WHERE d.user_id = $1 AND ba.key = $2 AND ba.value <> ''
		   UNION
		   SELECT da.value
		     FROM document_attributes da
		     JOIN documents d ON d.id = da.document_id
		    WHERE d.user_id = $1 AND da.key = $2 AND da.value <> ''
		 ) v
		 ORDER BY value`,
		userID, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := []string{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, rows.Err()
}
