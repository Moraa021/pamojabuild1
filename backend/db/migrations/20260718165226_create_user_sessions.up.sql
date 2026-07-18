ALTER TABLE users
    DROP COLUMN token_version;

CREATE TABLE user_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    CONSTRAINT ck_user_sessions_expiry CHECK (expires_at > created_at)
);

CREATE INDEX idx_user_sessions_active_user
    ON user_sessions (user_id, expires_at DESC)
    WHERE revoked_at IS NULL;

CREATE INDEX idx_user_sessions_expiry
    ON user_sessions (expires_at);
