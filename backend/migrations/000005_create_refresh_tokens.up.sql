-- CREATE REFRESH TOKENS TABLE

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_refresh_token_hash_length CHECK (char_length(token_hash) > 0),
    CONSTRAINT chk_refresh_token_expires_at CHECK (expires_at > created_at),
    CONSTRAINT chk_refresh_token_revoked_at CHECK (revoked_at IS NULL OR revoked_at > created_at)
);

-- Fast lookup when validating an incoming refresh token.
CREATE UNIQUE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);

-- Fast cleanup/lookup of all tokens belonging to a user
-- (used during logout-all and account deletion).
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- Used by the worker that periodically prunes expired tokens.
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);