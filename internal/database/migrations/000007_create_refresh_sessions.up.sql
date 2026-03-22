CREATE TABLE refresh_sessions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    family_id     TEXT        NOT NULL,
    token_jti     TEXT        NOT NULL UNIQUE,
    parent_jti    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMPTZ NOT NULL,
    revoked_at    TIMESTAMPTZ,
    revoked_reason TEXT
);

CREATE INDEX idx_refresh_sessions_user_family ON refresh_sessions (user_id, family_id);
CREATE INDEX idx_refresh_sessions_active_expires ON refresh_sessions (expires_at) WHERE revoked_at IS NULL;