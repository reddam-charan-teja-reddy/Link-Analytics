CREATE TABLE links (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    original_url    TEXT         NOT NULL,
    hash            VARCHAR(12)  NOT NULL UNIQUE,
    title           VARCHAR(255),
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_links_user_id ON links (user_id);
CREATE INDEX idx_links_hash    ON links (hash);
