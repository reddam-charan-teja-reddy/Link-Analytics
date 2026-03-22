CREATE TABLE source_links (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    link_id         UUID         NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    source_name     VARCHAR(100) NOT NULL,
    hash            VARCHAR(12)  NOT NULL UNIQUE,
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    UNIQUE(link_id, source_name)
);

CREATE INDEX idx_source_links_link_id ON source_links (link_id);
CREATE INDEX idx_source_links_hash    ON source_links (hash);
