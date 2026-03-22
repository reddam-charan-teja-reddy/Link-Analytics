CREATE TABLE click_events (
    id              BIGSERIAL    PRIMARY KEY,
    hash            VARCHAR(12)  NOT NULL,
    link_id         UUID         NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    source_link_id  UUID         REFERENCES source_links(id) ON DELETE SET NULL,
    ip_address      INET,
    user_agent      TEXT,
    referer         TEXT,
    browser         VARCHAR(100),
    os              VARCHAR(100),
    country         VARCHAR(100),
    city            VARCHAR(100),
    is_bot          BOOLEAN      NOT NULL DEFAULT FALSE,
    clicked_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_click_events_link_id    ON click_events (link_id);
CREATE INDEX idx_click_events_hash       ON click_events (hash);
CREATE INDEX idx_click_events_clicked_at ON click_events (clicked_at);
CREATE INDEX idx_click_events_source     ON click_events (source_link_id) WHERE source_link_id IS NOT NULL;
