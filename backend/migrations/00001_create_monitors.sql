-- +goose Up

CREATE TABLE IF NOT EXISTS monitors (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    url TEXT NOT NULL UNIQUE,
    interval_seconds INTEGER NOT NULL DEFAULT 60,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_interval CHECK (interval_seconds > 0)
);

-- +goose Down

DROP TABLE IF EXISTS monitors;