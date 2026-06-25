-- Records the Reddit (or other) posts published from a document. A document can
-- be posted more than once, so this is a child table rather than a column.
CREATE TABLE document_posts (
    document_id BIGINT      NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    url         TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (document_id, url)
);
