CREATE TABLE categories (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    UNIQUE(user_id, name)
);

ALTER TABLE links
    ADD COLUMN category_id UUID REFERENCES categories(id) ON DELETE SET NULL;

CREATE INDEX idx_links_category_id ON links (category_id);

CREATE TABLE groups (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    UNIQUE(user_id, name)
);

CREATE TABLE link_groups (
    group_id        UUID         NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    link_id         UUID         NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, link_id)
);

CREATE INDEX idx_link_groups_group_id ON link_groups (group_id);
CREATE INDEX idx_link_groups_link_id ON link_groups (link_id);