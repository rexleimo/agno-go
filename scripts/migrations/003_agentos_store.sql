-- Schema for pkg/hno/store/postgres (long-term memory store, slice 29 / D2 OPT-a1).
-- Rows are addressed by the canonical namespace encoding plus key; the embedding
-- column holds write-time float32 payload scored in the application layer.
CREATE TABLE IF NOT EXISTS agno_store_items (
    namespace TEXT NOT NULL,
    key TEXT NOT NULL,
    value BYTEA NOT NULL,
    embedding BYTEA,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (namespace, key)
);

CREATE INDEX IF NOT EXISTS idx_agno_store_items_namespace
    ON agno_store_items(namespace);
