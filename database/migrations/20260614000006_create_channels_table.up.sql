CREATE TABLE IF NOT EXISTS channels (
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    type       VARCHAR(255) NOT NULL UNIQUE,
    logo       JSONB,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
