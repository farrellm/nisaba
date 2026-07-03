-- Accelerates full-text search over the `story` document attribute (Search page).
-- Partial GIN index on the English tsvector of the value; the expression must
-- match the query's `to_tsvector('english', value)` exactly for the planner to
-- use it. Scoped to key='story' so it only covers rows Search actually queries.
CREATE INDEX IF NOT EXISTS idx_document_attributes_story_fts
    ON document_attributes
    USING GIN (to_tsvector('english', value))
    WHERE key = 'story';
