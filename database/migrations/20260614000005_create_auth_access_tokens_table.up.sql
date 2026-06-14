CREATE TABLE IF NOT EXISTS auth_access_tokens (
    id            SERIAL PRIMARY KEY,
    tokenable_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type          VARCHAR(255) NOT NULL,
    name          VARCHAR(255),
    hash          VARCHAR(255) NOT NULL,
    abilities     TEXT NOT NULL,
    created_at    TIMESTAMPTZ,
    updated_at    TIMESTAMPTZ,
    last_used_at  TIMESTAMPTZ,
    expires_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_auth_access_tokens_tokenable_id ON auth_access_tokens(tokenable_id);
