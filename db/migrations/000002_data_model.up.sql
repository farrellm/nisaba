CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS documents (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    selected_model TEXT NOT NULL DEFAULT '',
    metadata       JSONB NOT NULL DEFAULT '{}',
    is_archived    BOOLEAN NOT NULL DEFAULT FALSE,
    url            TEXT
);

CREATE INDEX IF NOT EXISTS idx_documents_user_id ON documents(user_id);

CREATE TABLE IF NOT EXISTS document_attributes (
    document_id BIGINT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    key         TEXT NOT NULL,
    value       TEXT NOT NULL,
    PRIMARY KEY (document_id, key)
);

CREATE TABLE IF NOT EXISTS blocks (
    id          BIGSERIAL PRIMARY KEY,
    document_id BIGINT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    mode        TEXT NOT NULL DEFAULT '',
    position    INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_blocks_document_id ON blocks(document_id);

CREATE TABLE IF NOT EXISTS block_attributes (
    block_id BIGINT NOT NULL REFERENCES blocks(id) ON DELETE CASCADE,
    key      TEXT NOT NULL,
    value    TEXT NOT NULL,
    PRIMARY KEY (block_id, key)
);

CREATE TABLE IF NOT EXISTS responses (
    id       BIGSERIAL PRIMARY KEY,
    block_id BIGINT NOT NULL REFERENCES blocks(id) ON DELETE CASCADE,
    value    TEXT NOT NULL,
    model    TEXT NOT NULL,
    position INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_responses_block_id ON responses(block_id);

CREATE TABLE IF NOT EXISTS labels (
    id      BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name    TEXT NOT NULL,
    UNIQUE (user_id, name)
);

CREATE TABLE IF NOT EXISTS document_labels (
    document_id BIGINT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    label_id    BIGINT NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    PRIMARY KEY (document_id, label_id)
);
